# Repository Guidelines

## Project Structure & Module Organization

`mdwalker` is a Go terminal UI for browsing Markdown. The entry point is `main.go`, with application logic under `internal/`: `app` wires the Bubble Tea model (includes file watcher), `config` handles CLI arguments and YAML whitelist loading, `discover` finds Markdown files with whitelist-driven scanning, `markdown` provides code/image/mermaid/semantic extraction and rendering, and UI packages `filelist`, `preview`, `outline`, `search` each own a Bubble Tea model with update/view. Markdown fixtures live in `testdata/`. Design notes and implementation plans are under `docs/superpowers/`.

## Build, Test, and Development Commands

- `go run .` starts the TUI against the current directory.
- `go run . README.md` previews a single Markdown file.
- `go run . -- --no-watch` runs without filesystem watching.
- `go build ./...` compiles all packages and catches integration errors.
- `go test ./...` runs the full Go test suite; use it before opening a PR.
- `go install github.com/bairea/mdwalker@latest` installs the published CLI.

Optional runtime tools improve features: `fd` speeds file discovery, `mmdc` enables Mermaid rendering, and `chafa` improves terminal image fallback.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on changed `.go` files before committing. Keep packages small and named after their responsibility, matching the existing `internal/<feature>` pattern. Prefer clear exported names only where cross-package use requires them; keep helpers unexported. Use tabs for Go indentation as produced by `gofmt`.

## Testing Guidelines

Add `_test.go` files next to the package being tested. Prefer table-driven tests for parsers, discovery, search, and rendering helpers, using `testdata/*.md` fixtures when behavior depends on Markdown structure. For TUI changes, test pure state transitions and parsing functions where possible, then run `go test ./...` and a local `go run . testdata` smoke check.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit prefixes such as `feat:`, `fix:`, and `docs:`; keep using that style with a concise imperative summary. For agent-authored commits, include decision trailers when useful: `Constraint:`, `Rejected:`, `Confidence:`, `Scope-risk:`, `Tested:`, and `Not-tested:`. PRs should describe user-visible behavior, list verification commands, link related issues or plans, and include terminal screenshots or recordings for UI changes.

## 运行错误记录

- 2026-05-27：在 Windows 环境运行 `go test ./internal/discover -run TestScanSkipSubdirs -count=1 -v` 时，测试失败于 `TempDir RemoveAll cleanup: unlinkat ... The process cannot access the file because it is being used by another process.`。必要环境信息：`go version go1.26.3 windows/amd64`，PowerShell，工作目录 `D:\Desktopfile\chores\mdwalker`。原因是测试中 `os.Chdir` 进入 `t.TempDir()` 后没有恢复当前目录，导致 Windows 无法清理临时目录；已改为 `t.Chdir`。
- 2026-05-27：运行 `go test ./...` 时，`internal/preview` 中 `TestPreviewImageFileUsesImagePlaceholder`、`TestPreviewImageFileRendersTerminalImageWhenRendererAvailable`、`TestPreviewMarkdownImageResolvesRelativePathAndRenders`、`TestPreviewRendersMermaidThroughDiagramAndImagePipeline`、`TestPreviewImageRendererUsesViewportBoundedSize` 失败。必要环境信息：`go version go1.26.3 windows/amd64`，PowerShell，工作目录 `D:\Desktopfile\chores\mdwalker`。失败摘要：OSC 1337 图片序列在输出中被换行或拆分，Mermaid 测试中的临时 `mmdc` 可执行文件未被 Windows PATH 解析找到；该问题与本次白名单路径修复无直接关系，未在本次任务中修改 `internal/preview`。
- 2026-05-27：修复图片协议时运行目标测试，`internal/markdown` 暴露 Kitty 序列错误地使用 `f=24` 标记文件字节为 raw RGB，WezTerm 的 iTerm2 序列缺少稳定光标参数；`internal/preview` 暴露 Glamour/viewport 会拆分 OSC 1337 图片序列，Windows 下测试用 fake `chafa`/`mmdc` shell 脚本不可执行。必要环境信息：`go version go1.26.3 windows/amd64`，PowerShell，`TERM_PROGRAM=WezTerm`，工作目录 `D:\Desktopfile\chores\mdwalker`。已改为 Kitty PNG graphics protocol、WezTerm/iTerm2 inline image protocol、含图片控制序列跳过 Markdown 渲染，并为 Windows 测试 helper 使用 `.cmd`。
- 2026-05-27：继续修复图片显示时，目标测试暴露 Kitty/WezTerm 仍嵌入整图 base64 内容导致直接看图慢，Markdown 图片 `![pic](assets/pic.png "caption")` 会把 title 一起当成路径而回退占位符；全量 `go test ./...` 暴露旧测试仍期望 WezTerm 使用 iTerm2 OSC 序列。必要环境信息：`go version go1.26.3 windows/amd64`，PowerShell，`TERM_PROGRAM=WezTerm`，工作目录 `D:\Desktopfile\chores\mdwalker`。已改为 Kitty/WezTerm 使用 Kitty `t=f` 文件路径传输，并剥离 Markdown 图片 target 中的 title。
- 2026-05-27：按截图继续诊断图片显示时，Bubble Tea/app 级反馈环路暴露文件列表轻量预览 `LoadFileLight` 对 Markdown 引用图片只显示 `[Image: ...]`，未走原生图片协议；相对引用图按 `i` 打开时未按 Markdown 文件所在目录解析；真实 PNG 使用 `t=f` 普通文件路径传输在 WezTerm/Kitty 下兼容性不足。必要环境信息：`go version go1.26.3 windows/amd64`，PowerShell，`TERM_PROGRAM=WezTerm`，工作目录 `D:\Desktopfile\chores\mdwalker`。已改为轻量预览在原生协议可用时也渲染图片但不调用外部 `chafa/viu`，按 `i` 打开图片前解析为 Markdown 文件相对路径，真实图片使用 Kitty `t=t` 临时文件传输。
