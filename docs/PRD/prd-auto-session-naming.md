# PRD: Automatic Session Directory Naming from Notes Title

## Introduction/Overview

Currently, session directories in Trani use the format `YYYY-MM-DD-{title}` where the title is either manually provided via `trani start 'title'` or defaults to `sesion_HH-MM`. This PRD proposes an automatic naming system that:

1. Uses a timestamp-based unique identifier as the primary directory name
2. Automatically appends a slugified version of the first markdown H1 heading from `notas.md`
3. Simplifies the user experience by eliminating the need to provide a session name upfront

**Problem**: Users must decide on a session name before starting, but the session's actual topic/content often becomes clear only after writing notes. The current date prefix format `YYYY-MM-DD` also doesn't provide sufficient uniqueness or sortability for multiple sessions in the same day.

**Goal**: Automate session directory naming based on note content while maintaining chronological sorting and uniqueness through timestamp-based identifiers.

## Goals

1. Eliminate manual session naming requirement from `trani start` command
2. Use timestamp as unique session identifier with format `YYYYMMDD-HHMM`
3. Automatically extract and append slugified H1 heading from `notas.md` when present
4. Maintain chronological sortability of session directories
5. Minimize code complexity and lines of code added
6. Apply consistent naming logic across both `trani start` and `trani process` commands

## User Stories

### Story 1: Starting a Session Without Predefined Name
**As a** Trani user
**I want to** run `trani start` without providing a session name
**So that** I can focus on capturing ideas first and let the directory name reflect the actual content

**Acceptance**: Running `trani start` creates a directory named `20251015-1430/`, opens notes editor, and after closing the editor, renames directory to `20251015-1430-awesome-session/` if notes start with `# Awesome Session`

### Story 2: Session Without H1 Heading
**As a** Trani user
**I want to** start a session and write notes without an H1 heading
**So that** the session is still properly saved with a unique timestamp identifier

**Acceptance**: If `notas.md` doesn't start with `# `, directory remains as `20251015-1430/`

### Story 3: Processing Existing Audio
**As a** Trani user
**I want to** run `trani process audio.wav` and optionally provide notes
**So that** the session directory is named based on my notes content if I provide them

**Acceptance**: `trani process` creates timestamped directory and extracts H1 if notes file is provided

## Functional Requirements

### FR1: Timestamp-Based Directory Naming
The system must create session directories using the format `YYYYMMDD-HHMM` where:
- `YYYYMMDD`: Current date (e.g., `20251015`)
- `HHMM`: Current time in 24-hour format (e.g., `1430`)
- Single dash separator between date and time
- This serves as the unique session identifier

**Example**: `20251015-1430/`

### FR2: Remove Title Argument from `trani start`
The system must remove the ability to pass a title argument to `trani start`:
- Current: `trani start 'meeting-notes'` ❌
- New: `trani start` ✅

### FR3: H1 Heading Extraction
After the user closes the notes editor (Neovim), the system must:
1. Read the first line of `notas.md`
2. Check if it starts with `# ` (H1 markdown syntax)
3. If yes, extract the heading text (everything after `# `)
4. If no, skip to FR7 (keep timestamp-only name)

### FR4: Heading Slugification
When an H1 heading is found, the system must slugify it using these rules:
1. Convert to lowercase
2. Replace spaces with hyphens (`-`)
3. Remove all special characters (keep only `a-z`, `0-9`, `-`)
4. Collapse multiple consecutive hyphens into single hyphen
5. Trim hyphens from start and end
6. Truncate to maximum 50 characters
7. Trim hyphens again after truncation

**Examples**:
- `# Awesome Session` → `awesome-session`
- `# Meeting 🚀 Notes!` → `meeting-notes`
- `# Project Planning: Q4 2025` → `project-planning-q4-2025`
- `# Very Long Session Title That Exceeds Fifty Characters Limit` → `very-long-session-title-that-exceeds-fifty-char` (50 chars)

### FR5: Directory Renaming
After slugification, the system must:
1. Rename the directory from `YYYYMMDD-HHMM/` to `YYYYMMDD-HHMM-{slug}/`
2. Update any internal state references to the new path
3. Continue normal session processing (recording, transcription, summary)

**Example**: `20251015-1430/` → `20251015-1430-awesome-session/`

### FR6: Fallback to Timestamp-Only
If any of these conditions are true, keep the directory name as timestamp-only:
- `notas.md` is empty
- First line doesn't start with `# `
- H1 text is empty (just `#` or `# ` with whitespace)
- Slugified result is empty after removing special characters

