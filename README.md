# 本地文件浏览器部署

程序默认监听 `127.0.0.1:8080`，默认浏览目录为 `~/project`。服务只绑定本机，FRP、Nginx 和 HTTPS 由部署环境单独配置。

## 公网部署（FRP + Nginx）

本项目可通过 `https://files.zhangjianyong.top` 对外访问。链路为：浏览器 -> ECS2 Nginx -> ECS2 FRP HTTP 虚拟主机 -> 本机 FRP 客户端 -> `127.0.0.1:8080`。

本机将 [deploy/frp/local-file-browser.ini](deploy/frp/local-file-browser.ini) 中的代理段追加到已有的 `~/.frpc/frpc.ini`，保留原有的 `server_addr`、`server_port` 和认证配置，不要将认证信息写入仓库。重启后验证：

```bash
systemctl --user restart frpc.service
systemctl --user status frpc.service
```

ECS2 使用 [deploy/nginx/files.zhangjianyong.top.conf](deploy/nginx/files.zhangjianyong.top.conf)。该服务器的实际 Nginx 配置根目录为 `/usr/local/nginx/conf/`，重载前必须执行：

```bash
/usr/sbin/nginx -t -c /usr/local/nginx/conf/nginx.conf
kill -HUP "$(cat /usr/local/nginx/logs/nginx.pid)"
```

证书使用 Certbot webroot 模式，ACME 目录为 `/var/www/letsencrypt`：

```bash
certbot certonly --webroot -w /var/www/letsencrypt -d files.zhangjianyong.top
```

生产环境必须将 `config.json` 中的 `cookie_secure` 设置为 `true`，并限制配置权限：

```bash
chmod 600 config.json
```

验证入口：

```bash
curl -I http://files.zhangjianyong.top
curl -I https://files.zhangjianyong.top
```

回滚时，删除 ECS2 上的 `files.zhangjianyong.top.conf` 并通过上述测试后重载 Nginx；再从本机 `~/.frpc/frpc.ini` 删除 `local-file-browser` 段并重启 `frpc.service`。这不会影响其他 FRP 代理。

## 配置文件

复制示例并修改路径和密码：

```bash
cp config.example.json config.json
chmod 600 config.json
```

`config.json` 默认位于可执行文件同目录，也可以通过 `-config /path/to/config.json` 指定。字段为 `root`、`listen`、`password` 和 `cookie_secure`。配置文件中的 `root` 必须写绝对路径，例如 `/home/zhangjianyong/project`；`~/project` 仅是未设置配置时的程序默认目录，不会在 JSON 中展开。不要把真实密码提交到代码仓库。

配置优先级从高到低为：命令行参数、环境变量、JSON 配置文件、内置默认值。环境变量包括 `FILE_BROWSER_ROOT`、`FILE_BROWSER_LISTEN`、`FILE_BROWSER_PASSWORD` 和 `FILE_BROWSER_COOKIE_SECURE`。

## 手动启停

先构建：

```bash
go build -o local-file-browser .
```

启动和停止：

```bash
./scripts/start.sh
./scripts/stop.sh
```

脚本会在项目目录创建 PID 文件和日志文件。可用 `FILE_BROWSER_BINARY`、`FILE_BROWSER_CONFIG`、`FILE_BROWSER_PID_FILE`、`FILE_BROWSER_LOG_FILE` 覆盖默认路径。

## 注册用户级 systemd 服务

不需要 `sudo`。确认 unit 中的项目路径与实际位置一致后执行：

```bash
mkdir -p ~/.config/systemd/user
cp deploy/local-file-browser.service ~/.config/systemd/user/local-file-browser.service
systemctl --user daemon-reload
systemctl --user enable --now local-file-browser.service
systemctl --user status local-file-browser.service
```

查看日志：

```bash
journalctl --user -u local-file-browser.service -f
```

用户未登录时是否继续运行取决于 systemd linger 设置。需要开机即运行时，可由管理员或用户按系统策略执行 `loginctl enable-linger "$USER"`；本项目不会自动执行该权限操作。停止并取消自启：

```bash
systemctl --user disable --now local-file-browser.service
```
