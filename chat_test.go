package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestHubCreationWithHistory prueba la creación correcta de un nuevo hub con historial
func TestHubCreationWithHistory(t *testing.T) {
	hub := NewHub()

	if hub.clients == nil {
		t.Error("El mapa de clientes no se inicializó correctamente")
	}

	if hub.broadcast == nil {
		t.Error("El canal de difusión no se inicializó")
	}

	if hub.register == nil {
		t.Error("El canal de registro no se inicializó")
	}

	if hub.unregister == nil {
		t.Error("El canal de cancelación no se inicializó")
	}

	if hub.messageHistory == nil {
		t.Error("El historial de mensajes no se inicializó")
	}

	if hub.userHistory == nil {
		t.Error("El historial de usuarios no se inicializó")
	}

	if hub.maxHistorySize != 200 {
		t.Errorf("Se esperaba maxHistorySize=200, pero se encontró %d", hub.maxHistorySize)
	}

	if hub.messageCounter != 0 {
		t.Errorf("Se esperaba messageCounter=0, pero se encontró %d", hub.messageCounter)
	}

	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes inicialmente, pero se encontraron %d", hub.GetClientCount())
	}
}

// TestPersistentHistory prueba el historial persistente para usuarios que se reconectan
func TestPersistentHistory(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Simular usuario que se conecta, envía mensajes y se desconecta
	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "usuario1",
	}

	// Registrar primer usuario
	hub.register <- client1
	time.Sleep(100 * time.Millisecond)

	// Limpiar mensajes del sistema
	for len(client1.send) > 0 {
		<-client1.send
	}

	// Enviar algunos mensajes
	msg1 := NewMessage("usuario1", "Primer mensaje")
	msg2 := NewMessage("usuario1", "Segundo mensaje")

	msg1Bytes, _ := json.Marshal(msg1)
	msg2Bytes, _ := json.Marshal(msg2)

	hub.broadcast <- msg1Bytes
	hub.broadcast <- msg2Bytes
	time.Sleep(200 * time.Millisecond)

	// Verificar que los mensajes están en el historial
	history := hub.GetMessageHistory()
	if len(history) != 2 {
		t.Errorf("Se esperaban 2 mensajes en el historial, pero se encontraron %d", len(history))
	}

	// Desconectar el usuario
	hub.unregister <- client1
	time.Sleep(100 * time.Millisecond)

	// Verificar que el usuario se marcó como desconectado
	userHistory := hub.GetUserHistory()
	if userStatus, exists := userHistory["usuario1"]; !exists {
		t.Error("El usuario no se encontró en el historial de usuarios")
	} else if userStatus.Connected {
		t.Error("El usuario debería estar marcado como desconectado")
	} else if userStatus.LastMessageIndex != 2 {
		t.Errorf("Se esperaba LastMessageIndex=2, pero se encontró %d", userStatus.LastMessageIndex)
	}

	// Enviar más mensajes mientras el usuario está desconectado
	msg3 := NewMessage("otroUsuario", "Mensaje perdido 1")
	msg4 := NewMessage("otroUsuario", "Mensaje perdido 2")

	msg3Bytes, _ := json.Marshal(msg3)
	msg4Bytes, _ := json.Marshal(msg4)

	hub.broadcast <- msg3Bytes
	hub.broadcast <- msg4Bytes
	time.Sleep(200 * time.Millisecond)

	// Reconectar el usuario
	client1Reconnected := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "usuario1",
	}

	hub.register <- client1Reconnected
	time.Sleep(300 * time.Millisecond)

	// Verificar que recibió los mensajes perdidos
	messagesReceived := 0
	historyMessages := 0

	for len(client1Reconnected.send) > 0 {
		msgBytes := <-client1Reconnected.send
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)

		if msgType == "historyStart" {
			t.Logf("Recibió notificación de inicio de historial")
		} else if msgType == "historyEnd" {
			t.Logf("Recibió notificación de fin de historial")
		} else if msgType == "message" {
			messagesReceived++
			if content, exists := msg["content"].(string); exists {
				if strings.Contains(content, "perdido") {
					historyMessages++
				}
			}
		}
	}

	if historyMessages != 2 {
		t.Errorf("Se esperaban 2 mensajes de historial, pero se recibieron %d", historyMessages)
	}

	t.Logf("Usuario reconectado recibió %d mensajes perdidos correctamente", historyMessages)
}

