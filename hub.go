package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// UserStatus representa el estado de un usuario
type UserStatus struct {
	Username    string    `json:"username"`
	Connected   bool      `json:"connected"`
	LastSeen    time.Time `json:"lastSeen"`
	ConnectedAt time.Time `json:"connectedAt"`
	// ⭐ NUEVO: Timestamp del último mensaje que vio
	LastMessageSeen time.Time `json:"lastMessageSeen"`
	// ⭐ NUEVO: Indica si es la primera vez que se conecta
	IsFirstTime bool `json:"isFirstTime"`
}

// Hub mantiene el conjunto de clientes activos y difunde mensajes a los clientes
type Hub struct {
	// Clientes registrados - mapa protegido por mutex
	clients map[*Client]bool

	// Historial de todos los usuarios que se han conectado
	userHistory map[string]*UserStatus

	// ⭐ NUEVO: Historial persistente de mensajes con timestamps
	messageHistory []*Message
	maxHistorySize int
	maxHistoryAge  time.Duration // Tiempo máximo que guardamos mensajes

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
		broadcast:      make(chan []byte, 1000), // Buffer para evitar bloqueos
		register:       make(chan *Client, 100),
		unregister:     make(chan *Client, 100),
		clients:        make(map[*Client]bool),
		userHistory:    make(map[string]*UserStatus),
		messageHistory: make([]*Message, 0),
		maxHistorySize: 100,            // Mantener últimos 100 mensajes
		maxHistoryAge:  24 * time.Hour, // Mantener mensajes por 24 horas
	}
}

// Run inicia el loop principal del hub
func (h *Hub) Run() {
	log.Println("🚀 Hub iniciado con soporte para mensajes perdidos...")

	// Limpiar mensajes antiguos cada hora
	go h.cleanupOldMessages()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client) //nuevo usuario

		case client := <-h.unregister:
			h.unregisterClient(client) //usuario desconectado

		case message := <-h.broadcast:
			h.broadcastMessage(message) //mensaje entrante
		}
	}
}

// cleanupOldMessages elimina mensajes antiguos periódicamente
func (h *Hub) cleanupOldMessages() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		cutoff := time.Now().Add(-h.maxHistoryAge)
		newHistory := make([]*Message, 0)

		for _, msg := range h.messageHistory {
			if msg.Timestamp.After(cutoff) {
				newHistory = append(newHistory, msg)
			}
		}

		cleaned := len(h.messageHistory) - len(newHistory)
		h.messageHistory = newHistory
		h.mu.Unlock()

		if cleaned > 0 {
			log.Printf("🧹 Limpieza automática: eliminados %d mensajes antiguos", cleaned)
		}
	}
}

// isUsernameAvailable verifica si un nombre de usuario está disponible (método privado)
func (h *Hub) isUsernameAvailable(username string) bool {
	h.mu.RLock()         // Bloquear lectura del mapa de clientes
	defer h.mu.RUnlock() // Desbloquear al final de la función

	// Verificar si hay algún cliente conectado con ese nombre EXACTO
	for client := range h.clients {
		if client.username == username {
			return false
		}
	}

	// También verificar en el historial si está conectado actualmente
	if userStatus, exists := h.userHistory[username]; exists && userStatus.Connected {
		return false
	}

	return true
}

