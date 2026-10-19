package domain

type Chat struct {
	ID          string   `json:"id"`
	Topic       string   `json:"topic,omitempty"`
	Type        string   `json:"type,omitempty"`
	Members     []Person `json:"members,omitempty"`
	LastMessage string   `json:"last_message,omitempty"`
	// Hidden is set when the signed-in user has hidden the chat; teams list skips those by default.
	Hidden bool `json:"hidden,omitempty"`
}

type ChatMessage struct {
	ID          string       `json:"id"`
	ChatID      string       `json:"chat_id"`
	From        string       `json:"from,omitempty"`
	Created     string       `json:"created,omitempty"`
	Text        string       `json:"text,omitempty"`
	System      bool         `json:"system,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type ChatPage struct {
	Limit    int     `json:"limit"`
	Count    int     `json:"count"`
	NextPage *string `json:"next_page"`
	Items    []Chat  `json:"items"`
}

type ChatMessagePage struct {
	Limit    int           `json:"limit"`
	Count    int           `json:"count"`
	NextPage *string       `json:"next_page"`
	Items    []ChatMessage `json:"items"`
}
