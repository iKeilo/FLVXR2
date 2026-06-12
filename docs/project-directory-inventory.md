# FLVXR2 项目目录清单

更新日期：2026-06-12  
仓库：`iKeilo/FLVXR2`  
本地分支：`main`  
当前基线：`861b1251`  
最新本地标签：`3.0.24`  
主远程：`https://github.com/iKeilo/FLVXR2.git`

## 扫描口径

这份目录清单用于后续功能更新、发布前核对和代码定位。统计时排除了 `.git`、`.codegraph`、`build`、`node_modules`、`dist`、`docs/assets` 等缓存、索引、构建产物和第三方依赖目录。

当前纳入统计的源码、脚本、文档和配置文件约 `991` 个。其中顶层业务目录统计如下：

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `.github` | 7 | GitHub Actions、Release 模板、Docs/镜像发布流程 |
| `go-backend` | 149 | 面板后端、API、数据库、授权、发布升级、节点部署下发 |
| `go-gost` | 540 | 节点 Agent、GOST 核心适配、WebSocket 上报、nftables、转发服务 |
| `vite-frontend` | 183 | React/Vite 前端、页面、布局、主题、API 类型与全局样式 |
| `docs` | 25 | 当前维护文档、迁移方案、功能设计、项目目录记录 |
| `doc` | 8 | 文档站源内容或兼容文档 |
| `plans` | 36 | 阶段方案、需求设计、历史改造计划 |
| `scripts` | 2 | 发布与同步辅助脚本 |
| `skills` | 16 | Codex 技能与 API 参考资料 |
| `landing` | 7 | 文档站/落地页相关内容 |

## 顶层目录树

```text
FLVXR2/
|-- .github/
|   |-- ISSUES/
|   `-- workflows/
|-- doc/
|-- docs/
|   `-- .vitepress/
|-- go-backend/
|   |-- cmd/paneld/
|   |-- internal/
|   |   |-- app/
|   |   |-- auth/
|   |   |-- config/
|   |   |-- health/
|   |   |-- http/
|   |   |-- license/
|   |   |-- metrics/
|   |   |-- middleware/
|   |   |-- monitoring/
|   |   |-- payment/
|   |   |-- security/
|   |   |-- store/
|   |   `-- ws/
|   |-- migrations/
|   `-- tests/
|-- go-gost/
|   `-- x/
|       |-- api/
|       |-- config/
|       |-- dialer/
|       |-- handler/
|       |-- listener/
|       |-- nftables/
|       |-- service/
|       |-- socket/
|       |-- stats/
|       `-- traffic/
|-- landing/
|-- plans/
|-- scripts/
|-- skills/
|   `-- flvx-api/
`-- vite-frontend/
    |-- public/
    `-- src/
        |-- api/
        |-- components/
        |-- config/
        |-- hooks/
        |-- layouts/
        |-- pages/
        |-- shadcn-bridge/
        |-- styles/
        |-- themes/
        |-- types/
        `-- utils/
```

## 根目录关键文件

| 文件 | 作用 | 更新注意 |
| --- | --- | --- |
| `panel_install.sh` | 面板一键安装/更新脚本 | 发布新版本后确认仓库、镜像名、固定版本和下载地址 |
| `install.sh` | 节点 Agent 安装脚本 | 节点二进制、GitHub Release 地址和服务名改动时必须同步 |
| `install-auto.sh` | 自动安装辅助脚本 | 面板安装入口变化时同步检查 |
| `docker-compose-v4.yml` | IPv4 面板部署编排 | 镜像名、端口、环境变量变化时同步 |
| `docker-compose-v6.yml` | IPv6 面板部署编排 | 与 v4 compose 保持同等能力 |
| `README.md` | 项目说明和安装入口 | Release、安装命令、仓库名变更后同步 |
| `mkdocs.yml` | 文档站配置 | 文档目录变化时检查导航 |
| `AGENTS.md` | 项目协作说明 | Codex 后续工作前优先阅读 |
| `skills-lock.json` | 技能锁定信息 | 更新技能时检查 |

## 后端目录

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `go-backend/cmd/paneld` | 2 | 后端程序入口 |
| `go-backend/internal/app` | - | 应用初始化、服务装配 |
| `go-backend/internal/auth` | - | 登录、鉴权、会话相关逻辑 |
| `go-backend/internal/config` | - | 环境配置、运行配置 |
| `go-backend/internal/http/handler` | 54 | API Handler，覆盖节点、隧道、转发、TLS、授权、商城、升级、面板共享等业务 |
| `go-backend/internal/http/middleware` | - | HTTP 中间件 |
| `go-backend/internal/http/response` | - | API 响应封装 |
| `go-backend/internal/store/model` | 5 | 数据模型 |
| `go-backend/internal/store/repo` | 22 | 数据访问层、迁移兼容、业务查询 |
| `go-backend/internal/store/migrations` | - | 数据库迁移 |
| `go-backend/internal/license` | - | 授权端地址、授权校验相关逻辑 |
| `go-backend/internal/middleware` | 2 | 商业授权/功能开关底层校验 |
| `go-backend/internal/payment` | 5 | 商城、套餐、订单、支付配置 |
| `go-backend/internal/ws` | 1 | 节点 WebSocket 通信 |
| `go-backend/internal/monitoring` | - | 监控/探针配置 |
| `go-backend/internal/metrics` | - | 指标采集与处理 |
| `go-backend/tests` | - | 合约测试与回归验证 |

