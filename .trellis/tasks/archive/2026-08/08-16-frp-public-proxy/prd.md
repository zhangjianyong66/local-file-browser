# 配置 FRP 与公网反向代理

## Goal

将仅本机监听的文件浏览器安全发布为 `https://files.zhangjianyong.top`。公网请求经过 ECS2 的 Nginx 和 FRP，再抵达本机 `127.0.0.1:8080`；应用继续使用已有的单密码认证，不引入账户体系。

## Confirmed Facts

- 文件浏览器默认监听 `127.0.0.1:8080`，可通过 `config.json` 设置密码和 `cookie_secure`。
- 本机已安装 FRP `0.52.3`，现有 `frpc.service` 已启用，配置采用 HTTP 代理和 `custom_domains`。
- ECS2 为 Debian 13，运行 FRP 服务端 `0.52.3`；其 HTTP 虚拟主机端口为 `8080`，控制端口为 `7000`。
- ECS2 的实际 Nginx 进程读取 `/usr/local/nginx/conf/nginx.conf`，站点文件位于 `/usr/local/nginx/conf/conf.d/`，而非 Debian 默认 systemd 服务配置。
- Nginx 已监听公网 `80/443`，已有 HTTP 跳转 HTTPS、反代 FRP HTTP 虚拟主机的成熟模板。
- Certbot 已安装并启用自动续期；`files.zhangjianyong.top` 已解析至 ECS2，DNS 由阿里云托管。
- 现有 FRP 使用共享认证凭据；不得把其值写入仓库、文档或命令输出。

## Requirements

- 在本机现有 FRP 客户端配置中新增 `local-file-browser` HTTP 代理，使用 `custom_domains = files.zhangjianyong.top`，后端保持 `127.0.0.1:8080`。
- 将文件浏览器生产配置的 `cookie_secure` 设为 `true`，密码只保存在权限为 `0600` 的本机配置文件中。
- 在 ECS2 新增独立 Nginx 站点：HTTP 仅承载 ACME 校验并跳转 HTTPS；HTTPS 反向代理 `127.0.0.1:8080`，保留 `Host` 与标准转发头。
- 为 `files.zhangjianyong.top` 申请并配置独立 Let's Encrypt 证书，沿用现有 Certbot 自动续期。
- 提供逐层验证和回滚步骤；不暴露应用、FRP 凭据或文件内容。

## Acceptance Criteria

- [ ] `ss -ltnp` 在本机仅显示文件浏览器监听 `127.0.0.1:8080`。
- [ ] 本机 `frpc.service` 健康运行，且 ECS2 使用 `Host: files.zhangjianyong.top` 访问 FRP 虚拟主机可得到文件浏览器响应。
- [ ] `http://files.zhangjianyong.top` 重定向到 HTTPS，`https://files.zhangjianyong.top` 的证书名称匹配且链路有效。
- [ ] 浏览、下载、上传和媒体预览经 HTTPS 正常工作；登录 Cookie 带 `Secure` 属性。
- [ ] 现有 ECS2 站点和已有 FRP 代理保持可用。
- [ ] 文档包含验证、日志位置和安全回滚方法，且不包含真实密码、令牌或私钥。

## Out of Scope

- 修改文件浏览器的业务功能或新增用户体系。
- 更换现有 FRP 全局共享认证机制、公开 FRP 仪表盘，或改动其他已有代理。
- 自动执行 DNS 修改、管理员权限操作或影响其他服务的批量重启。

## Risks

- Nginx 不是由 Debian 的 `nginx.service` 管理，错误地对默认服务执行重载不会生效；实施必须使用实际二进制与配置路径。
- FRP HTTP 虚拟主机端口当前对外监听；Nginx 是该域名的 HTTPS 安全边界，实施时不得把文件浏览器直接配置为 TCP 公网端口。
