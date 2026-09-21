# Journal - zhangjianyong (Part 1)

> AI development session journal
> Started: 2026-08-16

---



## Session 1: 实现 Go 本地文件浏览器

**Date**: 2026-08-16
**Task**: 实现 Go 本地文件浏览器

### Summary

完成受密码保护的本地文件浏览器，覆盖安全浏览、上传、下载和预览。

### Main Changes

- 新增 Go 服务、嵌入式 Web 界面和自动化测试。
- 使用 os.Root 防止符号链接和路径替换导致的根目录越界。

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test -race ./...、go vet ./...、go build ./... 通过；本机 HTTP 登录与目录接口验收通过。

### Status

[OK] **Completed**

### Next Steps

- 在 Nginx HTTPS 反向代理下以 -cookie-secure 启动生产实例。


## Session 2: 修复 Markdown 纯文本预览

**Date**: 2026-08-16
**Task**: 修复 Markdown 纯文本预览

### Summary

明确支持 .md 与 .txt 在线预览，并保留二进制内容安全判定。

### Main Changes

- 增加 Markdown 详情/预览回归测试。

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test -race ./...、go vet ./...、go build ./... 通过；真实 .md 文件 API 返回 preview=text。

### Status

[OK] **Completed**


## Session 3: 渲染 Markdown 文件

**Date**: 2026-08-16
**Task**: 渲染 Markdown 文件

### Summary

为 Markdown 预览增加服务端安全渲染和原文查看。

### Main Changes

- 使用 goldmark 将 .md 渲染为 HTML，前端显示渲染结果与可折叠原文。

### Git Commits

(No commits - planning session)

### Testing

- [OK] go test -race ./...、go vet ./...、go build ./... 通过；真实 Markdown 预览接口返回 html。

### Status

[OK] **Completed**


## Session 4: 完成文件浏览器 UI 交互优化

**Date**: 2026-08-16
**Task**: 完成文件浏览器 UI 交互优化

### Summary

完成浅色工具化 UI、桌面双栏与移动端详情、上传反馈、可访问性和响应式样式优化；未修改 Go API。

### Main Changes

- 重构 web/index.html、web/login.html、web/app.js、web/style.css
- 新增桌面列表/详情双栏和移动端返回路径
- 补充加载、错误、上传状态和 Markdown/媒体预览层级

### Git Commits

(No commits - planning session)

### Testing

- [OK] node --check web/app.js
- [OK] go test -race ./...
- [OK] go vet ./...
- [OK] go build ./...

### Status

[OK] **Completed**

### Next Steps

- 当前服务地址：http://127.0.0.1:8080


## Session 5: 修复详情预览横向溢出

**Date**: 2026-08-16
**Task**: 修复详情预览横向溢出

### Summary

新增 overflow-fix.css，约束详情元信息、Markdown、代码块、表格与媒体预览宽度，避免页面级横向溢出。

### Main Changes

- index.html 与 login.html 引入溢出修复样式
- 长文本换行，表格/代码块局部滚动，媒体限制最大宽度

### Git Commits

(No commits - planning session)

### Testing

- [OK] node --check web/app.js
- [OK] go test -race ./...
- [OK] go vet ./...
- [OK] go build ./...

### Status

[OK] **Completed**

### Next Steps

- 服务地址：http://127.0.0.1:8080


## Session 6: 增加 Markdown 独立预览窗口

**Date**: 2026-08-16
**Task**: 增加 Markdown 独立预览窗口

### Summary

启用 Goldmark GFM 表格等安全扩展，Markdown 文件点击后在新窗口独立渲染，新增元信息、下载和关闭操作。

### Main Changes

- 新增 web/markdown.html 独立阅读页
- 主列表 Markdown 点击打开新窗口，其他文件行为不变
- 增加表格渲染与静态阅读页回归测试

### Git Commits

(No commits - planning session)

### Testing

- [OK] node --check web/app.js
- [OK] go test -race ./...
- [OK] go vet ./...
- [OK] go build ./...

### Status

[OK] **Completed**

### Next Steps

- 服务地址：http://127.0.0.1:8080


## Session 7: 增加配置文件与系统服务脚本

**Date**: 2026-08-16
**Task**: 增加配置文件与系统服务脚本

### Summary

实现 JSON 配置加载与参数优先级，新增示例配置、启停脚本、用户级 systemd 模板和部署文档；通过 race 测试、vet、build 与脚本语法检查。

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 8: 公开发布本地文件浏览器

**Date**: 2026-08-20
**Task**: 公开发布本地文件浏览器
**Branch**: `main`

### Summary

初始化 Git 仓库，排除真实配置与本机运行产物，创建并推送公开 GitHub 仓库。

### Git Commits

| Hash | Message |
|------|---------|
| `649b9b6` | (see git log) |

### Status

[OK] **Completed**


## Session 9: 完成 Trellis 项目规范初始化
<!-- trellis-session: v=2 fp=3d3be94230d643bf -->

**Date**: 2026-09-21
**Task**: 完成 Trellis 项目规范初始化
**Branch**: `main`

### Summary

基于 Go 后端与原生 JavaScript 前端补齐后端、前端开发规范和真实代码示例，完成校验并归档 00-bootstrap-guidelines 任务。

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 10: 清理 Git 忽略项
<!-- trellis-session: v=2 fp=4cf463a14a23d38e -->

**Date**: 2026-09-21
**Task**: 清理 Git 忽略项
**Branch**: `main`

### Summary

移除 .gitignore 中对 Trellis、Codex、Agent 配置和 AGENTS.md 的宽泛忽略，仅保留本地配置、日志、PID 与构建二进制；完成测试、vet 和构建验证。

### Git Commits

| Hash | Message |
|------|---------|
| `4600d08` | chore: 清理非必要的 Git 忽略项 |

### Status

[OK] **Completed**
