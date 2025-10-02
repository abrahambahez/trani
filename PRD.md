# PRD: Trani - Sistema de Transcripción y Notas Inteligentes

## 1. Visión General

Herramienta CLI minimalista para grabar audio del sistema, transcribirlo con Whisper.cpp, y usar Claude API para generar documentación estructurada que combina transcripción automática con notas manuales opcionales.

**Filosofía MVP:** Máximo valor con mínimo esfuerzo. Sin complicaciones innecesarias.

## 2. Arquitectura del Sistema

```
~/trani/
├── trani                    # Script ejecutable principal
├── config.json              # Configuración (opcional para futuro)
├── prompts/                 # Templates de prompts personalizables
│   ├── default.txt          # Prompt con notas
│   ├── default_no_notes.txt # Prompt sin notas
│   ├── meeting.txt          # Ejemplo: reuniones formales
│   └── brainstorm.txt       # Ejemplo: sesiones creativas
├── sessions/
│   └── YYYY-MM-DD-titulo/
│       ├── transcripcion.txt
│       ├── notas.md         # Opcional
│       └── resumen.md       # Generado por Claude
└── temp/                    # Archivos temporales
```

## 3. Comandos

```bash
trani start [título] [--prompt TEMPLATE]    # Inicia grabación
trani stop                                   # Detiene y procesa manualmente
trani toggle [título] [--prompt TEMPLATE]   # Toggle start/stop
```

**Opciones:**
- `--prompt TEMPLATE` - Usa un template de prompt personalizado (default: 'default')

**Nota:** La configuración de shortcuts globales (Super+S) queda fuera del scope. El usuario puede configurar su DE para ejecutar `trani toggle`.

## 4. Flujo de Trabajo

### 4.1. Inicio (start/toggle cuando no hay sesión activa)

```
1. Crear carpeta: sessions/YYYY-MM-DD-titulo/
2. Guardar estado de sesión (título, path, prompt template)
3. Iniciar grabación → temp/recording.wav
4. Notificar: "🎙️ Trani: Grabación iniciada - titulo"
5. Crear y abrir notas.md en neovim (bloqueante)
6. Cuando usuario cierra neovim → ejecutar stop_active_session
```

### 4.2. Durante la Grabación

- Audio se graba en background mientras neovim está abierto
- Usuario toma notas en `sessions/YYYY-MM-DD-titulo/notas.md`
- Al cerrar neovim (:wq), el script automáticamente procesa la sesión

### 4.3. Detención (automática al cerrar neovim, o manual con stop)

```
1. Detener grabación y descargar módulos de audio
2. Notificar: "⏸️ Trani: Grabación detenida. Procesando..."
3. Mover audio: temp/recording.wav → sesión/audio.wav
4. Transcribir con Whisper → transcripcion.txt
5. Verificar si notas.md tiene contenido
6. Cargar prompt template (personalizado o default)
7. Generar resumen con Claude → resumen.md
8. Si hay error de Claude API, guardar error en resumen.md y notificar
9. Eliminar audio.wav
10. Limpiar estado de sesión activa
11. Notificar: "✅ Trani: Sesión completada - titulo"
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

### 7.1. Sistema de Templates de Prompts

**Ubicación:** `~/trani/prompts/`

**Convención de nombres:**
- `{nombre}.txt` - Template con notas
- `{nombre}_no_notes.txt` - Template sin notas

**Variables en templates:**
- `{{TRANSCRIPTION}}` - Reemplazada con la transcripción completa
- `{{NOTES}}` - Reemplazada con las notas del usuario

**Ejemplo de template (`prompts/meeting.txt`):**
```
Tienes una transcripción de una reunión formal y las notas del usuario.

TRANSCRIPCIÓN:
{{TRANSCRIPTION}}

NOTAS DEL USUARIO:
{{NOTES}}

Genera un acta de reunión profesional con:
1. Resumen ejecutivo
2. Participantes mencionados
3. Puntos discutidos
4. Decisiones tomadas
5. Acciones asignadas con responsables
```

**Fallback:** Si no existe el template especificado, usa prompts hardcodeados en el script.

### 7.2. Caso: Con Notas y Template Personalizado

**Input:**
- `transcripcion.txt`
- `notas.md` (si tiene contenido)
- Template desde `prompts/{nombre}.txt`

**Proceso:**
1. Cargar template desde archivo
2. Reemplazar `{{TRANSCRIPTION}}` con contenido de transcripcion.txt
3. Reemplazar `{{NOTES}}` con contenido de notas.md
4. Enviar a Claude API

### 7.3. Caso: Prompts Hardcodeados (Fallback)

Si no existe el template, usa los prompts embebidos en el script.

**Prompt con notas (hardcoded):**
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

### 7.4. Caso: Sin Notas (Fallback Hardcoded)

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

### 7.5. API Call y Manejo de Errores

**API Call:**
```bash
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
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

