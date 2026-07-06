# 简化 mdwalker 为纯文本 Markdown 查看器 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除图片和 Mermaid 渲染功能，将 mdwalker 简化为纯文本 Markdown 查看器。

**Architecture:** 删除渲染相关代码，保留图片提取和占位符生成功能。Preview 包简化为直接使用 Glamour 渲染，不再处理媒体块。

**Tech Stack:** Go, Bubble Tea, Glamour

## Global Constraints

- 使用 `gofmt` 格式化所有修改的 `.go` 文件
- Go 版本 >= 1.21
- 所有测试必须通过 `go test ./...`
- Commit message 使用 Conventional Commits 格式

---

## File Structure

### 删除的文件
- `internal/markdown/mermaid.go`

### 修改的文件
- `internal/markdown/image.go` - 删除渲染函数，保留 `ExtractImages` 和 `RenderImagePlaceholder`
- `internal/preview/preview.go` - 简化渲染逻辑
- `internal/preview/preview_test.go` - 删除媒体渲染测试
- `internal/app/app.go` - 移除 'i' 键绑定
- `internal/config/config.go` - 移除图片/Mermaid 配置字段

---

### Task 1: 删除 Mermaid 渲染模块

**Files:**
- Delete: `internal/markdown/mermaid.go`

**Interfaces:**
- Produces: 无（此任务只删除代码，不产生新接口）

- [ ] **Step 1: 删除 mermaid.go 文件**

```bash
rm internal/markdown/mermaid.go
```

- [ ] **Step 2: 运行测试验证删除不影响其他模块**

运行: `go test ./internal/markdown/...`
预期: PASS（没有 mermaid 相关测试）

- [ ] **Step 3: Commit**

```bash
git add internal/markdown/mermaid.go
git commit -m "refactor(markdown): remove mermaid rendering module"
```

---

### Task 2: 简化 image.go - 删除渲染函数

**Files:**
- Modify: `internal/markdown/image.go`

**Interfaces:**
- Produces: 简化的 `image.go`，保留 `ImageRef`、`ExtractImages`、`RenderImagePlaceholder`

- [ ] **Step 1: 重写 image.go，删除渲染相关代码**

新文件内容:

```go
package markdown

import (
	"net/url"
	"regexp"
	"strings"
)

type ImageRef struct {
	Alt  string
	Path string
	Line int
}

var imgRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

func ExtractImages(content string) []ImageRef {
	var refs []ImageRef
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := imgRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			path := parseImageTarget(m[2])
			refs = append(refs, ImageRef{
				Alt:  m[1],
				Path: path,
				Line: i,
			})
		}
	}
	return refs
}

func parseImageTarget(target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "<") {
		if end := strings.Index(target, ">"); end >= 0 {
			return decodeImagePath(target[1:end])
		}
	}
	for i, r := range target {
		if r == ' ' || r == '\t' || r == '\n' {
			return decodeImagePath(target[:i])
		}
	}
	return decodeImagePath(target)
}

func decodeImagePath(path string) string {
	path = strings.TrimSpace(path)
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return decoded
}

func RenderImagePlaceholder(ref ImageRef) string {
	return "[Image: " + ref.Path + "]"
}
```

- [ ] **Step 2: 运行测试验证修改**

运行: `go test ./internal/markdown/... -v`
预期: `TestExtractImagesStripsTitleFromTarget` PASS，其他渲染测试 FAIL（将在 Task 4 处理）

- [ ] **Step 3: Commit**

```bash
git add internal/markdown/image.go
git commit -m "refactor(markdown): remove image rendering functions"
```

---

### Task 3: 简化 preview.go - 移除媒体渲染逻辑

**Files:**
- Modify: `internal/preview/preview.go`

**Interfaces:**
- Consumes: `markdown.ExtractImages`, `markdown.RenderImagePlaceholder`
- Produces: 简化的 `preview.Model`，不再有 `renderMedia`、`hasImage`、`needsClear` 字段

- [ ] **Step 1: 重写 preview.go，移除媒体处理**

新文件内容:

