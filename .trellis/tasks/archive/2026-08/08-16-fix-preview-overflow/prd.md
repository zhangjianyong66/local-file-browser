# 修复详情预览横向溢出

## Goal

修复文件详情面板和 Markdown 预览在长 MIME、长文件名、表格或代码内容下产生的横向溢出，确保内容留在面板内且页面不出现非预期横向滚动。

## Confirmed Facts

- 截图显示桌面双栏详情面板右侧内容被裁切/撑出边界，问题集中在详情元信息和 Markdown 预览区域。
- 前端使用原生 HTML/CSS/JavaScript；本修复不改变 Go API、Markdown 服务端渲染或文件功能。

## Requirements

- 详情面板、元信息网格、预览容器必须允许子元素收缩到面板宽度。
- 长 MIME、路径、文件名和普通文本必须换行或在局部容器内滚动，不得撑破页面。
- Markdown 表格、代码块和媒体预览最大宽度不得超过详情面板；代码块/表格可在自身区域横向滚动。
- 375px 和桌面宽度下不出现页面级横向滚动。

## Acceptance Criteria

- [x] 详情元信息中的长文本不会溢出右侧边界，字段仍可读。
- [x] Markdown 标题、段落、表格、代码块和媒体内容均被限制在详情面板内。
- [x] 页面在 375px、768px 和宽桌面视口下没有非预期横向滚动。
- [x] 现有目录、下载、上传和各种预览行为保持不变。

## Out of Scope

- 不改变 Markdown 解析规则、后端 API、布局结构或新增搜索/筛选能力。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
