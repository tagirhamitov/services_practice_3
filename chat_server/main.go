package main

import (
	"context"
	"log"

	"github.com/tagirhamitov/soa_project/chat_server/chat"
	"github.com/tagirhamitov/soa_project/proto/mafia"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const rabbitmqAddress string = "amqp://guest:guest@rabbitmq:5672"

func main() {
	amqpConn := chat.CreateConnection(context.Background(), rabbitmqAddress)
	defer amqpConn.Close()

	grpcConn, err := grpc.Dial(
		"mafia_server:9000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer grpcConn.Close()

	client := mafia.NewMafiaClient(grpcConn)

	chatServer := chat.NewChatServer(amqpConn, &client)

	errChan := make(chan error)
	go chatServer.ListenEvents(errChan)

	err = <-errChan
	log.Fatal(err)
}
