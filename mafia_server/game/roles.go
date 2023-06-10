package game

import (
	"math/rand"

	"github.com/tagirhamitov/soa_project/proto/mafia"
)

func putRoles(users []User) {
	p := rand.Perm(len(users))
	for i := range users {
		if i == p[0] {
			users[i].role = mafia.Role_COMISSAR
		} else if i == p[1] {
			users[i].role = mafia.Role_MAFIA
		} else {
			users[i].role = mafia.Role_USUAL
		}
	}
}