```go
package preview

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type Heading struct {
	Level int
	Text  string
	Line  int
}

type Model struct {
	viewport   viewport.Model
	renderer   *glamour.TermRenderer
	root       string
	filePath   string
	content    string
	headings   []Heading
	foldStates map[int]bool
	cursorLine int
	rendered   string
	width      int
	height     int
	ready      bool
}

func New() Model {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	return Model{
		renderer:   r,
		foldStates: make(map[int]bool),
	}
}

func (m *Model) LoadFile(root, path string) error {
	m.root = root
	fullPath := filepath.Join(root, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	m.filePath = path
	m.content = string(data)
	m.parseHeadings()
	m.cursorLine = 0
	m.renderFolded()
	m.viewport.GotoTop()
	return nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true
	m.renderFolded()
}

func (m *Model) parseHeadings() {
	m.headings = nil
	lines := strings.Split(m.content, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isFence := strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
		if isFence {
			if inFence {
				fenceChar := "`"
				if strings.HasPrefix(trimmed, "~~~") {
					fenceChar = "~"
				}
				rest := strings.TrimLeft(trimmed, fenceChar)
				if strings.TrimSpace(rest) == "" {
					inFence = false
				}
			} else {
				inFence = true
			}
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "#") {
			level := 0
			for _, c := range line {
				if c == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= 6 && len(line) > level && line[level] == ' ' {
				m.headings = append(m.headings, Heading{
					Level: level,
					Text:  strings.TrimSpace(line[level:]),
					Line:  i,
				})
			}
		}
	}
	m.foldStates = make(map[int]bool)
}

func (m *Model) contentRange(headingIdx int) (int, int) {
	startLine := m.headings[headingIdx].Line
	level := m.headings[headingIdx].Level
	endLine := len(strings.Split(m.content, "\n"))
	for j := headingIdx + 1; j < len(m.headings); j++ {
		if m.headings[j].Level <= level {
			endLine = m.headings[j].Line
			break
		}
	}
	return startLine, endLine
}

func (m *Model) ToggleFold(cursorLine int) {
	headingIdx := -1
	for i := range m.headings {
		start, end := m.contentRange(i)
		if cursorLine >= start && cursorLine < end {
			headingIdx = i
		}
	}
	if headingIdx < 0 || !m.IsLineVisible(cursorLine) {
		return
	}
	line := m.headings[headingIdx].Line
	m.foldStates[line] = !m.foldStates[line]
	m.renderFolded()
}

func (m *Model) IsLineVisible(line int) bool {
	for i, h := range m.headings {
		if m.foldStates[h.Line] {
			_, end := m.contentRange(i)
			if line > h.Line && line < end {
				return false
			}
		}
	}
	return true
}

func (m *Model) HeadingAtLine(line int) *Heading {
	for i := range m.headings {
		if m.headings[i].Line == line {
			return &m.headings[i]
		}
	}
	return nil
}

func (m *Model) CurrentHeading(cursorLine int) *Heading {
	var current *Heading
	for i := range m.headings {
		if m.headings[i].Line <= cursorLine {
			current = &m.headings[i]
		}
	}
	return current
}

func (m *Model) renderFolded() {
	if !m.ready || m.content == "" {
		return
	}
	lines := strings.Split(m.content, "\n")
	var visible []string
	for i, line := range lines {
		if m.IsLineVisible(i) {
			h := m.HeadingAtLine(i)
			if h != nil && m.foldStates[i] {
				visible = append(visible, line+"  ...")
			} else {
				visible = append(visible, line)
			}
		}
	}
	filtered := strings.Join(visible, "\n")
	rendered, err := m.renderer.Render(filtered)
	if err != nil {
		m.rendered = filtered
		m.viewport.SetContent(filtered)
		return
	}
	m.rendered = rendered
	m.viewport.SetContent(rendered)
}

func (m *Model) ScrollUp()           { m.viewport.LineUp(1) }
func (m *Model) ScrollDown()         { m.viewport.LineDown(1) }
func (m *Model) ScrollTop()          { m.viewport.GotoTop() }
func (m *Model) ScrollBottom()       { m.viewport.GotoBottom() }
func (m *Model) ScrollHalfPageUp()   { m.viewport.HalfViewUp() }
func (m *Model) ScrollHalfPageDown() { m.viewport.HalfViewDown() }

