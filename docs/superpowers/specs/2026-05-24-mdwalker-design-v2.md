# mdwalker 设计文档 v2

## 定位

**面向 AI agent 输出物的终端 Markdown 工作台。**

不是"更好看的 glow"。差异化在于：AI 输出物优先的文件发现、文档结构导航、实时刷新、语义高亮、命令块操作、标题折叠、双模式文件列表。

## 技术栈

Go:
- bubbletea — TUI 框架
- glamour — Markdown → ANSI 终端渲染
- lipgloss — 样式
- bubbles — viewport / textinput 等 TUI 组件
- fsnotify — watch 模式文件监听
- 外部工具：fd（可选加速文件搜索）、mmdc（可选 Mermaid 渲染）、chafa/viu（可选图片降级渲染）

## 两栏布局

```
┌─ Files ───────┬─ Preview ───────────────────────────────────────┐
│ AGENTS.md  ●  │ # Project Overview                              │
│ CLAUDE.md     │ ─────────────────                               │
│ .ai/summary   │                                                 │
│ reports/logs  │ ```bash                                         │
│ notes/idea.md │ cargo run                     [copy: y]         │
│               │ ```                                             │
├───────────────┴─────────────────────────────────────────────────┤
│ /search                            ● watch │ Tab:切换 Esc:取消  │
└─────────────────────────────────────────────────────────────────┘
```

- 左栏：文件列表，约 20% 宽度，AI 优先级排序
- 右栏：预览，约 80% 宽度
- 大纲：`o` 键切换浮动面板，叠加在右侧预览之上
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

## 交互

### 面板切换

| 按键 | 功能 |
|------|------|
| `h` / `←` | 焦点左移（预览 → 大纲 → 文件列表） |
| `l` / `→` | 焦点右移（文件列表 → 大纲 → 预览） |
| `Tab` | 文件列表 ↔ 预览快速切换（保留） |
| `j` / `k` / `↑` / `↓` | 当前聚焦面板内的上下移动 |
| 鼠标点击 | 直接切换焦点到被点击的面板 |

### 统一搜索（`/`）

`/` 键根据当前焦点自动切换搜索对象：

**焦点在文件列表：**
- 搜索栏显示在底部状态栏
- 匹配文件名高亮
- `Enter` — 打开当前匹配的文件
- `Tab` — 切换为预览内容搜索
- `Esc` — 取消搜索

**焦点在预览：**
- 搜索内容文本，实时高亮匹配项
- `n` / `N` — 下一个/上一个匹配
- `Tab` — 切换为文件名搜索
- `Esc` — 取消搜索

### 其他快捷键

| 按键 | 功能 |
|------|------|
| `Enter` | 文件列表：打开选中文件；大纲：跳转到标题 |
| `Space` | 预览中折叠/展开当前标题下的内容 |
| `o` | 打开/关闭大纲浮动面板 |
| `t` | 文件栏切换树模式 / 列表模式 |
| `y` | 复制当前光标所在代码块到剪贴板 |
| `i` | 用系统默认程序打开当前图片 |
| `b` | 返回上一个文件 |
| `r` | 重新扫描目录 |
| `g` | 跳到预览顶部 |
| `G` | 跳到预览底部 |
| `q` | 退出 |

### 鼠标支持

- 点击文件列表 → 选中并加载预览
- 滚轮 → 滚动预览
- 点击标题折叠 → 不实现（避免图标污染复制）
- 文本选中 → 鼠标事件透传，不触发任何应用逻辑

## 文件发现与排序

### fd 集成

启动时检测系统是否安装 `fd`：
- 有 `fd`：调用 `fd --type f --extension md` 快速搜索
- 无 `fd`：Go 原生 `filepath.WalkDir` 兜底

### AI 优先级排序

1. `AGENTS.md` / `CLAUDE.md` / `README.md` — 始终置顶
2. 最近 24 小时内修改的 `.md`
3. `.ai/`、`.claude/`、`.codex/` 目录下的 `.md`
4. `docs/`、`notes/`、`reports/` 目录下的 `.md`
5. 其余 `.md`，按修改时间倒序

## 文件列表 — 双模式

`t` 键切换。

### 树模式

按目录层级展示为可折叠的树形结构：

```
├─ AGENTS.md          3m
├─ .ai/
│  └─ summary.md      5m
├─ docs/
│  ├─ guide.md        2h
│  └─ api.md          1d
└─ README.md          7d
```

- 目录节点可 `Enter` 折叠/展开
- 选中文件 `Enter` 打开预览

### 列表模式（优化）

- 时间戳颜色使用浅色（`#6b6b6b`），不干扰文件名辨识
- 长路径软换行时，换行部分前加两个空格缩进，与下一文件的首行形成明显视觉差异
- 当前选中行反色高亮