// TestMessageCounterIncrement prueba que el contador de mensajes se incremente correctamente
func TestMessageCounterIncrement(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear cliente
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "testuser",
	}

	hub.register <- client
	time.Sleep(100 * time.Millisecond)

	initialCounter := hub.messageCounter

	// Enviar varios mensajes
	for i := 0; i < 5; i++ {
		msg := NewMessage("testuser", "Mensaje "+string(rune(i+'1')))
		msgBytes, _ := json.Marshal(msg)
		hub.broadcast <- msgBytes
		time.Sleep(50 * time.Millisecond)
	}

	expectedCounter := initialCounter + 5
	if hub.messageCounter != expectedCounter {
		t.Errorf("Se esperaba messageCounter=%d, pero se encontró %d", expectedCounter, hub.messageCounter)
	}

	// Verificar que el usuario tiene el índice correcto
	userHistory := hub.GetUserHistory()
	if userStatus, exists := userHistory["testuser"]; !exists {
		t.Error("El usuario no se encontró en el historial")
	} else if userStatus.LastMessageIndex != expectedCounter {
		t.Errorf("Se esperaba LastMessageIndex=%d, pero se encontró %d", expectedCounter, userStatus.LastMessageIndex)
	}
}

// TestHistoryTrimming prueba que el historial se mantenga dentro del límite máximo
func TestHistoryTrimming(t *testing.T) {
	hub := NewHub()
	hub.maxHistorySize = 5 // Reducir para prueba
	go hub.Run()

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "testuser",
	}

	hub.register <- client
	time.Sleep(100 * time.Millisecond)

	// Enviar más mensajes que el límite máximo
	for i := 0; i < 10; i++ {
		msg := NewMessage("testuser", "Mensaje "+string(rune(i+'1')))
		msgBytes, _ := json.Marshal(msg)
		hub.broadcast <- msgBytes
		time.Sleep(50 * time.Millisecond)
	}

	// Verificar que el historial no exceda el límite
	history := hub.GetMessageHistory()
	if len(history) > hub.maxHistorySize {
		t.Errorf("El historial excede el límite máximo: %d > %d", len(history), hub.maxHistorySize)
	}

	// Verificar que tiene exactamente el máximo permitido
	if len(history) != hub.maxHistorySize {
		t.Errorf("Se esperaban %d mensajes en el historial, pero se encontraron %d", hub.maxHistorySize, len(history))
	}

	// Verificar que se conservan los mensajes más recientes
	lastMessage := history[len(history)-1]
	if !strings.Contains(lastMessage.Content, "Mensaje 10") {
		t.Error("El último mensaje en el historial no es el más reciente")
	}
}

// TestNewUserNoHistory prueba que usuarios nuevos no reciban historial
func TestNewUserNoHistory(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Usuario existente envía mensajes
	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "usuario1",
	}

	hub.register <- client1
	time.Sleep(100 * time.Millisecond)

	// Limpiar mensajes del sistema
	for len(client1.send) > 0 {
		<-client1.send
	}

	// Enviar algunos mensajes
	for i := 0; i < 3; i++ {
		msg := NewMessage("usuario1", "Mensaje "+string(rune(i+'1')))
		msgBytes, _ := json.Marshal(msg)
		hub.broadcast <- msgBytes
		time.Sleep(50 * time.Millisecond)
	}

	// Usuario completamente nuevo se conecta
	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "usuarioNuevo",
	}

	hub.register <- client2
	time.Sleep(200 * time.Millisecond)

	// Verificar que NO recibió mensajes de historial
	historyMessagesReceived := 0
	for len(client2.send) > 0 {
		msgBytes := <-client2.send
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		if msgType == "historyStart" || msgType == "historyEnd" {
			t.Error("Usuario nuevo no debería recibir notificaciones de historial")
		} else if msgType == "message" {
			historyMessagesReceived++
		}
	}

	if historyMessagesReceived > 0 {
		t.Errorf("Usuario nuevo recibió %d mensajes de historial cuando no debería recibir ninguno", historyMessagesReceived)
	}

	// Verificar que el usuario nuevo tiene el índice correcto (actual)
	userHistory := hub.GetUserHistory()
	if userStatus, exists := userHistory["usuarioNuevo"]; !exists {
		t.Error("El usuario nuevo no se encontró en el historial")
	} else if userStatus.LastMessageIndex != hub.messageCounter {
		t.Errorf("Usuario nuevo debería tener LastMessageIndex=%d, pero tiene %d", hub.messageCounter, userStatus.LastMessageIndex)
	}
}

