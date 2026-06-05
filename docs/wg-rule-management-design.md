# WG 隧道规则管理设计

状态：方案文档  
适用项目：FLVX / iKeilo/flvxt2  
目标：让规则管理支持 WG 隧道的出入口转发、内网转发和链式路径承载。

## 1. 背景

当前 FLVX 已经有两类核心能力：

1. 规则管理：负责端口转发、隧道转发等业务规则的创建、更新、诊断、暂停、恢复、删除。
2. WG Path：负责节点之间的 WireGuard 链式隧道，例如 `A -> B -> C` 或 `A -> C`。

现在缺少的是二者之间的管理关系。WG Path 目前更像一条节点间网络链路，但规则管理还不能明确使用这条链路去承载：

```text
公网入口 -> WG Path -> 出口节点 -> 内网目标
公网入口 -> WG Path -> 出口节点本机端口
入口节点内网 -> WG Path -> 远端内网
```

因此这次设计不是再新增一个独立页面，而是在“规则管理”里把 WG Path 当成一种可选承载通道，并补充 WG 专属的内网转发参数。

## 2. 功能定位

WG 隧道规则管理的定位：

```text
规则负责业务入口和目标
WG Path 负责节点间传输路径
Agent 负责本机 nftables / route / policy rule 下发
```

规则管理不直接维护 WireGuard peer，不直接编辑 WG 私钥，不直接创建 WG interface。那些仍归 WG 隧道管理负责。

规则管理只引用已经存在的 WG Path，并决定流量如何进入、如何离开、是否转发到内网目标。

## 3. 使用场景

### 3.1 WG 出入口转发

典型场景：

```text
用户访问 A:公网端口
  -> A 进入 WG Path
  -> B/C 出口节点
  -> 出口节点转发到目标地址
```

示例：

```text
A:30080 -> WG Path 12 -> C:192.168.10.20:80
```

这里 A 是入口节点，C 是 WG Path 的出口节点，`192.168.10.20:80` 是 C 可达的内网服务。

### 3.2 WG 内网转发

典型场景：

```text
A 所在内网的某个来源网段
  -> A 通过策略路由进入 WG Path
  -> C 访问远端内网网段
```

示例：

```text
source: 10.0.1.0/24
path: A -> B -> C
target: 192.168.50.0/24
```

这类规则不是单个端口映射，而是网段级路由/转发规则。第一版建议先做“目标 CIDR + 可选 SNAT”，不要直接做复杂 ACL。

### 3.3 WG 本机服务转发

典型场景：

```text
A:公网端口 -> WG Path -> C:127.0.0.1:服务端口
```

适合把远端节点本机服务通过 WG Path 暴露给入口节点。

### 3.4 普通规则和 WG 规则共存

同一个节点可能同时承载：

```text
普通 nftables 端口转发
普通 gost 隧道规则
WG Path 入口
WG Path 中转
WG Path 出口
WG 内网转发规则
```

因此所有规则都必须进入统一资源占用模型，不能只靠 interface 名或端口号做临时判断。

## 4. 不做范围

第一版不做：

1. 单条规则内混用 WG/GOST/nftables 多种中转协议。
2. 自动动态路由协议，例如 OSPF/BGP。
3. 全网 Mesh 自动编排。
4. 用户侧 VPN 客户端配置生成。
5. 多租户复杂 ACL。
6. 基于域名的内网路由。
7. WG Path 自动选路。

第一版只做：

```text
规则 -> 选择一条 WG Path -> 下发入口/出口/nftables/route
```

## 5. 页面设计

### 5.1 规则管理新增字段

在新增/编辑规则窗口中增加“承载方式”：

```text
承载方式：
  - 普通转发
  - 隧道转发
  - WG 隧道
```

选择 `WG 隧道` 后展示 WG 专属配置：

```text
WG Path：下拉选择
入口节点：只允许选择该 WG Path 的入口节点
出口节点：默认使用该 WG Path 的出口节点
转发类型：
  - 端口转发
  - 内网转发
  - 本机服务转发
```

### 5.2 WG 端口转发配置

字段：

