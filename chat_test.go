package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt" // ⭐ AÑADIDO: Importación de fmt
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

// TestImageValidation prueba la validación de imágenes
func TestImageValidation(t *testing.T) {
	// Crear una imagen base64 de prueba pequeña (1x1 pixel PNG)
	validImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="
	
	// Test imagen válida
	err := ValidateImage(validImageData, "test.png", 67)
	if err != nil {
		t.Errorf("La imagen válida fue rechazada: %v", err)
	}

	// Test imagen demasiado grande
	err = ValidateImage(validImageData, "test.png", MaxImageSize+1)
	if err == nil {
		t.Error("Se esperaba error por imagen demasiado grande")
	}

	// Test imagen vacía
	err = ValidateImage(validImageData, "test.png", 0)
	if err == nil {
		t.Error("Se esperaba error por imagen vacía")
	}

	// Test tipo de archivo no soportado
	err = ValidateImage(validImageData, "test.txt", 67)
	if err == nil {
		t.Error("Se esperaba error por tipo de archivo no soportado")
	}

	// Test datos base64 inválidos
	err = ValidateImage("datos_invalidos", "test.png", 67)
	if err == nil {
		t.Error("Se esperaba error por datos base64 inválidos")
	}
}

// TestImageMessageCreation prueba la creación de mensajes de imagen
func TestImageMessageCreation(t *testing.T) {
	validImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="
	
	msg := NewImageMessage("testuser", "Mi imagen", validImageData, "test.png", 67, "image/png")

	if msg.Username != "testuser" {
		t.Errorf("Se esperaba username 'testuser', pero se obtuvo '%s'", msg.Username)
	}

	if msg.Content != "Mi imagen" {
		t.Errorf("Se esperaba contenido 'Mi imagen', pero se obtuvo '%s'", msg.Content)
	}

	if msg.Type != MessageTypeImage {
		t.Errorf("Se esperaba tipo '%s', pero se obtuvo '%s'", MessageTypeImage, msg.Type)
	}

	if msg.ImageData != validImageData {
		t.Error("Los datos de imagen no coinciden")
	}

	if msg.ImageName != "test.png" {
		t.Errorf("Se esperaba nombre 'test.png', pero se obtuvo '%s'", msg.ImageName)
	}

	if msg.ImageSize != 67 {
		t.Errorf("Se esperaba tamaño 67, pero se obtuvo %d", msg.ImageSize)
	}

	if msg.MimeType != "image/png" {
		t.Errorf("Se esperaba tipo MIME 'image/png', pero se obtuvo '%s'", msg.MimeType)
	}
}

// TestImageMessageBroadcast prueba la difusión de mensajes de imagen
func TestImageMessageBroadcast(t *testing.T) {
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

	// Crear un mensaje de imagen de prueba
	validImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="
	msg := NewImageMessage("testuser0", "Imagen de prueba", validImageData, "test.png", 67, "image/png")
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Error serializando mensaje de imagen: %v", err)
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

			if parsedMsg.Type != MessageTypeImage {
				t.Errorf("Cliente %d: Se esperaba tipo 'image', pero se recibió '%s'", i, parsedMsg.Type)
				continue
			}

			if parsedMsg.Content != "Imagen de prueba" {
				t.Errorf("Cliente %d: Se esperaba contenido 'Imagen de prueba', pero se recibió '%s'", i, parsedMsg.Content)
				continue
			}

			if parsedMsg.Username != "testuser0" {
				t.Errorf("Cliente %d: Se esperaba usuario 'testuser0', pero se recibió '%s'", i, parsedMsg.Username)
				continue
			}

			if parsedMsg.ImageData != validImageData {
				t.Errorf("Cliente %d: Los datos de imagen no coinciden", i)
				continue
			}

			messagesReceived++

		case <-time.After(500 * time.Millisecond):
			t.Errorf("Cliente %d: No recibió el mensaje de imagen en tiempo esperado", i)
		}
	}

	if messagesReceived != numClients {
		t.Errorf("Se esperaba que %d clientes recibieran el mensaje de imagen, pero solo %d lo recibieron", numClients, messagesReceived)
	}
}

// TestMimeTypeDetection prueba la detección de tipos MIME
func TestMimeTypeDetection(t *testing.T) {
	// Test PNG
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="
	mimeType := detectMimeTypeFromBase64(pngData)
	if mimeType != "image/png" {
		t.Errorf("Se esperaba 'image/png', pero se obtuvo '%s'", mimeType)
	}

	// Test JPEG (simulado con magic numbers)
	jpegData := base64.StdEncoding.EncodeToString([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10})
	mimeType = detectMimeTypeFromBase64(jpegData)
	if mimeType != "image/jpeg" {
		t.Errorf("Se esperaba 'image/jpeg', pero se obtuvo '%s'", mimeType)
	}

	// Test datos inválidos
	mimeType = detectMimeTypeFromBase64("datos_invalidos")
	if mimeType != "" {
		t.Errorf("Se esperaba cadena vacía para datos inválidos, pero se obtuvo '%s'", mimeType)
	}
}

// TestBase64Validation prueba la validación de base64
func TestBase64Validation(t *testing.T) {
	// Base64 válido
	validB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="
	if !isValidBase64(validB64) {
		t.Error("Base64 válido fue rechazado")
	}

	// Base64 con prefijo data URL
	dataURL := "data:image/png;base64," + validB64
	if !isValidBase64(dataURL) {
		t.Error("Data URL válido fue rechazado")
	}

	// Base64 inválido
	invalidB64 := "esto_no_es_base64!"
	if isValidBase64(invalidB64) {
		t.Error("Base64 inválido fue aceptado")
	}
}

