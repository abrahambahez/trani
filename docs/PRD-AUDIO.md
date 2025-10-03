# PRD: Refactor Sistema de Audio - Trani

## 1. Objetivo

Reemplazar PipeWire por **PulseAudio + SoX** hardcodeado para garantizar:
- ✅ Alta calidad de audio con niveles equilibrados
- ✅ Código lean (~50 líneas nuevas)
- ✅ Instalación automática de dependencias

**Plataforma:** Fedora Linux únicamente

---

## 2. Arquitectura

### 2.1. Estructura (sin cambios)

```
~/trani/
├── trani                    # Script principal (modificaciones mínimas)
├── lib/
│   └── audio.sh             # NUEVO: lógica de audio
└── ... (resto sin cambios)
```

### 2.2. Tecnologías

- **PulseAudio:** Routing de audio (loopbacks)
- **SoX:** Grabación con filtros de calidad
- **Hardcodeado:** No módulos intercambiables

---

## 3. Interfaz de Audio

### 3.1. Funciones Públicas

```bash
# lib/audio.sh

audio_check_and_install_deps()
# Verifica sox, pulseaudio-utils
# Intenta instalar con dnf si faltan
# Exit: 0 ok, 1 error fatal

audio_start(output_file)
# Setup: loopback system + mic → sink virtual
# Graba: sox desde sink virtual → output_file
# Return: PID del proceso sox
# Exit: 0 ok, 1 error

audio_stop(audio_pid)
# Kill proceso sox
# Cleanup: descargar módulos loopback
# Exit: 0 ok, 1 error
```

---

## 4. Implementación Técnica

### 4.1. Verificación e Instalación de Dependencias

```bash
audio_check_and_install_deps() {
    local missing=()
    
    command -v sox >/dev/null || missing+=("sox")
    command -v pactl >/dev/null || missing+=("pulseaudio-utils")
    
    if [ ${#missing[@]} -gt 0 ]; then
        echo "📦 Instalando dependencias: ${missing[*]}"
        
        if sudo dnf install -y "${missing[@]}" 2>/dev/null; then
            echo "✅ Dependencias instaladas"
        else
            notify-send -u critical "❌ Trani" "Error instalando: ${missing[*]}"
            echo "Error: No se pudieron instalar dependencias. Instala manualmente: sudo dnf install ${missing[*]}"
            return 1
        fi
    fi
    
    return 0
}
```

### 4.2. Inicio de Grabación

```bash
audio_start() {
    local output="$1"
    
    # 1. Crear sink virtual para mezclar audio
    local sink_id=$(pactl load-module module-null-sink \
        sink_name=trani_mix \
        sink_properties=device.description="Trani_Recording_Mix")
    
    # 2. Redirigir micrófono al mix
    local loop_mic=$(pactl load-module module-loopback \
        source=@DEFAULT_SOURCE@ \
        sink=trani_mix \
        latency_msec=1)
    
    # 3. Redirigir audio del sistema al mix
    local loop_sys=$(pactl load-module module-loopback \
        source=$(pactl get-default-sink).monitor \
        sink=trani_mix \
        latency_msec=1)
    
    # 4. Guardar IDs de módulos para cleanup
    echo "$sink_id $loop_mic $loop_sys" > "$TEMP_DIR/audio_modules"
    
    # 5. Grabar desde el mix con filtros de calidad
    sox -t pulseaudio trani_mix.monitor \
        -r 16000 -c 1 "$output" \
        norm -3 \
        highpass 80 \
        lowpass 8000 \
        2>"$TEMP_DIR/audio_error.log" &
    
    local pid=$!
    
    # 6. Verificar que sox inició correctamente
    sleep 0.5
    if ! kill -0 "$pid" 2>/dev/null; then
        notify-send -u critical "❌ Trani" "Error al iniciar grabación"
        audio_cleanup_modules
        return 1
    fi
    
    echo "$pid"
}
```

### 4.3. Detención y Limpieza

```bash
audio_stop() {
    local pid="$1"
    
    # Detener grabación
    kill "$pid" 2>/dev/null
    wait "$pid" 2>/dev/null
    
    # Descargar módulos de PulseAudio
    audio_cleanup_modules
}

audio_cleanup_modules() {
    if [ -f "$TEMP_DIR/audio_modules" ]; then
        read sink_id loop_mic loop_sys < "$TEMP_DIR/audio_modules"
        pactl unload-module "$loop_sys" 2>/dev/null
        pactl unload-module "$loop_mic" 2>/dev/null
        pactl unload-module "$sink_id" 2>/dev/null
        rm "$TEMP_DIR/audio_modules"
    fi
}
```

---

## 5. Integración con Script Principal

### 5.1. Modificaciones en `trani`

```bash
#!/bin/bash

# Agregar al inicio (después de definir TRANI_HOME, TEMP_DIR)
source "${TRANI_HOME}/lib/audio.sh"

# En start_session() - reemplazar sección de audio
start_session() {
    # ... código existente ...
    
    # Verificar dependencias
    audio_check_and_install_deps || exit 1
    
    # Iniciar grabación
    AUDIO_PID=$(audio_start "$TEMP_DIR/recording.wav")
    if [ $? -ne 0 ]; then
        exit 1
    fi
    
    echo "$AUDIO_PID" > "$TEMP_DIR/audio_pid"
    
    # Notificar inicio
    notify-send "🎙️ Trani" "Grabación iniciada: $TITLE"
    
    # ... resto del código (abrir neovim, etc) ...
}

# En stop_active_session() - reemplazar sección de audio
stop_active_session() {
    # ... código existente ...
    
    notify-send "⏸️ Trani" "Grabación detenida. Procesando..."
    
    # Detener grabación
    AUDIO_PID=$(cat "$TEMP_DIR/audio_pid")
    audio_stop "$AUDIO_PID"
    
    # Mover audio a sesión
    mv "$TEMP_DIR/recording.wav" "$SESSION_PATH/audio.wav"
    
    # ... resto del código (transcripción, Claude, etc) ...
}
```

