package chatting

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
	"github.com/tagirhamitov/soa_project/chat_server/chat"
)

func ListenEvents(
	conn *amqp.Connection,
	eventsChan <-chan chat.Event,
) {
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	q, err := ch.QueueDeclare(
		"events",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	for event := range eventsChan {
		eventBytes, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return
		}

		err = ch.Publish(
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        eventBytes,
			},
		)
		if err != nil {
			log.Printf("Error: %v\n", err)
			return
		}
	}
}
