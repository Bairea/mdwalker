# mdwalker 设计文档

## 定位

**面向 AI agent 输出物的终端 Markdown 工作台。**

不是"更好看的 glow"。差异化在于：AI 输出物优先的文件发现、文档结构导航、实时刷新、语义高亮、命令块操作、图片和 Mermaid 渐进增强。

目标用户场景：AI 终端工具在项目中生成了一堆 `.md` 文件（总结、报告、日志、错误分析等），用户需要快速找到并阅读这些内容。

## 技术栈

Go:
- bubbletea — TUI 框架
- glamour — Markdown → ANSI 终端渲染
- lipgloss — 样式
- bubblezone — 鼠标支持
- fsnotify — watch 模式文件监听
- 外部工具：fd（可选加速文件搜索）、mmdc（可选 Mermaid 渲染）、chafa/viu（可选图片降级渲染）

选 Go 的理由：Glamour 已有成熟 Markdown 终端渲染能力，标题、代码块、表格、列表等开箱即用，不需要从零造渲染管线。开发精力集中在差异化功能上。

## 两栏布局

```
┌─ Files ───────┬─ Preview ───────────────────────────────────────┐
│ AGENTS.md  ●  │ # Project Overview                              │
│ CLAUDE.md     │ ─────────────────                               │
│ README.md     │ ...                                             │
│ .ai/summary   │                                                 │
│ reports/logs  │ ```bash                                         │
│ notes/idea.md │ cargo run                     [copy: y]         │
│               │ ```                                             │
└───────────────┴─────────────────────────────────────────────────┘
```

- 左栏：文件列表，约 20% 宽度，AI 优先级排序
- 右栏：预览，约 80% 宽度
- 大纲：`o` 键切换浮动面板，叠加在右侧预览上方，不占用常驻空间
- 终端宽度 < 60 列时文件列表自动隐藏，预览占满全宽

## 启动方式

```
mdwalker                  # 扫描当前目录，进入 TUI（默认 watch）
mdwalker docs/            # 扫描指定目录，进入 TUI
mdwalker README.md        # 打开指定文件，单栏全屏预览
mdwalker a.md b.md        # 打开多个指定文件，进入 TUI
mdwalker --no-watch       # 关闭文件监听
mdwalker --mermaid auto   # Mermaid 渲染策略（默认 auto）
mdwalker --mermaid code   # 只显示 Mermaid 源码
mdwalker --mermaid browser # Mermaid 渲染后在浏览器打开
```

## 文件发现与排序

### fd 集成

启动时检测系统是否安装 `fd`：
- 有 `fd`：调用 `fd --type f --extension md` 快速搜索，毫秒级出结果
- 无 `fd`：Go 原生 `filepath.WalkDir` 兜底，goroutine 并发扫描，忽略 `.git/`、`node_modules/`、`target/`、`.direnv/`、`__pycache__/`

用户无需关心用哪种方式，无感切换。

### AI 优先级排序

按以下分组从上到下排列，组内按修改时间倒序：

1. `AGENTS.md` / `CLAUDE.md` / `README.md` — 项目级 AI 上下文文件，始终置顶
2. 最近 24 小时内修改的 `.md` — AI 刚生成的，最可能是用户想看的
3. `.ai/`、`.claude/`、`.codex/` 目录下的 `.md`
4. `docs/`、`notes/`、`reports/` 目录下的 `.md`
5. 其余 `.md`，按修改时间倒序

每个文件显示相对路径 + 人类可读修改时间（如"3分钟前"、"2小时前"）。

## 大纲

`o` 键切换浮动面板。从当前文件所有标题自动生成树形结构。方向键选择，Enter 跳转到对应位置。滚动预览时大纲自动高亮当前所在标题。

## 语义高亮

在 Glamour 渲染之外，对 AI 生成 Markdown 中的典型模式做行级正则识别：

| 模式 | 渲染效果 |
|------|----------|
| `Error:` / `[ERROR]` 开头行 | 红色前缀标记 |
| `Warning:` / `[WARN]` 开头行 | 黄色前缀标记 |
| `TODO:` / `[TODO]` 开头行 | 复选框样式 |
| `Next Steps:` / `Decision:` 开头行 | 蓝色前缀标记 |

不做复杂 NLP，行级正则匹配，零配置自动生效。

## 代码块增强

- 语言标签显示在代码块顶部
- 代码块语法高亮（Glamour 内置 + Chroma）
- 长行软换行，不加任何标记，不污染文本复制
- `y` 键复制当前光标所在代码块内容到系统剪贴板（包括 Mermaid 代码块）
- diff 代码块：`-` 行红色背景、`+` 行绿色背景
- 超长代码块（>100 行）默认折叠到前 30 行，Enter 展开/折叠

## 图片渐进增强

