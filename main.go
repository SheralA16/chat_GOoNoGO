package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Crear el hub de chat
	hub := NewHub()

	// Iniciar el hub en una goroutine separada
	go hub.Run()

	// Configurar rutas HTTP
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	// Servir archivos estáticos desde el directorio ./static/
	fs := http.FileServer(http.Dir("./static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// ⭐ RAILWAY: Obtener puerto de variable de entorno
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Puerto por defecto para desarrollo local
	}

	// Información de inicio
	log.Println("🚀 GO O NO GO - Servidor de chat iniciado")
	log.Printf("📡 Puerto: %s", port)
	log.Println("💬 WebSocket endpoint: /ws")
	log.Println("📁 Archivos estáticos servidos desde: ./static/")
	log.Println("✅ Servidor listo para recibir conexiones...")

	// ⭐ RAILWAY: Usar puerto dinámico
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("❌ Error iniciando servidor HTTP:", err)
	}
}

// serveHome sirve la página principal del chat
func serveHome(w http.ResponseWriter, r *http.Request) {
	// Verificar que sea la ruta raíz
	if r.URL.Path != "/" {
		http.Error(w, "Página no encontrada", http.StatusNotFound)
		return
	}

	// Solo permitir método GET
	if r.Method != "GET" {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Servir el archivo index.html
	http.ServeFile(w, r, "index.html")
}
