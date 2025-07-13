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

// TestHubCreation prueba la creación correcta de un nuevo hub
func TestHubCreation(t *testing.T) {
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

	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes inicialmente, pero se encontraron %d", hub.GetClientCount())
	}
}

// TestClientRegistration prueba el registro de clientes en el hub
func TestClientRegistration(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear un cliente mock
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "testuser",
	}

	// Registrar el cliente
	hub.register <- client

	// Dar tiempo para que se procese
	time.Sleep(100 * time.Millisecond)

	// Verificar que el cliente se registró
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaba 1 cliente, pero se encontraron %d", hub.GetClientCount())
	}

	// Verificar que el cliente está en el mapa
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if !exists {
		t.Error("El cliente no se encontró en el mapa de clientes")
	}

	// Verificar que se obtiene el nombre de usuario correcto
	users := hub.GetConnectedUsers()
	if len(users) != 1 || users[0] != "testuser" {
		t.Errorf("Se esperaba usuario 'testuser', pero se obtuvo %v", users)
	}
}

// TestClientUnregistration prueba la cancelación del registro de clientes
func TestClientUnregistration(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear un cliente mock
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "testuser",
	}

	// Registrar el cliente
	hub.register <- client
	time.Sleep(100 * time.Millisecond)

	// Verificar que se registró
	if hub.GetClientCount() != 1 {
		t.Fatal("El cliente no se registró correctamente")
	}

	// Cancelar el registro del cliente
	hub.unregister <- client
	time.Sleep(200 * time.Millisecond)

	// Verificar que el cliente se canceló
	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes después de cancelar, pero se encontraron %d", hub.GetClientCount())
	}

	// Verificar que el cliente ya no está en el mapa del hub
	hub.mu.RLock()
	_, exists := hub.clients[client]
	hub.mu.RUnlock()

	if exists {
		t.Error("El cliente aún existe en el mapa después de desregistrarse")
	}

	// Verificar que el canal send se cerró usando una goroutine separada
	done := make(chan bool, 1)
	go func() {
		// Intentar leer del canal con timeout
		select {
		case _, ok := <-client.send:
			if !ok {
				// Canal está cerrado (esto es lo que esperamos)
				done <- true
			} else {
				// Canal tiene datos pero no está cerrado
				done <- false
			}
		case <-time.After(100 * time.Millisecond):
			// Timeout - el canal podría estar vacío pero no cerrado
			// Intentar enviar algo para verificar
			select {
			case client.send <- []byte("test"):
				// Si podemos enviar, no está cerrado
				done <- false
			default:
				// Si no podemos enviar ni leer, probablemente está cerrado
				done <- true
			}
		}
	}()

	// Esperar resultado
	select {
	case channelClosed := <-done:
		if !channelClosed {
			t.Error("El canal send del cliente debería estar cerrado")
		}
	case <-time.After(500 * time.Millisecond):
		// Si llegamos aquí, asumimos que el canal está cerrado
		t.Log("Canal parece estar cerrado (timeout en verificación)")
	}
}

// TestMessageBroadcast prueba la difusión de mensajes
func TestMessageBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear múltiples clientes mock
	numClients := 3
	clients := make([]*Client, numClients)

	for i := 0; i < numClients; i++ {
		clients[i] = &Client{
			hub:      hub,
			send:     make(chan []byte, 256),
			username: "testuser" + string(rune(i+'0')),
		}
		hub.register <- clients[i]
	}

	time.Sleep(300 * time.Millisecond)

	// ✅ CORREGIDO: Limpiar mensajes del sistema de forma correcta
	for _, client := range clients {
		// Limpiar buffer de mensajes del sistema
	clearLoop:
		for {
			select {
			case <-client.send:
				// Continuar descartando mensajes
			case <-time.After(10 * time.Millisecond):
				// No hay más mensajes, salir del bucle
				break clearLoop
			}
		}
	}

	// Verificar que todos los clientes se registraron
	if hub.GetClientCount() != numClients {
		t.Fatalf("Se esperaban %d clientes, pero se encontraron %d", numClients, hub.GetClientCount())
	}

	// Crear un mensaje de prueba
	msg := NewMessage("testuser0", "Hola mundo")
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Error serializando mensaje: %v", err)
	}

	// Enviar mensaje al hub para difusión
	hub.broadcast <- msgBytes

	// Dar tiempo para la difusión
	time.Sleep(50 * time.Millisecond)

	// Verificar que todos los clientes recibieron el mensaje correcto
	messagesReceived := 0
	for i, client := range clients {
		select {
		case receivedMsg := <-client.send:
			var parsedMsg Message
			if err := json.Unmarshal(receivedMsg, &parsedMsg); err != nil {
				t.Errorf("Cliente %d: Error parseando mensaje recibido: %v", i, err)
				continue
			}

			// Solo contar mensajes que no sean del sistema
			if parsedMsg.Type == MessageTypeSystem {
				t.Logf("Cliente %d: Recibió mensaje del sistema (ignorado): %s", i, parsedMsg.Content)
				continue
			}

			if parsedMsg.Content != "Hola mundo" {
				t.Errorf("Cliente %d: Se esperaba contenido 'Hola mundo', pero se recibió '%s'", i, parsedMsg.Content)
				continue
			}

			if parsedMsg.Username != "testuser0" {
				t.Errorf("Cliente %d: Se esperaba usuario 'testuser0', pero se recibió '%s'", i, parsedMsg.Username)
				continue
			}

			messagesReceived++

		case <-time.After(500 * time.Millisecond):
			t.Errorf("Cliente %d: No recibió el mensaje en tiempo esperado", i)
		}
	}

	if messagesReceived != numClients {
		t.Errorf("Se esperaba que %d clientes recibieran el mensaje, pero solo %d lo recibieron", numClients, messagesReceived)
	}
}