```text
监听地址：默认 0.0.0.0 或 ::
监听端口：入口节点公网监听端口
目标地址：出口节点可达地址
目标端口：出口节点可达端口
协议：TCP / UDP / TCP+UDP
SNAT：默认启用
备注
```

行为：

```text
入口节点监听公网端口
流量进入 WG interface
出口节点 DNAT 到目标地址:端口
出口节点按需 SNAT，保证回程稳定
```

### 5.3 WG 内网转发配置

字段：

```text
源网段：可选，例如 10.0.1.0/24
目标网段：必填，例如 192.168.50.0/24
入口节点：WG Path 第一个节点
出口节点：WG Path 最后一个节点
SNAT：默认启用
允许转发协议：全部 / TCP / UDP / ICMP
备注
```

行为：

```text
入口节点添加 policy rule / route
中转节点只允许该 Path 的标记流量通过
出口节点添加 forward / masquerade
```

### 5.4 WG 本机服务转发配置

字段：

```text
监听地址
监听端口
目标地址：默认 127.0.0.1
目标端口
协议
```

行为：

```text
入口节点公网端口 -> WG Path -> 出口节点本机服务
```

## 6. 数据模型设计

### 6.1 扩展现有 forward 规则

建议优先复用现有 `forward` 表，不新建一套完全独立规则表。新增字段可以通过结构化扩展字段或明确列实现。

建议字段：

```text
transport_type        normal | tunnel | wg_path
path_id               WG Path ID
wg_rule_type          port | cidr | local
source_cidr           内网转发源网段，可空
target_cidr           内网转发目标网段，可空
snat_enabled          是否启用 SNAT
route_table           可选，默认从 path_id 派生
fwmark                可选，默认从 path_id + forward_id 派生
```

如果不希望立即改表，可以第一版使用 `extra_config` JSON：

```json
{
  "transportType": "wg_path",
  "pathId": 12,
  "wgRuleType": "cidr",
  "sourceCidr": "10.0.1.0/24",
  "targetCidr": "192.168.50.0/24",
  "snatEnabled": true
}
```

后续稳定后再迁移为正式列。

### 6.2 runtime resource 扩展

WG 规则必须进入 `node_runtime_resource`，避免和其它业务串线。

资源示例：

```text
owner_type=forward
owner_id=2001
resource_type=port
node_id=A
protocol=tcp
port=30080

owner_type=forward
owner_id=2001
resource_type=nft_chain
node_id=A
resource_key=flvx_forward_2001

owner_type=forward
owner_id=2001
resource_type=route_rule
node_id=A
resource_key=fwmark:0x120001

owner_type=forward
owner_id=2001
resource_type=cidr_route
node_id=C
resource_key=192.168.50.0/24
```

资源冲突必须在保存和部署前检查。

### 6.3 Path 与规则关系

一条 WG Path 可以被多条规则引用：

```text
Path 12: A -> B -> C
  Rule 2001: A:30080 -> C:192.168.10.20:80
  Rule 2002: A:30443 -> C:127.0.0.1:443
  Rule 2003: A:10.0.1.0/24 -> C:192.168.50.0/24
```

删除 WG Path 时需要提供规则处理策略：

```text
禁止删除：仍有规则引用时拒绝删除
级联删除：删除 Path 时删除引用规则
迁移规则：把规则迁移到另一条兼容 WG Path
```

第一版建议使用“禁止删除”，提示用户先处理规则。

## 7. 后端 API 设计

### 7.1 规则创建/更新

复用现有：

```text
POST /api/v1/forward/create
POST /api/v1/forward/update
```

请求增加：

```json
{
  "name": "wg-inner-lan",
  "transportType": "wg_path",
  "pathId": 12,
  "wgRuleType": "cidr",
  "sourceCidr": "10.0.1.0/24",
  "targetCidr": "192.168.50.0/24",
  "snatEnabled": true
}
```

校验规则：

1. `pathId` 必须存在。
2. Path `transport` 必须是 `wireguard`。
3. Path 状态建议为 `active`，否则保存允许但部署提示。
4. 入口节点必须是 Path 第一个节点。
5. 出口节点必须是 Path 最后一个节点。
6. `wgRuleType=port/local` 必须有监听端口和目标端口。
7. `wgRuleType=cidr` 必须有合法目标 CIDR。
8. 端口、CIDR、fwmark、nft chain 不得与现有资源冲突。

