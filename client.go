package main

import (
	"bytes"
	"encoding/json"
	"log"
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
	maxMessageSize = 512
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
		var incomingMsg struct {
			Content string `json:"content"`
		}

		if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
			log.Printf("Error parseando mensaje JSON de cliente '%s': %v", c.username, err)
			continue
		}

		// Crear mensaje completo con metadata
		msg := NewMessage(c.username, incomingMsg.Content)

		// Serializar mensaje completo
		messageJSON, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error serializando mensaje de '%s': %v", c.username, err)
			continue
		}

		// Enviar al hub para difusión
		select {
		case c.hub.broadcast <- messageJSON:
			log.Printf("💬 Mensaje de '%s' enviado al hub", c.username)
		default:
			log.Printf("⚠️ Hub ocupado, mensaje de '%s' descartado", c.username)
		}
	}
}

// ⭐ CORREGIDO: writePump - Un mensaje por WebSocket frame
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

			// ⭐ CORREGIDO: Enviar cada mensaje como un frame separado
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error escribiendo mensaje para '%s': %v", c.username, err)
				return
			}

			// ⭐ IMPORTANTE: NO concatenar mensajes adicionales
			// Cada mensaje debe ir en su propio WebSocket frame
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