func (m Model) FilePath() string { return m.filePath }
func (m Model) Content() string  { return m.content }
func (m Model) CurrentLine() int { return m.cursorLine }

func (m *Model) ScrollToLine(line int) {
	m.cursorLine = line
	m.viewport.SetYOffset(m.renderedOffsetForLine(line))
}

func (m Model) renderedOffsetForLine(line int) int {
	sourceLines := strings.Split(m.content, "\n")
	if line < 0 || line >= len(sourceLines) {
		return 0
	}

	fullLine := strings.TrimSpace(sourceLines[line])
	if fullLine == "" {
		return line
	}

	headingText := strings.TrimSpace(strings.TrimLeft(fullLine, "#"))

	renderedLines := strings.Split(m.rendered, "\n")

	searchFrom := func(text string, start int) int {
		if start >= len(renderedLines) {
			start = len(renderedLines) - 1
		}
		if start < 0 {
			start = 0
		}
		for i := start; i < len(renderedLines); i++ {
			if text != "" && strings.Contains(renderedLines[i], text) {
				return i
			}
		}
		for i := 0; i < start; i++ {
			if text != "" && strings.Contains(renderedLines[i], text) {
				return i
			}
		}
		return -1
	}

	approxStart := line - 5
	if approxStart < 0 {
		approxStart = 0
	}

	if idx := searchFrom(fullLine, approxStart); idx >= 0 {
		return idx
	}
	if headingText != fullLine {
		if idx := searchFrom(headingText, approxStart); idx >= 0 {
			return idx
		}
	}
	return line
}

func (m *Model) SetCursorFromVisibleRow(row int) {
	if row < 0 {
		row = 0
	}
	line := m.viewport.YOffset + row
	if len(m.headings) > 0 {
		line = m.nearestHeadingLine(line)
	}
	m.cursorLine = line
}

