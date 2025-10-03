# Problema: Captura de Audio del Sistema en Linux

## Contexto

**Objetivo:** Capturar audio del sistema (monitor) durante juntas/videoconferencias para transcripción con Whisper.

**Requisito crítico:** No arruinar el audio del usuario mientras graba (UX inaceptable para juntas en vivo).

## Estado Actual

### Síntoma
- Audio capturado está prácticamente vacío (Maximum amplitude: 0.000031)
- Whisper alucina completamente por falta de contenido de audio
- El comando usado: `pw-record --target @DEFAULT_MONITOR@ --rate 16000 --channels 1 output.wav`

### Intentos Fallidos

#### Intento 1: PulseAudio loopbacks + Sox en vivo
- **Problema:** Feedback loops horribles que arruinan el audio del sistema del usuario
- **Impacto UX:** Inaceptable - destruye audio durante juntas
- **Código:** `module-loopback` con sox grabando en tiempo real
- **Resultado:** Abandonado por UX

#### Intento 2: pw-record simple + normalización post
- **Comando:** `pw-record --target @DEFAULT_MONITOR@`
- **Problema:** Audio casi silencioso/vacío en la captura
- **Normalización intentada:** `sox norm -1`, `gain -n`, filtros highpass/lowpass
- **Resultado:** Audio destruido o vacío

## Por Qué Esto Debería Ser Simple (Pero No Lo Es)

### En teoría
1. PipeWire/PulseAudio expone monitores de sinks
2. Las apps graban de monitors sin problema (OBS, etc.)
3. Un simple `pw-record --target @DEFAULT_MONITOR@` debería funcionar

### En práctica
- `@DEFAULT_MONITOR@` parece no capturar nada útil
- Los loopbacks crean feedback loops
- No hay forma obvia de "solo escuchar" el audio sin interferir

## Preguntas Sin Responder

1. **¿Por qué `@DEFAULT_MONITOR@` captura audio vacío?**
   - ¿El sink correcto?
   - ¿Permisos?
   - ¿Configuración de PipeWire?

2. **¿Cómo OBS lo hace sin arruinar audio del usuario?**
   - ¿Usan un approach diferente?
   - ¿API específica de PipeWire?

3. **¿Hay alternativa a monitors que no requiera loopbacks?**
   - ¿Virtual cables?
   - ¿Módulos específicos de PipeWire?

## Datos de Diagnóstico

### Audio vacío capturado
```
Maximum amplitude:     0.000031
Minimum amplitude:    -0.000031
RMS amplitude:         0.000015
Volume adjustment:     32768.000  # Necesita 32k veces amplificación!
```

### Audio pre-normalización (funcionaba mejor)
```
Maximum amplitude:     0.572784
RMS amplitude:         0.011752
Volume adjustment:     1.690  # Solo necesitaba 1.7x
```

## Hipótesis

1. **Monitor incorrecto**: Tal vez `@DEFAULT_MONITOR@` no es el sink de audio activo
2. **Sample rate mismatch**: 16kHz captura vs 48kHz sistema causa problemas
3. **PipeWire routing**: Falta configuración específica de routing
4. **Permisos**: PipeWire requiere permisos específicos para monitor access

## Próximos Pasos a Investigar

1. Listar todos los sinks/sources disponibles: `pw-cli list-objects`
2. Verificar qué sink está realmente activo durante videoconferencias
3. Probar captura a 48kHz nativo y downsample después
4. Investigar cómo SimpleScreenRecorder/OBS capturan system audio
5. Revisar configuración de PipeWire en `/etc/pipewire/`
6. Considerar alternativa: `parec` (PulseAudio) vs `pw-record`

## Solución Temporal (Rollback?)

Volver a implementación original con `pw-record` que al menos capturaba algo audible (aunque bajo), y confiar en que Whisper puede manejar audio bajo.

**Trade-off aceptable:** Audio bajo pero funcional > Audio vacío con alucinaciones

## Referencias

- [PipeWire Wiki - Capture](https://gitlab.freedesktop.org/pipewire/pipewire/-/wikis/FAQ#how-do-i-record-audio-from-playback-stream)
- [Arch Wiki - PipeWire](https://wiki.archlinux.org/title/PipeWire)
- Comparar con: OBS Studio source code para screen recording
