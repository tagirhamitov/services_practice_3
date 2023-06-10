package chatting

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/streadway/amqp"
	"github.com/tagirhamitov/soa_project/chat_server/chat"
)

func ListenLocalMessages(
	ctx context.Context,
	conn *amqp.Connection,
	sessionID uint64,
	messagesChan <-chan chat.Message,
	errChan chan<- error,
) {
	ch, err := conn.Channel()
	if err != nil {
		errChan <- err
		return
	}
	defer ch.Close()

	queueName := strconv.FormatUint(sessionID, 10)
	q, err := ch.QueueDeclare(
		queueName,
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

		case msg := <-messagesChan:
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
				errChan <- err
				return
			}
		}
	}
}

func ListenRemoteMessages(
	ctx context.Context,
	conn *amqp.Connection,
	username string,
	errChan chan<- error,
) {
	ch, err := conn.Channel()
	if err != nil {
		errChan <- err
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		username,
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

	messages, err := ch.Consume(
		q.Name,
		"",
		true,
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

		case message := <-messages:
			var msg chat.Message
			err = json.Unmarshal(message.Body, &msg)
			if err != nil {
				errChan <- err
				return
			}

			fmt.Printf("%v> %v\n", msg.Sender, msg.Text)
		}
	}
}
