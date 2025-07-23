package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// UserStatus representa el estado de un usuario
type UserStatus struct {
	Username         string    `json:"username"`
	Connected        bool      `json:"connected"`
	LastSeen         time.Time `json:"lastSeen"`
	ConnectedAt      time.Time `json:"connectedAt"`
	LastMessageIndex int       `json:"-"` // ⭐ NUEVO: Índice del último mensaje que vio
}

// Hub mantiene el conjunto de clientes activos y difunde mensajes a los clientes
type Hub struct {
	// Clientes registrados - mapa protegido por mutex
	clients map[*Client]bool

	// Historial de todos los usuarios que se han conectado
	userHistory map[string]*UserStatus

	// ⭐ MEJORADO: Historial GLOBAL de mensajes con índices
	messageHistory []*Message
	maxHistorySize int
	messageCounter int // Contador global de mensajes

	// Mensajes entrantes de los clientes para difundir
	broadcast chan []byte

	// Solicitudes de registro de nuevos clientes
	register chan *Client

	// Solicitudes de cancelación de registro de clientes
	unregister chan *Client

	// Mutex para proteger acceso concurrente al mapa de clientes y historial
	mu sync.RWMutex
}

// NewHub crea una nueva instancia del hub de chat
func NewHub() *Hub {
	return &Hub{
		broadcast:      make(chan []byte, 1000),
		register:       make(chan *Client, 100),
		unregister:     make(chan *Client, 100),
		clients:        make(map[*Client]bool),
		userHistory:    make(map[string]*UserStatus),
		messageHistory: make([]*Message, 0),
		maxHistorySize: 200, // ⭐ AUMENTADO: Más mensajes en historial
		messageCounter: 0,
	}
}

