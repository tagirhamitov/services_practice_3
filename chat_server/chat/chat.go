package chat

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/streadway/amqp"
	"github.com/tagirhamitov/soa_project/proto/mafia"
)

type Chat struct {
	chatID               uint64
	members              []ChatMember
	cancelListenMessages func()
}

func NewChat(
	chatID uint64,
	members []ChatMember,
	conn *amqp.Connection,
	mafiaClient *mafia.MafiaClient,
	errChan chan<- error,
) Chat {
	ctx, cancel := context.WithCancel(context.Background())
	chat := Chat{
		chatID:               chatID,
		members:              members,
		cancelListenMessages: cancel,
	}
	go chat.listenMessages(ctx, conn, mafiaClient, errChan)

	return chat
}

func (c *Chat) StopListeners() {
	c.cancelListenMessages()
	for _, member := range c.members {
		member.cancelListenMessages()
	}
}

func (c *Chat) listenMessages(ctx context.Context, conn *amqp.Connection, mafiaClient *mafia.MafiaClient, errChan chan<- error) {
	ch, err := conn.Channel()
	if err != nil {
		errChan <- err
		return
	}
	defer ch.Close()

	queueName := strconv.FormatUint(c.chatID, 10)
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
			resp, err := (*mafiaClient).ActivePlayers(ctx, &mafia.ActivePlayersRequest{
				SessionID: c.chatID,
			})
			if err != nil {
				errChan <- err
				return
			}

			activePlayers := make(map[string]struct{})
			for _, player := range resp.Users {
				activePlayers[player.Username] = struct{}{}
			}

			var msg Message
			err = json.Unmarshal(message.Body, &msg)
			if err != nil {
				errChan <- err
				return
			}

			if _, ok := activePlayers[msg.Sender]; !ok {
				continue
			}

			for _, member := range c.members {
				if _, ok := activePlayers[member.username]; !ok {
					continue
				}
				if member.username != msg.Sender {
					member.messageChan <- msg
				}
			}
		}
	}
}
