# 🚀 GO O NO GO - Chat en Tiempo Real

**Desarrollado por: JUNIOR_ALVINES y SheralA16**  
**GitHub: [SheralA16/realtime-chat](https://github.com/SheralA16/realtime-chat)**

Un sistema de chat en tiempo real desarrollado en Go con WebSockets, interfaz Bootstrap y control de usuarios duplicados.

## 🌟 Características

✅ **Chat en tiempo real** con WebSockets  
✅ **Control de usuarios duplicados** - No permite nombres repetidos  
✅ **Interfaz moderna** con Bootstrap 5  
✅ **Lista de usuarios** conectados/desconectados  
✅ **Mensajes del sistema** para conexiones  
✅ **Responsive design** - Funciona en móviles  
✅ **Deploy en Railway** - Fácil y gratis  

## 📁 Estructura del Proyecto

```
realtime-chat/
├── main.go              # Servidor HTTP configurado para Railway
├── hub.go               # Gestión central de clientes y mensajes
├── client.go            # Manejo de clientes WebSocket individuales
├── message.go           # Estructuras de mensajes
├── websocket.go         # Configuración WebSocket
├── index.html           # Frontend con Bootstrap
├── chat_test.go         # Tests unitarios
├── go.mod              # Dependencias de Go
├── go.sum              # Checksums de dependencias
└── README.md           # Esta documentación
```

## 🚀 Deploy en Railway (Paso a Paso)

### **Paso 1: Preparar el Repositorio**

1. **Clonar tu repositorio:**
```bash
git clone https://github.com/JUNMPI/realtime-chat.git
cd realtime-chat
```

2. **Reemplazar archivos con las versiones corregidas:**
   - Reemplaza `main.go` con la versión que incluye `PORT` variable
   - Reemplaza `index.html` con la versión que detecta protocolo automáticamente

3. **Commit y push:**
```bash
git add .
git commit -m "🚀 Configurar para Railway deployment"
git push origin main
```

### **Paso 2: Deploy en Railway**

1. **Ve a [Railway.app](https://railway.app)**
2. **Haz clic en "Start a New Project"**
3. **Selecciona "Deploy from GitHub repo"**
4. **Conecta tu cuenta de GitHub si no lo has hecho**
5. **Busca y selecciona `JUNMPI/realtime-chat`**
6. **¡Railway detecta automáticamente que es Go y empieza el deploy!**

### **Paso 3: Obtener tu URL**

Railway te asignará una URL como:
```
https://realtime-chat-production-xxxx.up.railway.app
```

### **Paso 4: Compartir con Amigos**

¡Comparte la URL con tus amigos y prueben el chat juntos!

## 🧪 Pruebas Locales

Para probar en tu máquina antes de deployar:

```bash
# Ejecutar el servidor
go run *.go

# Abrir en navegador
http://localhost:8080
```

## 🎯 Funcionalidades del Chat

### **Control de Usuarios:**
- ✅ Nombres únicos (no permite duplicados)
- ✅ Validación de formato (solo letras, números, - y _)
- ✅ Longitud entre 2-20 caracteres

### **Mensajes:**
- ✅ Envío en tiempo real
- ✅ Timestamps automáticos
- ✅ Notificaciones de conexión/desconexión
- ✅ Diferenciación visual (propios vs otros)

### **Lista de Usuarios:**
- ✅ Estado online/offline
- ✅ Tiempo de conexión
- ✅ Última vez visto
- ✅ Contador de usuarios activos

## 🔧 Tecnologías Utilizadas

- **Backend:** Go 1.24.4
- **WebSockets:** Gorilla WebSocket
- **Frontend:** HTML5, Bootstrap 5, JavaScript ES6
- **Deploy:** Railway
- **Icons:** Bootstrap Icons

## 📱 Responsive Design

El chat funciona perfectamente en:
- 💻 **Desktop** (1200px+)
- 📱 **Tablet** (768px - 1199px)
- 📱 **Mobile** (< 768px)

## 🛠️ Desarrollo

### **Ejecutar tests:**
```bash
go test -v
go test -race -v  # Con detección de race conditions
```

### **Estructura de archivos Go:**
- `main.go` - Servidor HTTP y configuración Railway
- `hub.go` - Centro de gestión de clientes
- `client.go` - Lógica de clientes individuales
- `websocket.go` - Configuración WebSocket
- `message.go` - Estructuras de datos

## 🎨 Personalización

### **Cambiar colores:**
Modifica las variables CSS en `index.html`:
```css
.gradient-bg {
    background: linear-gradient(135deg, #198754 0%, #20c997 100%);
}
```

### **Modificar límites:**
En `websocket.go` y `index.html`:
```go
// Longitud de nombres de usuario
if len(username) < 2 || len(username) > 20 {
    return false
}
```

## 🚨 Solución de Problemas

### **Error: "No se pudo conectar al servidor"**
- ✅ Verifica que Railway haya deployado correctamente
- ✅ Revisa los logs en Railway dashboard
- ✅ Asegúrate de usar HTTPS/WSS en producción

### **Error: "Nombre ya está en uso"**
- ✅ Es normal - el sistema funciona correctamente
- ✅ Prueba con otro nombre de usuario

### **No aparecen otros usuarios:**
- ✅ Abre múltiples pestañas para probar
- ✅ Usa nombres diferentes en cada pestaña

## 📊 Logs y Monitoreo

Railway proporciona logs en tiempo real:
```
🚀 GO O NO GO - Servidor de chat iniciado
📡 Puerto: 34567
💬 WebSocket endpoint: /ws
✅ Servidor listo para recibir conexiones...
✅ Cliente 'JUNIOR_ALVINES' conectado exitosamente. Total de clientes: 1
```

## 🌍 Variables de Entorno

Railway maneja automáticamente:
- `PORT` - Puerto asignado dinámicamente
- Protocolo HTTPS/WSS para producción

## 🔒 Seguridad

- ✅ Validación de entrada en frontend y backend
- ✅ Escape de HTML para prevenir XSS
- ✅ Rate limiting natural por WebSocket
- ✅ Conexiones HTTPS/WSS en producción

## 🎯 Próximas Funcionalidades

- [ ] Salas de chat múltiples
- [ ] Envío de archivos/imágenes
- [ ] Historial de mensajes persistente
- [ ] Autenticación con GitHub
- [ ] Temas personalizables
- [ ] Comandos especiales (/help, /users, etc.)

## 📞 Soporte

**Desarrollador:** JUNIOR_ALVINES 
**Desarrollador:** SheralA16
**GitHub:** [github.com/JUNMPI](https://github.com/JUNMPI)  
**Proyecto:** [realtime-chat](https://github.com/JUNMPI/realtime-chat)

Para reportar bugs o sugerir mejoras, crea un Issue en GitHub.

---

**¡Disfruta tu chat en tiempo real! 🚀💬**