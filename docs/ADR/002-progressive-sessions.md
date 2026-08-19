# ADR 002: Progressive Sessions, Chunked Transcription, and Obsidian Integration

## Status
Accepted

## Context
Trani's original session flow was a single synchronous operation: record for the whole session, open nvim in the foreground, and only after the editor closed, transcribe the entire recording and generate a summary — all inline, in the same blocking CLI invocation. This meant:
- No feedback until the very end; a long meeting meant a long wait after closing the editor before the summary existed.
- No way to start a new session while the previous one was still transcribing/summarizing, since the whole pipeline was one blocking call.
- Only nvim was supported, and only system output was captured (see ADR-001), not the microphone.

The goal of this round of work was to move to progressive, chunked recording with incremental transcription, support recording the microphone (alone or together with system output, for meetings), and move note-taking to Obsidian.

## Decisions

### Recording lock vs. post-processing are separate lifecycles
`temp/active_recording.json` now exists only while audio is actively being captured. It's created (atomically, see below) when recording starts and removed the moment recording stops — before transcription of the last chunk or summary generation happens. Those steps run in a detached, backgrounded worker process. This is what lets a new session start immediately after stopping the previous one, even while its summary is still being generated.

### Lock acquisition must be atomic
A plain "check if a lock exists, then write one" has a TOCTOU race: two near-simultaneous `toggle` invocations can both see no lock and both start recording, and whichever writes last silently orphans the other (unreachable by `trani stop`, recording indefinitely). `RecordingLock.Acquire` uses `O_CREATE|O_EXCL` so only one caller can ever win, with a retry for the case where the existing lock's owning process has already died.

### Chunked capture via ffmpeg's segment muxer, not repeated `pw-record` restarts
Each stream (mic, and system output monitor in `mic_system` mode) is captured directly — never through a virtual sink, see ADR-001 — via `ffmpeg -f pulse -i <source> -f segment -segment_time <N> ...`. This produces gapless, sequentially numbered chunk files and a segment list file that ffmpeg appends to as each chunk closes. A chunker polls that list, and processes (sox normalize/filter, transcribe) each newly closed chunk while the session is still live, appending results into `sessions/.sources/<timestamp>.txt` and `.wav`. Restarting `pw-record` every N minutes was considered and rejected: it introduces a small gap in the recording at every chunk boundary.

ffmpeg's segment list entries are relative to ffmpeg's own working directory (in practice, just the basename), not to the list file's location — this tripped up the first implementation and is worth knowing if touching `readSegmentList`.

### mic_system mixing: both strategies, not one
For meetings (`audio.mode: mic_system`), two independent strategies for combining the mic and system streams are implemented and selectable via `audio.mix_strategy`:
- `post_mix` (default): each stream is normalized independently (reusing the sox pipeline from ADR-001) before being summed and renormalized into one chunk, transcribed once.
- `separate_transcribe`: each stream is transcribed independently and the text concatenated.

Both were kept, rather than picking one, to let real usage decide which holds up better — normalizing and mixing risks re-introducing an amplitude problem close to what ADR-001 rejected if the mix isn't renormalized carefully; transcribing separately avoids that risk but doubles transcription calls and can't interleave overlapping speech by time.

### The session note is the final note — summary appended, never overwritten
Sessions live as a single flat file, `sessions/<timestamp>.md`, instead of a per-session directory with `notas.md` and `resumen.md` as separate files. The user's raw notes (including any metadata a note-taking template already put there) and the generated summary are the same file: the summary is appended below the existing content under a `## Resumen` heading once generated, never overwriting what was already there. If summary generation fails, the note is left untouched (only a notification reports the failure) so a transient LLM error can't destroy the user's notes or metadata.

`trani process` (reprocessing an arbitrary external audio file, not a trani-managed session) was originally kept on its own three-file layout (`transcripcion.txt`, `notas.md`, `resumen.md`) on the grounds that it operates on unrelated, externally-provided input. That split turned out to be pure duplication with no benefit, and had a worse failure mode besides — a failed summary was written into `resumen.md` as if it were real output, while the command still reported success. It has since been converged onto the same convention as live sessions: `process` writes into the same `sessions_dir`, seeds the note from `--notes` (if given) exactly like a live session's note already holds the user's content before postprocessing, and shares the very same summary-writing code path (`writeSummary` in `internal/session/postprocess.go`) so both flows fail and succeed identically. The one deliberate difference: `process` propagates a postprocessing failure as a real command error (non-zero exit), since it's a synchronous foreground command instead of a detached background worker.

### Obsidian is required, no editor fallback
`obsidian.vault_path` must be configured for the live session flow; `Launch` fails immediately if it's empty, before touching the recording lock. The note opens via Obsidian's own `obsidian://open` URI, dispatched through `xdg-open` — not the [obsidian-cli](https://github.com/Yakitrak/obsidian-cli), whose IPC to a running instance proved unreliable on a flatpak install (failed outright while Obsidian was open, silently ignored the requested note on a cold start). Reading and writing the note's content is always plain file I/O against the vault path; a failure to open the URI (app not running, `xdg-open` missing) degrades to a logged warning rather than failing the session, bounded by a timeout so a stuck/unreachable Obsidian instance can't block the session from ever reaching the point where it listens for a stop signal.

Since Obsidian is a separate GUI app, a session runs entirely in a detached `__record-worker` process that waits for an explicit stop signal (`trani stop` or a second `trani toggle`) instead of waiting on an editor process.

## Consequences

### Positive
- Most transcription work happens while the meeting is still going, not all at once afterward.
- A new session can start immediately after stopping the previous one.
- Meetings can capture the microphone together with system output, not just system output.

### Negative
- More moving parts: two hidden worker subcommands (`__record-worker`, `__postprocess-worker`), a lock file, a chunk poller.
- No zero-configuration fallback: without `obsidian.vault_path` set, the live session flow (`start`/`toggle`) doesn't run at all.
- A `toggle` sent within the (roughly half-second) window while the recorder is still shutting down is absorbed as a redundant stop signal rather than starting a new session; the user has to toggle again slightly later. This is intentional (fails safe rather than racing to start something mid-shutdown) but is a minor rough edge worth knowing about.
- `mic_system` mode depends on the machine's audio routing behaving as expected (the current default sink/source actually carrying signal); this can't be fully validated outside a real desktop session with real hardware.

## Notes
- Chunk cadence defaults to `audio.chunk_seconds: 300` (5 minutes).
- Poll interval for picking up closed chunks is a fixed 20 seconds (`chunkPollInterval` in `internal/session/chunker.go`), well under the default chunk duration.
