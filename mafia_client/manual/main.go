package manual

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tagirhamitov/soa_project/chat_server/chat"
	"github.com/tagirhamitov/soa_project/mafia_client/chatting"
	"github.com/tagirhamitov/soa_project/mafia_client/parameters"
	"github.com/tagirhamitov/soa_project/proto/mafia"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Main(params parameters.ClientParameters, rabbitmqAddress string) error {
	// Set up a connection with server
	conn, err := grpc.Dial(
		params.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to establish a connection with server: %w", err)
	}
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
	fmt.Println("Выполнен вход в игру")

	// Set up event listener.
	sessionIDChan := make(chan uint64)
	gameFinishedChan := make(chan struct{})
	go listenEvents(events, sessionIDChan, gameFinishedChan)

	// Wait for the start of session.
	sessionID := <-sessionIDChan

	// Start chat message listener.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	amqpConn := chat.CreateConnection(ctx, rabbitmqAddress)
	defer amqpConn.Close()

	messagesChan := make(chan chat.Message)
	errChan := make(chan error)
	go chatting.ListenLocalMessages(ctx, amqpConn, sessionID, messagesChan, errChan)
	go chatting.ListenRemoteMessages(ctx, amqpConn, params.Username, errChan)

	// Set up user input reader.
	userInputChan := make(chan string)
	go getUserInput(userInputChan)

	for {
		select {
		case err = <-errChan:
			return err

		case <-gameFinishedChan:
			return nil

		case input := <-userInputChan:
			items := strings.Split(input, " ")
			switch items[0] {
			case "finish_day":
				_, err := client.FinishDay(context.Background(), &mafia.PlayingUser{
					SessionID: sessionID,
					Username:  params.Username,
				})
				if err != nil {
					return fmt.Errorf("failed to run command: %w", err)
				}

			case "kill":
				if len(items) < 2 {
					fmt.Println("Unknown command")
					continue
				}
				_, err := client.Kill(context.Background(), &mafia.UserAction{
					User: &mafia.PlayingUser{
						SessionID: sessionID,
						Username:  params.Username,
					},
					TargetUser: &mafia.User{
						Username: items[1],
					},
				})
				if err != nil {
					return fmt.Errorf("failed to run command: %w", err)
				}

			case "check_mafia":
				if len(items) < 2 {
					fmt.Println("Unknown command")
					continue
				}
				resp, err := client.CheckMafia(context.Background(), &mafia.UserAction{
					User: &mafia.PlayingUser{
						SessionID: sessionID,
						Username:  params.Username,
					},
					TargetUser: &mafia.User{
						Username: items[1],
					},
				})
				if err != nil {
					return fmt.Errorf("failed to run command: %w", err)
				}
				if resp.Success {
					fmt.Printf("Игрок %v - мафия\n", items[1])
				} else {
					fmt.Printf("Игрок %v - не мафия\n", items[1])
				}

			case "publish_mafia":
				if len(items) < 2 {
					fmt.Println("Unknown command")
					continue
				}
				_, err := client.PublishMafia(context.Background(), &mafia.UserAction{
					User: &mafia.PlayingUser{
						SessionID: sessionID,
						Username:  params.Username,
					},
					TargetUser: &mafia.User{
						Username: items[1],
					},
				})
				if err != nil {
					return fmt.Errorf("failed to run command: %w", err)
				}

			case "msg":
				text := strings.TrimPrefix(input, "msg ")
				messagesChan <- chat.Message{
					Sender: params.Username,
					Text:   text,
				}

			default:
				fmt.Println("Unknown command")
			}
		}
	}
}

func getUserInput(outChan chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		outChan <- scanner.Text()
	}
}