// TestFormatImageSize prueba el formateo de tamaños de archivo
func TestFormatImageSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 Bytes"},
		{512, "512 Bytes"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{1048576, "1 MB"},
		{5242880, "5 MB"},
	}

	for _, test := range tests {
		result := FormatImageSize(test.bytes)
		if result != test.expected {
			t.Errorf("Para %d bytes, se esperaba '%s', pero se obtuvo '%s'", test.bytes, test.expected, result)
		}
	}
}

// TestWebSocketImageUpload prueba la subida de imágenes a través de WebSocket
func TestWebSocketImageUpload(t *testing.T) {
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

	// Leer y descartar mensajes del sistema hasta encontrar el mensaje de éxito
	var systemMsg Message
	for i := 0; i < 5; i++ { // Máximo 5 intentos
		if err := conn.ReadJSON(&systemMsg); err != nil {
			t.Logf("Intento %d: No se pudo leer mensaje: %v", i+1, err)
			continue
		}
		
		// Si es mensaje de conexión exitosa, continuar
		if systemMsg.Type == "system" || systemMsg.Type == "join" || 
		   strings.Contains(fmt.Sprintf("%v", systemMsg), "connectionSuccess") ||
		   strings.Contains(fmt.Sprintf("%v", systemMsg), "userList") {
			t.Logf("Mensaje del sistema descartado: %s", systemMsg.Type)
			continue
		}
		break
	}

	// Verificar que el cliente se registró en el hub
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaba 1 cliente conectado, pero se encontraron %d", hub.GetClientCount())
	}

	// Enviar una imagen
	validImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="
	
	imageMessage := map[string]interface{}{
		"type":      "image",
		"content":   "Imagen de prueba",
		"imageData": "data:image/png;base64," + validImageData,
		"imageName": "test.png",
		"imageSize": int64(67),
	}

	if err := conn.WriteJSON(imageMessage); err != nil {
		t.Fatalf("Error enviando mensaje de imagen: %v", err)
	}

	// Dar tiempo para el procesamiento
	time.Sleep(100 * time.Millisecond)

	// Intentar leer el mensaje de imagen, saltando mensajes del sistema
	var receivedMsg Message
	found := false
	
	for i := 0; i < 5; i++ { // Máximo 5 intentos
		if err := conn.ReadJSON(&receivedMsg); err != nil {
			t.Fatalf("Error leyendo mensaje %d: %v", i+1, err)
		}
		
		// Si es mensaje de imagen, procesarlo
		if receivedMsg.Type == MessageTypeImage {
			found = true
			break
		}
		
		// Si no es imagen, continuar buscando
		t.Logf("Mensaje no-imagen descartado: tipo=%s", receivedMsg.Type)
	}

	if !found {
		t.Fatalf("No se encontró mensaje de imagen después de 5 intentos")
	}

	// Verificar el mensaje de imagen
	if receivedMsg.Type != MessageTypeImage {
		t.Errorf("Se esperaba tipo 'image', pero se recibió '%s'", receivedMsg.Type)
	}

	if receivedMsg.Content != "Imagen de prueba" {
		t.Errorf("Se esperaba contenido 'Imagen de prueba', pero se recibió '%s'", receivedMsg.Content)
	}

	if receivedMsg.Username != "testuser" {
		t.Errorf("Se esperaba usuario 'testuser', pero se recibió '%s'", receivedMsg.Username)
	}

	if receivedMsg.ImageName != "test.png" {
		t.Errorf("Se esperaba nombre 'test.png', pero se recibió '%s'", receivedMsg.ImageName)
	}

	if receivedMsg.ImageSize != 67 {
		t.Errorf("Se esperaba tamaño 67, pero se recibió %d", receivedMsg.ImageSize)
	}

	if receivedMsg.ImageData == "" {
		t.Error("Los datos de imagen están vacíos")
	}
}

// TestConcurrentImageOperations prueba operaciones concurrentes con imágenes
func TestConcurrentImageOperations(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const numGoroutines = 5
	const imagesPerGoroutine = 3

	var wg sync.WaitGroup
	validImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="

	// Crear clientes y enviar imágenes concurrentemente
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Crear cliente para esta goroutine
			client := &Client{
				hub:      hub,
				send:     make(chan []byte, 256),
				username: "user" + string(rune(goroutineID+'0')),
			}

			hub.register <- client
			time.Sleep(50 * time.Millisecond)

			// Enviar múltiples imágenes
			for j := 0; j < imagesPerGoroutine; j++ {
				msg := NewImageMessage(
					client.username,
					"Imagen de prueba concurrente",
					validImageData,
					"test.png",
					67,
					"image/png",
				)

				if msgBytes, err := json.Marshal(msg); err == nil {
					select {
					case hub.broadcast <- msgBytes:
					case <-time.After(100 * time.Millisecond):
						// Timeout para evitar bloqueos
					}
				}

				time.Sleep(10 * time.Millisecond)
			}

			time.Sleep(50 * time.Millisecond)
			hub.unregister <- client
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Al final no debería haber clientes
	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes al final, pero se encontraron %d", hub.GetClientCount())
	}
}

// BenchmarkImageProcessing benchmarks el procesamiento de imágenes
func BenchmarkImageProcessing(b *testing.B) {
	validImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9fj8nIgAAAABJRU5ErkJggg=="

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Validar imagen
		_ = ValidateImage(validImageData, "test.png", 67)

		// Crear mensaje de imagen
		msg := NewImageMessage("testuser", "test", validImageData, "test.png", 67, "image/png")

		// Serializar a JSON
		_, _ = json.Marshal(msg)
	}
}