package main

import (
	"encoding/base64"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

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

// Configuración para imágenes
const (
	MaxImageSize = 5 * 1024 * 1024 // 5MB máximo
)

// Tipos MIME permitidos para imágenes
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

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

// ValidateImage valida que la imagen sea válida para el chat
func ValidateImage(imageData, imageName string, imageSize int64) error {
	// Verificar tamaño
	if imageSize > MaxImageSize {
		return fmt.Errorf("la imagen es demasiado grande (máximo %d MB)", MaxImageSize/(1024*1024))
	}

	if imageSize == 0 {
		return fmt.Errorf("la imagen está vacía")
	}

	// Verificar tipo MIME
	mimeType := mime.TypeByExtension(filepath.Ext(imageName))
	if mimeType == "" {
		mimeType = detectMimeTypeFromBase64(imageData)
	}

	if !allowedImageTypes[mimeType] {
		return fmt.Errorf("tipo de imagen no soportado: %s (solo se permiten JPEG, PNG, GIF, WebP)", mimeType)
	}

	// Verificar que los datos base64 sean válidos
	if !isValidBase64(imageData) {
		return fmt.Errorf("datos de imagen inválidos")
	}

	return nil
}

// detectMimeTypeFromBase64 intenta detectar el tipo MIME desde los datos base64
func detectMimeTypeFromBase64(data string) string {
	// Remover prefijo data URL si existe
	if strings.Contains(data, ",") {
		parts := strings.Split(data, ",")
		if len(parts) > 1 {
			data = parts[1]
		}
	}

	// Decodificar los primeros bytes para detectar el tipo
	decoded, err := base64.StdEncoding.DecodeString(data[:min(32, len(data))])
	if err != nil {
		return ""
	}

	// Detectar tipo basado en magic numbers
	if len(decoded) >= 4 {
		// JPEG
		if decoded[0] == 0xFF && decoded[1] == 0xD8 && decoded[2] == 0xFF {
			return "image/jpeg"
		}
		// PNG
		if decoded[0] == 0x89 && decoded[1] == 0x50 && decoded[2] == 0x4E && decoded[3] == 0x47 {
			return "image/png"
		}
		// GIF
		if string(decoded[:3]) == "GIF" {
			return "image/gif"
		}
		// WebP
		if string(decoded[:4]) == "RIFF" && len(decoded) >= 12 && string(decoded[8:12]) == "WEBP" {
			return "image/webp"
		}
	}

	return ""
}

// isValidBase64 verifica si una cadena es base64 válido
func isValidBase64(s string) bool {
	// Remover prefijo data URL si existe
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		if len(parts) > 1 {
			s = parts[1]
		}
	}

	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// min devuelve el menor de dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FormatImageSize formatea el tamaño de la imagen para mostrar
func FormatImageSize(bytes int64) string {
	const unit = 1024
	if bytes == 0 {
		return "0 Bytes"
	}
	
	sizes := []string{"Bytes", "KB", "MB", "GB"}
	i := 0
	size := float64(bytes)
	
	for size >= unit && i < len(sizes)-1 {
		size /= unit
		i++
	}
	
	// Si es un número entero, no mostrar decimales
	if size == float64(int(size)) {
		return fmt.Sprintf("%.0f %s", size, sizes[i])
	}
	
	return fmt.Sprintf("%.1f %s", size, sizes[i])
}