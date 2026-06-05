# FLVX 项目目录清单

更新时间：2026-06-05  
仓库：`iKeilo/flvxt2`  
当前基线：`main`，Release `3.0.16`

## 扫描范围

本清单用于后续项目更新前的目录对照。统计时排除了构建缓存与第三方依赖目录，例如 `.git`、`build`、`vite-frontend/dist`、`vite-frontend/node_modules`。当前纳入统计的源码、脚本和文档文件约 `998` 个。

## 顶层目录

| 路径 | 文件数 | 主要职责 | 更新注意 |
| --- | ---: | --- | --- |
| `.github` | 7 | GitHub Actions、Release 模板、CI/镜像发布流程 | 发布版本前重点检查 `docker-build.yml` 是否能构建 Release 资产 |
| `go-backend` | 137 | 面板后端、API、授权、数据库、升级、节点部署下发 | 改配置、授权、节点部署、商城支付时优先检查 |
| `go-gost` | 537 | 节点 Agent、转发核心适配、WebSocket 上报、nftables/服务控制 | 改节点安装、离线部署、下发核心配置时优先检查 |
| `vite-frontend` | 174 | React/Vite 前端、页面、布局、API 类型、UI 样式 | 改 UI、节点页、TLS 管理、商业授权显示时优先检查 |
| `docs` | 68 | 项目文档、维护记录、静态文档站内容 | 后续方案和目录记录放这里 |
| `doc` | 8 | 旧文档或补充材料 | 改动前先确认是否仍被引用 |
| `plans` | 36 | 方案、阶段性计划、需求整理 | 新功能设计方案可继续放这里 |
| `scripts` | 1 | 辅助脚本 | 发布或维护脚本改动要同步验证 |
| `skills` | 16 | Codex/插件技能相关资料 | 通常不影响产品运行 |
| `landing` | 7 | 落地页/官网相关文件 | UI 或文档站发布时检查 |

## 根目录关键文件

| 文件 | 作用 | 后续更新注意 |
| --- | --- | --- |
| `panel_install.sh` | 面板一键安装/更新脚本 | 仓库地址应保持 `iKeilo/flvxt2`，Release 版会写入 `PINNED_VERSION` |
| `install.sh` | 节点 Agent 安装脚本 | 节点安装文件从 GitHub Release 下载 `gost-{ARCH}` |
| `install-auto.sh` | 自动安装辅助脚本 | 改安装入口时同步检查 |
| `docker-compose-v4.yml` | IPv4 面板部署编排 | Release 会替换镜像为指定版本 |
| `docker-compose-v6.yml` | IPv6 面板部署编排 | Release 会替换镜像为指定版本 |
| `README.md` | 项目说明与安装入口 | 发布后安装命令变化需同步 |
| `mkdocs.yml` | 文档站配置 | 文档结构变化时检查 |
| `AGENTS.md` | 项目协作/代理说明 | Codex 后续工作先读 |

## 后端目录表

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `go-backend/cmd/paneld` | 2 | 后端程序入口 |
| `go-backend/internal/http/handler` | 39 | API Handler，包含节点、转发、TLS、授权、商城、升级、部署等业务入口 |
| `go-backend/internal/http/middleware` | 1 | HTTP 中间件 |
| `go-backend/internal/http/response` | 1 | API 响应封装 |
| `go-backend/internal/http/client` | 1 | 后端 HTTP 客户端辅助 |
| `go-backend/internal/store/model` | 4 | 数据模型 |
| `go-backend/internal/store/repo` | 21 | 数据访问层和迁移测试 |
| `go-backend/internal/store/migrations` | 1 | SQL 迁移 |
| `go-backend/internal/license` | 2 | 授权校验相关逻辑 |
| `go-backend/internal/middleware` | 2 | 底层授权/许可检查 |
| `go-backend/internal/payment` | 5 | 支付配置、订单、产品相关能力 |
| `go-backend/internal/ws` | 1 | 节点 WebSocket 通信 |
| `go-backend/internal/monitoring` | 1 | 监控服务配置 |
| `go-backend/internal/metrics` | 2 | 指标采集与处理 |
| `go-backend/tests` | 多个 | 合约测试 |

### 后端重点文件

| 文件 | 职责 |
| --- | --- |
| `go-backend/internal/http/handler/handler.go` | 路由注册、配置保存、核心 Handler 结构 |
| `go-backend/internal/http/handler/node_deploy.go` | 节点入站部署、下发、复制信息、进度查询 |
| `go-backend/internal/store/repo/repository_node_deploy.go` | 节点部署记录和模板相关数据访问 |
| `go-backend/internal/http/handler/monitoring.go` | 节点监控与公开探针接口 |
| `go-backend/internal/http/handler/license_commercial.go` | 商业授权功能开关检查 |
| `go-backend/internal/http/handler/upgrade.go` | 面板/节点 Release 查询和升级入口 |
| `go-backend/internal/http/handler/system_upgrade.go` | 系统级升级、compose 资产下载 |
| `go-backend/internal/http/handler/product.go` | 商城状态和商品管理 |
| `go-backend/internal/store/repo/repository.go` | 数据库初始化、默认配置种子 |