### 后端重点文件

| 文件 | 职责 |
| --- | --- |
| `go-backend/cmd/paneld/main.go` | 后端启动入口、授权启动校验 |
| `go-backend/internal/http/handler/handler.go` | 路由注册、Handler 汇总 |
| `go-backend/internal/http/handler/node_deploy.go` | 节点入站部署、配置下发、进度查询、复制信息 |
| `go-backend/internal/http/handler/panel_sharing.go` | 面板共享/远程节点导入相关入口 |
| `go-backend/internal/http/handler/tunnel_wg.go` | WG 隧道相关后端逻辑 |
| `go-backend/internal/http/handler/license_commercial.go` | 商业授权功能检查 |
| `go-backend/internal/http/handler/upgrade.go` | 面板/节点 Release 查询与升级入口 |
| `go-backend/internal/http/handler/system_upgrade.go` | 系统级升级和 compose 资产下载 |
| `go-backend/internal/store/repo/repository.go` | 数据库初始化、默认配置种子 |
| `go-backend/internal/store/repo/repository_node_deploy.go` | 节点部署记录、模板和入站数据访问 |

## 前端目录

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `vite-frontend/src/pages` | 48 | 页面入口，包含主页、节点、隧道、规则、用户、TLS、商城、设置等 |
| `vite-frontend/src/pages/node` | 14 | 节点页拆分组件，包含部署弹窗、分组、入站/TLS 嵌入逻辑 |
| `vite-frontend/src/pages/forward` | 6 | 端口转发相关辅助逻辑和组件 |
| `vite-frontend/src/pages/tunnel` | 4 | 隧道表单、诊断、分组、WG 适配 |
| `vite-frontend/src/api` | 5 | API 请求、类型、错误处理 |
| `vite-frontend/src/layouts` | 3 | 管理端布局、H5 布局、默认布局 |
| `vite-frontend/src/components` | 32 | 通用组件、状态组件、版本提示等 |
| `vite-frontend/src/shadcn-bridge` | 24 | HeroUI/shadcn 兼容桥接组件 |
| `vite-frontend/src/styles` | 2 | 全局样式、主题样式、移动端/毛玻璃布局 |
| `vite-frontend/src/themes` | 9 | 主题系统和主题上下文 |
| `vite-frontend/src/utils` | 10 | 会话、授权判断、格式化、版本更新等工具 |
| `vite-frontend/src/hooks` | 7 | React Hooks |

### 前端重点文件

| 文件 | 职责 |
| --- | --- |
| `vite-frontend/src/App.tsx` | 路由和桌面/H5 布局选择 |
| `vite-frontend/src/main.tsx` | React 入口、PWA 注册 |
| `vite-frontend/src/layouts/admin.tsx` | 桌面端管理布局、导航、顶部栏、底部 Powered 标识 |
| `vite-frontend/src/layouts/h5.tsx` | 手机端布局、底部导航、移动端菜单 |
| `vite-frontend/src/styles/globals.css` | 全局视觉、毛玻璃、响应式、移动端独立样式 |
| `vite-frontend/src/pages/dashboard.tsx` | 首页统计、流量图、节点到期提醒 |
| `vite-frontend/src/pages/node.tsx` | 节点管理主页面，含卡片/分组/新增/TLS 入口 |
| `vite-frontend/src/pages/node/node-deploy-modal.tsx` | 入站添加/编辑/部署窗口，复制链接、二维码、进度轮询 |
| `vite-frontend/src/pages/tunnel.tsx` | 隧道管理主页面，含 WG 隧道入口和诊断 |
| `vite-frontend/src/pages/forward.tsx` | 规则/端口转发管理 |
| `vite-frontend/src/pages/tls.tsx` | TLS/Reality 模板管理 |
| `vite-frontend/src/pages/config.tsx` | 系统设置、站点品牌、商业授权、商城/注册/探针开关 |
| `vite-frontend/src/pages/panel-sharing.tsx` | 面板共享、远程节点管理 |
| `vite-frontend/src/api/index.ts` | API 方法集中定义 |
| `vite-frontend/src/api/types.ts` | 前端 API 类型定义 |
| `vite-frontend/src/config/site.ts` | 站点名称、Logo、背景、仓库地址默认配置 |

## 节点 Agent 目录

