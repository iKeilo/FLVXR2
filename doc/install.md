# 部署流程

本文档介绍如何部署 FLVXT2 面板端和节点端。

## 1. 面板端部署

### 环境要求

- Linux
- Docker
- Docker Compose

### 一键安装

```bash
curl -L https://raw.githubusercontent.com/iKeilo/flvxt2/main/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```

安装过程中会询问：

- 前端端口，默认 `6366`
- 后端端口，默认 `6365`
- 数据库类型，默认 `SQLite`

安装完成后访问：

```bash
http://<服务器IP>:6366
```

默认管理员账号：

- 用户名：`admin_user`
- 密码：`admin_user`

### 指定版本安装

从 Releases 页面复制对应版本的命令即可，例如：

```bash
curl -L https://github.com/iKeilo/flvxt2/releases/download/3.0.0/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```

### 更新与迁移

重新运行 `./panel_install.sh` 可进入管理菜单：

1. 安装面板
2. 更新面板
3. 卸载面板
4. 迁移到 PostgreSQL
5. 退出

### PostgreSQL

如果你选择 PostgreSQL，安装脚本会自动生成 `.env` 和数据库容器配置。

## 2. 节点端部署

### 获取节点接入信息

1. 登录面板
2. 进入 `节点管理`
3. 新增节点
4. 复制节点的 `Secret` 和面板地址

### 一键安装

```bash
curl -L https://raw.githubusercontent.com/iKeilo/flvxt2/main/install.sh -o install.sh && chmod +x install.sh && ./install.sh
```

安装时会提示输入：

- 面板地址，例如 `http://1.2.3.4:6365`
- 节点 Secret

也可以直接带参数安装：

```bash
./install.sh -a "http://1.2.3.4:6365" -s "your_node_secret"
```

## 3. 可选反向代理

如果你希望通过域名访问面板，可以在前面加一层 Caddy 或 Nginx 反代。反代后建议把前端暴露端口限制到本机地址，例如：

```bash
FRONTEND_PORT=127.0.0.1:6366
```
