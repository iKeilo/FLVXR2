# 节点批量化部署设计与进度

## 背景

这份文档用于回顾你提出的需求，并整理当前已经在 flvxt2 中完成的改造内容。

这次目标不是把整个 S-UI 面板塞进 flvxt2，也不是复制一套完整 S-UI 管理系统。真正目标是：把 S-uiR 中对本项目有价值的部分能力抽出来，整合进 flvxt2 的节点管理，让 flvxt2 能通过模板生成节点入站配置，并在每台服务器的关键字段上做随机化，最后把完整 core 配置文件下发到节点运行。

整体部署路径是“面板轻控制，节点跑配置”：

1. 面板负责编辑 TLS 模板、节点身份、入站模板和已部署入站。
2. 后端负责渲染完整 core 配置。
3. 配置通过现有节点 WebSocket 通道下发。
4. 节点写入配置文件，并用本地 core 运行。
5. 面板提供可复制的节点连接信息，方便直接使用。

## 你的需求回顾

你一路提出的核心需求可以整理为：

- 在 flvxt2 现有节点功能中增加“批量化部署”能力。
- 通过模板部署节点。
- 在关键字段上随机化，避免不同机器出现完全相同的配置。
- 只整合 S-uiR 的部分能力，不整合整个 S-UI 面板。
- 需要 TLS 管理，尤其是 Reality 等 TLS/伪装相关设置。
- 需要入站管理，类似 S-UI 的入站编辑能力。
- 需要类似 S-uiR 用户管理的能力，但这里每个服务器/节点都可以理解成一个用户。
- 在节点页面增加节点部署按钮。
- 部署按钮唤起二级部署菜单或弹窗。
- 部署时使用服务器绑定的 UUID 和其它随机凭据。
- 部署时可以选择协议、TLS 模板、端口、地址和入站参数。
- 默认监听地址使用回环地址。
- 节点部署名称如果不设置，默认为“节点名字 + 协议名字”。
- 如果部署名称重复，则自动追加数字。
- 节点页面需要能查看已经部署的节点入站。
- 下发方式不是通过 UI 直接控制运行逻辑，而是编辑节点上的配置文件。
- 节点以内核方式运行生成的配置，也就是直接跳过 UI。
- 面板只负责编辑节点、生成配置、下发配置。
- 已部署节点信息必须可以复制，从而直接使用。
- 二维码必须完全离线生成，不能调用外部 QR API。
- 新功能整体需要尽量离线可用。

## 从 S-uiR 借鉴的思路

当前实现不是照搬 S-uiR，而是借鉴了这些概念：

- TLS 模板：
  - 可复用的服务端 TLS JSON。
  - 可复用的客户端 TLS JSON。
- Reality 支持：
  - 服务端 Reality 字段。
  - 客户端 Reality 字段。
  - short ID 处理。
- 用户/客户端概念：
  - 在 flvxt2 中，每个节点拥有自己的固定身份。
  - 这个身份类似 S-uiR 里的用户或客户端。
- 入站动态参数：
  - 协议相关入站参数用 JSON 保存。
  - 渲染配置时合并到最终入站配置中。

## 当前架构

### 数据模型

新增了几类持久化模型：

- `NodeTLSTemplate`
  - 保存 TLS 或 Reality 模板。
  - 包含服务端 JSON 和客户端 JSON。
- `NodeIdentity`
  - 保存每个节点绑定的随机身份和凭据。
  - 包含 UUID、各协议密码、Reality short ID、路径后缀、服务后缀等。
- `NodeDeployedInbound`
  - 保存某个节点已经部署的入站定义。
  - 保存渲染后的服务端配置、客户端配置、分享链接、监听地址、发布地址、端口等。
- `NodeConfigRevision`
  - 保存生成过的 core 配置版本。
  - 包含 checksum 和部署状态。
- `NodeDeployLog`
  - 保存部署、回滚、删除、应用配置等操作日志。

这些模型已经加入仓库的自动迁移。

### 后端 API

新增了这些节点部署接口：

- `/api/v1/node/tls-template/list`
- `/api/v1/node/tls-template/save`
- `/api/v1/node/tls-template/delete`
- `/api/v1/node/deploy/detail`
- `/api/v1/node/deploy/identity/regenerate`
- `/api/v1/node/deploy/inbound/save`
- `/api/v1/node/deploy/inbound/delete`
- `/api/v1/node/deploy/apply`
- `/api/v1/node/deploy/rollback`

