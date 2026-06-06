# WG 链式隧道功能设计

状态：方案确认稿  
日期：2026-06-05  
适用项目：FLVX / `iKeilo/FLVXR2`

## 1. 功能定位

本功能不只做单纯的 WireGuard 节点组网，而是作为 FLVX 的 **链式隧道 Path 管理能力**。

第一版以 WireGuard 作为新增 Transport，实现节点之间的低延迟内核级链式隧道：

```text
A -> B -> C
```

其中 B 可以同时承载其它业务：

```text
Path 1: A -> B -> C    transport=wireguard
Path 2: D -> B -> E    transport=gost
Path 3: F -> B -> G    transport=nftables
```

但第一版明确不支持单条 Path 内跨协议桥接：

```text
A --WG--> B --nftables--> C     不做
A --GOST--> B --WG--> C         不做
```

也就是说：

```text
同一条 Path 内 transport 必须一致
不同 Path 可以使用不同 transport
同一个节点可以同时承载多条不同 transport 的 Path
```

## 2. 设计目标

1. 使用 Linux kernel WireGuard 获得低延迟、低 CPU 占用。
2. 支持 A -> B -> C 多节点串联。
3. B 节点可同时作为多条隧道的入口、中继或出口。
4. 防止多 Path、多协议、多端口在同一节点上串线。
5. 复用现有 FLVX 控制链路，不另建独立控制面。
6. 允许后续扩展到 GOST Path、nftables Path、自动路径选择。

## 3. 现有代码基础

当前项目已有可复用能力：

| 能力 | 现有位置 | 用途 |
| --- | --- | --- |
| 节点命令下发 | `go-backend/internal/http/handler/control_plane.go` | `sendNodeCommand` / `sendNodeCommandWithTimeout` |
| Agent WebSocket 控制 | `go-gost/x/socket/websocket_reporter.go` | 接收面板命令并执行 |
| nftables 下发模型 | `go-gost/x/socket/nftables_handler.go` | Linux 下执行规则添加/删除 |
| 非 Linux stub 模式 | `go-gost/x/socket/nftables_handler_stub.go` | 保持跨平台编译 |
| 节点部署记录 | `go-backend/internal/http/handler/node_deploy.go` | 可参考部署/回滚/状态模型 |
| 节点部署 Repo | `go-backend/internal/store/repo/repository_node_deploy.go` | 可参考数据访问方式 |
| 节点页弹窗模式 | `vite-frontend/src/pages/node/node-deploy-modal.tsx` | 可参考前端交互 |

WG 功能应复用这条链路：

```text
面板生成 Path Runtime Plan
  -> sendNodeCommand
  -> Agent 执行 WireGuard/nftables/iproute 操作
  -> Agent 返回状态与 runtime hash
  -> 面板记录 expected/actual 状态
```

## 4. 第一版范围

### 4.1 必做

1. 节点 WireGuard 身份生成。
2. 手动创建 WireGuard Path。
3. 支持 2 节点与 3 节点 Path：

```text
A -> C
A -> B -> C
```

4. 每条 WG Path 使用独立 WireGuard interface。
5. 每条 WG Path 独立端口、路由表、fwmark、nftables chain。
6. Agent 支持 WireGuard 环境检查、部署、删除、状态查询。
7. 面板展示每条 Path 的 Segment 状态。
8. 节点详情展示当前承载的 Path 和角色。
9. 部署前检查端口、接口名、路由表、mark、配置文件冲突。

### 4.2 不做

1. 单条 Path 内跨协议桥接。
2. 自动最短路径。
3. DHT 去中心发现。
4. WebRTC/ICE 打洞。
5. Exit Node 客户端 VPN。
6. 多租户复杂 ACL。
7. 共享 `wg-flvx0` 的大规模 mesh 优化。

## 5. 核心概念

### 5.1 Path

Path 表示一条完整链式隧道。

```text
Path 1001
  transport: wireguard
  nodes: A -> B -> C
```

### 5.2 Segment

Segment 表示 Path 中相邻两个节点之间的一段连接。

```text
Path 1001:
  Segment 1: A -> B
  Segment 2: B -> C
```

### 5.3 Node Role

节点在 Path 内的角色：

```text
entry   入口
relay   中继
exit    出口
```

B 节点可同时在不同 Path 中承担不同角色。

### 5.4 Runtime Resource

Runtime Resource 是面板对节点本机资源的占用记录，用于防止串线和冲突。

资源包括：

```text
udp/tcp port
wireguard interface
route table
fwmark
nftables chain
config path
```

## 6. 隔离策略

### 6.1 每条 Path 独立 interface

第一版不共享 `wg-flvx0`，而是：

