package chat

import (
	"encoding/json"
	"sync"

	"github.com/streadway/amqp"
	"github.com/tagirhamitov/soa_project/proto/mafia"
)

type ChatServer struct {
	chats       map[uint64]*Chat
	conn        *amqp.Connection
	mafiaClient *mafia.MafiaClient
	mutex       sync.Mutex
}

func NewChatServer(conn *amqp.Connection, mafiaClient *mafia.MafiaClient) ChatServer {
	return ChatServer{
		chats:       make(map[uint64]*Chat),
		conn:        conn,
		mafiaClient: mafiaClient,
	}
}

func (c *ChatServer) CreateChat(chatID uint64, members []ChatMember, errChan chan<- error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	chat := NewChat(chatID, members, c.conn, c.mafiaClient, errChan)
	c.chats[chatID] = &chat
}

func (c *ChatServer) RemoveChat(chatID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	chat, ok := c.chats[chatID]
	if !ok {
		return
	}

	chat.StopListeners()
	delete(c.chats, chatID)
}

func (c *ChatServer) ListenEvents(errChan chan<- error) {
	ch, err := c.conn.Channel()
	if err != nil {
		errChan <- err
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"events",
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

	for message := range messages {
		var event Event
		err = json.Unmarshal(message.Body, &event)
		if err != nil {
			errChan <- err
			return
		}

		if event.Create {
			var members []ChatMember
			for _, username := range event.Members {
				members = append(members, NewChatMember(username, c.conn, errChan))
			}
			c.CreateChat(event.ChatID, members, errChan)
		} else {
			c.RemoveChat(event.ChatID)
		}
	}
}
