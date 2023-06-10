package chat

import (
	"context"
	"time"

	"github.com/streadway/amqp"
)

const repeatInterval time.Duration = 100 * time.Millisecond

func CreateConnection(ctx context.Context, rabbitmqAddress string) *amqp.Connection {
	for {
		conn, err := amqp.Dial(rabbitmqAddress)
		if err == nil {
			return conn
		}

		select {
		case <-ctx.Done():
			return nil

		case <-time.After(repeatInterval):
			continue
		}
	}
}