### 5.2. Líneas Modificadas

**Total cambios en `trani`:**
- Agregar: 1 línea (source lib/audio.sh)
- Modificar: ~8 líneas en start_session()
- Modificar: ~4 líneas en stop_active_session()

**Total código nuevo:**
- `lib/audio.sh`: ~50 líneas

---

## 6. Filtros de Audio SoX

### 6.1. Parámetros Explicados

```bash
sox -t pulseaudio trani_mix.monitor \
    -r 16000        # Sample rate óptimo para Whisper
    -c 1            # Mono (mezcla estéreo → mono)
    "$output" \
    norm -3         # Normaliza a -3dB (previene clipping)
    highpass 80     # Elimina ruido de baja frecuencia
    lowpass 8000    # Anti-aliasing para 16kHz
```

### 6.2. Calidad Esperada

- **Sistema + Micrófono:** Mezclados con niveles equilibrados
- **Ruido:** Reducido por filtros highpass/lowpass
- **Volumen:** Normalizado automáticamente
- **Formato:** Óptimo para Whisper (16kHz mono WAV)

---

## 7. Manejo de Errores

### 7.1. Casos Cubiertos

| Error | Detección | Acción |
|-------|-----------|--------|
| Dependencias faltantes | `command -v` | Intentar `dnf install`, error si falla |
| Sox no inicia | `kill -0 $pid` | Notificación + cleanup + exit 1 |
| No hay dispositivos de audio | Implícito en pactl | Error de pactl → mostrar stderr |
| Módulos no se descargan | Verificar archivo existe | Cleanup silencioso (no crítico) |

### 7.2. Logs de Debug

```bash
# Errores de sox
cat "$TEMP_DIR/audio_error.log"

# Ver módulos cargados
pactl list modules short | grep trani
```

---

## 8. Testing

### 8.1. Comando de Prueba

```bash
# Agregar a trani
trani test-audio

# Implementación
test_audio() {
    echo "🎤 Probando grabación (5 segundos)..."
    
    audio_check_and_install_deps || exit 1
    
    local test_file="$TEMP_DIR/test_audio.wav"
    local pid=$(audio_start "$test_file")
    
    [ -z "$pid" ] && { echo "Error al iniciar"; exit 1; }
    
    sleep 5
    audio_stop "$pid"
    
    # Analizar niveles de audio
    echo "📊 Análisis de volumen:"
    sox "$test_file" -n stat 2>&1 | grep -E "Maximum|Mean"
    
    echo "✅ Archivo guardado en: $test_file"
    echo "   Reproducir con: ffplay $test_file"
}
```

### 8.2. Checklist de Validación

- [ ] `trani test-audio` genera archivo WAV
- [ ] Audio del sistema se escucha claramente
- [ ] Micrófono se escucha claramente
- [ ] Niveles equilibrados (ninguno domina)
- [ ] No hay ruido excesivo de fondo
- [ ] Whisper transcribe correctamente

---

## 9. Dependencias del Sistema

### 9.1. Requeridas

```bash
# Instalación automática por el script
sudo dnf install sox pulseaudio-utils
```

### 9.2. Ya Disponibles en Fedora

- `pulseaudio` - Sistema de audio (pre-instalado)
- `pactl` - Viene con pulseaudio-utils
- `curl`, `jq` - Ya requeridos por trani

---

## 10. Plan de Implementación

### Fase 1: Core (1 hora)
1. Crear `lib/audio.sh` con las 3 funciones
2. Modificar `trani` para usar nueva interfaz
3. Testing básico con `trani test-audio`

### Fase 2: Integración (30 min)
1. Reemplazar secciones de PipeWire en start/stop
2. Probar flujo completo: start → neovim → stop → transcripción
3. Validar que archivos se guardan correctamente

### Fase 3: Refinamiento (15 min)
1. Ajustar filtros de sox si es necesario
2. Verificar notificaciones funcionan
3. Cleanup de archivos temporales

**Total estimado: 1.75 horas**

---

## 11. Criterios de Éxito

- ✅ Audio con calidad superior a PipeWire actual
- ✅ Sistema y micrófono audibles y balanceados
- ✅ Instalación automática de dependencias
- ✅ Menos de 60 líneas de código nuevo
- ✅ Sin regresiones en el flujo existente
- ✅ Comando `trani test-audio` funcional

---

## 12. Rollback Plan

Si hay problemas críticos:

```bash
# Restaurar versión PipeWire
git checkout HEAD~1 trani
rm lib/audio.sh
```

**Criterio de rollback:**
- Calidad de audio peor que antes
- Errores que bloquean el flujo normal
- Incompatibilidad con Fedora específica

---

## 13. Próximos Pasos Post-MVP

**Fuera de scope pero considerado:**
- Soporte macOS (requiere BlackHole)
- Sistema modular de plugins
- Compresión dinámica avanzada
- Detección automática de niveles óptimos

**Estado:** Listo para implementación
**Esfuerzo:** ~2 horas
**Riesgo:** Bajo (cambios aislados, fácil rollback)
