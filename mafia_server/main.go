package main

import (
	"context"
	"log"
	"net"

	"github.com/tagirhamitov/soa_project/mafia_server/game"
	"github.com/tagirhamitov/soa_project/proto/mafia"
	"google.golang.org/grpc"
)

const rabbitmqAddress = "amqp://guest:guest@rabbitmq:5672"

var lobby = game.NewLobby(rabbitmqAddress)

type server struct {
	mafia.UnimplementedMafiaServer
}

// JoinGame implements mafia.MafiaServer
func (*server) JoinGame(request *mafia.User, stream mafia.Mafia_JoinGameServer) error {
	// Channels for event listener
	eventChan := make(chan *mafia.Event)
	errChan := make(chan error)

	// Run event listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go streamEvents(ctx, request.Username, eventChan, stream, errChan)

	// Add user to the game
	err := lobby.AddUser(request.Username, eventChan)
	if err != nil {
		return err
	}

	err = <-errChan
	return err
}

// FinishDay implements mafia.MafiaServer
func (*server) FinishDay(_ context.Context, user *mafia.PlayingUser) (*mafia.Empty, error) {
	session := lobby.GetSession(user.SessionID)
	session.FinishDay(user.Username)
	return &mafia.Empty{}, nil
}

// Kill implements mafia.MafiaServer
func (*server) Kill(_ context.Context, action *mafia.UserAction) (*mafia.Empty, error) {
	session := lobby.GetSession(action.User.SessionID)
	session.Kill(action.User.Username, action.TargetUser.Username)
	return &mafia.Empty{}, nil
}

// CheckMafia implements mafia.MafiaServer
func (*server) CheckMafia(_ context.Context, action *mafia.UserAction) (*mafia.CheckMafiaResponse, error) {
	session := lobby.GetSession(action.User.SessionID)
	success := session.CheckMafia(action.User.Username, action.TargetUser.Username)
	return &mafia.CheckMafiaResponse{
		Success: success,
	}, nil
}

// PublishMafia implements mafia.MafiaServer
func (*server) PublishMafia(_ context.Context, action *mafia.UserAction) (*mafia.Empty, error) {
	session := lobby.GetSession(action.User.SessionID)
	session.PublishMafia(action.User.Username, action.TargetUser.Username)
	return &mafia.Empty{}, nil
}

// ActivePlayers implements mafia.MafiaServer.
func (*server) ActivePlayers(ctx context.Context, req *mafia.ActivePlayersRequest) (*mafia.ActivePlayersResponse, error) {
	session := lobby.GetSession(req.SessionID)
	users := session.GetActiveUsers()
	return &mafia.ActivePlayersResponse{
		Users: users,
	}, nil
}

func streamEvents(
	ctx context.Context,
	username string,
	eventsChan <-chan *mafia.Event,
	stream mafia.Mafia_JoinGameServer,
	errChan chan<- error,
) {
	for {
		select {
		case event := <-eventsChan:
			err := stream.Send(event)
			if err != nil {
				errChan <- err
				return
			}

		case <-stream.Context().Done():
			// User was disconnected
			lobby.RemoveUser(username)
			return

		case <-ctx.Done():
			return
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	mafia.RegisterMafiaServer(srv, &server{})

	log.Fatalln(srv.Serve(lis))
}
