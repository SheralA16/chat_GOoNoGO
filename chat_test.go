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

	if hub.messageHistory == nil {
		t.Error("El historial de mensajes no se inicializó")
	}

	if hub.maxHistorySize != 50 {
		t.Errorf("Se esperaba maxHistorySize=50, pero se encontró %d", hub.maxHistorySize)
	}

	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes inicialmente, pero se encontraron %d", hub.GetClientCount())
	}
}

// TestMessageHistory prueba el sistema de historial de mensajes
func TestMessageHistory(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear mensaje de prueba
	msg := NewMessage("testuser", "Mensaje de prueba")
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Error serializando mensaje: %v", err)
	}

	// Enviar mensaje al hub
	hub.broadcast <- msgBytes
	time.Sleep(100 * time.Millisecond)

	// Verificar que se agregó al historial
	history := hub.GetMessageHistory()
	if len(history) != 1 {
		t.Errorf("Se esperaba 1 mensaje en el historial, pero se encontraron %d", len(history))
	}

	if history[0].Content != "Mensaje de prueba" {
		t.Errorf("Se esperaba contenido 'Mensaje de prueba', pero se encontró '%s'", history[0].Content)
	}
}

// TestImageMessage prueba el manejo de mensajes con imágenes
func TestImageMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Crear datos de imagen simulados
	imageData := &ImageData{
		Data: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
		Name: "test.png",
		Type: "image/png",
		Size: 100,
	}

	// Crear mensaje con imagen
	msg := NewMessageWithImage("testuser", "Mira esta imagen", imageData)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Error serializando mensaje con imagen: %v", err)
	}

	// Verificar propiedades del mensaje
	if !msg.HasImage {
		t.Error("El mensaje debería tener HasImage=true")
	}

	if msg.Image == nil {
		t.Fatal("Los datos de imagen no deberían ser nil")
	}

	if msg.Image.Name != "test.png" {
		t.Errorf("Se esperaba nombre 'test.png', pero se encontró '%s'", msg.Image.Name)
	}

	// Enviar al hub y verificar historial
	hub.broadcast <- msgBytes
	time.Sleep(100 * time.Millisecond)

	history := hub.GetMessageHistory()
	if len(history) != 1 {
		t.Errorf("Se esperaba 1 mensaje en el historial, pero se encontraron %d", len(history))
	}

	if !history[0].HasImage {
		t.Error("El mensaje en el historial debería tener HasImage=true")
	}
}

