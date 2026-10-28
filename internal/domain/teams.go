package domain

type Chat struct {
	ID          string   `json:"id"`
	Topic       string   `json:"topic,omitempty"`
	Type        string   `json:"type,omitempty"`
	Members     []Person `json:"members,omitempty"`
	LastMessage string   `json:"last_message,omitempty"`
}

type ChatMessage struct {
	ID                string       `json:"id"`
	ChatID            string       `json:"chat_id"`
	From              string       `json:"from,omitempty"`
	FromUserID        string       `json:"from_user_id,omitempty"`
	FromApplicationID string       `json:"from_application_id,omitempty"`
	SenderType        string       `json:"sender_type,omitempty"`
	Created           string       `json:"created,omitempty"`
	Text              string       `json:"text,omitempty"`
	System            bool         `json:"system,omitempty"`
	Attachments       []Attachment `json:"attachments,omitempty"`
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