### FR7: Apply to `trani process` Command
The `trani process` command must follow the same logic:
1. Create directory with timestamp format `YYYYMMDD-HHMM/`
2. If notes file path is provided as argument, read it and extract H1
3. Slugify and rename directory if H1 exists
4. Fallback to timestamp-only if no notes or no H1

### FR8: Update Session State Management
The session state file (`current_session.json`) must store the final directory name after any renaming occurs.

## Non-Goals (Out of Scope)

1. **Migration of existing sessions**: Existing session directories with old naming format (`YYYY-MM-DD-title`) will not be automatically renamed
2. **Custom title argument**: The ability to manually specify a session title is removed (breaking change)
3. **Interactive prompts**: No prompts asking user to add H1 if missing - silent fallback to timestamp
4. **Duplicate name handling**: Since timestamp provides uniqueness, no special handling for duplicate slugs is needed
5. **Configuration options**: No config settings for timestamp format, slug length, or naming strategy
6. **Validation of H1 quality**: No checks for meaningful titles (e.g., won't reject `# test` or `# a`)

## Technical Considerations

### Code Impact Analysis

**Estimated Lines of Code**: ~40-60 new lines, ~10-15 lines removed

#### Files to Modify

1. **`cmd/start.go`** (~5 lines removed, ~5 lines changed)
   - Remove title argument parsing (lines 29-31)
   - Remove title parameter from `session.New()` call

2. **`cmd/toggle.go`** (~5 lines removed, ~5 lines changed)
   - Same changes as `start.go`

3. **`internal/session/session.go`** (~25-35 new lines, ~5 lines changed)
   - Change `New()` signature to remove title parameter
   - Remove default title generation logic (lines 47-50)
   - Update directory creation to use timestamp format (lines 52-53)
   - **NEW**: Add `extractAndRenameIfNeeded()` function after Neovim closes
   - **NEW**: Add `slugify()` helper function
   - Update state path reference after potential rename

4. **`internal/session/processor.go`** (~15-20 new lines, ~3 lines changed)
   - Update directory creation to timestamp format (lines 26-27)
   - **NEW**: Add H1 extraction logic after notes file reading
   - **NEW**: Call `slugify()` helper (can import from session package or duplicate)

#### New Functions Required

```go
// internal/session/session.go
func (s *Session) extractAndRenameIfNeeded() error {
    // Read notas.md first line
    // Check for H1
    // Slugify
    // Rename directory
    // Update s.path
    // ~15-20 lines
}

func slugify(text string) string {
    // Lowercase
    // Replace spaces with hyphens
    // Remove special chars
    // Collapse multiple hyphens
    // Trim
    // Truncate to 50 chars
    // ~15-20 lines
}
```

### Implementation Complexity: LOW

**Reasoning**:
- No external dependencies required (use stdlib `strings`, `regexp`, `unicode`)
- Logic is straightforward string manipulation
- Minimal changes to existing flow (just timestamp format change + rename step)
- No breaking changes to file structure or state management
- All changes localized to session creation functions

### Dependencies

- `strings` package (stdlib) - for `ToLower`, `ReplaceAll`, `TrimSpace`
- `regexp` package (stdlib) - for special character removal
- `unicode` package (stdlib) - for character classification
- `os` package (stdlib) - for directory renaming via `os.Rename()`

### Edge Cases to Handle

1. **Empty notes file**: Check file size or content length before reading first line
2. **Unicode in headings**: Ensure slug preserves readability for non-ASCII letters (e.g., `café` → `caf`)
3. **Very long headings**: Truncate at 50 chars but ensure no partial UTF-8 sequences
4. **Session state consistency**: Ensure state file is updated AFTER successful rename
5. **Rename failure**: If `os.Rename()` fails, log error but continue (timestamp-only name is valid)

## Success Metrics

1. **Code simplicity**: Implementation requires ≤60 new lines of code
2. **User friction reduction**: Zero required arguments for `trani start` command
3. **Naming accuracy**: 95% of sessions with H1 headings get properly slugified directory names
4. **Backward compatibility**: Old session directories remain accessible (read-only, no migration needed)

## Open Questions

1. **Unicode handling**: Should we preserve accented characters or transliterate them? (e.g., `ñ` → `n` or keep `ñ`?)
2. **Number-only slugs**: If slug is all numbers after special char removal, should we prefix with something? (e.g., `# 2025` → `2025` vs `session-2025`)
3. **Error messaging**: Should we notify user if directory rename fails, or silently continue with timestamp-only?

---

**Document Version**: 1.0
**Created**: 2025-10-15
**Target Implementation**: Trani v2 (Go implementation)
