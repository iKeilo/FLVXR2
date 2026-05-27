# FLVXT2

> 基于 `go-gost` 的流量转发管理面板，包含 Go 后端、Vite/React 前端和 Go 节点代理。

## 项目特点

- 支持 `TCP` 和 `UDP` 转发
- 支持 `端口转发` 和 `隧道转发`
- 支持按用户、节点、分组进行权限和流量管理
- 支持节点分组、节点标签、流量统计和重置
- 支持 `gost` 和 `nftables` 两种运行模式
- 支持面板和节点一键安装、升级、Release 下载

## 快速开始

- [部署流程](./doc/install.md)
- [使用说明](./doc/usage.md)
- [PostgreSQL 指南](./doc/postgresql.md)
- [常见问题](./doc/faq.md)
- [AI Skill 接入](./doc/ai-skill.md)

## Release 与 Packages

- GitHub Releases: [iKeilo/flvxt2/releases](https://github.com/iKeilo/flvxt2/releases)
- 源码仓库: [iKeilo/flvxt2](https://github.com/iKeilo/flvxt2)
- Docker Packages:
  - `ghcr.io/ikeilo/flvx-svc-backend`
  - `ghcr.io/ikeilo/flvx-svc-frontend`

## 部署方式

### 1. 面板端

```bash
curl -L https://raw.githubusercontent.com/iKeilo/flvxt2/main/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```

安装脚本会自动下载对应版本的 `docker-compose` 文件，并根据系统环境配置 IPv6、数据库和基础参数。

### 2. 节点端

```bash
curl -L https://raw.githubusercontent.com/iKeilo/flvxt2/main/install.sh -o install.sh && chmod +x install.sh && ./install.sh
```

节点安装脚本会根据 Release 下载对应架构的 `flux_agent` 二进制，并自动写入节点配置。

### 3. 指定版本安装

从 Releases 页面复制对应版本的命令即可，示例：

```bash
curl -L https://github.com/iKeilo/flvxt2/releases/download/3.0.0/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```

```bash
curl -L https://github.com/iKeilo/flvxt2/releases/download/3.0.0/install.sh -o install.sh && chmod +x install.sh && ./install.sh
```

## 默认账号

- 用户名：`admin_user`
- 密码：`admin_user`

首次登录后请立即修改默认密码。

## 项目说明

FLVXT2 是在原有流量转发面板基础上继续演进的开源项目，当前仓库已把面板、节点代理和前端文档统一到了 `iKeilo/flvxt2`。

## 免责声明

本项目仅供学习、研究和合法合规用途使用。请确保你的部署和使用行为符合所在地法律法规，并自行承担使用风险。
