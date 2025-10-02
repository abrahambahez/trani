# PRD: Trani - Sistema de Transcripción y Notas Inteligentes

## 1. Visión General

Herramienta CLI minimalista para grabar audio del sistema, transcribirlo con Whisper.cpp, y usar Claude API para generar documentación estructurada que combina transcripción automática con notas manuales opcionales.

**Filosofía MVP:** Máximo valor con mínimo esfuerzo. Sin complicaciones innecesarias.

## 2. Arquitectura del Sistema

```
~/trani/
├── trani                    # Script ejecutable principal
├── config.json              # Configuración
├── sessions/
│   └── YYYY-MM-DD-titulo/
│       ├── transcripcion.txt
│       ├── notas.md         # Opcional
│       └── resumen.md       # Generado por Claude
└── temp/                    # Archivos temporales
```

## 3. Comandos

```bash
trani start [título]    # Inicia grabación
trani stop              # Detiene y procesa automáticamente
trani toggle [título]   # Toggle start/stop
```

**Nota:** La configuración de shortcuts globales (Super+S) queda fuera del scope. El usuario puede configurar su DE para ejecutar `trani toggle`.

## 4. Flujo de Trabajo

### 4.1. Inicio (start/toggle cuando no hay sesión activa)

```
1. Crear carpeta: sessions/YYYY-MM-DD-titulo/
2. Iniciar grabación → temp/recording.wav
3. Crear notas.md vacío en la carpeta de la sesión
4. Notificar: "🎙️ Trani: Grabación iniciada - titulo"
5. Guardar estado de sesión activa
```

### 4.2. Durante la Grabación

- Usuario toma notas editando `sessions/YYYY-MM-DD-titulo/notas.md` en neovim (o cualquier editor)
- Audio se graba continuamente en temp/

### 4.3. Detención (stop/toggle cuando hay sesión activa)

```
1. Detener grabación
2. Notificar: "⏸️ Trani: Grabación detenida. Procesando..."
3. Mover audio: temp/recording.wav → sesión/audio.wav
4. Transcribir con Whisper → transcripcion.txt
5. Verificar si notas.md tiene contenido
6. Generar resumen con Claude → resumen.md
7. Eliminar audio.wav
8. Limpiar estado de sesión activa
9. Notificar: "✅ Trani: Sesión completada - titulo"
```

## 5. Captura de Audio

**Sistema:** PipeWire (nativo en Fedora moderna)

**Requisitos:**
- Capturar simultáneamente audio del sistema + micrófono
- Formato: WAV, 16kHz mono (óptimo para Whisper)
- Mezclar ambas fuentes en un solo stream

**Implementación:**

### 5.1. Configuración de Virtual Sink

```bash
# Crear sink virtual para mezclar audio
pactl load-module module-null-sink \
    sink_name=trani_mix \
    sink_properties=device.description="Trani_Recording_Mix"

# Redirigir micrófono al mix
pactl load-module module-loopback \
    source=@DEFAULT_SOURCE@ \
    sink=trani_mix \
    latency_msec=1

# Redirigir audio del sistema al mix
pactl load-module module-loopback \
    source=@DEFAULT_MONITOR@ \
    sink=trani_mix \
    latency_msec=1

# Grabar desde el mix
pw-record --target trani_mix.monitor \
    --rate 16000 --channels 1 \
    temp/recording.wav
```

### 5.2. Limpieza después de grabar

```bash
# Descargar módulos (usar IDs guardados durante setup)
pactl unload-module [loop_mic_module_id]
pactl unload-module [loop_sys_module_id]
pactl unload-module [sink_module_id]
```

**Nota:** Los IDs de módulos se guardan al cargarlos para poder descargarlos correctamente después.

### 5.3. Ventajas de PipeWire

- ✅ Nativo en Fedora con Wayland
- ✅ Mejor integración con el sistema
- ✅ Captura de audio del sistema más simple
- ✅ Menor latencia
- ✅ Comando `pw-record` optimizado

## 6. Transcripción

