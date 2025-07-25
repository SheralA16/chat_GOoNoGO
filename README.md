# 🚀 GO O NO GO - Chat en Tiempo Real con Soporte para Imágenes

**Desarrollado por: JUNIOR_ALVINES y SheralA16**  
**GitHub: [JUNMPI/realtime-chat](https://github.com/JUNMPI/realtime-chat)**  
**GitHub: [SheralA16/realtime-chat](https://github.com/SheralA16/realtime-chat)**

Un sistema de chat en tiempo real desarrollado en Go con WebSockets, interfaz Bootstrap y **soporte completo para envío de imágenes**.

## 🌟 Características

✅ **Chat en tiempo real** con WebSockets  
✅ **Envío de imágenes** - JPEG, PNG, GIF, WebP (máx. 5MB)  
✅ **Vista previa de imágenes** - Modal con zoom y descarga  
✅ **Arrastrar y soltar** - Interfaz intuitiva para subir imágenes  
✅ **Control de usuarios duplicados** - No permite nombres repetidos  
✅ **Historial persistente** - Los mensajes no se borran durante la sesión  
✅ **Interfaz moderna** con Bootstrap 5  
✅ **Lista de usuarios** conectados/desconectados  
✅ **Mensajes del sistema** para conexiones  
✅ **Responsive design** - Funciona en móviles  
✅ **Deploy en Railway** - Fácil y gratis  
✅ **Manejo robusto de errores** - Sin pérdida de conversación  

## 🖼️ Características de Imágenes

### **Formatos Soportados:**
- **JPEG/JPG** - Fotos comprimidas
- **PNG** - Imágenes con transparencia
- **GIF** - Imágenes animadas
- **WebP** - Formato moderno optimizado

### **Funcionalidades:**
- 📤 **Subida por arrastrar y soltar**
- 📤 **Selector de archivos tradicional**
- 🔍 **Vista previa antes de enviar**
- 💬 **Captions opcionales para imágenes**
- 🖼️ **Modal de vista completa**
- 💾 **Descarga de imágenes recibidas**
- 📏 **Información de tamaño y formato**
- ⚡ **Validación en tiempo real**
- 🚫 **Sin pérdida de historial** - Las conversaciones se mantienen intactas

### **Limitaciones:**
- 📦 **Tamaño máximo:** 5MB por imagen
- 🔒 **Solo tipos permitidos:** JPEG, PNG, GIF, WebP
- 🌐 **Base64:** Las imágenes se envían codificadas

## 📁 Estructura del Proyecto

```
realtime-chat/
├── main.go              # Servidor HTTP configurado para Railway
├── hub.go               # Gestión central de clientes y mensajes
├── client.go            # Manejo de clientes WebSocket individuales (✅ MEJORADO)
├── message.go           # Estructuras de mensajes
├── image.go             # Funciones para manejo de imágenes (⭐ NUEVO)
├── websocket.go         # Configuración WebSocket
├── index.html           # Frontend con Bootstrap (✅ CORREGIDO)
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

2. **Actualizar archivos con las versiones corregidas:**
   - ✅ `index.html` con historial persistente corregido
   - ✅ `client.go` con soporte para imágenes
   - ✅ `message.go` con campos de imagen
   - ✅ `image.go` con funciones de validación
   - ✅ `chat_test.go` con tests de imágenes

3. **Commit y push:**
```bash
git add .
git commit -m "🐛 Arreglar historial persistente y mejorar funcionalidad de imágenes"
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

### **Paso 4: Probar Funcionalidad Completa**

¡Comparte la URL con tus amigos y prueben todas las características!

## 🧪 Pruebas Locales

Para probar en tu máquina antes de deployar:

```bash
# Ejecutar el servidor
go run *.go

# Abrir en navegador
http://localhost:8080
```

### **Probar Funcionalidad Completa:**
1. **Conectarte con un nombre de usuario único**
2. **Enviar mensajes de texto normales**
3. **Hacer clic en el botón de imagen** 📷
4. **Arrastrar una imagen o hacer clic para seleccionar**
5. **Añadir un caption opcional**
6. **Enviar la imagen**
7. **Continuar chateando** - ¡El historial se mantiene!
8. **Hacer clic en imágenes recibidas para vista completa**

## 🎯 Funcionalidades del Chat

### **Control de Usuarios:**
- ✅ Nombres únicos (no permite duplicados)
- ✅ Validación de formato (solo letras, números, - y _)
- ✅ Longitud entre 2-20 caracteres

### **Mensajes de Texto:**
- ✅ Envío en tiempo real
- ✅ Timestamps automáticos
- ✅ Notificaciones de conexión/desconexión
- ✅ Diferenciación visual (propios vs otros)

### **Mensajes de Imagen:**
- ✅ Subida por arrastrar y soltar
- ✅ Vista previa antes de enviar
- ✅ Captions opcionales
- ✅ Modal de vista completa
- ✅ Descarga de imágenes
- ✅ Validación de formato y tamaño
- ✅ Información de archivo (nombre, tamaño)
- ✅ **Historial persistente** - Sin pérdida de conversación

### **Lista de Usuarios:**
- ✅ Estado online/offline
- ✅ Tiempo de conexión
- ✅ Última vez visto
- ✅ Contador de usuarios activos

## 🔧 Tecnologías Utilizadas

- **Backend:** Go 1.24.4
- **WebSockets:** Gorilla WebSocket
- **Frontend:** HTML5, Bootstrap 5, JavaScript ES6
- **Imágenes:** Base64 encoding, File API, Drag & Drop API
- **Deploy:** Railway
- **Icons:** Bootstrap Icons

## 📱 Responsive Design

El chat funciona perfectamente en:
- 💻 **Desktop** (1200px+) - Vista completa con sidebar
- 📱 **Tablet** (768px - 1199px) - Layout adaptativo
- 📱 **Mobile** (< 768px) - Interfaz optimizada para móviles

## 🛠️ Desarrollo

### **Ejecutar tests:**
```bash
# Tests básicos
go test -v

# Tests con detección de race conditions
go test -race -v

# Tests específicos de imágenes
go test -v -run TestImage

# Benchmarks de rendimiento
go test -bench=.
```

### **Estructura de archivos Go:**
- `main.go` - Servidor HTTP y configuración Railway
- `hub.go` - Centro de gestión de clientes
- `client.go` - Lógica de clientes individuales (✅ con soporte de imágenes)
- `message.go` - Estructuras de datos (✅ con campos de imagen)
- `image.go` - Funciones de validación y procesamiento de imágenes
- `websocket.go` - Configuración WebSocket

## 🎨 Personalización

### **Cambiar límites de imagen:**
En `image.go`:
```go
const (
    MaxImageSize = 5 * 1024 * 1024 // Cambiar tamaño máximo
)

var allowedImageTypes = map[string]bool{
    "image/jpeg": true,
    "image/png":  true,
    // Añadir o quitar tipos
}
```

### **Modificar interfaz:**
En `index.html`:
```css
.message-image {
    max-width: 300px;  /* Tamaño de vista previa */
    max-height: 200px;
}
```

## 🚨 Solución de Problemas

### **Error: "Imagen demasiado grande"**
- ✅ Reduce el tamaño de la imagen (máx. 5MB)
- ✅ Usa herramientas de compresión de imagen
- ✅ Convierte a formatos más eficientes (WebP, JPEG)

### **Error: "Tipo de imagen no soportado"**
- ✅ Usa solo: JPEG, PNG, GIF, WebP
- ✅ Verifica la extensión del archivo
- ✅ Algunos formatos antiguos pueden no funcionar

### **Imágenes no se cargan:**
- ✅ Verifica la conexión a internet
- ✅ Revisa la consola del navegador (F12)
- ✅ Asegúrate de que el archivo no esté corrupto

### **El historial se borra:** ✅ **SOLUCIONADO**
- ✅ **Problema corregido** en la versión actual
- ✅ Ahora el historial es **persistente durante toda la sesión**
- ✅ Los mensajes **no se borran** al enviar imágenes

### **Interfaz de imagen no aparece:**
- ✅ Asegúrate de estar conectado al chat
- ✅ Verifica que JavaScript esté habilitado
- ✅ Usa un navegador moderno (Chrome, Firefox, Safari)

## 📊 Logs y Monitoreo

Railway proporciona logs en tiempo real:
```
🚀 GO O NO GO - Servidor de chat iniciado
📡 Puerto: 34567
💬 WebSocket endpoint: /ws
🖼️ Soporte para imágenes habilitado (máx. 5MB)
✅ Servidor listo para recibir conexiones...
📜 Mensaje agregado al historial local. Total: 15
🖼️ Imagen de 'JUNIOR_ALVINES' enviada al hub (2.3 MB)
```

## 🌍 Variables de Entorno

Railway maneja automáticamente:
- `PORT` - Puerto asignado dinámicamente
- Protocolo HTTPS/WSS para producción

## 🔒 Seguridad

- ✅ Validación de entrada en frontend y backend
- ✅ Escape de HTML para prevenir XSS
- ✅ Validación de tipos MIME y magic numbers
- ✅ Límites de tamaño de archivo
- ✅ Rate limiting natural por WebSocket
- ✅ Conexiones HTTPS/WSS en producción
- ✅ Historial seguro sin pérdida de datos

## 🎯 Próximas Funcionalidades

- [ ] Salas de chat múltiples
- [ ] Historial de mensajes persistente en base de datos
- [ ] Autenticación con GitHub
- [ ] Compresión automática de imágenes
- [ ] Soporte para más formatos (videos, documentos)
- [ ] Stickers y emojis personalizados
- [ ] Comandos especiales (/help, /users, /clear, etc.)
- [ ] Notificaciones push
- [ ] Modo oscuro/claro

## 🔄 Migración desde Versión Anterior

Si tienes la versión anterior:

1. **Backup de datos importantes**
2. **Actualizar todos los archivos con las nuevas versiones**
3. **Ejecutar tests para verificar funcionamiento:**
   ```bash
   go test -v
   ```
4. **Redeploy en Railway**

## ✨ Novedades en Esta Versión

### **🐛 Correcciones:**
- ✅ **Historial persistente** - Los mensajes ya no se borran
- ✅ **Sin duplicados** - Cada mensaje aparece solo una vez
- ✅ **Mejor gestión de memoria** - Optimización del frontend
- ✅ **Logs mejorados** - Mejor debugging y monitoreo

### **🚀 Mejoras:**
- ✅ **Flujo optimizado** - Menos operaciones redundantes
- ✅ **IDs únicos** - Sistema robusto de identificación de mensajes
- ✅ **Validación mejorada** - Mejor detección de duplicados
- ✅ **Experiencia de usuario** - Chat más fluido y confiable

## 📞 Soporte

**Desarrolladores:** JUNIOR_ALVINES & SheralA16  
**GitHub:** [github.com/JUNMPI](https://github.com/JUNMPI)  
**Proyecto:** [realtime-chat](https://github.com/JUNMPI/realtime-chat)

Para reportar bugs o sugerir mejoras, crea un Issue en GitHub.

### **Issues Comunes:**
- **Imágenes grandes:** Reporta problemas con archivos específicos
- **Compatibilidad:** Menciona navegador y sistema operativo
- **Performance:** Incluye detalles de red y dispositivo
- **Historial:** ✅ **Ya no es un problema** - Corregido en esta versión

### **Changelog:**
- **v1.2.0** - ✅ Historial persistente corregido
- **v1.1.0** - 🖼️ Soporte completo para imágenes
- **v1.0.0** - 💬 Chat básico en tiempo real

---

**¡Disfruta tu chat en tiempo real con imágenes y historial persistente! 🚀💬🖼️**