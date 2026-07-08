# Repository Guidelines

## Project Structure & Module Organization

`mdwalker` is a Go terminal UI for browsing Markdown. The entry point is `main.go`, with application logic under `internal/`: `app` wires the Bubble Tea model (includes file watcher), `config` handles CLI arguments and YAML whitelist loading, `discover` finds Markdown files with whitelist-driven scanning, `markdown` provides code block extraction and clipboard copy, and UI packages `filelist`, `preview`, `outline`, `search` each own a Bubble Tea model with update/view. Markdown fixtures live in `testdata/`. Design notes and implementation plans are under `docs/superpowers/`.

## Build, Test, and Development Commands

- `go run .` starts the TUI against the current directory.
- `go run . README.md` previews a single Markdown file.
- `go run . -- --no-watch` runs without filesystem watching.
- `go build ./...` compiles all packages and catches integration errors.
- `go test ./...` runs the full Go test suite; use it before opening a PR.
- `go install github.com/bairea/mdwalker@latest` installs the published CLI.

Optional runtime tools improve features: `fd` speeds file discovery.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt` on changed `.go` files before committing. Keep packages small and named after their responsibility, matching the existing `internal/<feature>` pattern. Prefer clear exported names only where cross-package use requires them; keep helpers unexported. Use tabs for Go indentation as produced by `gofmt`.

## Testing Guidelines

Add `_test.go` files next to the package being tested. Prefer table-driven tests for parsers, discovery, search, and rendering helpers, using `testdata/*.md` fixtures when behavior depends on Markdown structure. For TUI changes, test pure state transitions and parsing functions where possible, then run `go test ./...` and a local `go run . testdata` smoke check.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit prefixes such as `feat:`, `fix:`, and `docs:`; keep using that style with a concise imperative summary. For agent-authored commits, include decision trailers when useful: `Constraint:`, `Rejected:`, `Confidence:`, `Scope-risk:`, `Tested:`, and `Not-tested:`. PRs should describe user-visible behavior, list verification commands, link related issues or plans, and include terminal screenshots or recordings for UI changes.

## 运行错误记录

- 2026-05-27：在 Windows 环境运行 `go test ./internal/discover -run TestScanSkipSubdirs -count=1 -v` 时，测试失败于 `TempDir RemoveAll cleanup: unlinkat ... The process cannot access the file because it is being used by another process.`。必要环境信息：`go version go1.26.3 windows/amd64`，PowerShell，工作目录 `D:\Desktopfile\chores\mdwalker`。原因是测试中 `os.Chdir` 进入 `t.TempDir()` 后没有恢复当前目录，导致 Windows 无法清理临时目录；已改为 `t.Chdir`。
