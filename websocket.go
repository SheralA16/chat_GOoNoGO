package main

import (
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/gorilla/websocket"
)

// upgrader especifica parámetros para actualizar una conexión HTTP a WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// ⭐ RAILWAY: Permitir conexiones desde cualquier origen
	CheckOrigin: func(r *http.Request) bool {
		// En Railway necesitamos permitir conexiones cross-origin
		return true
	},
}

// validateUsername valida que el nombre de usuario sea válido
func validateUsername(username string) bool {
	// Verificar longitud
	if len(username) < 2 || len(username) > 20 {
		return false
	}

	// Verificar caracteres válidos (letras, números, guiones y guiones bajos)
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}

	return true
}

// serveWS maneja las solicitudes WebSocket del cliente
func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Obtener nombre de usuario de los parámetros de consulta
	username := strings.TrimSpace(r.URL.Query().Get("username"))

	log.Printf("🔌 Intento de conexión WebSocket desde %s con username: '%s'", r.RemoteAddr, username)

	// Validar nombre de usuario
	if username == "" {
		log.Printf("❌ Nombre de usuario vacío desde %s", r.RemoteAddr)
		http.Error(w, "Nombre de usuario requerido", http.StatusBadRequest)
		return
	}

	if !validateUsername(username) {
		log.Printf("❌ Nombre de usuario inválido: '%s' desde %s", username, r.RemoteAddr)
		http.Error(w, "Nombre de usuario inválido", http.StatusBadRequest)
		return
	}

	// Actualizar la conexión HTTP a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Error actualizando conexión a WebSocket: %v", err)
		return
	}

	// Crear cliente
	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		username: username,
	}

	// Registrar cliente en el hub (el hub manejará duplicados)
	client.hub.register <- client

	// Iniciar las goroutines
	go client.writePump()
	go client.readPump()

	log.Printf("✅ Cliente '%s' procesado desde %s", username, r.RemoteAddr)
}
