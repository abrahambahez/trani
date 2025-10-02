# Trani - Asistente de reuniones

Transcribe tus reuniones y genera resúmenes inteligentes automáticamente.

## ¿Qué hace?

Trani graba el audio de tu computadora (sistema + micrófono), lo transcribe con Whisper, y usa Claude para generar un resumen estructurado combinado con tus notas.

## Instalación rápida

```bash
# Añadir al PATH
echo 'export PATH="$HOME/trani:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Configurar API key de Claude
echo 'export ANTHROPIC_API_KEY="tu-api-key"' >> ~/.zshrc
source ~/.zshrc
```

## Uso

```bash
# Iniciar sesión
trani start "nombre_reunion"

# Se abre neovim para tomar notas
# Al cerrar neovim (:wq), automáticamente:
# - Detiene la grabación
# - Transcribe el audio
# - Genera resumen con Claude
# - Limpia archivos temporales
```

## Resultado

Cada sesión crea una carpeta con:

```
sessions/2025-10-01-nombre_reunion/
├── transcripcion.txt  # Todo lo que se dijo
├── notas.md          # Tus notas
└── resumen.md        # Resumen inteligente generado
```

## Requisitos

- Fedora con PipeWire (instalado por defecto)
- Whisper.cpp instalado en `~/whisper.cpp`
- Claude API key
- `jq`, `curl`, `notify-send`

## Comandos

```bash
trani start [título]   # Inicia sesión (abre neovim)
trani stop             # Detiene manualmente
trani toggle [título]  # Alterna start/stop
```

## Ejemplo real

```bash
$ trani start "planning_sprint"
🎙️ Trani: Grabación iniciada: planning_sprint

# [Neovim se abre para tomar notas]
# [Tomas notas durante la reunión]
# [Cierras neovim con :wq]

⏸️ Trani: Grabación detenida. Procesando...
✅ Trani: Sesión completada: planning_sprint
```

## Notas

- El audio original se elimina después de transcribir (solo quedan texto y resumen)
- Si no tomas notas, Claude genera el resumen solo con la transcripción
- Usa `Super+S` (o tu shortcut preferido) para ejecutar `trani toggle`

---

**Versión:** 1.0 MVP  
**Autor:** sabhz
**Licencia:** MIT
