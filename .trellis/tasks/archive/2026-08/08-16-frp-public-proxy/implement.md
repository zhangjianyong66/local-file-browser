# 执行计划

1. 读取本机 `frpc.service` 的实际 unit 和配置加载路径，备份相关配置后新增 `local-file-browser` HTTP 代理段。
2. 创建或校验文件浏览器的生产 `config.json`：根目录正确、设置单密码、`cookie_secure: true`、权限 `0600`；构建并启用用户级文件浏览器服务。
3. 在 ECS2 先写入只含 HTTP 和 ACME location 的 `files.zhangjianyong.top` Nginx 站点，使用实际运行的 Nginx 二进制执行配置测试和安全 reload。
4. 使用 Certbot 的 webroot 方式为该域名签发证书；验证 renewal 配置。
5. 补全 HTTPS server block，代理至 FRP 的本地 HTTP 虚拟主机；再次测试并 reload。
6. 重启本机 FRPC 与文件浏览器服务，按本机、FRP、HTTP 跳转、HTTPS、登录 Cookie、上传及预览逐层验证。
7. 写入脱敏部署文档和模板，保留每步回滚命令；最后检查现有站点的 Nginx 配置与 FRPC 服务状态。

## 验证

- 本机：`curl -I http://127.0.0.1:8080`、`ss -ltnp`、`systemctl --user status frpc.service local-file-browser.service`。
- ECS2：实际 Nginx 二进制配置测试、带 Host 的 `curl http://127.0.0.1:8080`、`curl -I http://files.zhangjianyong.top`、`curl -I https://files.zhangjianyong.top`。
- 浏览器：登录、目录浏览、下载、5 MiB 内上传、Markdown/图片/视频/PDF 预览。

## 回滚点

- FRPC 新段仅在 Nginx 与证书成功后启用；若失败，恢复备份并重启 FRPC。
- Nginx 每次 reload 前均执行配置测试；失败时不 reload。已启用后移除新增站点文件并 reload 即可撤销公网入口。
