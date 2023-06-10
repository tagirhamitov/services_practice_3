package chat

import (
	"context"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type ChatMember struct {
	username             string
	messageChan          chan Message
	cancelListenMessages func()
}

func NewChatMember(username string, conn *amqp.Connection, errChan chan<- error) ChatMember {
	messageChan := make(chan Message)

	ctx, cancel := context.WithCancel(context.Background())
	member := ChatMember{
		username:             username,
		messageChan:          messageChan,
		cancelListenMessages: cancel,
	}
	go member.listenMessages(ctx, conn, errChan)

	return member
}

func (c *ChatMember) listenMessages(ctx context.Context, conn *amqp.Connection, errChan chan<- error) {
	ch, err := conn.Channel()
	if err != nil {
		errChan <- err
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		c.username,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		errChan <- err
		return
	}

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.messageChan:
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				errChan <- err
				return
			}

			err = ch.Publish(
				"",
				q.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        msgBytes,
				},
			)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