### 7.2 规则列表

复用：

```text
POST /api/v1/forward/list
```

返回中增加：

```json
{
  "transportType": "wg_path",
  "pathId": 12,
  "pathName": "A-B-C WG",
  "wgRuleType": "cidr",
  "pathRoute": "A -> B -> C"
}
```

前端卡片展示：

```text
规则名称
WG 隧道 / 内网转发
A -> B -> C
目标：192.168.50.0/24
```

### 7.3 规则部署

复用现有规则同步入口：

```text
syncForwardServices(...)
```

内部根据 `transportType` 分发：

```text
normal/tunnel -> 现有逻辑
wg_path       -> syncWGForwardRule(...)
```

新增内部方法：

```text
buildWGForwardPlan(forward)
applyWGForwardRule(forward)
removeWGForwardRule(forward)
diagnoseWGForwardRule(forward)
```

### 7.4 规则诊断

复用现有诊断 UI 和接口风格：

```text
POST /api/v1/forward/diagnose
POST /api/v1/forward/diagnose/stream
```

WG 规则诊断项目：

```text
入口节点监听检查
WG Path 状态检查
入口 -> 出口连通检查
出口 -> 目标地址检查
nftables 规则存在检查
路由表/fwmark 检查
```

## 8. Agent 命令设计

新增或扩展 Agent 命令：

```text
ApplyWGForwardRule
RemoveWGForwardRule
GetWGForwardRuleStatus
ProbeWGForwardRule
```

### 8.1 ApplyWGForwardRule

请求：

```json
{
  "forward_id": 2001,
  "path_id": 12,
  "interface": "wg-flvx-12",
  "rule_type": "port",
  "role": "entry",
  "listen": {
    "address": "0.0.0.0",
    "port": 30080,
    "protocol": "tcp"
  },
  "target": {
    "address": "192.168.10.20",
    "port": 80
  },
  "mark": "0x120001",
  "table": 112001,
  "snat": true
}
```

入口节点动作：

```text
创建 forward 专属 nft chain
匹配监听端口
打 fwmark
把流量导向 WG interface / route table
```

出口节点动作：

```text
匹配 path_id + forward_id comment
DNAT 到目标地址
按需 SNAT/MASQUERADE
允许 forward
```

中转节点动作：

```text
默认不下发业务规则
只依赖 WG Path 自身的转发能力
必要时检查 ip_forward 和 path chain
```

### 8.2 CIDR 内网转发

请求：

```json
{
  "forward_id": 2003,
  "path_id": 12,
  "rule_type": "cidr",
  "source_cidr": "10.0.1.0/24",
  "target_cidr": "192.168.50.0/24",
  "mark": "0x120003",
  "table": 112003,
  "snat": true
}
```

入口节点：

```text
匹配 source_cidr -> target_cidr
打 mark
走 WG Path route table
```

出口节点：

```text
允许 WG interface -> 目标网段
按需 SNAT
添加目标网段回程策略提示/检查
```

## 9. 防串线设计

### 9.1 所有规则必须带 comment

nftables 规则统一 comment：

```text
flvx:forward=2001
flvx:path=12:forward=2001
flvx:path=12:forward=2001:entry
flvx:path=12:forward=2001:exit
```

删除时只删除带对应 comment 的规则。

### 9.2 独立 chain

每条 forward 使用独立 chain：

```text
flvx_forward_2001
flvx_wg_forward_2001
```

不要把多条业务规则直接堆在 WG Path 主 chain 里。

WG Path 主 chain 只负责 Path 层转发，业务规则 chain 负责业务 DNAT/SNAT/mark。

### 9.3 独立 fwmark

建议派生：

```text
path_id = 12
forward_id = 2001
fwmark = 0x0C07D1
```

实际实现可以用稳定算法：

```text
base = 0x100000
fwmark = base + path_id * 4096 + forward_id % 4096
```

冲突时拒绝部署，不自动覆盖。

### 9.4 资源占用表是唯一事实源

部署前必须检查：

```text
端口是否被其它 forward/tunnel/path 占用
CIDR 是否与同节点其它 WG 规则冲突
fwmark 是否占用
route table 是否占用
nft chain 是否占用
```