```text
wg-flvx-1001
wg-flvx-1002
wg-flvx-1003
```

优点：

1. AllowedIPs 不容易互相污染。
2. 路由表清晰。
3. 删除 Path 时可以按 interface 整体清理。
4. B 节点多 Path 并存时排障简单。

缺点：

1. 大规模 Path 会产生较多 interface。
2. 后续可优化为共享 interface + VRF/mark/table 隔离。

### 6.2 每条 Path 独立地址段

每条 Path 使用独立小网段。

2 节点：

```text
Path 1001: 10.88.1.0/30
A: 10.88.1.1
C: 10.88.1.2
```

3 节点：

```text
Path 1002: 10.88.2.0/29
A: 10.88.2.1
B: 10.88.2.2
C: 10.88.2.3
```

禁止第一版使用：

```text
AllowedIPs = 0.0.0.0/0
AllowedIPs = ::/0
```

### 6.3 每条 Path 独立 fwmark/table

建议分配规则：

```text
path_id = 1001
fwmark = 0x10010000
route_table = 101001
```

Segment 可追加低位：

```text
segment 1 mark = 0x10010001
segment 2 mark = 0x10010002
```

### 6.4 每条 Path 独立 nftables chain

命名：

```text
table inet flvx
chain flvx_wg_path_1001
```

规则必须带 comment：

```text
flvx:path=1001
flvx:path=1001:segment=1
```

删除时按 comment/interface/path_id 清理。

## 7. 数据模型

### 7.1 `wg_node_identity`

每个节点的 WG 身份。

```text
id
node_id
private_key_encrypted
public_key
default_listen_port
enabled
created_time
updated_time
```

说明：

1. 私钥必须加密保存。
2. 公钥可用于配置渲染和展示。
3. 节点重置身份时需要级联标记相关 Path 为待重新部署。

### 7.2 `path_tunnel`

Path 主表。

```text
id
name
transport
status
owner_user_id
created_by
remark
created_time
updated_time
```

`transport` 第一版仅支持：

```text
wireguard
```

预留：

```text
gost
nftables
```

### 7.3 `path_segment`

Path 分段表。

```text
id
path_id
sequence
from_node_id
to_node_id
transport
status
endpoint
listen_port
tunnel_ip_from
tunnel_ip_to
latest_handshake
latency_ms
rx_bytes
tx_bytes
created_time
updated_time
```

### 7.4 `node_runtime_resource`

节点资源占用表。

```text
id
node_id
owner_type
owner_id
resource_type
resource_key
protocol
port
status
created_time
updated_time
```

示例：

```text
node_id=2
owner_type=path
owner_id=1001
resource_type=wireguard_interface
resource_key=wg-flvx-1001

node_id=2
owner_type=path
owner_id=1001
resource_type=port
protocol=udp
port=51821

node_id=2
owner_type=path
owner_id=1001
resource_type=route_table
resource_key=101001
```

### 7.5 `path_runtime_version`

用于记录部署版本和回滚。

```text
id
path_id
version
expected_hash
actual_hash
status
message
created_time
```

## 8. 后端 API

新增文件：

```text
go-backend/internal/http/handler/wg_path.go
go-backend/internal/store/repo/repository_wg_path.go
```

新增路由：

```text
POST /api/v1/node/wg/identity
POST /api/v1/node/wg/identity/regenerate

POST /api/v1/path/list
POST /api/v1/path/detail
POST /api/v1/path/create
POST /api/v1/path/update
POST /api/v1/path/delete

POST /api/v1/path/apply
POST /api/v1/path/remove
POST /api/v1/path/status
POST /api/v1/path/probe
```

### 8.1 Path 创建流程

1. 校验节点在线状态。
2. 校验节点是否已生成 WG identity。
3. 校验节点顺序是否合法。
4. 分配 Path ID。
5. 分配接口名、端口、地址段、fwmark、route table。
6. 写入 `path_tunnel`、`path_segment`。
7. 写入 `node_runtime_resource`。
8. 状态置为 `pending`。

### 8.2 Path 部署流程

1. 查询 Path 和 Segment。
2. 生成每个节点的 Runtime Plan。
3. 计算 expected hash。
4. 使用 `sendNodeCommand(nodeID, "ApplyWireGuardPath", plan, ...)` 下发。
5. Agent 返回 actual hash 和状态。
6. 全部成功后 Path 状态置为 `active`。
7. 部分失败则置为 `degraded` 或 `failed`。

### 8.3 Path 删除流程

1. 状态置为 `removing`。
2. 对所有相关节点下发 `RemoveWireGuardPath`。
3. Agent 按 `path_id` 清理 interface、config、route、nftables。
4. 删除或释放 `node_runtime_resource`。
5. Path 状态置为 `removed` 或删除记录。