## 前端目录表

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `vite-frontend/src/pages` | 47 | 页面入口，包含节点、隧道、转发、TLS、商城、设置等 |
| `vite-frontend/src/pages/node` | 4 | 节点页拆分组件，包含部署弹窗、分组管理 |
| `vite-frontend/src/pages/forward` | 6 | 端口转发相关辅助逻辑 |
| `vite-frontend/src/pages/tunnel` | 3 | 隧道表单、诊断、分组管理 |
| `vite-frontend/src/api` | 5 | API 请求、类型、错误处理 |
| `vite-frontend/src/layouts` | 3 | 管理端/H5/默认布局 |
| `vite-frontend/src/components` | 32 | 通用组件 |
| `vite-frontend/src/shadcn-bridge` | 24 | HeroUI/shadcn 兼容桥接组件 |
| `vite-frontend/src/styles` | 2 | 全局样式和主题样式 |
| `vite-frontend/src/themes` | 9 | 主题系统 |
| `vite-frontend/src/utils` | 10 | 会话、版本、格式化等工具 |
| `vite-frontend/src/hooks` | 7 | React hooks |

### 前端重点文件

| 文件 | 职责 |
| --- | --- |
| `vite-frontend/src/pages/node.tsx` | 节点管理主页面，含卡片/分组/新增/TLS 二级入口 |
| `vite-frontend/src/pages/node/node-deploy-modal.tsx` | 入站添加/编辑/部署弹窗，复制链接、二维码、进度轮询 |
| `vite-frontend/src/pages/tls.tsx` | TLS/Reality 模板管理，可嵌入节点页弹窗 |
| `vite-frontend/src/pages/config.tsx` | 系统设置、站点品牌、授权、商城/注册/探针开关 |
| `vite-frontend/src/layouts/admin.tsx` | 管理端主布局、导航、底部 Powered 标识 |
| `vite-frontend/src/layouts/h5.tsx` | H5 布局 |
| `vite-frontend/src/api/index.ts` | API 方法集中定义 |
| `vite-frontend/src/api/types.ts` | 前端 API 类型定义 |
| `vite-frontend/src/styles/globals.css` | 毛玻璃、圆角、布局全局样式 |
| `vite-frontend/src/config/site.ts` | 版本、仓库地址、站点默认配置 |

## 节点 Agent 目录表

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `go-gost/main.go` | 1 | Agent 程序入口 |
| `go-gost/config.go` | 1 | Agent 配置读取 |
| `go-gost/program.go` | 1 | 服务运行封装 |
| `go-gost/register.go` | 1 | 节点注册/连接面板 |
| `go-gost/x/socket` | 13 | WebSocket、nftables、节点上报、服务控制 |
| `go-gost/x/service` | 5 | 转发服务与流量上报 |
| `go-gost/x/traffic` | 2 | 流量周期/基线 |
| `go-gost/x/stats` | 3 | 转发统计 |
| `go-gost/x/nftables` | 2 | nftables 适配 |
| `go-gost/x/api` | 25 | GOST API 配置结构 |
| `go-gost/x/handler` | 78 | 协议处理器 |
| `go-gost/x/listener` | 92 | 入站监听器 |
| `go-gost/x/dialer` | 67 | 出站拨号器 |
| `go-gost/x/config` | 24 | GOST 配置解析 |

## 发布与安装链路

| 链路 | 当前指向 | 用途 |
| --- | --- | --- |
| Git 远端 | `https://github.com/iKeilo/flvxt2.git` | 源码推送 |
| Release 仓库 | `iKeilo/flvxt2` | 安装脚本、compose、节点二进制下载 |
| GHCR 镜像 | `ghcr.io/ikeilo/flvx-svc-backend` | 后端面板镜像 |
| GHCR 镜像 | `ghcr.io/ikeilo/flvx-svc-frontend` | 前端面板镜像 |
| 面板安装脚本 | `panel_install.sh` | 下载 compose 并拉起面板 |
| 节点安装脚本 | `install.sh` | 下载 `gost-{ARCH}` 并部署 Agent |
| GitHub Action | `.github/workflows/docker-build.yml` | tag 推送时构建镜像、Release 资产和离线包 |

## 后续更新检查表

| 更新类型 | 优先检查目录 | 必跑检查 |
| --- | --- | --- |
| UI/布局/毛玻璃风格 | `vite-frontend/src/layouts`、`vite-frontend/src/styles`、`vite-frontend/src/pages` | `npm run build` |
| 节点部署/入站管理 | `vite-frontend/src/pages/node`、`go-backend/internal/http/handler/node_deploy.go`、`go-backend/internal/store/repo/repository_node_deploy.go`、`go-gost/x/socket` | 前端构建、后端编译、测试服部署验证 |
| TLS/Reality 模板 | `vite-frontend/src/pages/tls.tsx`、后端 TLS 模板 Handler/Repo | 前端构建、保存/复制/下发链路验证 |
| 商业授权限制 | `go-backend/internal/http/handler/license_commercial.go`、`go-backend/internal/middleware`、`vite-frontend/src/pages/config.tsx` | 未授权置灰、后端拒绝保存、公开接口不泄露 |
| 商城/订单/支付 | `go-backend/internal/http/handler/product.go`、`payment.go`、`order.go`、`vite-frontend/src/pages/admin-*` | 管理员/普通用户可见性验证 |
| 安装/发布 | `panel_install.sh`、`install.sh`、`docker-compose-*.yml`、`.github/workflows/docker-build.yml` | tag 发布后检查 Release 资产、GHCR 镜像、安装脚本固定版本 |

## 更新记录模板

| 日期 | 版本/Commit | 更新范围 | 影响目录 | 验证结果 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 2026-06-05 | `3.0.16` / `47da516a` | TLS 弹窗、商业授权限制、探针入口收紧、UI 细节、发布链路修复 | `go-backend`、`vite-frontend`、`.github` | 前端构建通过；后端 amd64/arm64 编译通过；Release 发布成功 | `go-backend/internal/http/handler` 仍有既有测试断言需后续单独清理 |

