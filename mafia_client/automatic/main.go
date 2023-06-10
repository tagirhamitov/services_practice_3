package automatic

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/tagirhamitov/soa_project/mafia_client/parameters"
	"github.com/tagirhamitov/soa_project/proto/mafia"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Main(params parameters.ClientParameters) error {
	log.Printf(
		"running automatic mafia client, username: %v, server: %v\n",
		params.Username,
		params.ServerAddress,
	)

	// Set up a connection with server
	conn, err := grpc.Dial(
		params.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to establish a connection with server: %w", err)
	}
	log.Println("established connection with the server")
	defer conn.Close()

	// Create GRPC client
	client := mafia.NewMafiaClient(conn)

	// Join game
	events, err := client.JoinGame(context.Background(), &mafia.User{
		Username: params.Username,
	})
	if err != nil {
		return fmt.Errorf("failed to join the game: %w", err)
	}

	// Player information.
	var sessionID uint64
	var role mafia.Role
	var users []string
	var knownMafia *string = nil
	alive := true

	dayTurn := func() error {
		// If mafia is known, kill him.
		if knownMafia != nil {
			log.Printf("Знаю, что мафия - %v, пытаюсь убить\n", *knownMafia)
			_, err := client.Kill(context.Background(), &mafia.UserAction{
				User: &mafia.PlayingUser{
					SessionID: sessionID,
					Username:  params.Username,
				},
				TargetUser: &mafia.User{
					Username: *knownMafia,
				},
			})
			if err != nil {
				return err
			}
		} else if rand.Int()%2 == 0 {
			// Randomly try to kill someone.
			targetIdx := rand.Int31n(int32(len(users)))
			target := users[targetIdx]
			log.Printf("Я не знаю, кто мафия, но пытаюсь убить %v\n", target)
			_, err := client.Kill(context.Background(), &mafia.UserAction{
				User: &mafia.PlayingUser{
					SessionID: sessionID,
					Username:  params.Username,
				},
				TargetUser: &mafia.User{
					Username: target,
				},
			})
			if err != nil {
				return err
			}
		}

		log.Println("Заканчиваю день")
		_, err := client.FinishDay(context.Background(), &mafia.PlayingUser{
			SessionID: sessionID,
			Username:  params.Username,
		})
		return err
	}

	nightTurn := func() error {
		switch role {
		case mafia.Role_COMISSAR:
			// Comissar tries to find the mafia.
			targetIdx := rand.Int31n(int32(len(users)))
			target := users[targetIdx]
			log.Printf("Я комиссар, пытаюсь проверить %v\n", target)
			resp, err := client.CheckMafia(context.Background(), &mafia.UserAction{
				User: &mafia.PlayingUser{
					SessionID: sessionID,
					Username:  params.Username,
				},
				TargetUser: &mafia.User{
					Username: target,
				},
			})
			if err != nil {
				return err
			}
			if resp.Success {
				knownMafia = &target
			}
		case mafia.Role_MAFIA:
			// Mafia randomly kills someone.
			targetIdx := rand.Int31n(int32(len(users)))
			target := users[targetIdx]
			log.Printf("Я мафия, убиваю %v\n", target)
			_, err := client.Kill(context.Background(), &mafia.UserAction{
				User: &mafia.PlayingUser{
					SessionID: sessionID,
					Username:  params.Username,
				},
				TargetUser: &mafia.User{
					Username: target,
				},
			})
			if err != nil {
				return err
			}
		default:
			log.Println("Я не могу ходить ночью")
		}
		return nil
	}

	for {
		event, err := events.Recv()
		if err != nil {
			return fmt.Errorf("failed to receive an event: %w", err)
		}

		switch event := event.EventType.(type) {
		case *mafia.Event_SessionStarted:
			// Fill in the player information.
			sessionID = event.SessionStarted.SessionID
			role = event.SessionStarted.Role
			for _, user := range event.SessionStarted.Users {
				if user.Username != params.Username {
					users = append(users, user.Username)
				}
			}
			dayTurn()
		case *mafia.Event_DayFinished:
			if alive {
				nightTurn()
			}
		case *mafia.Event_NightFinished:
			if alive {
				dayTurn()
			}
		case *mafia.Event_UserKilled:
			killed := event.UserKilled.Username

			// This bot was killed.
			if killed == params.Username {
				log.Println("Меня убили")
				alive = false
				continue
			}

			// Remove killed user.
			var newUsers []string
			for _, user := range users {
				if user != killed {
					newUsers = append(newUsers, user)
				}
			}
			users = newUsers
		case *mafia.Event_PublishedMafia:
			knownMafia = &event.PublishedMafia.TargetUser.Username
			log.Printf("Я узнал, что %v - мафия\n", *knownMafia)
		case *mafia.Event_GameFinished:
			return nil
		}
	}
}