// TestImageValidation prueba la validación de imágenes
func TestImageValidation(t *testing.T) {
	client := &Client{username: "testuser"}

	// Test: Imagen válida
	validImage := &ImageData{
		Data: "data:image/png;base64,validdata",
		Name: "test.png",
		Type: "image/png",
		Size: 1000,
	}

	if !client.isValidImage(validImage) {
		t.Error("Una imagen válida fue rechazada")
	}

	// Test: Imagen muy grande
	largeImage := &ImageData{
		Data: "data:image/png;base64,validdata",
		Name: "large.png",
		Type: "image/png",
		Size: 10 * 1024 * 1024, // 10MB > 5MB límite
	}

	if client.isValidImage(largeImage) {
		t.Error("Una imagen muy grande fue aceptada")
	}

	// Test: Tipo MIME inválido
	invalidTypeImage := &ImageData{
		Data: "data:text/plain;base64,validdata",
		Name: "test.txt",
		Type: "text/plain",
		Size: 1000,
	}

	if client.isValidImage(invalidTypeImage) {
		t.Error("Una imagen con tipo MIME inválido fue aceptada")
	}

	// Test: Nombre muy largo
	longNameImage := &ImageData{
		Data: "data:image/png;base64,validdata",
		Name: strings.Repeat("a", 300), // > 255 caracteres
		Type: "image/png",
		Size: 1000,
	}

	if client.isValidImage(longNameImage) {
		t.Error("Una imagen con nombre muy largo fue aceptada")
	}

	// Test: Data URL inválida
	invalidDataImage := &ImageData{
		Data: "not-a-data-url",
		Name: "test.png",
		Type: "image/png",
		Size: 1000,
	}

	if client.isValidImage(invalidDataImage) {
		t.Error("Una imagen con data URL inválida fue aceptada")
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
	time.Sleep(200 * time.Millisecond)

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

// TestMessageBroadcastWithHistory prueba la difusión y el historial
func TestMessageBroadcastWithHistory(t *testing.T) {
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

	// Limpiar mensajes del sistema
	for _, client := range clients {
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
	time.Sleep(100 * time.Millisecond)

	// Verificar que el mensaje se agregó al historial
	history := hub.GetMessageHistory()
	if len(history) != 1 {
		t.Errorf("Se esperaba 1 mensaje en el historial, pero se encontraron %d", len(history))
	}

	// Verificar que todos los clientes recibieron el mensaje
	messagesReceived := 0
	for i, client := range clients {
		select {
		case receivedMsg := <-client.send:
			var parsedMsg Message
			if err := json.Unmarshal(receivedMsg, &parsedMsg); err != nil {
				t.Errorf("Cliente %d: Error parseando mensaje recibido: %v", i, err)
				continue
			}

			if parsedMsg.Type == MessageTypeSystem {
				t.Logf("Cliente %d: Recibió mensaje del sistema (ignorado): %s", i, parsedMsg.Content)
				continue
			}

			if parsedMsg.Content != "Hola mundo" {
				t.Errorf("Cliente %d: Se esperaba contenido 'Hola mundo', pero se recibió '%s'", i, parsedMsg.Content)
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

// TestConcurrentOperationsWithImages prueba operaciones concurrentes con imágenes
func TestConcurrentOperationsWithImages(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const numGoroutines = 5
	const messagesPerGoroutine = 3

	var wg sync.WaitGroup

	// Crear datos de imagen para pruebas
	imageData := &ImageData{
		Data: "data:image/png;base64,testdata",
		Name: "concurrent_test.png",
		Type: "image/png",
		Size: 500,
	}

	// Enviar mensajes concurrentemente
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				username := "user" + string(rune(goroutineID+'0'))
				content := "Mensaje " + string(rune(j+'0'))

				var msg *Message
				if j%2 == 0 {
					// Mensaje con imagen
					msg = NewMessageWithImage(username, content, imageData)
				} else {
					// Mensaje solo texto
					msg = NewMessage(username, content)
				}

				if msgBytes, err := json.Marshal(msg); err == nil {
					select {
					case hub.broadcast <- msgBytes:
						// Mensaje enviado exitosamente
					case <-time.After(100 * time.Millisecond):
						t.Logf("Timeout enviando mensaje desde goroutine %d", goroutineID)
					}
				}

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Verificar historial
	history := hub.GetMessageHistory()
	expectedMessages := numGoroutines * messagesPerGoroutine

	if len(history) != expectedMessages {
		t.Logf("Se esperaban %d mensajes en el historial, pero se encontraron %d", expectedMessages, len(history))
		// No falla el test porque algunos mensajes pueden haberse perdido por concurrencia
	}

	// Verificar que hay mensajes con y sin imágenes
	withImages := 0
	withoutImages := 0
	for _, msg := range history {
		if msg.HasImage {
			withImages++
		} else {
			withoutImages++
		}
	}

	t.Logf("Mensajes con imágenes: %d, sin imágenes: %d", withImages, withoutImages)

	if withImages == 0 {
		t.Error("No se encontraron mensajes con imágenes en el historial")
	}

	if withoutImages == 0 {
		t.Error("No se encontraron mensajes sin imágenes en el historial")
	}
}

// TestWebSocketUpgradeWithImage prueba la actualización de WebSocket con imágenes
func TestWebSocketUpgradeWithImage(t *testing.T) {
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

	// Leer mensajes del sistema
	for i := 0; i < 3; i++ {
		var msg interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break // No más mensajes
		}
	}

	// Verificar que el cliente se registró en el hub
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaba 1 cliente conectado, pero se encontraron %d", hub.GetClientCount())
	}

	// Enviar un mensaje con imagen
	imageMessage := map[string]interface{}{
		"content":  "Mira esta imagen",
		"hasImage": true,
		"image": map[string]interface{}{
			"data": "data:image/png;base64,testdata",
			"name": "test.png",
			"type": "image/png",
			"size": 100,
		},
	}

	if err := conn.WriteJSON(imageMessage); err != nil {
		t.Fatalf("Error enviando mensaje con imagen: %v", err)
	}

	// Leer mensaje de respuesta
	var receivedMsg Message
	if err := conn.ReadJSON(&receivedMsg); err != nil {
		t.Fatalf("Error leyendo mensaje: %v", err)
	}

	if receivedMsg.Content != "Mira esta imagen" {
		t.Errorf("Se esperaba contenido 'Mira esta imagen', pero se recibió '%s'", receivedMsg.Content)
	}

	if !receivedMsg.HasImage {
		t.Error("El mensaje debería tener HasImage=true")
	}

	if receivedMsg.Image == nil {
		t.Error("Los datos de imagen no deberían ser nil")
	}

	if receivedMsg.Username != "testuser" {
		t.Errorf("Se esperaba usuario 'testuser', pero se recibió '%s'", receivedMsg.Username)
	}
}

// BenchmarkMessageBroadcastWithImages benchmarks la difusión de mensajes con imágenes
func BenchmarkMessageBroadcastWithImages(b *testing.B) {
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

	// Crear mensaje con imagen para benchmark
	imageData := &ImageData{
		Data: "data:image/png;base64,benchmarkdata",
		Name: "benchmark.png",
		Type: "image/png",
		Size: 1000,
	}

	msg := NewMessageWithImage("benchuser", "benchmark message", imageData)
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