后端当前渲染的是 sing-box 风格配置：

```json
{
  "log": {
    "level": "warning"
  },
  "inbounds": [],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
```

启用状态的已部署入站会被渲染进 `inbounds`。

每个入站会保存：

- 服务端配置 JSON。
- 客户端配置 JSON。
- 分享 URI。
- 入站参数 JSON。
- 选中的 TLS 模板。

### 节点身份

每个节点都会获得一个稳定的随机身份。除非手动重新生成，否则这个身份会保持不变。

当前生成的字段包括：

- VLESS UUID。
- mixed/SOCKS/HTTP 使用的密码。
- Trojan 密码。
- Hysteria2 密码。
- TUIC UUID 和密码。
- Reality short ID。
- 路径后缀。
- 服务后缀。
- seed。

这满足了“把每一个节点都当成一个用户/客户端”的需求。

### 名称生成

如果部署表单没有指定名称，后端会按你的要求生成默认名称：

```text
节点名字 + 协议名字
```

当前代码中的格式是：

```text
<节点名>-<协议名大写>
```

例如：

```text
hk-node-VLESS
```

如果同一个节点下已经存在相同部署名称，后端会自动追加数字后缀。

### TLS 和 Reality 模板

TLS 模板分成两段 JSON：

- `serverJson`
  - 合并到服务端入站配置的 `tls` 字段。
- `clientJson`
  - 写入生成出来的客户端配置。

Reality 模板有额外处理：

- 强制开启 `reality.enabled`。
- 如果服务端模板没有指定 short ID，则自动注入节点绑定的 Reality short ID。
- 如果客户端模板没有指定 short ID，也自动注入节点绑定的 Reality short ID。

前端 TLS 表单支持通过常见字段生成 JSON：

- SNI / server name。
- fingerprint。
- 证书文件路径。
- 私钥文件路径。
- 证书内容。
- 私钥内容。
- insecure 开关。
- Reality 握手服务器。
- Reality 握手端口。
- Reality private key。
- Reality public key。
- Reality short IDs。

### 入站管理

部署弹窗当前支持这些协议：

- VLESS。
- Hysteria2。
- Trojan。
- Shadowsocks。
- TUIC。
- SOCKS。
- HTTP。

当前已暴露的协议字段包括：

- VLESS：
  - flow。
  - transport type。
  - transport path。
- Hysteria2：
  - up Mbps。
  - down Mbps。
  - obfs password。
- Shadowsocks：
  - method。
- TUIC：
  - congestion control。
- Trojan / SOCKS / HTTP：
  - 默认使用节点绑定凭据。

此外还保留了高级 JSON 编辑器，用于填写更细的协议参数。

### 配置下发与 core 运行

节点侧 WebSocket reporter 现在支持这些命令：

- `ApplyCoreConfig`
- `GetCoreConfig`
- `RestartCore`

执行 `ApplyCoreConfig` 时，节点会：

1. 接收 `coreType`、`configJson` 和 `checksum`。
2. 如果提供了 checksum，则校验配置内容。
3. 校验配置必须是合法 JSON。
4. 写入临时配置文件。
5. 如果是 sing-box 配置，则执行 `sing-box check -c`。
6. 备份旧配置。
7. 替换目标配置文件。
8. 创建或更新本地 systemd 服务。
9. 重启 core 服务。
10. 如果重启失败，回滚配置文件。

Linux 下 sing-box 配置路径是：

```text
/etc/flvxt2/sing-box.json
```

托管的 systemd 服务名是：

```text
flvxt2-sing-box
```

服务运行命令是：

```text
sing-box run -c /etc/flvxt2/sing-box.json
```

离线规则：

- 节点不会自动下载 `sing-box`。
- 如果节点本机没有安装 `sing-box`，部署会明确失败并提示。

### 前端页面

节点页面现在增加了部署入口：

- 节点卡片/网格视图中有部署入口。
- 节点列表视图中也有部署入口。

部署弹窗包含：

- 节点身份查看。
- 节点身份重新生成。
- 入站创建和编辑。
- TLS/Reality 模板创建。
- 已部署入站列表。
- 配置预览。
- 配置版本列表。
- 配置回滚。
- 部署日志。
- 多种复制按钮。
- 离线二维码预览。

