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

# 上下移动浏览文件
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

# 按 / 搜内容（当前文件搜"用户"，overview.md 有 2 处匹配）
Type "/"
Sleep 300ms
Type "用户"
Sleep 1s

# Tab 切换到全文件内容搜索（匹配全部文件 4 处，标题变为 Search all files）
Tab
Sleep 1s

# Enter 打开第一个匹配项对应的文件并跳转
Enter
Sleep 800ms

# 折叠标题
Space
Sleep 500ms

# 退出
Type "q"
Sleep 500ms