Agent 侧也要二次检查本机实际状态，防止数据库和系统状态漂移。

## 10. 删除与变更策略

### 10.1 修改规则

修改 WG 规则时流程：

```text
生成新 plan
检查资源冲突
下发新规则
确认成功
清理旧规则
更新数据库
```

如果失败：

```text
保留旧规则
返回失败原因
不释放旧资源
```

### 10.2 删除规则

删除流程：

```text
RemoveWGForwardRule
释放 node_runtime_resource
删除 forward 记录
```

如果节点不在线：

```text
规则标记 pending_delete
节点上线后自动清理
允许管理员强制删除数据库记录，但保留风险提示
```

### 10.3 删除 WG Path

如果存在引用规则：

```text
拒绝删除
提示引用规则数量和名称
提供跳转到规则管理筛选
```

第一版不做自动迁移。

## 11. 权限与授权

WG 规则管理属于商业能力。

权限建议：

```text
管理员：
  创建/编辑/部署/删除所有 WG 规则

普通用户：
  只能管理自己名下规则
  只能选择自己有权限使用的 WG Path

未授权：
  WG 隧道承载方式置灰
  已存在 WG 规则只读展示
  API 返回 403
```

需要注意：前端置灰只是体验，后端必须强校验。

## 12. UI 交互方案

### 12.1 规则管理列表

卡片/表格新增显示：

```text
承载：WG 隧道
Path：A -> B -> C
类型：端口转发 / 内网转发 / 本机服务
目标：192.168.50.0/24 或 192.168.10.20:80
```

操作仍然使用现有：

```text
编辑
诊断
暂停/恢复
删除
```

### 12.2 新增规则窗口

布局：

```text
基础信息
  名称
  分组
  承载方式

WG 隧道
  WG Path
  链路预览 A -> B -> C
  当前状态 Active/Failed

入口
  监听地址
  监听端口
  协议

出口
  转发类型
  目标地址/目标端口
  目标 CIDR
  SNAT
```

### 12.3 诊断 UI

复用现有规则诊断 UI，结果分组：

```text
入口检查
WG Path 检查
出口检查
目标检查
```

不要再使用弹窗打印 JSON。

## 13. 实施阶段

### 阶段 1：模型和文档

1. 明确 forward 扩展字段。
2. 明确 WG 规则类型。
3. 明确资源占用模型。
4. 明确删除 Path 时的规则保护。

### 阶段 2：后端规则扩展

1. 扩展 `ForwardMutationPayload`。
2. 扩展后端 forward create/update/list。
3. 增加 WG Path 引用校验。
4. 增加资源冲突检查。
5. 增加 Path 删除前引用检查。

### 阶段 3：Agent 下发

1. 新增 `ApplyWGForwardRule`。
2. 新增 `RemoveWGForwardRule`。
3. 新增 `GetWGForwardRuleStatus`。
4. 增加 Linux nftables/iproute 实现和非 Linux stub。

### 阶段 4：前端接入

1. 规则新增/编辑窗口增加承载方式。
2. WG 模式显示 Path 选择和内网转发字段。
3. 规则列表展示 WG Path 信息。
4. 诊断 UI 复用现有规则诊断窗口。

### 阶段 5：测试

测试矩阵：

```text
A -> C 端口转发
A -> B -> C 端口转发
A -> C 内网 CIDR 转发
A -> B -> C 内网 CIDR 转发
同一个 B 同时承载多条 WG Path
同一个 B 同时承载普通转发和 WG 规则
删除规则清理 nftables/route
删除 Path 时阻止仍有规则引用
节点离线后的 pending_delete 清理
```

## 14. 最终建议

WG 不应该变成规则管理之外的另一套转发系统。最稳的做法是：

```text
WG Path 负责建网
规则管理负责业务流量
资源表负责防串线
Agent 负责本机执行
```

第一版优先实现：

```text
WG Path 选择
WG 端口转发
WG 内网 CIDR 转发
现有诊断 UI 复用
删除 Path 时引用保护
```

这样既能覆盖出入口内网转发需求，又不会把 WG Path、普通隧道、端口转发三套模型搅在一起。