// registerClient registra un nuevo cliente en el hub
func (h *Hub) registerClient(client *Client) {
	// ⭐ VALIDACIÓN: Verificar si el nombre de usuario ya está en uso
	if !h.isUsernameAvailable(client.username) {
		log.Printf("❌ Intento de conexión con nombre duplicado: '%s'", client.username)

		// Enviar mensaje de error al cliente
		errorMsg := map[string]interface{}{
			"type":    "error",
			"message": "El nombre de usuario '" + client.username + "' ya está en uso. Por favor, elige otro nombre.",
			"code":    "USERNAME_TAKEN",
		}

		if msgBytes, err := json.Marshal(errorMsg); err == nil {
			select {
			case client.send <- msgBytes:
				log.Printf("📤 Mensaje de error enviado a cliente con nombre duplicado")
			default:
				log.Printf("❌ No se pudo enviar mensaje de error al cliente")
			}
		}

		// Cerrar la conexión después de un breve delay para que el mensaje llegue
		go func() {
			time.Sleep(100 * time.Millisecond)
			client.conn.Close()
		}()

		return // Salir si el nombre ya está en uso
	}

	// Si llegamos aquí, el nombre está disponible
	h.mu.Lock()              // Bloquear escritura del mapa de clientes
	h.clients[client] = true // Registrar cliente en el mapa

	// ⭐ LÓGICA MEJORADA: Determinar si es reconexión o primera vez
	now := time.Now()
	var isReconnection bool
	var lastSeen time.Time

	if userStatus, exists := h.userHistory[client.username]; exists {
		// Usuario existente - es una reconexión
		isReconnection = true
		lastSeen = userStatus.LastSeen
		userStatus.Connected = true
		userStatus.ConnectedAt = now
		userStatus.LastSeen = now
		userStatus.IsFirstTime = false
		log.Printf("🔄 Reconexión detectada para '%s', última vez visto: %s",
			client.username, lastSeen.Format("15:04:05"))
	} else {
		// Usuario nuevo - primera vez
		isReconnection = false
		h.userHistory[client.username] = &UserStatus{
			Username:        client.username,
			Connected:       true,
			ConnectedAt:     now,
			LastSeen:        now,
			LastMessageSeen: now, // Para nuevos usuarios, empezar desde ahora
			IsFirstTime:     true,
		}
		log.Printf("✨ Nuevo usuario detectado: '%s'", client.username)
	}

	clientCount := len(h.clients)
	h.mu.Unlock()

	log.Printf("✅ Cliente '%s' conectado exitosamente. Total de clientes: %d", client.username, clientCount)

	// Enviar mensaje de éxito
	successMsg := map[string]interface{}{
		"type":     "connectionSuccess",
		"message":  "Conectado exitosamente como " + client.username,
		"username": client.username,
	}

	if msgBytes, err := json.Marshal(successMsg); err == nil {
		select {
		case client.send <- msgBytes:
		default:
		}
	}

	// ⭐ NUEVA FUNCIONALIDAD: Enviar mensajes perdidos solo a reconexiones
	if isReconnection {
		h.sendMissedMessages(client, lastSeen)
	}

	h.broadcastUserList() // Enviar lista de usuarios actualizada

	// Mensaje de sistema
	joinMsg := NewSystemMessage(client.username + " se ha unido al chat")
	joinMsg.Type = MessageTypeJoin

	if msgBytes, err := json.Marshal(joinMsg); err == nil {
		h.broadcastMessage(msgBytes)
	} else {
		log.Printf("Error serializando mensaje de conexión: %v", err)
	}
}

// ⭐ NUEVA FUNCIÓN: Enviar mensajes perdidos a usuario que se reconecta
func (h *Hub) sendMissedMessages(client *Client, lastSeen time.Time) {
	h.mu.RLock()

	// Encontrar mensajes enviados después de la última vez que estuvo online
	var missedMessages []*Message
	for _, msg := range h.messageHistory {
		// Solo incluir mensajes normales (no de sistema) enviados después de lastSeen
		if msg.Type == MessageTypeMessage && msg.Timestamp.After(lastSeen) {
			// No incluir sus propios mensajes
			if msg.Username != client.username {
				missedMessages = append(missedMessages, msg)
			}
		}
	}

	h.mu.RUnlock()

	if len(missedMessages) == 0 {
		log.Printf("📭 No hay mensajes perdidos para '%s'", client.username)
		return
	}

	log.Printf("📬 Enviando %d mensajes perdidos a '%s'", len(missedMessages), client.username)

	// Enviar notificación de mensajes perdidos
	notificationMsg := map[string]interface{}{
		"type":    "missedMessagesNotification",
		"count":   len(missedMessages),
		"message": "Tienes mensajes perdidos mientras estuviste ausente",
	}

	if msgBytes, err := json.Marshal(notificationMsg); err == nil {
		select {
		case client.send <- msgBytes:
		default:
		}
	}

	// Enviar cada mensaje perdido con un pequeño delay para mejor UX
	go func() {
		for i, msg := range missedMessages {
			if msgBytes, err := json.Marshal(msg); err == nil {
				select {
				case client.send <- msgBytes:
					log.Printf("📤 Mensaje perdido %d/%d enviado a '%s': %.30s...",
						i+1, len(missedMessages), client.username, msg.Content)
				default:
					log.Printf("❌ No se pudo enviar mensaje perdido a '%s'", client.username)
				}

				// Pequeño delay entre mensajes para mejor UX
				if i < len(missedMessages)-1 {
					time.Sleep(50 * time.Millisecond)
				}
			}
		}

		// Actualizar lastMessageSeen después de enviar todos los mensajes
		h.mu.Lock()
		if userStatus, exists := h.userHistory[client.username]; exists {
			userStatus.LastMessageSeen = time.Now()
		}
		h.mu.Unlock()

		log.Printf("✅ Todos los mensajes perdidos enviados a '%s'", client.username)
	}()
}

