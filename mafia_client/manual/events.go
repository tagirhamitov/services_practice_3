package manual

import (
	"fmt"
	"log"

	"github.com/tagirhamitov/soa_project/proto/mafia"
)

func listenEvents(
	events mafia.Mafia_JoinGameClient,
	sessionIDChan chan<- uint64,
	gameFinishedChan chan<- struct{},
) {
	var sessionID *uint64 = nil
	for sessionID == nil {
		event, err := events.Recv()
		if err != nil {
			log.Fatal(err)
		}

		switch event := event.EventType.(type) {

		case *mafia.Event_UserJoin:
			fmt.Printf("%v вошел в игру.\n", event.UserJoin.Username)

		case *mafia.Event_UserLeave:
			fmt.Printf("%v покинул игру.\n", event.UserLeave.Username)

		case *mafia.Event_SessionStarted:
			fmt.Println("Началась игровая сессия с игроками:")
			for _, user := range event.SessionStarted.Users {
				fmt.Printf(" - %v\n", user.Username)
			}

			var role string
			switch event.SessionStarted.Role {

			case *mafia.Role_USUAL.Enum():
				role = "мирный житель"

			case *mafia.Role_MAFIA.Enum():
				role = "мафия"

			case *mafia.Role_COMISSAR.Enum():
				role = "комиссар"

			}
			fmt.Printf("Ваша роль: %v\n", role)
			sessionIDChan <- event.SessionStarted.SessionID
			fmt.Println("Начинается день.")

		case *mafia.Event_DayFinished:
			fmt.Println("День закончен. Начинается ночь.")

		case *mafia.Event_NightFinished:
			fmt.Println("Ночь закончена. Начинается день.")

		case *mafia.Event_UserKilled:
			fmt.Printf("Игрок %v убит.\n", event.UserKilled.Username)

		case *mafia.Event_PublishedMafia:
			fmt.Printf(
				"Комиссар %v объявляет, что игрок %v - мафия.\n",
				event.PublishedMafia.User.Username,
				event.PublishedMafia.TargetUser.Username,
			)

		case *mafia.Event_GameFinished:
			if event.GameFinished.MafiaWin {
				fmt.Println("Игра окончена. Выиграла мафия.")
			} else {
				fmt.Println("Игра окончена. Выиграли мирные жители.")
			}
			gameFinishedChan <- struct{}{}
			return

		case *mafia.Event_Voted:
			fmt.Printf(
				"%v проголосовал за убийство %v.\n",
				event.Voted.User.Username,
				event.Voted.TargetUser.Username,
			)
		}

	}
}
