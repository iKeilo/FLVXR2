# 项目整理与入侵痕迹检查记录

检查时间：2026-06-13

## 检查范围

- Git 工作区、最近提交、远程仓库配置。
- Git hooks、GitHub Actions 工作流、发布脚本、安装脚本。
- 近三天内变更文件。
- 常见入侵痕迹关键字：硬编码 token、私钥、反连命令、`eval`、Base64 解码执行、`authorized_keys`、`curl | bash`、旧授权端、旧仓库链接等。
- 前端依赖审计与 Go 依赖版本检查。
- Git 对象完整性检查。

## 当前项目状态

- 当前分支：`main`
- 当前远程：
  - `origin` -> `https://github.com/iKeilo/FLVXR2.git`
  - `flvxr2` -> `https://github.com/iKeilo/FLVXR2.git`
  - `old-origin` -> `https://github.com/iKeilo/flvxt2.git`
- 最近提交：`0fa2d8f7 Add sidebar panel update entry`
- 当前存在未提交改动，主要集中在限速规则、用户限速、隧道限速、规则绑定相关文件。
- 当前存在未跟踪文档：`docs/speed-limit-scope-design.md`

## 未发现的高危入侵迹象

本轮检查未发现以下典型入侵痕迹：

- 未发现启用的恶意 Git hook，`.git/hooks` 下只有默认 `.sample` 文件。
- 未发现硬编码 GitHub token、AWS key、Slack token、私钥文件等高风险凭据。
- 未发现 `/dev/tcp`、`nc -e`、`ncat -e`、`authorized_keys` 写入等反连或持久化痕迹。
- 未发现异常 `eval`、`new Function`、`FromBase64String`、`base64 -d` 后执行代码的模式。
- 未发现 `LICENSE_SERVER_URL=https://sq.abai.eu.org` 残留。
- `git fsck --full --no-reflogs` 未报告对象损坏，只报告了 dangling commit/blob，这通常来自 rebase、reset、临时提交或历史操作，不等于入侵。

## 需要整理的残留项

### 1. 旧远程仍保留

`.git/config` 中仍存在：

```text
old-origin -> https://github.com/iKeilo/flvxt2.git
```

这不是入侵痕迹，但后续发布时容易误推或误拉旧项目。建议确认不再需要后执行：

```bash
git remote remove old-origin
```

### 2. 文档构建产物仍有旧仓库链接

扫描到以下旧链接残留：

- `docs/.vitepress/config.ts` 仍指向 `https://github.com/iKeilo/flvxt2`
- `docs/assets/*.js` 中仍有旧 `iKeilo/flvxt2` 的构建产物
- `landing/doc/install.md` 中仍有 `Sagit-chu/flux-panel`
- `landing/doc/ai-skill.md` 中仍有 `Sagit-chu/flvx`

这些更像历史文档/上游兼容说明残留，不像被植入。若要彻底品牌独立，建议统一替换为 `iKeilo/FLVXR2`，并重新构建 docs/landing。

### 3. 仍有兼容旧项目的协议命名

代码中存在 `flvxt2-v1`、`Sagit-chu compatible`，主要出现在面板共享兼容逻辑中。这个属于之前设计的兼容行为，不是入侵痕迹。

### 4. 发布辅助脚本中仍有国内同步脚本

`scripts/sync-gost-to-domestic.sh` 仍存在。当前主安装脚本已经走 GitHub，但如果后续策略是完全取消国内加速，可以考虑删除或改为归档说明，避免误用。

## 安装与升级脚本检查

主脚本当前关键仓库指向正常：

- `install.sh`：`REPO="iKeilo/FLVXR2"`
- `panel_install.sh`：`REPO="iKeilo/FLVXR2"`
- Docker 镜像：`ghcr.io/ikeilo/flvxr2-svc-backend`、`ghcr.io/ikeilo/flvxr2-svc-frontend`
- 后端升级逻辑：下载 `https://raw.githubusercontent.com/iKeilo/FLVXR2/main/panel_install.sh`

`panel_install.sh` 会删除 `.env` 中的 `LICENSE_SERVER_URL`：

```bash
sed -i '/^LICENSE_SERVER_URL=/d' ".env"
```

后端 `env_writer.go` 也会移除 `LICENSE_SERVER_URL` 行。

## 依赖检查

### 前端 npm audit

`npm audit --audit-level=high --omit=dev` 返回 3 个 moderate 级别问题：

- `postcss < 8.5.10`
- `react-router / react-router-dom <= 6.30.3`

这些不是入侵痕迹，但属于依赖安全维护项。建议后续单独升级并回归测试。

### Go 依赖

`go list -m -u all` 显示若干依赖有新版本，例如：

- `golang.org/x/crypto`
- `golang.org/x/net`
- `golang.org/x/sys`
- `modernc.org/sqlite`
- `github.com/jackc/pgx/v5`

本轮未发现明显异常来源依赖。是否升级建议单独开任务处理，避免影响 SQLite/PostgreSQL 兼容性。

## 本轮结论

当前项目没有发现明确被入侵或被植入后门的证据。

更像是项目长期迁移、合并和发布过程中留下的整理问题：

- 旧远程仍存在。
- 文档和 landing 构建产物有旧仓库链接。
- 发布脚本目录保留了国内同步脚本。
- 依赖存在中等级别安全更新需求。
- 当前有大量未提交功能改动，应在继续发布前先完成测试、修复旧契约测试或明确跳过原因，然后提交。

## 建议下一步

1. 删除不再需要的 `old-origin` 远程。
2. 统一 docs/landing 中的旧项目链接并重新构建文档。
3. 决定是否删除或归档 `scripts/sync-gost-to-domestic.sh`。
4. 修正 `/forward/list` 旧契约测试，使其读取当前分页结构 `data.items`。
5. 单独升级 `postcss`、`react-router-dom`，并跑前端构建与核心页面回归。
6. 在发布前提交当前限速功能改动，避免工作区长期处于混杂状态。