// Run inicia el loop principal del hub
func (h *Hub) Run() {
	log.Println("🚀 Hub iniciado con historial persistente...")

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// isUsernameAvailable verifica si un nombre de usuario está disponible
func (h *Hub) isUsernameAvailable(username string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Verificar si hay algún cliente conectado con ese nombre EXACTO
	for client := range h.clients {
		if client.username == username {
			return false
		}
	}

	return true
}

// registerClient registra un nuevo cliente en el hub
func (h *Hub) registerClient(client *Client) {
	// ⭐ VALIDACIÓN: Verificar si el nombre de usuario ya está en uso
	if !h.isUsernameAvailable(client.username) {
		log.Printf("❌ Intento de conexión con nombre duplicado: '%s'", client.username)
		h.sendErrorToClient(client, "El nombre de usuario '"+client.username+"' ya está en uso.", "USERNAME_TAKEN")
		return
	}

	h.mu.Lock()
	h.clients[client] = true

	// ⭐ MEJORADO: Verificar si es un usuario que regresa
	now := time.Now()
	isReturningUser := false

	if userStatus, exists := h.userHistory[client.username]; exists {
		// Usuario que regresa
		userStatus.Connected = true
		userStatus.ConnectedAt = now
		userStatus.LastSeen = now
		isReturningUser = true
		log.Printf("🔄 Usuario '%s' reconectado. Último mensaje visto: %d", client.username, userStatus.LastMessageIndex)
	} else {
		// Usuario completamente nuevo
		h.userHistory[client.username] = &UserStatus{
			Username:         client.username,
			Connected:        true,
			ConnectedAt:      now,
			LastSeen:         now,
			LastMessageIndex: h.messageCounter, // ⭐ Empezar desde el mensaje actual
		}
		log.Printf("✨ Usuario nuevo '%s' registrado", client.username)
	}

	clientCount := len(h.clients)
	h.mu.Unlock()

	log.Printf("✅ Cliente '%s' conectado exitosamente. Total: %d", client.username, clientCount)

	// Enviar mensaje de éxito al cliente
	h.sendSuccessToClient(client, "Conectado exitosamente como "+client.username)

	// ⭐ ENVIAR HISTORIAL PERDIDO si es usuario que regresa
	if isReturningUser {
		h.sendMissedMessages(client)
	}

	// Enviar lista de usuarios actualizada
	h.broadcastUserList()

	// Enviar mensaje de sistema
	joinMsg := NewSystemMessage(client.username + " se ha unido al chat")
	joinMsg.Type = MessageTypeJoin

	if msgBytes, err := json.Marshal(joinMsg); err == nil {
		h.broadcastMessage(msgBytes)
	}
}

// ⭐ NUEVA FUNCIÓN: Enviar mensajes perdidos a usuario que se reconecta
func (h *Hub) sendMissedMessages(client *Client) {
	h.mu.RLock()
	userStatus := h.userHistory[client.username]
	lastSeenIndex := userStatus.LastMessageIndex

	// Encontrar mensajes que el usuario se perdió
	missedMessages := make([]*Message, 0)
	for i, msg := range h.messageHistory {
		// Solo enviar mensajes DESPUÉS del último que vio
		if i > lastSeenIndex && msg.Type == MessageTypeMessage {
			missedMessages = append(missedMessages, msg)
		}
	}
	h.mu.RUnlock()

	if len(missedMessages) == 0 {
		log.Printf("📜 No hay mensajes perdidos para '%s'", client.username)
		return
	}

	log.Printf("📜 Enviando %d mensajes perdidos a '%s'", len(missedMessages), client.username)

	// Enviar notificación de historial
	historyNotification := map[string]interface{}{
		"type":    "historyStart",
		"message": "Recuperando conversación perdida...",
		"count":   len(missedMessages),
	}

	if notifBytes, err := json.Marshal(historyNotification); err == nil {
		select {
		case client.send <- notifBytes:
		default:
		}
	}

	// Enviar cada mensaje perdido
	for _, msg := range missedMessages {
		if msgBytes, err := json.Marshal(msg); err == nil {
			select {
			case client.send <- msgBytes:
			default:
				log.Printf("⚠️ No se pudo enviar mensaje perdido a '%s'", client.username)
			}
			// Pequeña pausa para evitar saturar el cliente
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Notificación de fin de historial
	endNotification := map[string]interface{}{
		"type":    "historyEnd",
		"message": "¡Ya estás al día! Conversación recuperada.",
	}

	if endBytes, err := json.Marshal(endNotification); err == nil {
		select {
		case client.send <- endBytes:
		default:
		}
	}
}

// sendErrorToClient envía un mensaje de error a un cliente específico
func (h *Hub) sendErrorToClient(client *Client, message, code string) {
	errorMsg := map[string]interface{}{
		"type":    "error",
		"message": message,
		"code":    code,
	}

	if msgBytes, err := json.Marshal(errorMsg); err == nil {
		select {
		case client.send <- msgBytes:
		default:
		}
	}

	// Cerrar conexión después de un breve delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.conn.Close()
	}()
}

// sendSuccessToClient envía un mensaje de éxito a un cliente específico
func (h *Hub) sendSuccessToClient(client *Client, message string) {
	successMsg := map[string]interface{}{
		"type":     "connectionSuccess",
		"message":  message,
		"username": client.username,
	}

	if msgBytes, err := json.Marshal(successMsg); err == nil {
		select {
		case client.send <- msgBytes:
		default:
		}
	}
}

// unregisterClient cancela el registro de un cliente del hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		// Eliminar cliente del mapa y cerrar su canal de envío
		delete(h.clients, client)
		close(client.send)

		// ⭐ MEJORADO: Actualizar índice del último mensaje visto
		if userStatus, exists := h.userHistory[client.username]; exists {
			userStatus.Connected = false
			userStatus.LastSeen = time.Now()
			userStatus.LastMessageIndex = h.messageCounter // ⭐ Guardar hasta dónde leyó
			log.Printf("💾 Usuario '%s' desconectado. Último mensaje guardado: %d", client.username, userStatus.LastMessageIndex)
		}

		clientCount := len(h.clients)
		h.mu.Unlock()

		log.Printf("🔌 Cliente '%s' desconectado. Total: %d", client.username, clientCount)

		// Enviar lista de usuarios actualizada
		h.broadcastUserList()

		// Enviar mensaje de sistema
		leaveMsg := NewSystemMessage(client.username + " ha salido del chat")
		leaveMsg.Type = MessageTypeLeave

		if msgBytes, err := json.Marshal(leaveMsg); err == nil {
			h.broadcastMessage(msgBytes)
		}
	} else {
		h.mu.Unlock()
	}
}

