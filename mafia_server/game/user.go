package game

import (
	"github.com/tagirhamitov/soa_project/proto/mafia"
)

type User struct {
	username  string
	eventChan chan<- *mafia.Event
	role      mafia.Role
	alive     bool
}

func NewUser(username string, eventChan chan<- *mafia.Event) User {
	return User{
		username:  username,
		eventChan: eventChan,
		alive:     true,
	}
}
