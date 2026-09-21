# 技术设计

## 架构

```text
浏览器
  -> https://files.zhangjianyong.top:443
  -> ECS2 Nginx (/usr/local/nginx/conf/conf.d/files.zhangjianyong.top.conf)
  -> 127.0.0.1:8080 (frps HTTP 虚拟主机，按 Host 路由)
  -> 本机 frpc.service (local-file-browser 代理)
  -> 127.0.0.1:8080 (Go 文件浏览器)
```

FRP 使用现有 HTTP 代理模型。Nginx 必须将 `Host` 设置为原始域名，FRPS 才能根据 `custom_domains` 路由至本机客户端。TLS 在 Nginx 终止，FRP 与本机应用之间保留 HTTP，但流量经已认证的 FRP 控制连接承载。

## 配置边界

- 本机：`config.json` 设定浏览根目录、应用密码、`cookie_secure: true`；`~/.frpc/frpc.ini` 新增代理段；重启 `frpc.service` 和文件浏览器用户服务。
- ECS2：新增一个 Nginx `conf.d` 文件与 Let's Encrypt 证书，不修改现有 FRPS 配置，不占用新的 FRP TCP 端口。
- 仓库：增加脱敏配置模板和部署文档，不保存真实 FRP token、应用密码或证书私钥。

## Nginx 契约

- `80`：`/.well-known/acme-challenge/` 指向 `/var/www/letsencrypt`，其他请求 `301` 至对应 HTTPS 地址。
- `443`：证书使用 `/etc/letsencrypt/live/files.zhangjianyong.top/`；代理目标为 `http://127.0.0.1:8080`。
- 请求头：传递 `Host`、`X-Real-IP`、`X-Forwarded-For`、`X-Forwarded-Proto`；上传上限至少为应用的 5 MiB，建议配置为 `10m` 以保留协议余量。

## 安全与回滚

- 应用密码与 FRP 认证信息仅保存在主机权限受限的现有配置中，不进入仓库。
- 证书签发前先以 HTTP 站点通过 `nginx -t` 验证；签发后再启用 HTTPS 站点并 reload 实际运行的 Nginx master。
- 回滚可删除新增 Nginx 站点文件、reload Nginx，并从本机 FRPC 配置移除新增段后重启 FRPC。上述操作均不修改已有服务段。
