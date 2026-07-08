# Cross-File Content Search Design

Date: 2026-07-08

## Problem

Current content search (`ModeContent`) only searches within the file currently open in the preview panel. Users often don't remember which file contains a keyword and need to search across all files in the directory.

## Decision

Add a third search mode `ModeAllContent` that searches the content of every file in the file list. Three modes cycle via Tab: `ModeFileName → ModeContent → ModeAllContent → ModeFileName`.

## Search Modes

| Mode | Scope | Result Type | Enter Behavior |
|------|-------|-------------|----------------|
| ModeFileName | File names | `FileMatch{Index, Entry}` | Open file, close search |
| ModeContent | Current preview file content | `Match{Line, Text}` | Jump to line, close search |
| ModeAllContent | All files content | `AllContentMatch{Path, Line, Text}` | Open file + jump to line, close search |

- `/` from file list → `ModeFileName` (unchanged)
- `/` from preview → `ModeContent` (unchanged)
- `ModeAllContent` is only reachable via Tab cycling

## Data Model

### New type

```go
type AllContentMatch struct {
    Path string  // relative path, e.g. "docs/api.md"
    Line int     // 0-based line number
    Text string  // matched line content
}
```

### Model additions

```go
type Model struct {
    // existing fields...
    AllMatches []AllContentMatch
    AllCurrent int
}
```

## Search Logic

### SearchAllContent

New method `SearchAllContent(root string, entries []discover.FileEntry)`:

1. Iterate over entries
2. Read each file at `filepath.Join(root, entry.Path)`
3. Skip files that fail to open (permission, missing)
4. Per-file: split by `\n`, `strings.Contains` per line (same logic as existing `Search()`)
5. Append matches with `Path` to `AllMatches`

Synchronous scan — acceptable for mdwalker's typical scale (10–100 small .md files).

### ToggleMode update

Cycle through three modes:

```go
func (m *Model) ToggleMode() {
    switch m.Mode {
    case ModeFileName:
        m.Mode = ModeContent
        m.input.Placeholder = "search..."
    case ModeContent:
        m.Mode = ModeAllContent
        m.input.Placeholder = "search..."
    case ModeAllContent:
        m.Mode = ModeFileName
        m.input.Placeholder = "search files..."
    }
    m.Matches = nil
    m.FileMatches = nil
    m.AllMatches = nil
    m.Current = 0
    m.FileCurrent = 0
    m.AllCurrent = 0
}
```

### UpdateSearch signature change

```go
func (m *Model) UpdateSearch(root string, files []discover.FileEntry, content string)
```

New `root` parameter needed for `SearchAllContent` to construct full file paths.

### Next / Prev

`ModeAllContent` operates on `AllCurrent`, symmetric to `ModeContent`'s use of `Current`.

## UI

### Search panel header

| Mode | Title | Meta label |
|------|-------|------------|
| ModeFileName | Search files | `files · 1/3` |
| ModeContent | Search content | `content · 2/5` |
| ModeAllContent | Search all files | `all · 2/15` |

### Match display (ModeAllContent)

```
› docs/api.md:L8  获取用户列表。
  docs/api.md:L16 创建新用户。
  overview.md:L4  快速浏览 AI 生成的文档
```

Max 8 entries shown; overflow shows `... N more matches`. Selected entry uses `selectedCandidateStyle`.

### Help line

Unchanged: `Tab switch  Enter open  Esc cancel`

## App Layer Changes

### Tab behavior

Simplified — Tab always cycles mode, no longer has "select file + switch" behavior in ModeFileName:

```go
case "tab":
    if m.search.Active {
        m.search.ToggleMode()
        m.search.UpdateSearch(m.root, m.files.Entries, m.preview.Content())
        skipSearchInput = true
    }
```

Selecting a file in ModeFileName is done with Enter (already supported).

### Enter for ModeAllContent

```go
if m.search.Active && m.search.Mode == search.ModeAllContent {
    match := m.search.CurrentAllMatch()
    m.openFile(match.Path)
    m.preview.ScrollToLine(match.Line)
    m.search.Deactivate()
    m.focus = focusPreview
}
```

### n / N for ModeAllContent

Move `AllCurrent` pointer, then open the matching file and scroll to line:

```go
case "n":
    if m.search.Active && m.search.Mode == search.ModeAllContent {
        m.search.Next()
        match := m.search.CurrentAllMatch()
        m.openFile(match.Path)
        m.preview.ScrollToLine(match.Line)
    }
```

### Status bar

Search mode label: `files` / `content` / `all`

## Edge Cases

- **File read failure**: skip the file, don't abort the scan
- **Single-file mode** (`mdwalker README.md`): `ModeAllContent` works identically to `ModeContent` (1 entry)
- **Empty query**: no matches, same as existing modes
- **Large result set**: capped at 8 displayed + overflow count, same as existing modes

## Tests

1. `TestAllContentSearchViewShowsMatches` — View output contains file paths and matched text
2. `TestToggleModeCyclesThroughThreeModes` — Tab cycles FileName → Content → AllContent → FileName
3. `TestAllContentSearchEmptyQuery` — empty query produces no matches
4. `TestAllContentSearchNoMatch` — no crash when nothing matches

## Files Changed

| File | Change |
|------|--------|
| `internal/search/search.go` | Add `ModeAllContent`, `AllContentMatch`, `AllMatches`/`AllCurrent` fields, `SearchAllContent()`, `CurrentAllMatch()`, update `ToggleMode()`/`Next()`/`Prev()`/`UpdateSearch()`/`View()`/`Activate()`/`Deactivate()` |
| `internal/search/search_test.go` | Add 4 new test cases |
| `internal/app/app.go` | Update Tab/Enter/n/N handlers, pass `m.root` to `UpdateSearch()`, update status bar |
