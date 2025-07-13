package main

import "time"

// Message representa un mensaje de chat (texto o imagen)
type Message struct {
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "system", "join", "leave", "image"

	// Campos específicos para imágenes
	ImageData string `json:"imageData,omitempty"`
	ImageName string `json:"imageName,omitempty"`
	ImageSize int64  `json:"imageSize,omitempty"`
	MimeType  string `json:"mimeType,omitempty"`
}

// MessageType define los tipos de mensajes
const (
	MessageTypeMessage = "message"
	MessageTypeSystem  = "system"
	MessageTypeJoin    = "join"
	MessageTypeLeave   = "leave"
	MessageTypeImage   = "image"
)

// NewMessage crea un nuevo mensaje de chat de texto
func NewMessage(username, content string) *Message {
	return &Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeMessage,
	}
}

// NewImageMessage crea un nuevo mensaje de imagen
func NewImageMessage(username, content, imageData, imageName string, imageSize int64, mimeType string) *Message {
	return &Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeImage,
		ImageData: imageData,
		ImageName: imageName,
		ImageSize: imageSize,
		MimeType:  mimeType,
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