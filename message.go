package main

import "time"

// ImageData representa los datos de una imagen
type ImageData struct {
	Data string `json:"data"` // Base64 data URL
	Name string `json:"name"` // Nombre del archivo
	Type string `json:"type"` // MIME type
	Size int64  `json:"size"` // Tamaño en bytes
}

// Message representa un mensaje de chat
type Message struct {
	Username  string     `json:"username"`        // Quién envió
	Content   string     `json:"content"`         // Texto del mensaje
	Timestamp time.Time  `json:"timestamp"`       // Cuándo se envió
	Type      string     `json:"type"`            // Tipo: message/system/join/leave
	Image     *ImageData `json:"image,omitempty"` // Datos de imagen opcionales
	HasImage  bool       `json:"hasImage"`        // Indica si el mensaje tiene imagen
}

// MessageType define los tipos de mensajes
const (
	MessageTypeMessage = "message"
	MessageTypeSystem  = "system"
	MessageTypeJoin    = "join"
	MessageTypeLeave   = "leave"
)

//Funciones para crear mensajes:

// NewMessage crea un nuevo mensaje de chat
func NewMessage(username, content string) *Message {
	return &Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeMessage,
		HasImage:  false,
	}
}

// NewMessageWithImage crea un nuevo mensaje de chat con imagen
func NewMessageWithImage(username, content string, imageData *ImageData) *Message {
	return &Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeMessage,
		Image:     imageData,
		HasImage:  true,
	}
}

// NewSystemMessage crea un nuevo mensaje del sistema
func NewSystemMessage(content string) *Message {
	return &Message{
		Username:  "Sistema",
		Content:   content,
		Timestamp: time.Now(),
		Type:      MessageTypeSystem,
		HasImage:  false,
	}
}