```
检测终端能力
  ├── 支持 kitty/iTerm2 协议 → 内联显示（高分辨率）
  ├── tmux 且父终端支持 → 透传显示
  ├── 安装了 chafa/viu → 调外部工具转半块像素显示
  └── 都不行 → 显示 [Image: path.png] (1200x800)  按 i 用系统默认程序打开
```

- `i` 键：用系统默认程序打开当前图片
- 不强制依赖任何图形协议

## Mermaid 渲染

```
检测到 mermaid 代码块
       │
       ▼
  检查 --mermaid 参数
       │
       ├── auto（默认）：有 mmdc 则渲染为图片显示，没有则显示源码
       ├── code：始终显示 Mermaid 源码
       └── browser：调 mmdc 生成图片后在浏览器打开
```

- Mermaid 源码块同样支持 `y` 键复制原始内容
- 安装文档中提醒用户安装 mmdc：`npm install -g @mermaid-js/mermaid-cli`
- 缓存：mmdc 生成的图片以内容哈希命名缓存在 `~/.cache/mdwalker/mermaid/`，每次启动清理 24 小时前的缓存条目，源内容变化时自动重新生成

## watch 模式

默认启用。通过 fsnotify 监听目标目录中 `.md` 文件的变更（创建、修改、删除）。文件列表实时更新，当前打开的文件如有变化自动刷新预览内容。

用 `--no-watch` 关闭。

AI agent 边运行边写 Markdown 时，用户不需要反复退出重进。

## 交互

### 键盘快捷键

| 按键 | 功能 |
|------|------|
| `j` / `k` / `↑` / `↓` | 文件列表移动选中；预览区上下滚动 |
| `h` / `l` | 左右切换焦点（文件列表 ↔ 预览） |
| `Tab` | 文件列表/大纲/预览之间切换焦点 |
| `Enter` | 打开选中文件；大纲中跳转到标题 |
| `o` | 打开/关闭大纲浮动面板 |
| `/` | 打开搜索栏，实时高亮匹配 |
| `n` / `N` | 跳转到下一个/上一个匹配 |
| `Esc` | 关闭搜索栏 / 关闭大纲面板 |
| `y` | 复制当前光标所在代码块到剪贴板（包括 Mermaid 代码块） |
| `i` | 用系统默认程序打开当前图片 |
| `b` | 返回上一个位置 |
| `r` | 重新扫描目录 |
| `g` | 跳到预览顶部 |
| `G` | 跳到预览底部 |
| `q` | 退出 |

### 鼠标支持

- 点击文件列表 → 选中并加载预览
- 滚轮 → 滚动预览
- 文本选中 → 鼠标事件透传，不触发任何应用逻辑

## 搜索

在渲染后的预览内容上做纯文本搜索。匹配结果反色高亮。搜索栏在屏幕底部，单行。焦点在文件列表时按 `/` 则在文件名中搜索。

## 配置

可选的 `~/.config/mdwalker/config.toml`：

```toml
image_protocol = "auto"           # auto | kitty | halfblock | off
mermaid_mode = "auto"             # auto | code | browser
mmdc_path = "/usr/local/bin/mmdc"
```

零配置可用。

## 错误处理

- `fd` 未安装：自动使用 Go 原生文件搜索
- `mmdc` 未安装：Mermaid 代码块显示为普通代码块，菜单提示安装
- 图片文件不存在：显示 `[Image not found: path]` 占位
- 文件编码非 UTF-8：显示错误信息，程序继续运行
- 终端不支持图形协议：渐进降级到占位符 + `i` 键外部打开

## v0.1 范围

- 当前目录递归发现 .md，AI 优先级排序
- fd 集成（可选加速）
- 两栏布局：文件列表 + 预览
- 大纲浮动面板
- 搜索
- watch 模式
- 代码块增强（语言标签、语法高亮、diff 渲染、长代码折叠、`y` 复制）
- 语义高亮（Error/Warning/TODO/Decision）
- 图片路径识别 + `i` 键外部打开 + 协议检测内联显示
- Mermaid 代码块识别 + `y` 复制源码 + mmdc 渲染（如有安装）
- OSC 8 超链接（Glamour 已有支持）
- 位置导航（`b` 返回、`r` 刷新）

## v0.2 范围（待定）

- 图片/Mermaid 内联渲染完善
- 超长文档分段加载
- 多目录索引
- 导出 HTML

## 依赖

| 包 | 用途 |
|----|------|
| github.com/charmbracelet/bubbletea | TUI 框架 |
| github.com/charmbracelet/glamour | Markdown ANSI 渲染 |
| github.com/charmbracelet/lipgloss | 终端样式 |
| github.com/charmbracelet/bubbles | TUI 组件（viewport 等） |
| github.com/fsnotify/fsnotify | watch 模式文件监听 |
| github.com/alecthomas/chroma/v2 | 代码语法高亮 |

## 不做的事情

- Markdown 编辑
- Frontmatter 解析
- PDF/HTML 导出（v0.1 不做）
- 插件系统
- WikiLinks 或 Obsidian 特有语法