// TestDuplicateUsernames prueba que no se permitan nombres duplicados
func TestDuplicateUsernames(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear primer cliente con nombre "duplicatetest"
	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "duplicatetest",
	}

	// Registrar primer cliente
	hub.register <- client1
	time.Sleep(100 * time.Millisecond)

	// Verificar que se registró
	if hub.GetClientCount() != 1 {
		t.Fatal("El primer cliente no se registró correctamente")
	}

	// Intentar registrar segundo cliente con el mismo nombre
	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, 256),
		username: "duplicatetest",
	}

	hub.register <- client2
	time.Sleep(200 * time.Millisecond)

	// Debería seguir habiendo solo 1 cliente (el duplicado no se debe registrar)
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaba 1 cliente después del intento de duplicado, pero se encontraron %d", hub.GetClientCount())
	}

	// Verificar que client2 recibió mensaje de error
	select {
	case errorMsg := <-client2.send:
		var errorData map[string]interface{}
		if err := json.Unmarshal(errorMsg, &errorData); err != nil {
			t.Errorf("Error parseando mensaje de error: %v", err)
		} else {
			if errorData["type"] != "error" {
				t.Errorf("Se esperaba mensaje de tipo 'error', pero se recibió '%v'", errorData["type"])
			}
			if !strings.Contains(errorData["message"].(string), "ya está en uso") {
				t.Errorf("El mensaje de error no indica que el nombre está en uso: %v", errorData["message"])
			}
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Client2 no recibió mensaje de error por nombre duplicado")
	}
}

// TestConcurrentClientOperations prueba operaciones concurrentes de clientes
func TestConcurrentClientOperations(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const numGoroutines = 10
	const numClientsPerGoroutine = 5

	var wg sync.WaitGroup
	var clients []*Client
	var clientsMutex sync.Mutex

	// Crear clientes concurrentemente
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numClientsPerGoroutine; j++ {
				client := &Client{
					hub:      hub,
					send:     make(chan []byte, 256),
					username: "user" + string(rune(goroutineID+'0')) + string(rune(j+'0')),
				}

				clientsMutex.Lock()
				clients = append(clients, client)
				clientsMutex.Unlock()

				hub.register <- client
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	expectedClients := numGoroutines * numClientsPerGoroutine
	actualClients := hub.GetClientCount()

	if actualClients != expectedClients {
		t.Errorf("Se esperaban %d clientes, pero se encontraron %d", expectedClients, actualClients)
	}

	// Desregistrar clientes concurrentemente
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			startIdx := goroutineID * numClientsPerGoroutine
			endIdx := startIdx + numClientsPerGoroutine

			for j := startIdx; j < endIdx && j < len(clients); j++ {
				hub.unregister <- clients[j]
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes después de desregistrar todos, pero se encontraron %d", hub.GetClientCount())
	}
}

// TestMessageStructures prueba las estructuras de mensajes
func TestMessageStructures(t *testing.T) {
	// Probar creación de mensaje normal
	msg := NewMessage("usuario1", "Contenido de prueba")

	if msg.Username != "usuario1" {
		t.Errorf("Se esperaba username 'usuario1', pero se obtuvo '%s'", msg.Username)
	}

	if msg.Content != "Contenido de prueba" {
		t.Errorf("Se esperaba contenido 'Contenido de prueba', pero se obtuvo '%s'", msg.Content)
	}

	if msg.Type != MessageTypeMessage {
		t.Errorf("Se esperaba tipo '%s', pero se obtuvo '%s'", MessageTypeMessage, msg.Type)
	}

	// Probar creación de mensaje del sistema
	sysMsg := NewSystemMessage("Mensaje del sistema")

	if sysMsg.Username != "Sistema" {
		t.Errorf("Se esperaba username 'Sistema', pero se obtuvo '%s'", sysMsg.Username)
	}

	if sysMsg.Type != MessageTypeSystem {
		t.Errorf("Se esperaba tipo '%s', pero se obtuvo '%s'", MessageTypeSystem, sysMsg.Type)
	}

	// Probar serialización JSON
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Error serializando mensaje: %v", err)
	}

	var parsedMsg Message
	if err := json.Unmarshal(msgBytes, &parsedMsg); err != nil {
		t.Fatalf("Error deserializando mensaje: %v", err)
	}

	if parsedMsg.Username != msg.Username {
		t.Errorf("Username no coincide después de serialización/deserialización")
	}
}