**Manejo de errores:**
- Verifica si la respuesta contiene campo `error`
- Si hay error:
  - Guarda mensaje en `resumen.md`: "Error de Claude API: {mensaje}"
  - Notifica al usuario con el error específico
  - Imprime en consola
  - Continúa guardando transcripción
- Si no hay error:
  - Extrae `response.content[0].text`
  - Guarda en `resumen.md`

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
  "session_path": "sessions/2025-10-01-sprint_planning",
  "prompt_template": "default"
}
```

Permite saber si hay una sesión en curso y qué template de prompt usar.

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
→ Extraer mensaje de error específico
→ Guardar en resumen.md: "Error de Claude API: {mensaje}"
→ Notificar error con mensaje específico
→ Mantener transcripcion.txt
→ Continuar limpieza normal

# Si no hay sesión activa al hacer stop
→ Notificar: "No hay sesión activa"

# Si ya hay sesión activa al hacer start
→ Notificar: "Ya hay una sesión en curso: [titulo]"

# Si falta ANTHROPIC_API_KEY
→ Error de curl
→ Claude API devuelve error de autenticación
→ Se guarda en resumen.md y se notifica
```

## 14. Casos de Uso MVP

### Caso 1: Reunión con notas y prompt default
```bash
trani start "reunion_equipo"
# [Se abre neovim, usuario escribe notas durante la reunión]
# [Usuario hace :wq cuando termina]
# → Resultado en sessions/2025-10-01-reunion_equipo/
```

### Caso 2: Reunión sin notas
```bash
trani start "daily_standup"
# [Se abre neovim, usuario no escribe nada]
# [Usuario hace :wq]
# → Resumen generado solo con transcripción
```

### Caso 3: Con prompt personalizado
```bash
trani start "brainstorming" --prompt brainstorm
# [Se abre neovim con notas]
# [Usuario hace :wq]
# → Usa template prompts/brainstorm.txt
```

### Caso 4: Con toggle y prompt personalizado
```bash
trani toggle "planning" --prompt technical
# Primera vez → Inicia con template technical
# [Transcurre tiempo, usuario trabaja en neovim]
# Segunda vez → Detiene y procesa con template technical
```

### Caso 5: Stop manual (sin usar neovim)
```bash
# Si prefieres no usar neovim o hubo un problema
trani stop
# → Procesa sesión actual manualmente
```

### Caso 6: Integración con scripts
```bash
#!/bin/bash
trani start "automated_meeting" --prompt meeting
# Script espera aquí hasta que neovim se cierre
# Procesamiento automático después
```

## 15. Fuera de Scope (Versiones Futuras)

- Shortcuts globales del sistema (usuario lo configura manualmente)
- Verificar si archivo está abierto antes de procesar (neovim es bloqueante, no hay necesidad)
- Detección automática de título desde calendario
- Búsqueda en sesiones
- Diarización (identificar speakers)
- Web UI
- Editor configurable (actualmente solo neovim)

## 16. Criterios de Éxito MVP

✅ Grabar audio del sistema + micrófono con PipeWire
✅ Transcribir con Whisper.cpp
✅ Generar resumen inteligente con Claude API
✅ Sistema de prompts personalizables con templates
✅ Manejo robusto de errores de API con mensajes específicos
✅ Auto-procesamiento al cerrar neovim
✅ Organización automática de archivos
✅ Comandos simples: start, stop, toggle
✅ Soporte para --prompt flag
✅ Notificaciones desktop informativas
✅ Fallback a prompts hardcodeados si no hay templates

## 17. Próximos Pasos de Implementación

1. ✅ Script básico de captura de audio con PipeWire
2. ✅ Integración con Whisper.cpp
3. ✅ Integración con Claude API
4. ✅ Sistema de gestión de sesiones
5. ✅ Comandos start/stop/toggle
6. ✅ Notificaciones
7. ✅ Sistema de templates de prompts
8. ✅ Manejo robusto de errores
9. ✅ Auto-apertura de neovim
10. ⏳ Testing con casos reales
11. ⏳ Documentación de uso completa
12. ⏳ Creación de templates de ejemplo

---

**Estado:** MVP Completado - En fase de testing
**Versión:** 1.0
**Fecha:** Octubre 2025
**Lenguaje de implementación:** Bash
