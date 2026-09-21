# 修复 Markdown 纯文本预览

## Goal

让本地文件浏览器明确支持 `.txt` 和 `.md` 文件在线查看。

## Requirements

- `.md` 与 `.txt` 文件在文件详情中识别为纯文本并显示预览入口。
- 预览内容沿用现有 5 MiB 上限、字符数、行数和 MIME 信息。
- 不改变图片、视频、PDF、二进制文件和上传行为。

## Acceptance Criteria

- [ ] `.md` 文件详情返回 `preview: "text"`，并能通过预览接口读取内容。
- [ ] `.txt` 现有预览行为继续通过测试。
- [ ] 超过 5 MiB 的 `.md` 文件只返回截断提示，不读取完整内容。
- [ ] `go test -race ./...`、`go vet ./...` 和 `go build ./...` 通过。

## Out of Scope

- Markdown 渲染为 HTML、编辑、语法高亮或文件内容修改。
