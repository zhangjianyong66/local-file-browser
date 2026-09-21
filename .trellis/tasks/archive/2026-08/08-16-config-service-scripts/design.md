# 技术设计

## 配置加载

- 扩展现有 `config` 结构与 `parseConfig`，使用标准库 `encoding/json` 读取配置文件。
- 增加 `-config` 参数；默认路径为可执行文件所在目录的 `config.json`，文件不存在时沿用现有环境变量/参数行为。
- 配置字段覆盖 `root`、`listen`、`password`、`cookie_secure`；字段值先作为候选默认，再由环境变量和 flag 覆盖。
- 启动时校验必需密码和根目录；配置文件解析错误给出文件路径和字段错误。

## 脚本与服务

- `scripts/start.sh`：解析项目目录，检查二进制和配置文件，使用 `nohup`/PID 文件启动并输出日志路径。
- `scripts/stop.sh`：读取 PID 文件，发送 TERM，等待退出并清理 PID，避免误杀其他进程。
- `deploy/local-file-browser.service`：用户级 systemd 模板，使用 `%h`、项目绝对路径占位说明、`Restart=on-failure`、`WantedBy=default.target` 和 journald 日志。
- 文档说明复制服务文件到 `~/.config/systemd/user/`、`daemon-reload`、`enable --now`、`status`、`journalctl` 及 linger 注意事项。

## 兼容性与安全

- 保留现有 `FILE_BROWSER_PASSWORD`、`-password` 和其他 flags 的兼容性。
- 服务默认继续监听 `127.0.0.1`；不在脚本或 unit 中暴露公网端口。
- 不把真实密码写入示例配置；示例使用空值/占位说明并提示设置权限。