| 路径 | 文件数 | 主要职责 |
| --- | ---: | --- |
| `go-gost/main.go` | 1 | Agent 程序入口 |
| `go-gost/config.go` | 1 | Agent 配置读取 |
| `go-gost/program.go` | 1 | 服务运行封装 |
| `go-gost/register.go` | 1 | 节点注册、面板连接 |
| `go-gost/x/socket` | 15 | WebSocket、nftables、节点上报、服务控制、核心配置应用 |
| `go-gost/x/service` | 5 | 转发服务、流量上报 |
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
| Git 远程 | `https://github.com/iKeilo/FLVXR2.git` | 源码推送 |
| Release 仓库 | `iKeilo/FLVXR2` | 安装脚本、compose、节点二进制和离线包下载 |
| GHCR 后端镜像 | `ghcr.io/ikeilo/flvxr2-svc-backend` | 面板后端容器 |
| GHCR 前端镜像 | `ghcr.io/ikeilo/flvxr2-svc-frontend` | 面板前端容器 |
| 面板安装脚本 | `panel_install.sh` | 下载 compose 并拉起面板 |
| 节点安装脚本 | `install.sh` | 下载 `gost-{ARCH}` 并部署 Agent |
| 发布脚本 | `scripts/release.ps1` | 构建、打包、推送、Release/Packages 发布工作流 |
| GitHub Action | `.github/workflows/docker-build.yml` | tag 推送时构建镜像、Release 资产和离线包 |

## 后续更新定位表

| 更新类型 | 优先检查目录 | 必跑检查 |
| --- | --- | --- |
| 桌面 UI/主题/毛玻璃 | `vite-frontend/src/layouts/admin.tsx`、`vite-frontend/src/styles/globals.css`、`vite-frontend/src/themes` | `npm run build` |
| 手机端 UI | `vite-frontend/src/layouts/h5.tsx`、`vite-frontend/src/pages/dashboard.tsx`、`vite-frontend/src/styles/globals.css` | 手机视口截图、`npm run build` |
| 节点部署/入站管理 | `vite-frontend/src/pages/node`、`go-backend/internal/http/handler/node_deploy.go`、`go-gost/x/socket` | 前端构建、后端编译、测试服部署验证 |
| TLS/Reality 模板 | `vite-frontend/src/pages/tls.tsx`、后端 TLS Handler/Repo | 保存、复制、下发链路验证 |
| 面板共享/远程节点 | `vite-frontend/src/pages/panel-sharing.tsx`、`go-backend/internal/http/handler/panel_sharing.go`、Repo 相关文件 | 导入、删除、隧道/规则影响确认 |
| WG 隧道 | `vite-frontend/src/pages/tunnel*`、`go-backend/internal/http/handler/tunnel_wg.go`、`go-gost/x/socket` | 创建、编辑、诊断、规则联动验证 |
| 用户/权限/商业授权 | `go-backend/internal/http/handler/license_commercial.go`、`go-backend/internal/middleware`、`vite-frontend/src/pages/config.tsx`、`vite-frontend/src/pages/user.tsx` | 未授权置灰、后端拒绝保存、普通用户权限验证 |
| 商城/订单/支付 | `go-backend/internal/payment`、`go-backend/internal/http/handler/product.go`、`vite-frontend/src/pages/admin-*`、`vite-frontend/src/pages/shop.tsx` | 管理员/普通用户可见性验证 |
| 安装/发布 | `panel_install.sh`、`install.sh`、`docker-compose-*.yml`、`.github/workflows/docker-build.yml`、`scripts/release.ps1` | Release 资产、GHCR 镜像、一键安装命令验证 |

## 当前工作区注意事项

本次目录生成时，工作区存在未提交的手机端 UI 调整改动：

```text
vite-frontend/src/components/animated-page.tsx
vite-frontend/src/hooks/useScrollTopOnPathChange.ts
vite-frontend/src/layouts/h5.tsx
vite-frontend/src/main.tsx
vite-frontend/src/pages/dashboard.tsx
vite-frontend/src/styles/globals.css
```

这些改动和本目录文档属于不同工作面。后续提交时建议分开提交：先提交手机端 UI 修复，再提交目录文档更新。

## 清理建议

当前仓库中可见一些运行或构建痕迹，后续发布前建议单独确认是否需要保留：

| 路径 | 类型 | 建议 |
| --- | --- | --- |
| `build/` | 构建/调试输出 | 通常不纳入发布提交 |
| `gcm-diagnose.log` | Git 凭据诊断日志 | 如无排障需要，可不纳入提交 |
| `go-backend/paneld.exe` | 本地编译产物 | 通常由 Release 构建生成 |
| `go-gost/gost.exe` | 本地编译产物 | 通常由 Release 构建生成 |
| `go-gost/gost-linux-amd64` | 本地编译产物 | 通常由 Release 构建生成 |
| `go-gost/flux_agent` | 本地编译产物 | 需要确认是否仍作为资产使用 |
| `vite-frontend/preview-*.log` | 前端预览日志 | 如无排障需要，可不纳入提交 |

## 更新记录模板

| 日期 | 版本/Commit | 更新范围 | 影响目录 | 验证结果 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 2026-06-12 | `861b1251` | 重新生成项目目录清单 | `docs/project-directory-inventory.md` | 目录统计完成 | 当前工作区仍有手机端 UI 未提交改动 |