func (m Model) nearestHeadingLine(visibleRow int) int {
	approxSourceLine := m.viewport.YOffset + max(0, (visibleRow-1)/2)
	best := approxSourceLine
	bestDistance := len(strings.Split(m.content, "\n")) + 1
	for _, h := range m.headings {
		distance := h.Line - approxSourceLine
		if distance < 0 {
			distance = -distance
		}
		if distance < bestDistance {
			bestDistance = distance
			best = h.Line
		}
	}
	return best
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
```

- [ ] **Step 2: 运行编译检查**

运行: `go build ./...`
预期: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/preview/preview.go
git commit -m "refactor(preview): remove media rendering logic"
```

---

### Task 4: 清理 preview_test.go - 删除媒体渲染测试

**Files:**
- Modify: `internal/preview/preview_test.go`

**Interfaces:**
- Consumes: 简化的 `preview.Model`

- [ ] **Step 1: 删除所有媒体渲染相关测试**

保留的测试:
- `TestToggleFoldFoldsCurrentSectionWhenCursorIsInsideIt`
- `TestScrollToLinePlacesRenderedHeadingAtTop`
- `TestScrollToLineUsesRenderedOffsetNotSourceLineApproximation`
- `TestGlamourPreservesMediaPlaceholder`（修改为验证图片占位符）

删除的测试:
- 所有 `*Image*` 测试
- 所有 `*Mermaid*` 测试
- `disableNativeImageProtocol`
- `installFakeChafa`
- `installFakeMMDC`
- `writeExecutable`
- `readFile`

新文件内容:

```go
package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToggleFoldFoldsCurrentSectionWhenCursorIsInsideIt(t *testing.T) {
	root := t.TempDir()
	content := "# First\nintro\n## Child\nchild body\n# Second\nsecond body\n"
	if err := os.WriteFile(filepath.Join(root, "headings.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 20)
	if err := m.LoadFile(root, "headings.md"); err != nil {
		t.Fatal(err)
	}

	m.ToggleFold(1)

	for _, hidden := range []int{1, 2, 3} {
		if m.IsLineVisible(hidden) {
			t.Fatalf("line %d remained visible after folding the current section", hidden)
		}
	}
	if !m.IsLineVisible(4) {
		t.Fatalf("next top-level heading should remain visible")
	}
}

func TestScrollToLinePlacesRenderedHeadingAtTop(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n\n")
	for i := 0; i < 20; i++ {
		b.WriteString("filler paragraph with enough text to render\n\n")
	}
	targetLine := strings.Count(b.String(), "\n")
	b.WriteString("## Target Heading\nbody\n")
	for i := 0; i < 20; i++ {
		b.WriteString("after target filler\n\n")
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 8)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	m.ScrollToLine(targetLine)

	firstLine := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(firstLine, "Target Heading") {
		t.Fatalf("ScrollToLine did not place rendered heading at top; first line=%q view=%q", firstLine, m.View())
	}
}

func TestScrollToLineUsesRenderedOffsetNotSourceLineApproximation(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("# Top\n\n")
	for i := 0; i < 8; i++ {
		b.WriteString("- list item before target\n")
		b.WriteString("  - nested item before target\n")
	}
	targetLine := strings.Count(b.String(), "\n")
	b.WriteString("## Precise Target\n")
	b.WriteString("body that should not be the first visible line\n")
	for i := 0; i < 8; i++ {
		b.WriteString("after target filler\n\n")
	}
	if err := os.WriteFile(filepath.Join(root, "doc.md"), []byte(b.String()), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	m.SetSize(80, 8)
	if err := m.LoadFile(root, "doc.md"); err != nil {
		t.Fatal(err)
	}

	m.ScrollToLine(targetLine)

	firstLine := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(firstLine, "Precise Target") {
		t.Fatalf("ScrollToLine first visible line = %q, want target heading; view=%q", firstLine, m.View())
	}
	if strings.Contains(firstLine, "body that should not") {
		t.Fatalf("ScrollToLine skipped past target heading: %q", m.View())
	}
}
```

- [ ] **Step 2: 运行测试验证**

运行: `go test ./internal/preview/... -v`
预期: 3 个测试 PASS

- [ ] **Step 3: Commit**

```bash
git add internal/preview/preview_test.go
git commit -m "refactor(preview): remove media rendering tests"
```

---

### Task 5: 简化 app.go - 移除 'i' 键绑定和图片打开功能

**Files:**
- Modify: `internal/app/app.go`

**Interfaces:**
- Consumes: 简化的 `preview.Model`（不再有 `LoadFileLight` 和 `ResolveAssetPath`）

- [ ] **Step 1: 修改 app.go**

删除:
- `openImage` 变量定义
- `openCurrentImage` 函数
- 'i' 键绑定处理（第 331-333 行）
- `markdown.CleanMermaidCache()` 调用（第 68 行）
- `LoadFileLight` 和 `loadFileWithMedia` 相关调用

修改 `loadFile` 相关逻辑，合并 `LoadFile` 和 `LoadFileLight`。

需要修改的具体位置:
- 第 64-65 行: 删除 `openImage` 变量
- 第 68 行: 删除 `markdown.CleanMermaidCache()`
- 第 331-333 行: 删除 `case "i":` 整个块
- 第 448-458 行: 删除 `openCurrentImage` 函数
- 第 414-438 行: 简化 `previewSelectedFile` 和 `loadFile` 函数

简化后的相关代码:

```go
// 删除第 64-65 行的 openImage 变量定义

// 第 67-84 行的 New 函数，删除 CleanMermaidCache 调用
func New(root string) *Model {
	cfg := config.Load()
	wl := config.LoadWhitelist()
	files := filelist.New()
	files.ShowTime = cfg.ShowTime

	return &Model{
		cfg:     cfg,
		wl:      wl,
		files:   files,
		preview: preview.New(),
		outline: outline.New(),
		search:  search.New(),
		root:    root,
	}
}

// 删除 case "i": 整个块（原第 331-333 行）

// 简化 previewSelectedFile（原第 414-420 行）
func (m *Model) previewSelectedFile() {
	path := m.files.SelectedFile()
	if path == "" {
		return
	}
	m.loadFile(path, false)
}

// 简化 loadFile（原第 422-438 行）
func (m *Model) loadFile(path string, recordHistory bool) {
	currentPath := m.preview.FilePath()
	if recordHistory && currentPath != "" && currentPath != path {
		m.history = append(m.history, currentPath)
	}
	m.preview.LoadFile(m.root, path)
	m.outline.SetContent(m.preview.Content())
	m.codeBlocks = markdown.ExtractBlocks(m.preview.Content())
}

// 删除 openCurrentImage 函数（原第 448-458 行）
```

- [ ] **Step 2: 运行编译检查**

运行: `go build ./...`
预期: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "refactor(app): remove image open keybinding and media loading"
```

---

### Task 6: 简化 config.go - 移除图片/Mermaid 配置字段

**Files:**
- Modify: `internal/config/config.go`

**Interfaces:**
- Produces: 简化的 `config.Config`，只保留 `ShowTime`

- [ ] **Step 1: 重写 config.go**

新文件内容:

```go
package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ShowTime bool
}