## 标题折叠

- 预览中光标停在标题行时，按 `Space` 或 `Enter` 折叠/展开该标题下所有内容（直到遇到同级或更高级标题）
- 折叠后标题行末尾追加 `…`（省略号），表示内容已隐藏
- 无任何图标字符（`▼`/`▶` 等），保证文本复制完全纯净
- 折叠状态视觉提示：标题颜色变暗
- 嵌套折叠：折叠父标题同时隐藏所有子内容。展开父标题后，子标题各自保持自己的折叠状态
- 光标移动时跳过折叠内容：折叠后 `j`/`k` 直接跳到下一可见行

## 大纲

`o` 键切换浮动面板。从当前文件所有标题自动生成树形结构。渲染为独立层，确保在任何预览位置都能正确叠加显示。

- 方向键选择，`Enter` 跳转到对应位置
- `Esc` 或 `o` 关闭
- 滚动预览时大纲自动高亮当前所在标题

## 语义高亮

| 模式 | 渲染效果 |
|------|----------|
| `Error:` / `[ERROR]` / `❌` 开头行 | 红色前缀标记 |
| `Warning:` / `[WARN]` / `⚠️` 开头行 | 黄色前缀标记 |
| `TODO:` / `[TODO]` 开头行 | 复选框样式 |
| `Next Steps:` / `Decision:` 开头行 | 蓝色前缀标记 |

## 代码块增强

- 语言标签显示在代码块顶部
- 长行软换行，不加标记，不污染复制
- `y` 键复制当前代码块到系统剪贴板
- diff 代码块：`-` 行红色、`+` 行绿色
- 超长代码块（>100 行）默认折叠到前 30 行，`Enter` 展开/折叠

## 图片渐进增强

```
检测终端能力
  ├── 支持 kitty/iTerm2 协议 → 内联显示
  ├── tmux 且父终端支持 → 透传显示
  ├── 安装了 chafa/viu → 半块像素显示
  └── 都不行 → 显示 [Image: path.png]  按 i 用系统默认程序打开
```

## Mermaid 渲染

- 默认 `auto`：有 mmdc 则渲染，没有则显示源码
- Mermaid 缓存：`~/.cache/mdwalker/mermaid/`，内容哈希去重，24h 过期清理
- 安装提醒：README 中提示 `npm install -g @mermaid-js/mermaid-cli`

## watch 模式

默认启用。fsnotify 监听目标目录中 `.md` 文件的创建、修改、删除。文件列表实时更新，当前文件变化自动刷新预览。用 `--no-watch` 关闭。

## 配置

可选的 `~/.config/mdwalker/config.toml`：

```toml
image_protocol = "auto"           # auto | kitty | halfblock | off
mermaid_mode = "auto"             # auto | code | browser
mmdc_path = "/usr/local/bin/mmdc"
```

零配置可用。

## 测试用例

生成测试用 markdown 文件到 `testdata/` 目录，覆盖：

- `testdata/basic.md` — 各层级标题、段落、加粗斜体删除线、行内代码
- `testdata/codeblocks.md` — 多语言代码块、diff、超长代码块、mermaid
- `testdata/lists.md` — 有序/无序/嵌套列表、任务列表
- `testdata/tables.md` — GFM 表格、超宽表格
- `testdata/images.md` — 内联图片引用
- `testdata/semantic.md` — Error/Warning/TODO/Decision 语义标注
- `testdata/links.md` — 内联链接、引用链接
- `testdata/headings.md` — 多层级标题、深层嵌套，用于测试折叠和目录

## v0.2 范围（本次）

- 左右方向键面板切换 + 鼠标点击切换
- 统一搜索（`/` 按焦点切换搜索目标 + `Tab` 切换模式）
- 文件列表双模式（树模式 + 优化列表模式）
- 标题折叠（预览中 `Space`/`Enter`，无图标）
- 大纲面板修复（独立渲染层）
- 测试用例

## 依赖

| 包 | 用途 |
|----|------|
| github.com/charmbracelet/bubbletea | TUI 框架 |
| github.com/charmbracelet/glamour | Markdown ANSI 渲染 |
| github.com/charmbracelet/lipgloss | 终端样式 |
| github.com/charmbracelet/bubbles | TUI 组件（viewport、textinput） |
| github.com/fsnotify/fsnotify | watch 模式文件监听 |

## 不做的事情

- Markdown 编辑
- Frontmatter 解析
- PDF/HTML 导出
- 插件系统
- WikiLinks 或 Obsidian 特有语法
- 折叠图标（`▼`/`▶`）
