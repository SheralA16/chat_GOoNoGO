package main

import "time"

// Message representa un mensaje de chat
type Message struct {
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "system", "join", "leave"
}

// MessageType define los tipos de mensajes
const (
	MessageTypeMessage = "message"
	MessageTypeSystem  = "system"
	MessageTypeJoin    = "join"
	MessageTypeLeave   = "leave"
)

// NewMessage crea un nuevo mensaje de chat
func NewMessage(username, content string) *Message {
	return &Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeMessage,
	}
}

// NewSystemMessage crea un nuevo mensaje del sistema
func NewSystemMessage(content string) *Message {
	return &Message{
		Username:  "Sistema",
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeSystem,
	}
}