func Default() Config {
	return Config{
		ShowTime: false,
	}
}

func Load() Config {
	cfg := Default()
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}
	path := filepath.Join(home, ".config", "mdwalker", "config.toml")
	_ = path
	return cfg
}
```

- [ ] **Step 2: 运行编译检查**

运行: `go build ./...`
预期: 无编译错误

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor(config): remove image and mermaid config fields"
```

---

### Task 7: 清理 image_test.go - 删除渲染相关测试

**Files:**
- Modify: `internal/markdown/image_test.go`

**Interfaces:**
- Consumes: 简化的 `image.go`

- [ ] **Step 1: 删除渲染相关测试，保留 ExtractImages 测试**

新文件内容:

```go
package markdown

import (
	"testing"
)

func TestExtractImagesStripsTitleFromTarget(t *testing.T) {
	refs := ExtractImages(`![pic](assets/pic%201.png "caption")`)

	if len(refs) != 1 {
		t.Fatalf("ExtractImages() returned %d refs, want 1", len(refs))
	}
	if refs[0].Path != "assets/pic 1.png" {
		t.Fatalf("ExtractImages() path = %q, want decoded path without title", refs[0].Path)
	}
}
```

- [ ] **Step 2: 运行测试验证**

运行: `go test ./internal/markdown/... -v`
预期: 1 个测试 PASS

- [ ] **Step 3: Commit**

```bash
git add internal/markdown/image_test.go
git commit -m "refactor(markdown): remove image rendering tests"
```

---

### Task 8: 全量测试验证

**Files:**
- 无文件修改

- [ ] **Step 1: 运行全量测试**

运行: `go test ./...`
预期: 所有测试 PASS

- [ ] **Step 2: 运行编译检查**

运行: `go build ./...`
预期: 无编译错误

- [ ] **Step 3: 运行程序验证**

运行: `go run . testdata`
预期: 程序正常启动，可以浏览 Markdown 文件

---

### Task 9: 更新项目文档

**Files:**
- Modify: `README.md`（如果存在）
- Modify: `CLAUDE.md`

- [ ] **Step 1: 检查 README.md 是否需要更新**

运行: `cat README.md | grep -i "image\|mermaid"`
如果有相关描述，需要更新删除

- [ ] **Step 2: 更新 CLAUDE.md 中的错误记录**

删除与图片/Mermaid 渲染相关的错误记录（第 27-31 行），添加简化说明。

- [ ] **Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: update documentation for text-only viewer"
```

---

## Self-Review Checklist

**1. Spec coverage:**
- Task 1-2: 移除 Mermaid 渲染 ✓
- Task 2: 移除图片渲染函数 ✓
- Task 3: 简化 preview 渲染逻辑 ✓
- Task 5: 移除 'i' 键绑定 ✓
- Task 6: 移除配置字段 ✓
- Task 4,7: 删除相关测试 ✓

**2. Placeholder scan:**
- 无 TBD、TODO 等占位符 ✓
- 所有步骤包含完整代码 ✓

**3. Type consistency:**
- `ImageRef` 结构体在 Task 2 定义，Task 4 使用 ✓
- `preview.Model` 在 Task 3 简化，Task 5 使用 ✓
- 函数签名一致 ✓