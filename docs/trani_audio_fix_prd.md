# PRD: Fix Captura de Audio - Trani

## Problema
`@DEFAULT_MONITOR@` captura audio vacío (amplitude: 0.000031). Whisper alucina por falta de contenido.

## Solución
Captura directa del monitor correcto sin loopbacks ni virtual sinks.

---

## Cambios de Código

### 1. Nueva Función: Detección de Monitor
```bash
get_active_monitor_source() {
    local default_sink=$(pactl get-default-sink)
    [ -z "$default_sink" ] && return 1
    echo "${default_sink}.monitor"
}
```

### 2. Reemplazar: start_audio_recording()
```bash
start_audio_recording() {
    local output_file="$1"
    local monitor_source=$(get_active_monitor_source) || return 1
    
    pw-record --target "$monitor_source" \
        --rate 48000 --channels 2 \
        "$output_file" &
    
    local pid=$!
    echo $pid > "$RECORDING_PID_FILE"
    
    sleep 0.5
    kill -0 "$pid" 2>/dev/null || return 1
}
```

### 3. Simplificar: stop_audio_recording()
```bash
stop_audio_recording() {
    [ -f "$RECORDING_PID_FILE" ] || return
    kill $(cat "$RECORDING_PID_FILE") 2>/dev/null
    wait $(cat "$RECORDING_PID_FILE") 2>/dev/null
    rm -f "$RECORDING_PID_FILE"
}
```

### 4. Agregar Downsample en stop_active_session()
```bash
# Después de move_recording_to_session
sox "$session_path/audio.wav" -r 16000 -c 1 /tmp/temp.wav \
    norm -3 highpass 80 lowpass 8000
mv /tmp/temp.wav "$session_path/audio.wav"
```

### 5. Actualizar: start_new_session()
```bash
# ELIMINAR:
# setup_audio_virtual_sink
# setup_microphone_loopback  
# setup_system_audio_loopback

# REEMPLAZAR CON:
start_audio_recording "$TEMP_DIR/recording.wav" || exit 1
```

### 6. Eliminar
- `setup_audio_virtual_sink()`
- `setup_microphone_loopback()`
- `setup_system_audio_loopback()`
- `unload_audio_modules()`
- Variables: `SINK_MODULE_ID_FILE`, `LOOP_MIC_MODULE_ID_FILE`, `LOOP_SYS_MODULE_ID_FILE`

---

## Testing Manual

### Test 1: Detección
```bash
pactl get-default-sink
# Debe retornar: alsa_output.XXX.analog-stereo
```

### Test 2: Captura
```bash
MONITOR=$(pactl get-default-sink).monitor
pw-record --target "$MONITOR" --rate 48000 --channels 2 test.wav &
# Reproduce audio 10 seg
killall pw-record
sox test.wav -n stat | grep Maximum
# Esperado: Maximum > 0.1
```

### Test 3: Downsample
```bash
sox test.wav -r 16000 -c 1 test_16k.wav norm -3 highpass 80 lowpass 8000
soxi test_16k.wav
# Esperado: Sample Rate: 16000, Channels: 1
```

### Test 4: End-to-end
```bash
trani start "test"
# Reproduce audio, escribe notas, :wq
# Verificar: transcripción correcta, no alucinaciones
```

---

## Dependencias
```bash
sudo dnf install sox
```

---

## Métricas de Éxito
- ✅ Maximum amplitude > 0.1 en Test 2
- ✅ Whisper no alucina en Test 4
- ✅ Transcripción precisa del audio reproducido

---

## Rollback
```bash
cp trani trani.backup  # Antes de cambios
cp trani.backup trani  # Si falla
```

---

## Tiempo Estimado
- Implementación: 30 min
- Testing: 25 min
- **Total: 55 min**

---

## Decisión
Después de Test 4:
- **Funciona mejor** → Commit
- **Funciona igual/peor** → Rollback, evaluar Opción B/C