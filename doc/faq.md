# 常见问题

### Q1: 安装脚本提示 `Docker command not found`
**A**: 请先确认系统已经安装 `Docker` 和 `Docker Compose`。

```bash
curl -fsSL https://get.docker.com | bash
```

### Q2: 面板打不开
**A**:
1. 确认前端端口已经放行，默认是 `6366`
2. 确认容器已经启动：`docker ps`
3. 查看日志：`docker logs flvx-svc-backend` 或 `docker logs flvx-svc-frontend`

### Q3: 节点显示离线
**A**:
1. 确认节点服务器能访问面板地址
2. 确认安装时输入的面板地址和 Secret 正确
3. 查看服务状态：`systemctl status flvx_agent`
4. 查看日志：`journalctl -u flvx_agent -f`

### Q4: 只有 TCP 通，UDP 不通
**A**: 检查服务器防火墙和安全组是否同时放行对应端口的 `TCP` 和 `UDP` 协议。

### Q5: IPv6 没生效
**A**: 面板安装脚本会自动尝试配置 Docker IPv6。如果失败，请手动检查 `/etc/docker/daemon.json` 是否启用了 `ipv6` 并配置了正确的 `fixed-cidr-v6`。

### Q6: 如何切换到 PostgreSQL
**A**: 在 `.env` 里设置 `DB_TYPE=postgres`，并同步配置 `DATABASE_URL` 和 `POSTGRES_*`，然后执行 `docker compose up -d` 重启服务。

### Q7: SQLite 迁移到 PostgreSQL 后数据丢失
**A**:
1. 确认迁移前已经备份了 `gost.db`
2. 确认 `pgloader` 执行成功
3. 确认 `.env` 里的 `DATABASE_URL` 和 `POSTGRES_PASSWORD` 一致

### Q8: PostgreSQL 容器启动失败
**A**:
1. 确认 `POSTGRES_PASSWORD` 不为空
2. 查看容器日志：`docker logs flvx-svc-postgres`
3. 如果首次启动后修改过密码，可能需要清理数据卷后重新初始化