## 9. Agent 命令

新增 Linux 实现与 stub：

```text
go-gost/x/socket/wireguard_handler.go
go-gost/x/socket/wireguard_handler_stub.go
go-gost/x/socket/wireguard_interface.go
```

### 9.1 `CheckWireGuardSupport`

返回：

```json
{
  "supported": true,
  "kernel": true,
  "wgTool": true,
  "ipTool": true,
  "nftables": true,
  "message": ""
}
```

检查项：

```bash
ip link help
wg --version
nft --version
modprobe wireguard
```

注意：如果系统没有 `wireguard-tools`，第一版只提示错误，不自动安装。后续再接入“补全环境”。

### 9.2 `ApplyWireGuardPath`

请求：

```json
{
  "path_id": 1001,
  "interface": "wg-flvx-1001",
  "listen_port": 51821,
  "private_key": "...",
  "addresses": ["10.88.1.2/29"],
  "peers": [
    {
      "node_id": 1,
      "public_key": "...",
      "endpoint": "1.1.1.1:51821",
      "allowed_ips": ["10.88.1.1/32"],
      "persistent_keepalive": 25
    },
    {
      "node_id": 3,
      "public_key": "...",
      "endpoint": "3.3.3.3:51821",
      "allowed_ips": ["10.88.1.3/32"],
      "persistent_keepalive": 25
    }
  ],
  "routes": [
    {
      "dst": "10.88.1.0/29",
      "table": 101001
    }
  ],
  "nftables": {
    "enabled": true,
    "chain": "flvx_wg_path_1001",
    "snat": true
  },
  "expected_hash": "..."
}
```

执行步骤：

1. 检查接口名是否已存在。
2. 如果存在且属于同 `path_id`，先清理旧配置。
3. 写入 `/etc/flvx_agent/wg/path-1001.conf`。
4. 创建 WireGuard interface。
5. 设置地址、peer、路由、fwmark。
6. 写入 nftables forward/SNAT 规则。
7. 执行 `wg show` 验证。
8. 返回 actual hash。

### 9.3 `RemoveWireGuardPath`

请求：

```json
{
  "path_id": 1001,
  "interface": "wg-flvx-1001"
}
```

清理：

```text
ip link del wg-flvx-1001
nft delete rules with comment flvx:path=1001
ip rule/table cleanup
rm /etc/flvx_agent/wg/path-1001.conf
```

### 9.4 `GetWireGuardPathStatus`

返回：

```json
{
  "path_id": 1001,
  "interface": "wg-flvx-1001",
  "up": true,
  "peers": [
    {
      "public_key": "...",
      "latest_handshake": 12,
      "rx_bytes": 123456,
      "tx_bytes": 654321
    }
  ],
  "actual_hash": "..."
}
```

### 9.5 `ProbeWireGuardPath`

执行：

```bash
ping -c 3 -I wg-flvx-1001 10.88.1.3
```

返回：

```json
{
  "latency_ms": 18,
  "packet_loss": 0
}
```

## 10. WireGuard Runtime 细节

### 10.1 2 节点 Path

```text
A -> C
```

A：

```text
interface: wg-flvx-1001
address: 10.88.1.1/30
peer C allowed_ips: 10.88.1.2/32
```

C：

```text
interface: wg-flvx-1001
address: 10.88.1.2/30
peer A allowed_ips: 10.88.1.1/32
```

### 10.2 3 节点 Path

```text
A -> B -> C
```

A：

```text
peer B allowed_ips: 10.88.2.2/32, 10.88.2.3/32
route C via wg-flvx-1002
```

B：

```text
peer A allowed_ips: 10.88.2.1/32
peer C allowed_ips: 10.88.2.3/32
ip_forward=1
SNAT enabled
```

C：

```text
peer B allowed_ips: 10.88.2.2/32, 10.88.2.1/32
```

第一版默认 B 做 SNAT，保证回程稳定。

## 11. 防串线检查

部署前面板检查：

1. `node_runtime_resource` 是否已有同端口。
2. 是否已有同 interface。
3. 是否已有同 fwmark。
4. 是否已有同 route table。
5. Path 节点是否重复或非法。
6. Path 是否跨协议。

部署前 Agent 检查：

```bash
ip link show wg-flvx-1001
ss -lnup
ss -lntp
nft list ruleset
wg show
```

如果资源冲突：

```text
拒绝部署
返回冲突资源
不自动覆盖其它 Path 或现有业务
```

## 12. 状态模型

Path 状态：

```text
pending
applying
active
degraded
failed
removing
removed
```

Segment 状态：