// TestImageMessagesInHistory prueba que los mensajes con imágenes se manejen correctamente en el historial
func TestImageMessagesInHistory(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "testuser",
	}

	hub.register <- client
	time.Sleep(100 * time.Millisecond)

	// Limpiar mensajes del sistema
	for len(client.send) > 0 {
		<-client.send
	}

	// Crear mensaje con imagen
	imageData := &ImageData{
		Data: "data:image/png;base64,testdata",
		Name: "test.png",
		Type: "image/png",
		Size: 1000,
	}

	msg := NewMessageWithImage("testuser", "Mensaje con imagen", imageData)
	msgBytes, _ := json.Marshal(msg)

	hub.broadcast <- msgBytes
	time.Sleep(100 * time.Millisecond)

	// Desconectar usuario
	hub.unregister <- client
	time.Sleep(100 * time.Millisecond)

	// Enviar más mensajes con imágenes
	msg2 := NewMessageWithImage("otrouser", "Otra imagen", imageData)
	msg2Bytes, _ := json.Marshal(msg2)
	hub.broadcast <- msg2Bytes
	time.Sleep(100 * time.Millisecond)

	// Reconectar usuario
	clientReconnected := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "testuser",
	}

	hub.register <- clientReconnected
	time.Sleep(200 * time.Millisecond)

	// Verificar que recibió el mensaje con imagen
	imageMessagesReceived := 0
	for len(clientReconnected.send) > 0 {
		msgBytes := <-clientReconnected.send
		var receivedMsg Message
		if err := json.Unmarshal(msgBytes, &receivedMsg); err != nil {
			continue
		}

		if receivedMsg.Type == "message" && receivedMsg.HasImage {
			imageMessagesReceived++
			if receivedMsg.Image == nil {
				t.Error("Mensaje con imagen no tiene datos de imagen")
			}
		}
	}

	if imageMessagesReceived != 1 {
		t.Errorf("Se esperaba 1 mensaje con imagen en el historial, pero se recibieron %d", imageMessagesReceived)
	}
}

// TestConcurrentReconnections prueba reconexiones concurrentes
func TestConcurrentReconnections(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const numUsers = 5
	var wg sync.WaitGroup

	// Crear usuarios y enviar mensajes
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			username := "user" + string(rune(userID+'0'))

			// Primera conexión
			client := &Client{
				hub:      hub,
				send:     make(chan []byte, 256),
				username: username,
			}

			hub.register <- client
			time.Sleep(100 * time.Millisecond)

			// Enviar un mensaje
			msg := NewMessage(username, "Mensaje de "+username)
			msgBytes, _ := json.Marshal(msg)
			hub.broadcast <- msgBytes
			time.Sleep(50 * time.Millisecond)

			// Desconectar
			hub.unregister <- client
			time.Sleep(50 * time.Millisecond)

			// Reconectar después de un tiempo aleatorio
			time.Sleep(time.Duration(userID*50) * time.Millisecond)

			clientReconnected := &Client{
				hub:      hub,
				send:     make(chan []byte, 256),
				username: username,
			}

			hub.register <- clientReconnected
			time.Sleep(100 * time.Millisecond)

			// Contar mensajes recibidos
			messagesReceived := 0
			for len(clientReconnected.send) > 0 {
				<-clientReconnected.send
				messagesReceived++
			}

			t.Logf("Usuario %s recibió %d mensajes al reconectarse", username, messagesReceived)
		}(i)
	}

	wg.Wait()

	// Verificar estado final del hub
	userHistory := hub.GetUserHistory()
	if len(userHistory) != numUsers {
		t.Errorf("Se esperaban %d usuarios en el historial, pero se encontraron %d", numUsers, len(userHistory))
	}

	connectedCount := 0
	for _, status := range userHistory {
		if status.Connected {
			connectedCount++
		}
	}

	if connectedCount != numUsers {
		t.Errorf("Se esperaban %d usuarios conectados, pero se encontraron %d", numUsers, connectedCount)
	}
}

