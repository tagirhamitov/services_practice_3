package chat

type Message struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
}

type Event struct {
	ChatID  uint64   `json:"chat_id"`
	Create  bool     `json:"create"` // true for creation, false for deletion
	Members []string `json:"members"`
}
