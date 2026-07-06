# 简化 mdwalker 为纯文本 Markdown 查看器

## 背景

mdwalker 最初设计时计划支持图片和 Mermaid 图表的实时渲染，但由于 Bubble Tea 框架在终端图片渲染方面的限制，这些功能无法可靠工作。为保持项目定位清晰，决定移除这些功能，将 mdwalker 重新定位为纯文本 Markdown 查看器。

## 目标

1. 移除所有图片渲染代码（Kitty、iTerm2、WezTerm 协议支持）
2. 移除所有 Mermaid 渲染代码（mmdc 调用、缓存机制）
3. Mermaid 代码块作为普通代码块展示
4. 图片引用显示为简单占位符 `[Image: path]`
5. 移除 'i' 键打开图片功能

## 架构变更

### 删除的文件

- `internal/markdown/mermaid.go` - Mermaid 渲染相关所有代码

### 修改的文件

#### `internal/markdown/image.go`

保留：
- `ImageRef` 结构体
- `ExtractImages` 函数（用于生成占位符）
- `RenderImagePlaceholder` 函数

删除：
- `OpenImage`
- `HasTerminalImageSequence`
- `ClearAllImages`
- `ImageToHalfblock`
- `TerminalSupportsImages`
- `RenderImageInline`
- `imageProtocol`
- `isWezTerm`
- `renderITerm2`
- `renderKitty`
- `renderKittyFile`
- `parseImageTarget`
- `decodeImagePath`

#### `internal/preview/preview.go`

移除的字段：
- `renderMedia`
- `hasImage`
- `needsClear`

简化的函数：
- `LoadFile` 和 `LoadFileLight` 合并
- `renderFolded` - 移除媒体块处理，直接用 Glamour 渲染
- `View` - 移除图片序列处理

删除的函数：
- `extractMediaBlocks`
- `replaceMermaidWithPlaceholders`
- `replaceImagesWithPlaceholders`
- `renderMermaidBlock`
- `renderImage`
- `resolveAssetPath`
- `mediaWidth`
- `mediaHeight`
- `isImagePath`
- `mediaSafeView`

#### `internal/app/app.go`

删除：
- `openImage` 变量
- `openCurrentImage` 函数
- 'i' 键绑定处理
- `markdown.CleanMermaidCache()` 调用

#### `internal/config/config.go`

删除字段：
- `ImageProtocol`
- `MermaidMode`
- `MmdcPath`

## 数据流变更

变更前：
```
Markdown 文件 → 提取媒体块 → 替换为占位符 → 渲染 Mermaid → 渲染图片 → 替换回渲染结果 → 显示
```

变更后：
```
Markdown 文件 → Glamour 渲染 → 显示
```

## 行为变更

### Mermaid 代码块

变更前：尝试调用 mmdc 渲染为 PNG，再使用终端图片协议显示

变更后：作为普通代码块展示，由 Glamour 提供语法高亮

### 图片引用

变更前：`![alt](image.png)` 渲染为终端图片或显示 `[Image: path] press i to open`

变更后：显示 `[Image: path]` 占位符，无交互功能

### 键绑定

变更前：'i' 键打开当前行图片

变更后：'i' 键无绑定

## 测试变更

### 删除的测试

- `internal/markdown/image_test.go` 中所有渲染相关测试
- `internal/preview/preview_test.go` 中 Mermaid/图片渲染测试

### 保留的测试

- `internal/markdown/image_test.go` 中 `ExtractImages` 相关测试（如果有）

## 影响范围

- 用户将无法在终端内预览图片
- 用户将无法在终端内预览 Mermaid 图表
- 外部工具依赖（mmdc、chafa、viu）不再需要
- 配置文件中的 ImageProtocol 和 MermaidMode 选项将不再生效