```text
pending
active
handshake_timeout
probe_failed
failed
```

节点承载状态：

```text
entry
relay
exit
```

Path 状态计算：

```text
全部 Segment active -> active
部分 Segment failed -> degraded
关键 Segment failed -> failed
```

## 13. 前端设计

节点页二级菜单：

```text
卡片 / 分组 / TLS / WG隧道 / 新增
```

新增组件：

```text
vite-frontend/src/pages/node/wg-path-manager.tsx
vite-frontend/src/pages/node/wg-path-modal.tsx
```

WG 隧道管理弹窗：

```text
顶部：Path 数量、Active、Degraded、Failed
主体：Path 列表
右侧/弹窗：创建或编辑 Path
```

Path 卡片：

```text
名称：日本-香港-美国
Transport：WireGuard
Route：A -> B -> C
状态：Active
延迟：A-B 12ms / B-C 28ms
按钮：部署 / 状态 / 诊断 / 删除
```

节点详情增加：

```text
承载链路
  WG Path 1001: A -> B -> C，中继
  GOST Forward 2001: 入口
  nftables Forward 3001: 中继
```

## 14. 权限与授权

建议第一版将 WG Path 管理划入商业授权功能。

权限：

```text
管理员：创建、编辑、部署、删除所有 Path
普通用户：只查看自己拥有或被分配的 Path
节点部署者：可查看自己部署节点参与的 Path 状态
```

未授权：

```text
WG 隧道入口置灰
API 返回 403
Agent 不执行新增部署
```

## 15. 失败与回滚

`ApplyWireGuardPath` 必须尽量可重入。

部署顺序：

1. 所有节点做 precheck。
2. 写入 pending runtime version。
3. 按 exit -> relay -> entry 顺序部署。
4. 任一节点失败，已部署节点执行 rollback。
5. 状态写入 failed，保留错误详情。

回滚清理：

```text
RemoveWireGuardPath(path_id)
release node_runtime_resource
path_runtime_version status=rollback
```

## 16. 测试计划

### 16.1 单节点检查

1. `CheckWireGuardSupport`
2. `ApplyWireGuardPath` 创建 interface
3. `GetWireGuardPathStatus`
4. `RemoveWireGuardPath`

### 16.2 双节点 Path

```text
A -> C
```

验证：

```text
握手成功
ping 成功
rx/tx 增长
删除后 interface/rule/config 清理
```

### 16.3 三节点 Path

```text
A -> B -> C
```

验证：

```text
A-B 握手
B-C 握手
A ping C 成功
B 重启后重新部署
删除 Path 后 B 不影响其它 Path
```

### 16.4 B 节点复用

准备：

```text
Path 1001: A -> B -> C wireguard
Path 1002: D -> B -> E wireguard
已有 GOST/nftables 业务在 B 上运行
```

验证：

```text
端口不冲突
interface 不冲突
Path 1001 删除不影响 Path 1002
WG 删除不影响 GOST/nftables
统计不串线
```

## 17. 实施步骤

### 阶段 1：后端数据与 API

1. 增加 model。
2. 增加 repo。
3. 增加 API handler。
4. 注册路由。
5. 增加资源分配和冲突检测。

### 阶段 2：Agent WireGuard 执行器

1. 增加 Linux handler。
2. 增加 stub handler。
3. 增加 support check。
4. 增加 apply/remove/status/probe。
5. 增加清理和回滚。

### 阶段 3：前端 WG 管理

1. 节点页新增 WG 隧道入口。
2. 新增 Path 列表。
3. 新增创建/编辑弹窗。
4. 新增部署进度与状态展示。
5. 节点详情显示承载链路。

### 阶段 4：测试服验证

1. 双节点测试。
2. 三节点测试。
3. B 节点复用测试。
4. 删除/回滚测试。
5. Release 资产构建。

## 18. 后续扩展

1. 共享 `wg-flvx0` 模式。
2. Partial Mesh。
3. 自动路径选择。
4. 故障转移 Path。
5. Path ACL。
6. GOST Path 管理。
7. nftables Path 管理。
8. Path 流量计费。
9. MTU 自动探测。
10. 环境自动补全安装。

## 19. 最终结论

第一版应该做成：

```text
Path 管理 + WireGuard Transport
```

而不是单纯的 WG 配置页面。

执行策略：

```text
单条 Path 内协议统一
B 节点允许多 Path、多协议并存
每条 WG Path 独立 interface/table/mark/chain
面板负责资源锁和状态
Agent 负责 Linux 本机执行
默认 SNAT 保证回程稳定
```

这个方案能满足多节点串联、B 节点复用、低延迟和防串线需求，同时不会把第一版复杂度推到不可控。

