package game

import (
	"fmt"
	"sync"

	"github.com/tagirhamitov/soa_project/chat_server/chat"
	"github.com/tagirhamitov/soa_project/proto/mafia"
)

type Session struct {
	sessionID       uint64
	lobby           *Lobby
	users           []User
	dayTime         DayTime
	usersReady      []bool
	firstDay        bool
	publishedMafia  bool
	voted           []bool
	votes           []int
	killedDuringDay bool
	finished        bool
	mutex           sync.Mutex
}

func NewSession(sessionID uint64, lobby *Lobby, users []User) *Session {
	return &Session{
		sessionID:       sessionID,
		lobby:           lobby,
		users:           users,
		dayTime:         Day,
		usersReady:      make([]bool, len(users)),
		firstDay:        true,
		publishedMafia:  false,
		voted:           make([]bool, len(users)),
		votes:           make([]int, len(users)),
		killedDuringDay: false,
		finished:        false,
	}
}

func (s *Session) FinishDay(username string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.finished {
		return
	}

	userIdx := s.getUserIdx(username)
	if userIdx == -1 {
		return
	}
	if s.usersReady[userIdx] {
		return
	}
	if !s.users[userIdx].alive {
		return
	}
	if s.dayTime != Day {
		return
	}

	s.usersReady[userIdx] = true

	if !s.areAllReady() {
		return
	}

	s.startDayTime(Night)
}

func (s *Session) Kill(username string, targetUsername string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.finished {
		return
	}

	userIdx := s.getUserIdx(username)
	targetUserIdx := s.getUserIdx(targetUsername)
	if userIdx == -1 || targetUserIdx == -1 {
		return
	}
	if s.usersReady[userIdx] {
		return
	}
	if !s.users[userIdx].alive || !s.users[targetUserIdx].alive {
		return
	}

	if s.dayTime == Day {
		if s.killedDuringDay {
			return
		}
		if s.voted[userIdx] {
			return
		}
		s.voted[userIdx] = true
		s.votes[targetUserIdx]++
		s.broadcast(&mafia.Event{
			EventType: &mafia.Event_Voted{
				Voted: &mafia.Voted{
					User: &mafia.User{
						Username: username,
					},
					TargetUser: &mafia.User{
						Username: targetUsername,
					},
				},
			},
		})
		if s.votes[targetUserIdx] >= s.aliveUsers()/2+1 {
			s.killedDuringDay = true
			s.users[targetUserIdx].alive = false
			s.broadcast(&mafia.Event{
				EventType: &mafia.Event_UserKilled{
					UserKilled: &mafia.User{
						Username: targetUsername,
					},
				},
			})
			s.checkFinished()
			if s.finished {
				return
			}
		}
	} else {
		if s.users[userIdx].role != mafia.Role_MAFIA {
			return
		}
		s.usersReady[userIdx] = true
		s.users[targetUserIdx].alive = false
		s.broadcast(&mafia.Event{
			EventType: &mafia.Event_UserKilled{
				UserKilled: &mafia.User{
					Username: targetUsername,
				},
			},
		})
		s.checkFinished()
		if s.finished {
			return
		}

		if !s.areAllReady() {
			return
		}

		s.startDayTime(Day)
	}
}

func (s *Session) CheckMafia(username string, targetUsername string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.finished {
		return false
	}

	userIdx := s.getUserIdx(username)
	targetUserIdx := s.getUserIdx(targetUsername)
	if userIdx == -1 || targetUserIdx == -1 {
		return false
	}
	if s.usersReady[userIdx] {
		return false
	}
	if !s.users[userIdx].alive {
		return false
	}
	if s.dayTime != Night {
		return false
	}
	if s.users[userIdx].role != mafia.Role_COMISSAR {
		return false
	}

	success := s.users[targetUserIdx].role == mafia.Role_MAFIA
	s.usersReady[userIdx] = true

	if !s.areAllReady() {
		return success
	}

	s.startDayTime(Day)
	return success
}