// TestWebSocketWithHistory prueba la funcionalidad completa via WebSocket con historial
func TestWebSocketWithHistory(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear servidor de prueba
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	}))
	defer server.Close()

	// Primer usuario se conecta
	wsURL1 := "ws" + strings.TrimPrefix(server.URL, "http") + "?username=user1"
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL1, nil)
	if err != nil {
		t.Fatalf("Error conectando primer usuario: %v", err)
	}
	defer conn1.Close()

	time.Sleep(200 * time.Millisecond)

	// Leer mensajes del sistema para user1
	for i := 0; i < 5; i++ {
		var msg interface{}
		if err := conn1.ReadJSON(&msg); err != nil {
			break
		}
	}

	// Primer usuario envía un mensaje
	testMessage := map[string]interface{}{
		"content":  "Mensaje de prueba",
		"hasImage": false,
	}

	if err := conn1.WriteJSON(testMessage); err != nil {
		t.Fatalf("Error enviando mensaje: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Leer el mensaje de vuelta
	var echoMsg Message
	if err := conn1.ReadJSON(&echoMsg); err != nil {
		t.Fatalf("Error leyendo eco del mensaje: %v", err)
	}

	// Cerrar primera conexión
	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	// Segundo usuario se conecta y envía mensaje
	wsURL2 := "ws" + strings.TrimPrefix(server.URL, "http") + "?username=user2"
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL2, nil)
	if err != nil {
		t.Fatalf("Error conectando segundo usuario: %v", err)
	}
	defer conn2.Close()

	time.Sleep(200 * time.Millisecond)

	// Leer mensajes del sistema para user2
	for i := 0; i < 5; i++ {
		var msg interface{}
		if err := conn2.ReadJSON(&msg); err != nil {
			break
		}
	}

	// Segundo usuario envía mensaje
	testMessage2 := map[string]interface{}{
		"content":  "Mensaje mientras user1 está desconectado",
		"hasImage": false,
	}

	if err := conn2.WriteJSON(testMessage2); err != nil {
		t.Fatalf("Error enviando segundo mensaje: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Primer usuario se reconecta
	conn1Reconnected, _, err := websocket.DefaultDialer.Dial(wsURL1, nil)
	if err != nil {
		t.Fatalf("Error reconectando primer usuario: %v", err)
	}
	defer conn1Reconnected.Close()

	time.Sleep(300 * time.Millisecond)

	// Verificar que user1 recibe historial
	historyReceived := false
	messagesReceived := 0

	// Leer hasta 10 mensajes o timeout
	for i := 0; i < 10; i++ {
		conn1Reconnected.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var msg map[string]interface{}
		if err := conn1Reconnected.ReadJSON(&msg); err != nil {
			break // Timeout o conexión cerrada
		}

		msgType, _ := msg["type"].(string)
		if msgType == "historyStart" {
			historyReceived = true
			t.Log("Usuario reconectado recibió notificación de inicio de historial")
		} else if msgType == "message" {
			content, _ := msg["content"].(string)
			if strings.Contains(content, "desconectado") {
				messagesReceived++
				t.Log("Usuario reconectado recibió mensaje perdido:", content)
			}
		}
	}

	if !historyReceived {
		t.Error("Usuario reconectado no recibió notificación de historial")
	}

	if messagesReceived != 1 {
		t.Errorf("Se esperaba 1 mensaje perdido, pero se recibieron %d", messagesReceived)
	}
}

// BenchmarkHistoryRetrieval benchmarks la recuperación de historial
func BenchmarkHistoryRetrieval(b *testing.B) {
	hub := NewHub()
	go hub.Run()

	// Llenar historial con muchos mensajes
	for i := 0; i < 1000; i++ {
		msg := NewMessage("benchuser", "Mensaje "+string(rune(i)))
		msgBytes, _ := json.Marshal(msg)
		hub.broadcast <- msgBytes
	}

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simular reconexión
		client := &Client{
			hub:      hub,
			send:     make(chan []byte, 1000),
			username: "benchuser",
		}

		// Marcar como usuario existente
		hub.mu.Lock()
		hub.userHistory["benchuser"] = &UserStatus{
			Username:         "benchuser",
			Connected:        false,
			LastMessageIndex: 500, // Simular que vio hasta el mensaje 500
		}
		hub.mu.Unlock()

		hub.register <- client
		time.Sleep(10 * time.Millisecond)
		hub.unregister <- client
	}
}