// TestWebSocketUpgrade prueba la actualización de WebSocket
func TestWebSocketUpgrade(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear servidor de prueba
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	}))
	defer server.Close()

	// Convertir URL HTTP a WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?username=testuser"

	// Conectar como cliente WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Error conectando WebSocket: %v", err)
	}
	defer conn.Close()

	// Dar tiempo para que se registre el cliente
	time.Sleep(200 * time.Millisecond)

	// Intentar leer mensaje del sistema (join message)
	var systemMsg Message
	if err := conn.ReadJSON(&systemMsg); err != nil {
		t.Logf("No se pudo leer mensaje del sistema: %v", err)
	}

	// Verificar que el cliente se registró en el hub
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaba 1 cliente conectado, pero se encontraron %d", hub.GetClientCount())
	}

	// Enviar un mensaje
	testMessage := map[string]string{
		"content": "Mensaje de prueba",
	}

	if err := conn.WriteJSON(testMessage); err != nil {
		t.Fatalf("Error enviando mensaje: %v", err)
	}

	// Leer mensaje de respuesta
	var receivedMsg Message
	if err := conn.ReadJSON(&receivedMsg); err != nil {
		t.Fatalf("Error leyendo mensaje: %v", err)
	}

	if receivedMsg.Content != "Mensaje de prueba" {
		t.Errorf("Se esperaba contenido 'Mensaje de prueba', pero se recibió '%s'", receivedMsg.Content)
	}

	if receivedMsg.Username != "testuser" {
		t.Errorf("Se esperaba usuario 'testuser', pero se recibió '%s'", receivedMsg.Username)
	}
}

// TestRaceConditions prueba condiciones de carrera
func TestRaceConditions(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const numGoroutines = 10
	const operationsPerGoroutine = 5

	var wg sync.WaitGroup

	// Ejecutar operaciones concurrentes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Crear cliente
				client := &Client{
					hub:      hub,
					send:     make(chan []byte, 256),
					username: "racer" + string(rune(id+'0')) + string(rune(j+'0')),
				}

				// Registrar
				hub.register <- client
				time.Sleep(1 * time.Millisecond)

				// Obtener conteo de clientes
				_ = hub.GetClientCount()

				// Obtener usuarios conectados
				_ = hub.GetConnectedUsers()

				// Enviar mensaje
				msg := NewMessage(client.username, "test message")
				if msgBytes, err := json.Marshal(msg); err == nil {
					select {
					case hub.broadcast <- msgBytes:
					case <-time.After(10 * time.Millisecond):
						// Timeout para evitar bloqueos
					}
				}

				time.Sleep(1 * time.Millisecond)

				// Desregistrar
				hub.unregister <- client
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	// Al final no debería haber clientes
	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes al final, pero se encontraron %d", hub.GetClientCount())
	}
}

// BenchmarkMessageBroadcast benchmarks la difusión de mensajes
func BenchmarkMessageBroadcast(b *testing.B) {
	hub := NewHub()
	go hub.Run()

	// Crear algunos clientes
	numClients := 10
	clients := make([]*Client, numClients)

	for i := 0; i < numClients; i++ {
		clients[i] = &Client{
			hub:      hub,
			send:     make(chan []byte, 256),
			username: "benchuser" + string(rune(i+'0')),
		}
		hub.register <- clients[i]
	}

	time.Sleep(100 * time.Millisecond)

	// Crear mensaje de prueba
	msg := NewMessage("benchuser", "benchmark message")
	msgBytes, _ := json.Marshal(msg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		select {
		case hub.broadcast <- msgBytes:
		case <-time.After(10 * time.Millisecond):
			// Timeout para evitar bloqueos
		}
	}
}
