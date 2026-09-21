# 增加 Markdown 独立预览窗口

## Goal

让 Markdown 文件在独立窗口中以完整、可阅读的渲染页面打开，避免详情双栏空间过窄导致表格、代码和正文难以阅读；主窗口继续用于目录浏览。

## Confirmed Facts

- 当前 Markdown 在详情栏内展示，截图中表格语法未完整转换且阅读区域拥挤。
- 服务端已使用 Goldmark 返回 Markdown 原文和 HTML；前端资源为无框架嵌入式 HTML/CSS/JavaScript。
- 用户明确要求点击 Markdown 文件后打开新窗口查看内容。

## Requirements

- 点击 `.md` 文件时在新窗口/新标签打开独立 Markdown 阅读页，主窗口不跳转。
- 独立页面展示文件名、路径、大小、修改时间、字符数/行数、下载入口和完整渲染内容。
- 服务端启用常用 GFM 表格等 Markdown 扩展，使表格、标题、列表、代码块和链接正确渲染。
- 独立页面在桌面和移动宽度下保持可读内容宽度；长表格或代码块只在自身区域滚动。
- 保留安全渲染约束：不执行原始 HTML、脚本或不安全链接协议。
- 非 Markdown 文件的现有详情预览行为保持不变。

## Acceptance Criteria

- [x] 从目录点击 `.md` 后打开新窗口/标签，原目录页面保持在原窗口。
- [x] 新窗口显示 Markdown 标题、段落、列表、表格、代码块和链接的渲染结果，而非原始表格标记。
- [x] 新窗口提供文件元信息、下载按钮、返回/关闭提示，并在 375px 和桌面宽度下无页面级横向溢出。
- [x] 不安全 HTML、脚本和 `javascript:` 链接不会执行或注入页面。
- [x] `go test -race ./...`、`go vet ./...`、`go build ./...` 和前端脚本语法检查通过。

## Out of Scope

- Markdown 在线编辑、保存、目录搜索和新的用户权限体系。
- 非 Markdown 文件改为新窗口预览。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