// unregisterClient cancela el registro de un cliente del hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		// Eliminar cliente del mapa y cerrar su canal de envío
		delete(h.clients, client)
		close(client.send)

		// Actualizar estado del usuario a desconectado
		if userStatus, exists := h.userHistory[client.username]; exists {
			userStatus.Connected = false
			userStatus.LastSeen = time.Now()
			// ⭐ NUEVO: Actualizar último mensaje visto al desconectarse
			userStatus.LastMessageSeen = time.Now()
		}

		clientCount := len(h.clients)
		h.mu.Unlock()

		log.Printf("🔌 Cliente '%s' desconectado. Total de clientes: %d", client.username, clientCount)

		// Enviar lista de usuarios actualizada
		h.broadcastUserList()

		// Enviar mensaje de sistema
		leaveMsg := NewSystemMessage(client.username + " ha salido del chat")
		leaveMsg.Type = MessageTypeLeave

		if msgBytes, err := json.Marshal(leaveMsg); err == nil {
			h.broadcastMessage(msgBytes)
		} else {
			log.Printf("Error serializando mensaje de desconexión: %v", err)
		}
	} else {
		h.mu.Unlock()
	}
}

// broadcastMessage envía un mensaje a todos los clientes conectados
func (h *Hub) broadcastMessage(message []byte) {
	// ⭐ NUEVO: Parsear mensaje para agregarlo al historial si es necesario
	var msg Message
	if err := json.Unmarshal(message, &msg); err == nil {
		if msg.Type == MessageTypeMessage {
			h.addToMessageHistory(&msg)
		}
	}

	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	// Debug: mostrar qué se está enviando
	log.Printf("Enviando mensaje a %d clientes", len(clients))

	// Enviar mensaje a cada cliente
	for _, client := range clients {
		select {
		case client.send <- message:
			// Mensaje enviado exitosamente
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

// ⭐ NUEVA FUNCIÓN: Agregar mensaje al historial persistente
func (h *Hub) addToMessageHistory(msg *Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Agregar mensaje al historial
	h.messageHistory = append(h.messageHistory, msg)

	// Mantener solo los últimos N mensajes
	if len(h.messageHistory) > h.maxHistorySize {
		// Eliminar mensajes más antiguos
		excess := len(h.messageHistory) - h.maxHistorySize
		h.messageHistory = h.messageHistory[excess:]
		log.Printf("🗑️ Historial recortado: eliminados %d mensajes antiguos", excess)
	}

	log.Printf("💾 Mensaje guardado en historial persistente. Total: %d/%d",
		len(h.messageHistory), h.maxHistorySize)
}

// broadcastUserList envía la lista actualizada de usuarios a todos los clientes
func (h *Hub) broadcastUserList() {
	h.mu.RLock()
	users := make([]*UserStatus, 0, len(h.userHistory))
	for _, userStatus := range h.userHistory {
		// Crear copia para evitar problemas de concurrencia
		userCopy := &UserStatus{
			Username:        userStatus.Username,
			Connected:       userStatus.Connected,
			LastSeen:        userStatus.LastSeen,
			ConnectedAt:     userStatus.ConnectedAt,
			LastMessageSeen: userStatus.LastMessageSeen,
			IsFirstTime:     userStatus.IsFirstTime,
		}
		users = append(users, userCopy)
	}
	h.mu.RUnlock()

	// Crear mensaje con la lista de usuarios
	userListMsg := map[string]interface{}{
		"type":  "userList",
		"users": users,
	}

	log.Printf("Enviando lista de %d usuarios a todos los clientes", len(users))

	if msgBytes, err := json.Marshal(userListMsg); err == nil {
		h.broadcastMessage(msgBytes)
	} else {
		log.Printf("Error serializando lista de usuarios: %v", err)
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

// GetMessageHistory devuelve el historial de mensajes (para debugging y tests)
func (h *Hub) GetMessageHistory() []*Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Crear copia del slice para evitar problemas de concurrencia
	history := make([]*Message, len(h.messageHistory))
	copy(history, h.messageHistory)
	return history
}

// ⭐ NUEVA FUNCIÓN: Obtener estadísticas del historial de mensajes
func (h *Hub) GetMessageHistoryStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"totalMessages": len(h.messageHistory),
		"maxSize":       h.maxHistorySize,
		"maxAge":        h.maxHistoryAge.String(),
	}

	if len(h.messageHistory) > 0 {
		oldest := h.messageHistory[0].Timestamp
		newest := h.messageHistory[len(h.messageHistory)-1].Timestamp
		stats["oldestMessage"] = oldest
		stats["newestMessage"] = newest
		stats["timeSpan"] = newest.Sub(oldest).String()
	}

	return stats
}
