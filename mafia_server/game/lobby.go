package game

import (
	"context"
	"fmt"
	"sync"

	"github.com/tagirhamitov/soa_project/chat_server/chat"
	"github.com/tagirhamitov/soa_project/mafia_server/chatting"
	"github.com/tagirhamitov/soa_project/proto/mafia"
)

type Lobby struct {
	users         map[string]User
	sessions      map[uint64]*Session
	nextSessionID uint64
	eventsChan    chan<- chat.Event
	mutex         sync.Mutex
}

func NewLobby(rabbitmqAddress string) Lobby {
	conn := chat.CreateConnection(context.Background(), rabbitmqAddress)

	eventsChan := make(chan chat.Event)
	go chatting.ListenEvents(conn, eventsChan)
	return Lobby{
		users:         make(map[string]User),
		sessions:      make(map[uint64]*Session),
		nextSessionID: 0,
		eventsChan:    eventsChan,
	}
}

func (l *Lobby) AddUser(username string, eventChan chan<- *mafia.Event) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Check username uniqueness.
	if _, ok := l.users[username]; ok {
		return fmt.Errorf("username %v is taken", username)
	}

	// Add user to the lobby.
	user := NewUser(username, eventChan)
	for _, otherUser := range l.users {
		otherUser.eventChan <- &mafia.Event{
			EventType: &mafia.Event_UserJoin{
				UserJoin: &mafia.User{
					Username: username,
				},
			},
		}
		user.eventChan <- &mafia.Event{
			EventType: &mafia.Event_UserJoin{
				UserJoin: &mafia.User{
					Username: otherUser.username,
				},
			},
		}
	}
	l.users[username] = user

	if len(l.users) == 4 {
		l.startSession()
	}

	return nil
}

func (l *Lobby) RemoveUser(username string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if _, ok := l.users[username]; !ok {
		return
	}

	delete(l.users, username)

	for _, user := range l.users {
		user.eventChan <- &mafia.Event{
			EventType: &mafia.Event_UserLeave{
				UserLeave: &mafia.User{
					Username: username,
				},
			},
		}
	}
}

func (l *Lobby) GetSession(sessionID uint64) *Session {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.sessions[sessionID]
}

func (l *Lobby) startSession() {
	// Move users from lobby.
	var users []User
	for _, user := range l.users {
		users = append(users, user)
	}
	l.users = make(map[string]User)

	// Distribute roles.
	putRoles(users)

	// Add new session to the lobby.
	sessionID := l.nextSessionID
	l.nextSessionID += 1
	l.sessions[sessionID] = NewSession(sessionID, l, users)

	// Convert users to protobuf format.
	var pbUsers []*mafia.User
	var usernames []string
	for _, user := range users {
		pbUsers = append(pbUsers, &mafia.User{
			Username: user.username,
		})
		usernames = append(usernames, user.username)
	}

	// Notify users about started session.
	for _, user := range users {
		user.eventChan <- &mafia.Event{
			EventType: &mafia.Event_SessionStarted{
				SessionStarted: &mafia.SessionStarted{
					SessionID: sessionID,
					Role:      user.role,
					Users:     pbUsers,
				},
			},
		}
	}

	l.eventsChan <- chat.Event{
		ChatID:  sessionID,
		Create:  true,
		Members: usernames,
	}
}
