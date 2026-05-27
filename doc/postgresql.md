# PostgreSQL 指南

FLVXT2 默认使用 SQLite，也完整支持 PostgreSQL。本文档说明如何切换到 PostgreSQL、如何迁移，以及常见运维命令。

## 1. 适用场景

- SQLite：适合单机、小规模部署
- PostgreSQL：适合多节点、较高并发或需要更稳健备份的场景

## 2. 环境变量

后端与数据库相关配置通常放在 `.env` 中：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_TYPE` | `sqlite` 或 `postgres` | `sqlite` |
| `DATABASE_URL` | PostgreSQL 连接串 | 空 |
| `DB_PATH` | SQLite 数据文件路径 | `/app/data/gost.db` |
| `POSTGRES_DB` | PostgreSQL 数据库名 | `flvx_svc` |
| `POSTGRES_USER` | PostgreSQL 用户名 | `flvx_svc` |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 | `flvx_svc_change_me` |

## 3. Docker Compose 部署

### 创建 `.env`

```bash
JWT_SECRET=replace_with_your_secret
BACKEND_PORT=6365
FRONTEND_PORT=6366

DB_TYPE=postgres
DATABASE_URL=postgresql://flvx_svc:replace_with_strong_password@postgres:5432/flvx_svc?sslmode=disable

POSTGRES_DB=flvx_svc
POSTGRES_USER=flvx_svc
POSTGRES_PASSWORD=replace_with_strong_password
```

### 启动服务

```bash
docker compose up -d
```

### 查看状态

```bash
docker ps
docker logs flvx-svc-backend
docker logs flvx-svc-postgres
```

## 4. 从 SQLite 迁移到 PostgreSQL

### 推荐方式：安装脚本菜单

```bash
./panel_install.sh
# 选择 4. 迁移到 PostgreSQL
```

脚本会自动：

- 备份 SQLite 数据
- 启动 PostgreSQL
- 通过 `pgloader` 导入数据
- 更新 `.env` 中的 `DB_TYPE` 和 `DATABASE_URL`
- 重启服务

### 手动方式

#### 备份 SQLite

```bash
docker compose down
docker run --rm -v sqlite_data:/data -v "$(pwd)":/backup alpine sh -c "cp /data/gost.db /backup/gost.db.bak"
```

#### 启动 PostgreSQL

```bash
docker compose up -d postgres
```

#### 导入数据

```bash
source .env
docker run --rm \
  --network gost-network \
  -v sqlite_data:/sqlite \
  dimitri/pgloader:latest \
  pgloader /sqlite/gost.db "postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}"
```

#### 切换并重启

```bash
source .env
export DB_TYPE=postgres
export DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
docker compose up -d
```

## 5. 备份与恢复

### 手动备份

```bash
docker exec flvx-svc-postgres pg_dump -U flvx_svc flvx_svc > backup_$(date +%Y%m%d_%H%M%S).sql
```

### 手动恢复

```bash
docker exec -i flvx-svc-postgres psql -U flvx_svc flvx_svc < backup_20260101_120000.sql
```

### 定时备份

```bash
#!/bin/bash
BACKUP_DIR="/opt/flvxt2/backups"
mkdir -p "$BACKUP_DIR"
docker exec flvx-svc-postgres pg_dump -U flvx_svc flvx_svc | gzip > "$BACKUP_DIR/flvxt2_$(date +%Y%m%d_%H%M%S).sql.gz"
find "$BACKUP_DIR" -name "flvxt2_*.sql.gz" -mtime +30 -delete
```

## 6. 常见问题

### 如何确认当前使用 SQLite 还是 PostgreSQL

```bash
docker exec flvx-svc-backend printenv DB_TYPE
```

### PostgreSQL 容器数据在哪里

Docker Compose 默认使用名为 `postgres_data` 的 volume：

```bash
docker volume inspect postgres_data
```
