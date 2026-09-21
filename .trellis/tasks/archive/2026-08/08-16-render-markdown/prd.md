# 渲染 Markdown 文件

## Goal

让 `.md` 文件在浏览器中以 Markdown 格式渲染，便于阅读项目文档。

## Requirements

- 服务端将 `.md` 内容解析为 HTML，支持常见标题、段落、列表、链接、代码块和表格等 Markdown 语法。
- 默认不渲染原始 HTML，禁止脚本和不安全协议进入页面。
- 文件仍受现有 5 MiB 预览上限约束，渲染失败或超限时提供原文/下载提示。
- 页面保留下载入口，并提供查看 Markdown 原文的方式。
- `.txt`、图片、视频、PDF 和其他文件行为不变。

## Acceptance Criteria

- [ ] `.md` 详情显示渲染后的标题、列表和代码块，而不是 Markdown 标记文本。
- [ ] Markdown 中的 `<script>`、原始 HTML 和 `javascript:` 链接不会执行或注入页面。
- [ ] 渲染接口对 5 MiB 内文件返回 HTML 和原文；超限返回截断提示。
- [ ] 下载 `.md` 仍返回原始文件内容。
- [ ] `go test -race ./...`、`go vet ./...` 和 `go build ./...` 通过。

## Out of Scope

- 在线编辑、保存或修改 Markdown。
- 复杂扩展语法和客户端实时渲染。
