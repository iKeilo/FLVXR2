# 014-license-trial-mode.md

## 背景

当前授权模式对想体验面板的用户极不友好。

## 目标

零门槛安装体验 + 面板内在线授权配置 + 体验模式资源限制。

---

## 任务清单

### Phase 1: 安装脚本改造 - 授权码/域名改为可选

- [ ] 1.1 panel_install.sh 改造
  - LICENSE_KEY 改为可选，允许留空跳过
  - 填了授权码则必须手动填 SERVER_DOMAIN
  - 跳过授权码则 SERVER_DOMAIN 也跳过

### Phase 2: 后端新增授权配置 API

- [ ] 2.1 新增 go-backend/internal/http/handler/license_config.go
  - POST /api/v1/license/config 保存到 config 表（license_key + domain）
  - 服务器 URL 后端默认 https://sq.sbplay.eu.org
  - 参数：{ license_key, domain }
  - 保存后触发 TriggerAsyncCheck

- [ ] 2.2 license_info.go 返回 has_license_key + domain

- [ ] 2.3 handler.go 注册新路由

- [ ] 2.4 license_guard.go 白名单新增 /api/v1/license/config

### Phase 3: 前端授权配置 UI

- [ ] 3.1 api/index.ts 新增 updateLicenseConfig(licenseKey, domain) + 扩展 has_license_key / domain 字段

- [ ] 3.2 config.tsx 新增授权码配置区块
  - 输入框1：授权码 UUID
  - 输入框2：面板域名
  - 保存按钮 + 状态显示
  - 仅 admin 可见

### Phase 4: 状态提示优化

- [ ] 4.1 admin.tsx 底部状态栏
  - 未配置：黄色 体验模式（已限制资源）
  - 有效：绿色 授权有效剩余X天
  - 无效：红色 授权无效+链接到/config

- [ ] 4.2 h5.tsx 同理

### Phase 5: 体验模式资源限制

- [ ] 5.1 新增 trial_guard.go 中间件
  - 节点 <= 5
  - 隧道 <= 5
  - 规则 不限
  - 用户 <= 1 (除admin外)

- [ ] 5.2 router.go 集成 TrialGuard

---

## 改动文件清单

| 文件 | 改动 |
|------|------|
| panel_install.sh | 授权码/域名改为可选 |
| go-backend/internal/http/handler/license_config.go | 新增：授权配置 API |
| go-backend/internal/http/handler/license_info.go | 返回 has_license_key + domain |
| go-backend/internal/http/handler/handler.go | 注册新路由 |
| go-backend/internal/http/middleware/license_guard.go | 白名单新增 |
| go-backend/internal/http/middleware/trial_guard.go | 新增：体验模式限制 |
| vite-frontend/src/api/index.ts | 新增 API + 扩展类型 |
| vite-frontend/src/pages/config.tsx | 新增授权码+域名输入 |
| vite-frontend/src/layouts/admin.tsx | 状态提示优化 |
| vite-frontend/src/layouts/h5.tsx | 状态提示优化 |