### 可复制的节点信息

每个已部署入站支持复制：

- 分享 URI。
- 客户端 JSON。
- sing-box outbound JSON。
- Mihomo/Clash proxy 片段。
- QR payload。

二维码预览使用本地 SVG 渲染，不调用任何外部二维码服务。

## 完全离线状态

新节点部署功能本身按完全离线目标设计：

- 不调用外部二维码 API。
- 部署弹窗不依赖 CDN。
- 节点应用配置时不远程下载 core。
- 节点使用本机已经安装的 core 可执行文件。
- 面板通过现有 WebSocket 通道下发渲染好的配置。

需要注意一个边界：

- flvxt2 项目中其它旧功能仍然包含在线能力，例如升级检查、安装链接、IP 探测、支付接口、文档链接等。
- 本次新增的节点部署功能自身已经按离线方式处理。

## 已新增或修改的文件

### 后端

- `go-backend/internal/store/model/model.go`
  - 新增部署相关模型。
- `go-backend/internal/store/repo/repository.go`
  - 注册自动迁移。
- `go-backend/internal/store/repo/repository_node_deploy.go`
  - 新增部署相关仓库操作。
- `go-backend/internal/http/handler/handler.go`
  - 注册部署路由。
- `go-backend/internal/http/handler/node_deploy.go`
  - 新增 TLS 模板、节点身份、入站、下发、回滚等 handler。

### 节点 Agent

- `go-gost/x/socket/websocket_reporter.go`
  - 新增 core 配置下发、读取、重启命令。
  - 新增离线 sing-box 校验。
  - 新增 systemd 服务创建和管理。

### 前端

- `vite-frontend/src/api/types.ts`
  - 新增部署相关 API 类型。
- `vite-frontend/src/api/index.ts`
  - 新增部署相关 API 方法。
- `vite-frontend/src/pages/node.tsx`
  - 新增部署弹窗状态和部署入口。
- `vite-frontend/src/pages/node/node-list-view.tsx`
  - 新增列表视图部署按钮。
- `vite-frontend/src/pages/node/node-deploy-modal.tsx`
  - 新增部署弹窗。
  - 新增离线二维码渲染器。

## 已完成验证

已经完成的检查：

- `go build ./cmd/paneld`
- `go test -count=1 ./socket`
- `go test ./internal/store/repo`
- `git diff --check`
- 确认部署弹窗没有引用外部二维码服务。

已知验证限制：

- 当前环境无法完成前端 build：
  - `node.exe` 执行时报 `Access is denied`。
  - `vite-frontend/node_modules` 不存在。
- `go test ./internal/http/handler` 当前失败：
  - 失败原因是现有测试仍在用旧的 `New(repo, secret)` 两参数构造函数。
  - 当前代码库中的 handler 构造函数需要三个参数。
  - 这个问题看起来不是本次节点部署功能引入的。

## 剩余工作

建议下一步继续做：

- 在具备前端依赖的环境中运行 TypeScript 和 Vite build。
- 用浏览器实际测试部署弹窗交互。
- 找一台已经本地安装 `sing-box` 的节点做真实下发测试。
- 补后端测试：
  - 入站参数 JSON 解析。
  - 默认名称生成。
  - 重名数字后缀。
  - Reality short ID 注入。
  - 配置渲染。
- 补节点 Agent 测试：
  - checksum 不匹配。
  - 非法 JSON。
  - 本机缺少 `sing-box`。
  - systemd unit 生成。
- 可以增加服务端 Reality keypair 生成功能。
- 决定 TLS 模板是否需要独立的全局管理页面，而不是只放在节点部署弹窗中。
- 根据客户端兼容性，继续完善各协议分享 URI 的参数生成。

## 当前实现状态

当前实现已经不是 Demo，而是节点部署功能的第一个真实版本：

- 面板可以定义 TLS/Reality 模板。
- 面板可以为每个节点绑定随机身份和凭据。
- 面板可以为节点创建已部署入站。
- 后端可以渲染完整 sing-box 配置版本。
- 面板可以把配置下发到节点。
- 节点可以应用配置，并通过本地 sing-box systemd 服务运行。
- 面板可以展示并复制已部署节点信息。

当前主要风险是前端 build 和浏览器运行时还没有在本环境完成验证，因为当前本地环境缺少前端依赖。