// broadcastMessage envía un mensaje a todos los clientes conectados
func (h *Hub) broadcastMessage(message []byte) {
	// ⭐ MEJORADO: Agregar al historial global con índice
	h.addToGlobalHistory(message)

	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	log.Printf("📤 Enviando mensaje a %d clientes", len(clients))

	// Enviar mensaje a cada cliente conectado
	for _, client := range clients {
		select {
		case client.send <- message:
			// ⭐ ACTUALIZAR: Marcar que este usuario vio este mensaje
			h.updateUserLastMessage(client.username)
		default:
			// El canal del cliente está lleno o cerrado
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
			close(client.send)
			log.Printf("Cliente '%s' eliminado por canal bloqueado", client.username)
		}
	}
}

// ⭐ MEJORADO: Agregar mensaje al historial global con índices
func (h *Hub) addToGlobalHistory(messageBytes []byte) {
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		log.Printf("❌ Error parseando mensaje para historial: %v", err)
		return
	}

	// Solo agregar mensajes de chat al historial (no sistema de conexión/desconexión)
	if msg.Type == MessageTypeMessage {
		h.mu.Lock()
		h.messageHistory = append(h.messageHistory, &msg)
		h.messageCounter++ // ⭐ Incrementar contador global

		// Mantener solo los últimos N mensajes
		if len(h.messageHistory) > h.maxHistorySize {
			// Eliminar mensajes más antiguos pero actualizar índices
			removed := len(h.messageHistory) - h.maxHistorySize
			h.messageHistory = h.messageHistory[removed:]

			// Ajustar índices de usuarios para compensar mensajes eliminados
			for _, userStatus := range h.userHistory {
				userStatus.LastMessageIndex -= removed
				if userStatus.LastMessageIndex < 0 {
					userStatus.LastMessageIndex = 0
				}
			}
		}
		h.mu.Unlock()

		log.Printf("📜 Mensaje #%d agregado al historial global. Total: %d", h.messageCounter, len(h.messageHistory))
	}
}

// ⭐ NUEVA FUNCIÓN: Actualizar el último mensaje visto por un usuario
func (h *Hub) updateUserLastMessage(username string) {
	h.mu.Lock()
	if userStatus, exists := h.userHistory[username]; exists {
		userStatus.LastMessageIndex = h.messageCounter
		userStatus.LastSeen = time.Now()
	}
	h.mu.Unlock()
}

// broadcastUserList envía la lista actualizada de usuarios a todos los clientes
func (h *Hub) broadcastUserList() {
	h.mu.RLock()
	users := make([]*UserStatus, 0, len(h.userHistory))
	for _, userStatus := range h.userHistory {
		// Crear copia para evitar problemas de concurrencia
		userCopy := &UserStatus{
			Username:    userStatus.Username,
			Connected:   userStatus.Connected,
			LastSeen:    userStatus.LastSeen,
			ConnectedAt: userStatus.ConnectedAt,
		}
		users = append(users, userCopy)
	}
	h.mu.RUnlock()

	// Crear mensaje con la lista de usuarios
	userListMsg := map[string]interface{}{
		"type":  "userList",
		"users": users,
	}

	if msgBytes, err := json.Marshal(userListMsg); err == nil {
		h.broadcastMessage(msgBytes)
	}
}

// GetClientCount devuelve el número actual de clientes conectados de forma thread-safe
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	count := len(h.clients)
	h.mu.RUnlock()
	return count
}

// GetConnectedUsers devuelve una lista de nombres de usuarios conectados
func (h *Hub) GetConnectedUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.clients))
	for client := range h.clients {
		users = append(users, client.username)
	}
	return users
}

// GetUserHistory devuelve el historial de todos los usuarios
func (h *Hub) GetUserHistory() map[string]*UserStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Crear copia del mapa para evitar problemas de concurrencia
	history := make(map[string]*UserStatus)
	for username, status := range h.userHistory {
		statusCopy := *status // Copia el valor
		history[username] = &statusCopy
	}
	return history
}

// GetMessageHistory devuelve el historial de mensajes (para debugging)
func (h *Hub) GetMessageHistory() []*Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Crear copia del slice
	history := make([]*Message, len(h.messageHistory))
	copy(history, h.messageHistory)
	return history
}

// ⭐ NUEVA FUNCIÓN: Obtener estadísticas del historial
func (h *Hub) GetHistoryStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"totalMessages":  len(h.messageHistory),
		"messageCounter": h.messageCounter,
		"totalUsers":     len(h.userHistory),
		"connectedUsers": len(h.clients),
		"maxHistorySize": h.maxHistorySize,
	}

	return stats
}
