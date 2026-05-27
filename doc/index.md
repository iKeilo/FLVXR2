# FLVXT2 文档

**FLVXT2** 是一个基于 [go-gost/gost](https://github.com/go-gost/gost) 和 [go-gost/x](https://github.com/go-gost/x) 的流量转发管理系统。

> 仓库地址: [iKeilo/flvxt2](https://github.com/iKeilo/flvxt2)

## 核心特性

- 支持 `TCP` / `UDP` 转发
- 支持 `端口转发` 和 `隧道转发`
- 支持 `gost` 与 `nftables` 双运行模式
- 支持用户、节点、分组、标签、流量历史和配额管理
- 支持面板、节点一键安装与 Release 指定版本部署
- 支持 Docker Compose、SQLite 和 PostgreSQL

## 快速入口

- [部署流程](./install.md)
- [使用说明](./usage.md)
- [PostgreSQL 指南](./postgresql.md)
- [AI Skill 接入](./ai-skill.md)
- [常见问题](./faq.md)

## 版本发布

- Releases: [github.com/iKeilo/flvxt2/releases](https://github.com/iKeilo/flvxt2/releases)
- Docker Packages:
  - `ghcr.io/ikeilo/flvx-svc-backend`
  - `ghcr.io/ikeilo/flvx-svc-frontend`

## 免责声明

本项目仅供学习与合法合规用途使用。部署者需自行确认使用方式符合所在地法律法规。
