# 公开发布到 GitHub

## Goal

将本地文件浏览器以公开 GitHub 仓库 `zhangjianyong66/local-file-browser` 发布，同时不泄露当前部署环境的访问凭据或运行产物。

## Confirmed Facts

- 当前目录尚未初始化为 Git 仓库；GitHub CLI 已登录账号 `zhangjianyong66`。
- `config.json` 保存本机部署配置，包含真实访问密码；README 已规定真实密码不得提交。
- `config.example.json` 已提供不含真实密码的配置模板。
- `local-file-browser.log` 是运行日志，不应进入版本控制。
- 程序二进制 `local-file-browser` 是本机构建产物，不应进入版本控制。

## Requirements

- R1: 将真实 `config.json`、运行日志、PID 文件和本机构建二进制排除在版本控制之外。
- R2: 保留 `config.example.json` 作为可公开的配置模板。
- R3: 初始化 Git 仓库，创建并关联公开 GitHub 仓库 `zhangjianyong66/local-file-browser`。
- R4: 在提交前扫描暂存内容中的高风险文件名和常见凭据模式；扫描通过后创建中文提交信息并推送 `main` 分支。

## Acceptance Criteria

- [ ] `git ls-files` 不包含 `config.json`、`local-file-browser.log`、`local-file-browser` 或 PID 文件。
- [ ] 暂存内容不含真实访问密码或其他高风险凭据。
- [ ] GitHub 上存在公开仓库 `zhangjianyong66/local-file-browser`，`origin` 指向该仓库，`main` 已推送。
- [ ] `go test ./...` 通过。

## Out of Scope

- 不修改应用功能、部署拓扑或远程服务器配置。
- 不轮换线上访问密码；该操作由部署所有者在发布后单独完成。

## Risks

- 真实密码已在本地配置文件中出现过，公开发布前必须从提交范围中移除；建议发布后由用户尽快轮换该密码。
