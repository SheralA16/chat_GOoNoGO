package main

import (
	"bytes"
	"encoding/json"
	"log"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Tiempo máximo de espera para escribir un mensaje al cliente
	writeWait = 10 * time.Second

	// Tiempo máximo de espera para leer el siguiente pong del cliente
	pongWait = 60 * time.Second

	// Enviar pings al cliente con este período. Debe ser menor que pongWait
	pingPeriod = (pongWait * 9) / 10

	// Tamaño máximo del mensaje permitido del cliente
	maxMessageSize = 10 * 1024 * 1024 // Aumentado a 10MB para imágenes
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client representa un cliente WebSocket activo
type Client struct {
	// El hub de chat al que pertenece este cliente
	hub *Hub

	// La conexión WebSocket
	conn *websocket.Conn

	// Canal con buffer para mensajes salientes
	send chan []byte

	// Nombre de usuario del cliente
	username string
}

// IncomingMessage representa la estructura de mensajes entrantes del cliente
type IncomingMessage struct {
	Type      string `json:"type"`      // "text" o "image"
	Content   string `json:"content"`   // Contenido del mensaje o caption
	ImageData string `json:"imageData"` // Datos base64 de la imagen
	ImageName string `json:"imageName"` // Nombre del archivo
	ImageSize int64  `json:"imageSize"` // Tamaño del archivo
}

// readPump bombea mensajes desde la conexión WebSocket al hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Error estableciendo deadline de lectura para '%s': %v", c.username, err)
		return
	}

	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Error estableciendo deadline en pong handler para '%s': %v", c.username, err)
		}
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error inesperado de WebSocket para '%s': %v", c.username, err)
			} else {
				log.Printf("Cliente '%s' cerró conexión: %v", c.username, err)
			}
			break
		}

		messageBytes = bytes.TrimSpace(bytes.Replace(messageBytes, newline, space, -1))

		// Intentar parsear el mensaje como JSON
		var incomingMsg IncomingMessage
		if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
			log.Printf("Error parseando mensaje JSON de cliente '%s': %v", c.username, err)
			continue
		}

		var msg *Message
		var messageJSON []byte

		// Procesar según el tipo de mensaje
		switch incomingMsg.Type {
		case "image":
			// Procesar mensaje de imagen
			msg, err = c.processImageMessage(incomingMsg)
			if err != nil {
				log.Printf("Error procesando imagen de '%s': %v", c.username, err)
				c.sendErrorMessage(err.Error())
				continue
			}

		case "text":
		default:
			// Procesar mensaje de texto normal
			msg = NewMessage(c.username, incomingMsg.Content)
		}

		// Serializar mensaje completo
		messageJSON, err = json.Marshal(msg)
		if err != nil {
			log.Printf("Error serializando mensaje de '%s': %v", c.username, err)
			continue
		}

		// Enviar al hub para difusión
		select {
		case c.hub.broadcast <- messageJSON:
			if msg.Type == MessageTypeImage {
				log.Printf("🖼️ Imagen de '%s' enviada al hub (%s)", c.username, FormatImageSize(msg.ImageSize))
			} else {
				log.Printf("💬 Mensaje de '%s' enviado al hub", c.username)
			}
		default:
			log.Printf("⚠️ Hub ocupado, mensaje de '%s' descartado", c.username)
		}
	}
}

// processImageMessage procesa un mensaje de imagen entrante
func (c *Client) processImageMessage(incomingMsg IncomingMessage) (*Message, error) {
	// Validar la imagen
	err := ValidateImage(incomingMsg.ImageData, incomingMsg.ImageName, incomingMsg.ImageSize)
	if err != nil {
		return nil, err
	}

	// Detectar tipo MIME
	mimeType := mime.TypeByExtension(filepath.Ext(incomingMsg.ImageName))
	if mimeType == "" {
		mimeType = detectMimeTypeFromBase64(incomingMsg.ImageData)
	}

	// Limpiar datos base64 (remover prefijo data URL si existe)
	imageData := incomingMsg.ImageData
	if strings.Contains(imageData, ",") {
		parts := strings.Split(imageData, ",")
		if len(parts) > 1 {
			imageData = parts[1]
		}
	}

	// Crear mensaje de imagen
	msg := NewImageMessage(
		c.username,
		incomingMsg.Content, // Caption opcional
		imageData,
		incomingMsg.ImageName,
		incomingMsg.ImageSize,
		mimeType,
	)

	return msg, nil
}

// sendErrorMessage envía un mensaje de error al cliente
func (c *Client) sendErrorMessage(errorMsg string) {
	errorResponse := map[string]interface{}{
		"type":    "error",
		"message": errorMsg,
	}

	if msgBytes, err := json.Marshal(errorResponse); err == nil {
		select {
		case c.send <- msgBytes:
		default:
			log.Printf("No se pudo enviar mensaje de error a '%s'", c.username)
		}
	}
}

// writePump bombea mensajes del hub al cliente WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Error estableciendo deadline de escritura para '%s': %v", c.username, err)
				return
			}

			if !ok {
				// El hub cerró el canal
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Error enviando mensaje de cierre para '%s': %v", c.username, err)
				}
				return
			}

			// Enviar cada mensaje como un frame separado
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error escribiendo mensaje para '%s': %v", c.username, err)
				return
			}

			// Procesar mensajes adicionales en el buffer sin concatenar
		additionalMessages:
			for {
				select {
				case nextMessage := <-c.send:
					// Enviar cada mensaje adicional como frame separado
					if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
						log.Printf("Error estableciendo deadline para mensaje adicional: %v", err)
						return
					}
					if err := c.conn.WriteMessage(websocket.TextMessage, nextMessage); err != nil {
						log.Printf("Error enviando mensaje adicional para '%s': %v", c.username, err)
						return
					}
				default:
					// No hay más mensajes en buffer
					break additionalMessages
				}
			}

		case <-ticker.C:
			// Enviar ping
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Error estableciendo deadline para ping para '%s': %v", c.username, err)
				return
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error enviando ping para '%s': %v", c.username, err)
				return
			}
		}
	}
}