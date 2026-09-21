# 实现计划

1. 读取后端规范并补充配置加载单元测试，覆盖 JSON、优先级、缺失配置和密码校验。
2. 实现 `-config` 与 JSON 配置解析，保持默认启动与现有 flags/env 行为。
3. 添加 `config.example.json`、启动/停止脚本、用户级 systemd unit 模板和部署文档。
4. 脚本做路径、权限、PID 和进程状态检查；unit 使用非 root 当前用户和自动重启。
5. 运行 `gofmt`、`go test -race ./...`、`go vet ./...`、`go build ./...`，并 shellcheck/手工验证脚本语法。
