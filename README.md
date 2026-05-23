# mdwalker

AI agent 输出物专用终端 Markdown 工作台。

## 安装

```bash
go install github.com/bairea/mdwalker@latest
```

## 推荐安装（可选增强）

```bash
# 更快的文件搜索
brew install fd

# Mermaid 图表渲染
npm install -g @mermaid-js/mermaid-cli

# 终端图片渲染（降级方案）
brew install chafa
```

## 使用

```bash
mdwalker                  # 当前目录，默认 watch
mdwalker docs/            # 指定目录
mdwalker README.md        # 单文件预览
mdwalker --no-watch       # 关闭文件监听
mdwalker --mermaid code   # Mermaid 只显示源码
```

## 快捷键

| 按键 | 功能 |
|------|------|
| `j`/`k`/`↑`/`↓` | 当前面板内移动 |
| `h`/`←` | 焦点左移（预览 → 大纲 → 文件列表） |
| `l`/`→` | 焦点右移（文件列表 → 大纲 → 预览） |
| `Tab` | 文件列表 ↔ 预览快速切换；搜索中切换搜索模式 |
| `Enter` | 打开选中文件；搜索中打开匹配文件 |
| `Space` | 折叠/展开当前标题内容 |
| `o` | 打开/关闭大纲浮动面板 |
| `t` | 文件栏切换树模式 / 列表模式 |
| `/` | 统一搜索（文件栏搜文件名，预览搜内容） |
| `n`/`N` | 下一个/上一个匹配 |
| `y` | 复制当前代码块 |
| `i` | 打开当前图片 |
| `b` | 返回上一个文件 |
| `r` | 重新扫描目录 |
| `g`/`G` | 跳转到顶部/底部 |
| `q` | 退出 |
