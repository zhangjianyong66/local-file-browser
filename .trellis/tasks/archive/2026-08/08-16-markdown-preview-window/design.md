# 技术设计

## 架构边界

- 后端仅调整 Goldmark 初始化参数，启用安全的常用 Markdown 扩展；不改变预览 API 返回结构。
- 新增嵌入式 `web/markdown.html` 独立阅读页及对应脚本，复用 `/api/info`、`/api/preview` 和 `/api/download`。
- 主页面只在 Markdown 文件点击处理器中打开新窗口；其他文件仍使用原详情面板。

## 数据流

1. 目录列表点击 `.md` 文件，使用 `window.open('/markdown.html?path=...', '_blank', 'noopener')`。
2. 阅读页从 URL 读取编码后的路径，调用 info 和 preview 接口，显示元信息与服务端生成的安全 HTML。
3. 页面加载失败、超出 5 MiB 或未登录时显示明确状态；下载链接继续指向原下载接口。

## 安全与兼容

- Goldmark 不启用原始 HTML；保留现有安全 URL 过滤行为。
- 新窗口页面使用同源 API 和 `target` 下载链接，不引入外部字体或运行时依赖。
- GFM 表格启用后仅扩大渲染能力，不改变原文下载结果。
