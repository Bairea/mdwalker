Output demo.gif
Set Theme "Catppuccin Mocha"
Set FontFamily "Noto Sans Mono CJK SC"
Set Width 1200
Set Height 800
Set FontSize 16
Set Padding 60
Set Margin 60

# 启动 mdwalker
Type "mdwalker demo/docs/"
Enter
Sleep 2s

# 上下移动浏览文件，预览跟着变化
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# 打开树形模式
Type "t"
Sleep 500ms

# 在树形模式下移动
Down
Sleep 300ms
Down
Sleep 300ms

# 打开文件
Enter
Sleep 1s

# 打开大纲面板
Type "o"
Sleep 500ms

# 在大纲中导航并跳转
Down
Enter
Sleep 800ms

# 关闭大纲
Type "o"
Sleep 300ms

# 按 / 搜索内容
Type "/"
Sleep 300ms
Type "用户"
Sleep 500ms

# Tab 切换搜索模式（内容 → 全文件内容）
Tab
Sleep 500ms

# 再按 Tab 切换搜索模式（全文件内容 → 文件名）
Tab
Sleep 500ms

# 搜索词保留，跨所有文件匹配
Type "n"
Sleep 300ms

# Enter 打开匹配文件并跳转到行
Enter
Sleep 800ms

# 折叠标题
Space
Sleep 500ms

# 退出
Type "q"
Sleep 500ms