func (s *Session) PublishMafia(username string, targetUsername string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.finished {
		return
	}

	userIdx := s.getUserIdx(username)
	targetUserIdx := s.getUserIdx(targetUsername)
	if userIdx == -1 || targetUserIdx == -1 {
		return
	}
	if s.usersReady[userIdx] {
		return
	}
	if !s.users[userIdx].alive {
		return
	}
	if s.dayTime != Day {
		return
	}
	if s.firstDay {
		return
	}
	if s.users[userIdx].role != mafia.Role_COMISSAR {
		return
	}
	if s.publishedMafia {
		return
	}

	s.publishedMafia = true
	s.broadcast(&mafia.Event{
		EventType: &mafia.Event_PublishedMafia{
			PublishedMafia: &mafia.PublishMafia{
				User: &mafia.User{
					Username: username,
				},
				TargetUser: &mafia.User{
					Username: targetUsername,
				},
			},
		},
	})
}

func (s *Session) GetActiveUsers() []*mafia.User {
	var result []*mafia.User
	for _, user := range s.users {
		if !user.alive {
			continue
		}
		if s.dayTime == Day || user.role == mafia.Role_MAFIA {
			result = append(result, &mafia.User{
				Username: user.username,
			})
		}
	}
	return result
}

func (s *Session) getUserIdx(username string) int {
	for i := range s.users {
		if s.users[i].username == username {
			return i
		}
	}
	return -1
}

func (s *Session) areAllReady() bool {
	for i, user := range s.users {
		if !user.alive {
			continue
		}
		if !s.usersReady[i] {
			fmt.Printf("Waiting for %v\n", user.username)
			return false
		}
	}
	return true
}

func (s *Session) startDayTime(dayTime DayTime) {
	if dayTime == Day {
		for i, user := range s.users {
			s.usersReady[i] = false
			s.voted[i] = false
			s.votes[i] = 0
			user.eventChan <- &mafia.Event{
				EventType: &mafia.Event_NightFinished{
					NightFinished: &mafia.Empty{},
				},
			}
		}
		s.publishedMafia = false
		s.killedDuringDay = false
		s.dayTime = Day
	} else {
		for i, user := range s.users {
			if user.role == mafia.Role_USUAL {
				s.usersReady[i] = true
			} else {
				s.usersReady[i] = false
			}
			user.eventChan <- &mafia.Event{
				EventType: &mafia.Event_DayFinished{
					DayFinished: &mafia.Empty{},
				},
			}
		}
		s.firstDay = false
		s.dayTime = Night
	}
}

func (s *Session) aliveUsers() int {
	result := 0
	for _, user := range s.users {
		if user.alive {
			result++
		}
	}
	return result
}

func (s *Session) checkFinished() {
	peacefulCount := 0
	mafiaCount := 0
	for _, user := range s.users {
		if !user.alive {
			continue
		}
		if user.role == mafia.Role_MAFIA {
			mafiaCount++
		} else {
			peacefulCount++
		}
	}

	if mafiaCount == 0 {
		s.finished = true
		s.broadcast(&mafia.Event{
			EventType: &mafia.Event_GameFinished{
				GameFinished: &mafia.GameFinished{
					MafiaWin: false,
				},
			},
		})
		s.lobby.eventsChan <- chat.Event{
			ChatID: s.sessionID,
			Create: false,
		}
	} else if mafiaCount == peacefulCount {
		s.finished = true
		s.broadcast(&mafia.Event{
			EventType: &mafia.Event_GameFinished{
				GameFinished: &mafia.GameFinished{
					MafiaWin: true,
				},
			},
		})
		s.lobby.eventsChan <- chat.Event{
			ChatID: s.sessionID,
			Create: false,
		}
	}
}

func (s *Session) broadcast(event *mafia.Event) {
	for _, user := range s.users {
		user.eventChan <- event
	}
}
