# Tasks: Automatic Session Directory Naming

Generated from: `prd-auto-session-naming.md`

## Relevant Files

- `internal/session/session.go` - Main session management logic; contains session creation, directory naming, and state management. Will add `slugify()` function and `extractAndRenameIfNeeded()` method.
- `internal/session/processor.go` - Handles `trani process` command; creates sessions from existing audio files. Will implement H1 extraction logic.
- `cmd/start.go` - CLI command handler for `trani start`. Will remove title argument parsing.
- `cmd/toggle.go` - CLI command handler for `trani toggle`. Will remove title argument parsing.

### Notes

- This is a breaking change: the ability to manually specify session titles via `trani start 'title'` will be removed.
- All changes use Go standard library only (no external dependencies).
- Target implementation: ≤60 new lines of code total.
- Existing session directories with old naming format remain unchanged (no migration).

## Tasks

- [x] 1.0 Implement slugify utility function
  - [x] 1.1 Create `slugify(text string) string` function in `internal/session/session.go`
  - [x] 1.2 Implement lowercase conversion using `strings.ToLower()`
  - [x] 1.3 Replace spaces with hyphens using `strings.ReplaceAll()`
  - [x] 1.4 Remove special characters using regex (keep only `a-z`, `0-9`, `-`)
  - [x] 1.5 Collapse multiple consecutive hyphens into single hyphen using regex
  - [x] 1.6 Trim hyphens from start and end using `strings.Trim()`
  - [x] 1.7 Truncate to maximum 50 characters (handle UTF-8 properly with rune slicing)
  - [x] 1.8 Trim hyphens again after truncation to avoid trailing hyphens
  - [x] 1.9 Return empty string if result is empty after processing

- [x] 2.0 Update timestamp format for session directories
  - [x] 2.1 In `internal/session/session.go`, change `datePrefix` format from `2006-01-02` to `20060102-1504`
  - [x] 2.2 Update directory path creation to use timestamp only (remove title concatenation)
  - [x] 2.3 Remove default title generation logic (`sesion_HH-MM` fallback)
  - [x] 2.4 In `internal/session/processor.go`, change `datePrefix` format from `2006-01-02` to `20060102-1504`
  - [x] 2.5 Update directory path creation in processor to use timestamp only

- [x] 3.0 Remove title argument from command interfaces
  - [x] 3.1 [depends on: 2.0] In `cmd/start.go`, remove title argument parsing logic (lines ~29-31)
  - [x] 3.2 [depends on: 2.0] Update `session.New()` call in `cmd/start.go` to remove title parameter
  - [x] 3.3 [depends on: 2.0] In `cmd/toggle.go`, remove title argument parsing logic (lines ~34-37)
  - [x] 3.4 [depends on: 2.0] Update `session.New()` call in `cmd/toggle.go` to remove title parameter
  - [x] 3.5 [depends on: 2.0] Update `session.New()` function signature in `internal/session/session.go` to remove `title` parameter

- [x] 4.0 Implement H1 extraction and directory renaming for `trani start`
  - [x] 4.1 [depends on: 1.0, 3.0] Create `extractAndRenameIfNeeded() error` method on `Session` struct in `internal/session/session.go`
  - [x] 4.2 [depends on: 4.1] Read first line of `notas.md` file (handle file not found gracefully)
  - [x] 4.3 [depends on: 4.2] Check if first line starts with `# ` (H1 markdown syntax)
  - [x] 4.4 [depends on: 4.3] If no H1 found, return nil (keep timestamp-only directory name)
  - [x] 4.5 [depends on: 4.3] Extract heading text (everything after `# `)
  - [x] 4.6 [depends on: 4.5] Trim whitespace from heading text
  - [x] 4.7 [depends on: 4.6] Check if heading text is empty; if yes, return nil
  - [x] 4.8 [depends on: 4.7, 1.0] Call `slugify()` on heading text
  - [x] 4.9 [depends on: 4.8] Check if slug is empty; if yes, return nil
  - [x] 4.10 [depends on: 4.9] Construct new directory path: `{timestamp}-{slug}`
  - [x] 4.11 [depends on: 4.10] Rename directory using `os.Rename()` from old path to new path
  - [x] 4.12 [depends on: 4.11] If rename fails, log error but continue (don't fail the session)
  - [x] 4.13 [depends on: 4.11] Update `s.path` field to new directory path
  - [x] 4.14 [depends on: 3.5] Call `extractAndRenameIfNeeded()` after user closes Neovim editor in `Run()` method
  - [x] 4.15 [depends on: 4.13] Save updated session state to `current_session.json` after successful rename

- [x] 5.0 Implement H1 extraction and directory renaming for `trani process`
  - [x] 5.1 [depends on: 1.0, 2.5] In `internal/session/processor.go`, after directory creation, check if notes file was provided
  - [x] 5.2 [depends on: 5.1] If notes file provided, read first line
  - [x] 5.3 [depends on: 5.2] Check if first line starts with `# `
  - [x] 5.4 [depends on: 5.3] If no H1, continue with timestamp-only directory name
  - [x] 5.5 [depends on: 5.3] Extract heading text (everything after `# `)
  - [x] 5.6 [depends on: 5.5] Trim whitespace and check if empty
  - [x] 5.7 [depends on: 5.6, 1.0] Call `slugify()` on heading text
  - [x] 5.8 [depends on: 5.7] Check if slug is empty; if yes, skip renaming
  - [x] 5.9 [depends on: 5.8] Construct new directory path: `{timestamp}-{slug}`
  - [x] 5.10 [depends on: 5.9] Rename directory using `os.Rename()`
  - [x] 5.11 [depends on: 5.10] Update `sessionPath` variable to new path for subsequent file operations
  - [x] 5.12 [depends on: 5.10] Handle rename errors gracefully (log but continue)

- [ ] 6.0 Verify implementation and edge cases
  - [ ] 6.1 [depends on: 4.0, 5.0] Test with H1 heading: `# Awesome Session` → directory `20251015-1430-awesome-session/`
  - [ ] 6.2 [depends on: 4.0, 5.0] Test without H1 heading → directory remains `20251015-1430/`
  - [ ] 6.3 [depends on: 4.0, 5.0] Test with special characters: `# Meeting 🚀 Notes!` → `20251015-1430-meeting-notes/`
  - [ ] 6.4 [depends on: 4.0, 5.0] Test with long heading (>50 chars) → slug truncated to 50 chars
  - [ ] 6.5 [depends on: 4.0, 5.0] Test with empty notes file → directory remains `20251015-1430/`
  - [ ] 6.6 [depends on: 4.0, 5.0] Test with heading that becomes empty after slugification (e.g., `# 🚀🎉`) → directory remains timestamp-only
  - [ ] 6.7 [depends on: 4.0] Verify session state file contains correct path after rename
  - [ ] 6.8 [depends on: 4.0, 5.0] Verify all session files (audio.wav, transcripcion.txt, resumen.md) are created in renamed directory
  - [ ] 6.9 [depends on: 3.0] Verify `trani start 'custom-title'` no longer works (breaking change confirmed)
  - [ ] 6.10 [depends on: 4.0, 5.0] Check that code changes are ≤60 new lines as per success metric
