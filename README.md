# mdwalker

AI agent 输出物专用终端 Markdown 工作台 — 在终端中快速浏览、导航、搜索 AI 生成的大量 .md 文件。

![demo](docs/demo.gif)

## 核心功能

- **双面板布局** — 左侧文件列表 + 右侧 Markdown 预览，支持鼠标点击和键盘操作
- **浮动大纲面板** — 按 `o` 打开/关闭，显示文档标题结构，Enter 跳转到对应位置
- **统一搜索** — 按 `/` 打开搜索，文件列表模式搜文件名，预览模式搜内容，`Tab` 切换搜索模式（文件名 → 单文件内容 → 全文件内容）
- **树形文件视图** — 按 `t` 切换，按目录层级展示文件，j/k 按树形顺序移动
- **标题折叠** — 预览中按 `Space` 折叠/展开当前标题下的内容
- **代码块操作** — 按 `y` 复制当前代码块到剪贴板
- **文件监听** — 自动检测目录变化，实时更新文件列表
- **历史导航** — 按 `b` 返回上一个打开的文件
- **目录白名单** — 自动发现 `.claude/`、`.agents/` 等 AI 工具目录中被 `.gitignore` 忽略的文档，排除 `*/skills/` 子目录

## 安装

### macOS

```bash
# Homebrew
brew tap bairea/tap
brew install mdwalker

# 或者 go install
go install github.com/bairea/mdwalker@latest
```

### Linux

```bash
go install github.com/bairea/mdwalker@latest
```

或者从 [GitHub Releases](https://github.com/Bairea/mdwalker/releases/latest) 下载对应架构的 tar.gz。

### Windows

```bash
# Scoop
scoop bucket add bairea https://github.com/Bairea/scoop-bucket
scoop install mdwalker

# 或者 go install
go install github.com/bairea/mdwalker@latest
```

也可以从 [GitHub Releases](https://github.com/Bairea/mdwalker/releases/latest) 下载 zip 解压后直接运行。

## 使用

```bash
mdwalker                  # 当前目录
mdwalker docs/            # 指定目录
mdwalker README.md        # 单文件预览
mdwalker --show-time      # 显示文件修改时间
mdwalker --no-watch       # 关闭文件监听
```

## 快捷键

### 导航

| 按键 | 功能 |
|------|------|
| `j` / `k` / `↓` / `↑` | 当前面板内移动 |
| `h` / `←` | 焦点左移 |
| `l` / `→` | 焦点右移 |
| `Tab` | 文件列表 ↔ 预览快速切换 |
| `g` | 跳转到顶部 |
| `G` | 跳转到底部 |

### 文件操作

| 按键 | 功能 |
|------|------|
| `Enter` | 打开选中文件 |
| `t` | 文件栏切换树模式 / 列表模式 |
| `r` | 重新扫描目录 |
| `b` | 返回上一个文件 |

### 大纲

| 按键 | 功能 |
|------|------|
| `o` | 打开 / 关闭大纲面板 |
| `Enter` | 跳转到选中标题（焦点保留在大纲） |

### 搜索

| 按键 | 功能 |
|------|------|
| `/` | 打开搜索（文件栏搜文件名，预览搜内容） |
| `Tab` | 切换搜索模式（文件名 → 单文件内容 → 全文件内容） |
| `n` / `N` | 下一个 / 上一个匹配 |
| `Enter` | 打开匹配文件或跳转到匹配行 |
| `Esc` | 关闭搜索 |

### 内容操作

| 按键 | 功能 |
|------|------|
| `Space` | 折叠 / 展开标题内容 |
| `y` | 复制当前代码块 |

### 全局

| 按键 | 功能 |
|------|------|
| `q` / `Ctrl+C` | 退出 |
| 鼠标左键 | 点击文件列表打开文件、点击大纲跳转标题 |

## 配置

### config.toml

`~/.config/mdwalker/config.toml`：

```toml
# 显示文件修改时间（默认 false）
show_time = false
```

### whitelist.yaml

`~/.config/mdwalker/whitelist.yaml`（全局）和项目根目录 `mdwalker-whitelist.yaml`（可选，与全局合并）：

```yaml
# 被 .gitignore 但需要 mdwalker 扫描的目录
unignore:
  dot_dirs:    # 隐藏 AI 工具目录
    - .claude
    - .agents
  paths:       # 普通目录
    - docs/superpowers
  files:       # 单独文件
    - AGENTS.md
    - CLAUDE.md

# unignore 扫描时跳过的子目录
skip_subdirs:
  - "*/skills"
```

内置默认值覆盖 31 个 AI 工具的 `.X` 目录（`.claude`、`.agents`、`.pi`、`.trae`、`.augment`、`.windsurf` 等），无需手动配置即可使用。
