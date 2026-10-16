# HTML 预览独立检查记录

## 检查结论

实现符合当前 PRD、设计及前后端契约，未发现需要阻止交付的功能问题。没有提交、部署或重启正式服务。

## 检查中修复

- 恢复被意外删除的 `/index.html` 登录跳转回归覆盖，保留根目录与 index 两种入口。
- 补充资源 GET/HEAD、忽略查询参数、无效令牌、非 HTML 返回 415、跨站管理 API 返回 403，以及合法会话从外部链接打开浏览器 UI 返回 200 的测试。

## 自动验证

- `go test -race ./...`：通过。
- `go vet ./...`：通过。
- `go build -o /tmp/local-file-browser-html-check .`：通过，没有覆盖仓库二进制。
- `node --test scripts/*.test.cjs`：10 项通过。
- `git diff --check`：通过。

## 真实浏览器验证

使用临时根目录和 127.0.0.1:18083 的新构建服务：

- 点击中文、空格、`#`、`%`、`?` 目录中的嵌套 `PAGE.HTML`，实际出现独立预览标签。
- 页面通过 `../` 引用 CSS、SVG 图片和经典 JS；脚本文字为 `script loaded`，背景色为 `rgb(12, 34, 56)`，图片自然宽度为 10。
- iframe 的 `sandbox` 为 `allow-scripts`，文档发送消息的 origin 为 `null`。
- 访问 parent.document 和 localStorage 均返回 SecurityError；访问管理 API 的 fetch 返回 TypeError，与 CSP 限制一致。
- 包装页有对应文件的编码下载地址。
- 同一预览资源在注销前返回 200、注销接口返回 200、注销后返回 401。
- 检查后已通过 Ctrl-C 停止本次创建的临时服务，未影响正式服务。

## 验证边界

没有独立复测 HTML 自行导航的浏览器场景；代码检查确认包装 iframe 的 sandbox 属性不会被页面代码撤销，资源响应也带独立 CSP sandbox。Markdown 点击、目录历史与请求顺序等兼容性由已有前端与 Go 回归测试覆盖。