```bash
~/whisper.cpp/build/bin/whisper-cli \
  -m ~/whisper.cpp/models/ggml-large-v3-turbo.bin \
  -f sesion/audio.wav \
  -l es \
  -t 12 \
  -otxt \
  -of sesion/transcripcion
```

## 7. Generación con Claude

### 7.1. Caso: Con Notas

**Input:**
- `transcripcion.txt`
- `notas.md` (si tiene contenido)

**Prompt:**
```
Tienes una transcripción de una sesión y las notas tomadas por el usuario.

TRANSCRIPCIÓN:
[contenido transcripcion.txt]

NOTAS DEL USUARIO:
[contenido notas.md]

Genera un documento markdown estructurado con:

1. RESUMEN EJECUTIVO (2-3 párrafos)
   - Contexto general de la sesión
   - Puntos clave discutidos
   - Conclusiones principales

2. DETALLES POR TEMA
   Usa los temas de las notas del usuario como estructura.
   Para cada tema identifica en la transcripción:
   - Detalles específicos mencionados
   - Datos, fechas, números relevantes
   - Procesos o procedimientos descritos
   - Decisiones tomadas
   - Contexto adicional importante

3. ACCIONES Y PENDIENTES
   - Action items identificados
   - Responsables (si se mencionan)
   - Fechas límite (si se mencionan)

4. DATOS IMPORTANTES
   - Fechas clave mencionadas
   - Números, métricas, estadísticas
   - Nombres de personas referenciadas
   - Documentos, sistemas o herramientas mencionadas

Mantén el formato limpio y profesional. Usa encabezados claros.
```

### 7.2. Caso: Sin Notas

**Input:**
- Solo `transcripcion.txt`

**Prompt:**
```
Tienes la transcripción de una sesión. Analízala y genera un documento estructurado.

TRANSCRIPCIÓN:
[contenido transcripcion.txt]

Genera un documento markdown con:

1. RESUMEN EJECUTIVO (2-3 párrafos)
   - Tema principal de la sesión
   - Puntos clave discutidos
   - Conclusiones principales

2. TEMAS PRINCIPALES
   Identifica los temas principales discutidos y para cada uno incluye:
   - Contexto y detalles
   - Puntos específicos mencionados
   - Decisiones o conclusiones

3. ACCIONES Y PENDIENTES
   - Action items identificados
   - Responsables (si se mencionan)
   - Fechas límite (si se mencionan)

4. DATOS IMPORTANTES
   - Fechas mencionadas
   - Números, métricas
   - Nombres de personas
   - Referencias a documentos/sistemas

Mantén el formato limpio y profesional.
```

**API Call:**
```bash
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "content-type: application/json" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 4000,
    "messages": [{
      "role": "user",
      "content": "[prompt con transcripción y notas]"
    }]
  }'
```

## 8. Estructura de Archivos

### 8.1. Nombrado de Sesiones

```
sessions/YYYY-MM-DD-titulo/
```

**Lógica:**
- Si se proporciona título: `2025-10-01-sprint_planning/`
- Si no se proporciona título: `2025-10-01-sesion_14-30/` (usando hora)

### 8.2. Contenido de Sesión

```
2025-10-01-sprint_planning/
├── transcripcion.txt    # Output de Whisper
├── notas.md            # Notas del usuario (puede estar vacío)
└── resumen.md          # Output de Claude
```

**Nota:** `audio.wav` se elimina después de transcribir para ahorrar espacio.

## 9. Configuración

**`config.json`:**
```json
{
  "whisper": {
    "model_path": "~/whisper.cpp/models/ggml-large-v3-turbo.bin",
    "binary_path": "~/whisper.cpp/build/bin/whisper-cli",
    "threads": 12,
    "language": "es"
  },
  "audio": {
    "sample_rate": 16000,
    "channels": 1
  },
  "claude": {
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 4000
  },
  "paths": {
    "sessions_dir": "~/trani/sessions",
    "temp_dir": "~/trani/temp"
  }
}
```

**API Key de Claude:**
Variable de entorno `ANTHROPIC_API_KEY` (más seguro para MVP, no se commitea accidentalmente).

```bash
# Añadir a ~/.zshrc o ~/.bashrc
export ANTHROPIC_API_KEY="sk-ant-..."
```

## 10. Estado de Sesión

**`temp/current_session.json`:**
```json
{
  "active": true,
  "title": "sprint_planning",
  "started_at": "2025-10-01T14:30:00",
  "session_path": "sessions/2025-10-01-sprint_planning"
}
```

Permite saber si hay una sesión en curso para el comando `toggle`.

## 11. Notificaciones

```bash
# Inicio
notify-send "🎙️ Trani" "Grabación iniciada: titulo"

# Stop - Procesando
notify-send "⏸️ Trani" "Grabación detenida. Procesando..."

# Completado
notify-send "✅ Trani" "Sesión completada: titulo\nUbicación: sessions/YYYY-MM-DD-..."

# Error
notify-send -u critical "❌ Trani" "Error: [mensaje]"
```

## 12. Dependencias

### Sistema
- `pipewire` - Sistema de audio (ya instalado en Fedora por defecto)
- `pipewire-utils` - Incluye `pw-record` y herramientas CLI
- `pipewire-pulse` - Compatibilidad con comandos `pactl`
- `curl` - Llamadas a Claude API
- `jq` - Parse JSON
- `libnotify` - Notificaciones (`notify-send`)

### Aplicaciones
- Whisper.cpp (instalado en `~/whisper.cpp`)
- Claude API key en `$ANTHROPIC_API_KEY`

### Verificar instalación
```bash
# Verificar PipeWire
systemctl --user status pipewire pipewire-pulse

# Verificar herramientas
which pw-record pactl notify-send curl jq
```

## 13. Manejo de Errores

```bash
# Si Whisper falla
→ Notificar error
→ NO eliminar audio.wav (para debug)
→ Guardar log en sesion/error.log

# Si Claude API falla
→ Notificar error
→ Mantener transcripcion.txt
→ Usuario puede regenerar resumen manualmente después

# Si no hay sesión activa al hacer stop
→ Notificar: "No hay sesión activa"

# Si ya hay sesión activa al hacer start
→ Notificar: "Ya hay una sesión en curso: [titulo]"
```

## 14. Casos de Uso MVP

### Caso 1: Reunión con notas
```bash
trani start "reunion_equipo"
# [Usuario edita notas.md en neovim durante la reunión]
trani stop
# → Resultado en sessions/2025-10-01-reunion_equipo/
```

### Caso 2: Reunión sin notas
```bash
trani start "daily_standup"
# [Usuario no toma notas]
trani stop
# → Resumen generado solo con transcripción
```

### Caso 3: Con toggle
```bash
trani toggle "planning"    # Inicia
# [transcurre tiempo]
trani toggle               # Detiene y procesa
```

### Caso 4: Integración con scripts
```bash
#!/bin/bash
trani start "automated_meeting"
sleep 900  # 15 minutos
trani stop
```

## 15. Fuera de Scope (Versiones Futuras)

- Shortcuts globales del sistema (usuario lo configura manualmente)
- Auto-abrir neovim al crear notas.md
- Verificar si archivo está abierto antes de procesar
- Detección automática de título desde calendario
- Templates de resumen personalizables
- Búsqueda en sesiones
- Diarización (identificar speakers)
- Web UI

## 16. Criterios de Éxito MVP

✅ Grabar audio del sistema + micrófono
✅ Transcribir con Whisper.cpp
✅ Generar resumen inteligente con Claude
✅ Organización automática de archivos
✅ Comandos simples: start, stop, toggle
✅ Notificaciones desktop
✅ Manejo básico de errores
✅ Configuración simple

## 17. Próximos Pasos de Implementación

1. Script básico de captura de audio (probar PipeWire)
2. Integración con Whisper.cpp
3. Integración con Claude API
4. Sistema de gestión de sesiones
5. Comandos start/stop/toggle
6. Notificaciones
7. Testing con casos reales
8. Documentación de uso

---

**Versión:** 1.0 MVP
**Fecha:** Octubre 2025
**Lenguaje de implementación:** Bash (simplicidad para MVP)
