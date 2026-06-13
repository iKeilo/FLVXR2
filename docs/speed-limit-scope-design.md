# 限速规则作用域改造方案

## 背景

当前 FLVXR2 已经有“限速规则”管理页，并且底层转发下发已经支持 `forward.speed_id` 和 `user_tunnel.speed_id`。上一轮已经补上了 `user.speed_limit_id`，用于创建转发规则时继承用户默认限速。

这次需求继续扩展：

- 在“编辑隧道”中增加“隧道限速”。
- 位置放在“协议过滤”下面，文案为“选择规则限速的规则来限制”。
- 在“编辑限速规则”窗口中增加用户、隧道、规则、节点选择能力。
- 增加权限管控：普通用户不可见“规则”绑定选项卡。
- 限速规则增加“是否显示提示标签”选项；启用后，被限速的用户、隧道、规则显示限速大小标签，例如 `100Mbps`。
- 限速规则增加“竞技场”子开关；它只能属于单条限速规则，开启后，该限速规则命中的同一作用域对象共享同一个限速大小。
- 限速规则详情/编辑中的排序固定为：

```text
用户
---
隧道
---
规则
---
节点
```

核心目标是让限速规则不只是一个孤立数值，而是可以从多个管理入口绑定到不同对象。

## 现状

已有字段和能力：

- `speed_limit`：限速规则表，保存名称、速率、状态。
- `user.speed_limit_id`：用户默认限速规则。
- `user_tunnel.speed_id`：用户对某个隧道的限速规则。
- `forward.speed_id`：单条转发规则限速。
- 控制面下发当前优先级为：
  - 先看 `forward.speed_id`
  - 再看 `user_tunnel.speed_id`
  - 最后兼容旧字段 `forward.speed_limit_enabled/speed_limit`

缺失能力：

- `tunnel` 自身没有 `speed_id` 字段。
- `node` 自身没有限速规则绑定。
- 限速规则编辑页只能编辑名称、速度、状态，不能反向选择绑定对象。
- 现有“用户默认限速”是创建规则时继承，不是运行时强制覆盖已有规则。

## 设计原则

1. 限速规则本身仍然只保存策略：名称、速度、状态。
2. 绑定关系保存到被限制对象上，不把用户、隧道、规则、节点 ID 列表塞进 `speed_limit` 主表。
3. “编辑限速规则”里的选择器本质是反向批量维护绑定关系。
4. 运行时下发必须有清晰优先级，避免同一条流量链路被多个入口随机覆盖。
5. 默认模式不共享总带宽池；只有在某一条限速规则内开启“竞技场”后，该规则命中的同一作用域对象才共享同一个限速桶。
6. 竞技场第一版只做同一入口节点内共享，不做跨节点全局共享，避免引入中心化限速服务和额外延迟。
7. 竞技场不能做成系统总开关、限速模块总开关、隧道总开关或用户总开关，只能做成“限速规则”的子开关。

## 数据模型

### 1. Tunnel 增加限速字段

在 `model.Tunnel` 增加：

```go
SpeedID sql.NullInt64 `gorm:"column:speed_id"`
```

API 出参增加：

```json
{
  "speedId": 12
}
```

创建/编辑隧道接口支持：

```json
{
  "speedId": 12
}
```

其中 `null` 表示清空隧道限速。

### 2. Node 限速字段

节点限速需要谨慎，因为“节点”可能同时承担多条隧道、多个协议、多个入口/出口角色。建议第一版只做“节点默认限速绑定数据和 UI 展示”，不立即接入运行时下发，除非明确要限制整个节点上的所有转发服务。

建议字段：

```go
SpeedID sql.NullInt64 `gorm:"column:speed_id"`
```

第一版行为：

- 在限速规则编辑窗口可以绑定节点。
- 节点列表/节点详情可以展示该绑定。
- 不作为转发服务运行时优先级的一部分。

后续如果要启用节点运行时限速，需要单独确定：

- 限制入口节点上的所有监听服务；
- 限制出口节点上的所有出口连接；
- 或限制该节点参与的所有链路。

这三种语义不同，不能混在一起默认实现。

### 3. 规则绑定沿用 Forward

规则即转发规则，继续使用：

```go
Forward.SpeedID
```

不新增表。

### 4. 用户绑定沿用 User

用户默认限速继续使用：

```go
User.SpeedLimitID
```

注意：这是新规则创建时的默认策略，不主动批量覆盖用户已有规则。

### 5. 用户-隧道绑定沿用 UserTunnel

用户对某个隧道的权限限速继续使用：

```go
UserTunnel.SpeedID
```

它属于“用户在指定隧道上的默认限速”，不是本次“隧道自身限速”的替代。

### 6. SpeedLimit 增加提示标签开关

在 `model.SpeedLimit` 增加：

```go
ShowTag int `gorm:"column:show_tag;not null;default:0"`
```

含义：

- `0`：不显示提示标签。
- `1`：显示提示标签。

API 字段：

```json
{
  "showTag": 1
}
```

标签文本由后端或前端统一格式化：

```text
100Mbps
1Gbps
```

建议第一版统一用 Mbps 展示，即 `100Mbps`，避免单位换算导致用户误解。

### 7. SpeedLimit 增加竞技场子开关

竞技场模式是 `SpeedLimit` 的规则级属性，只能跟随具体限速规则保存。它不能出现在全局设置、系统设置、隧道管理全局配置、节点管理全局配置或规则列表工具栏中。

正确入口：

```text
限速规则 -> 新增/编辑某一条限速规则 -> 竞技场模式
```

错误入口：

```text
系统设置 -> 全局竞技场
限速规则列表 -> 统一开启竞技场
隧道管理 -> 全部隧道竞技场
用户管理 -> 全部用户竞技场
```

在 `model.SpeedLimit` 增加：

```go
ArenaMode int `gorm:"column:arena_mode;not null;default:0"`
```

含义：

- `0`：普通限速。多个对象使用同一个限速规则时，各自拥有自己的限速能力。
- `1`：该限速规则启用竞技场限速。命中该规则同一作用域的对象共享同一个限速桶。

API 字段：

```json
{
  "arenaMode": 1
}
```

说明示例：

```text
如果限速规则设置为 100Mbps，并绑定 A、B、C 三个用户：
- 普通模式：A/B/C 各自最高 100Mbps。
- 竞技场模式：A/B/C 合计共享 100Mbps。
```

如果绑定隧道：

```text
限速规则设置为 100Mbps，并绑定隧道 T1：
所有命中 T1 作用域的流量共享 100Mbps。
```

### 8. 竞技场作用域模型

竞技场不是简单把所有绑定对象混在一起限速，而是按“作用域”生成共享限速桶。第一版把作用域拆成三层：

```text
节点层级 node scope
隧道层级 tunnel scope
用户层级 user scope
```

为了避免和示例用户 A/B/C 混淆，文档中使用 `node/tunnel/user` 表示层级，使用 `User A/B/C` 表示具体用户。

核心规则：

- 未开启竞技场：每个被限速对象独立拥有该限速大小。
- 开启竞技场：命中同一个作用域的对象共享该限速大小。
- 同时选择多个层级时，不做并集，而是做交集。
- 空层级不参与匹配；非空层级必须全部命中。
- 如果所有层级都为空，视为无效配置，不允许保存或不参与下发。

典型场景：

```text
限速规则：100Mbps
竞技场：开启
绑定用户：User A / User B / User C

结果：
User A / User B / User C 在同一个入口节点内共享 100Mbps。
其它用户不受影响。
```

```text
限速规则：100Mbps
竞技场：开启
绑定隧道：Tunnel T1

结果：
所有使用 Tunnel T1 的匹配流量共享 100Mbps。
不使用 Tunnel T1 的流量不受影响。
```

```text
限速规则：100Mbps
竞技场：开启
绑定隧道：Tunnel T1
绑定用户：User A / User B / User C

结果：
只有 User A / User B / User C 使用 Tunnel T1 时共享 100Mbps。
其它用户使用 Tunnel T1 不受影响。
User A / User B / User C 使用其它隧道也不受影响。
```

如果同时绑定节点：

```text
限速规则：100Mbps
竞技场：开启
绑定节点：Node N1
绑定隧道：Tunnel T1
绑定用户：User A / User B / User C

结果：
只有入口节点为 Node N1，隧道为 Tunnel T1，用户属于 A/B/C 的流量共享 100Mbps。
```

第一版建议把节点层级解释为“入口节点”。原因是当前限速真正发生在 Agent/入口转发服务本地，入口节点最容易形成明确、低延迟、可验证的限速边界。

### 9. 规则层级与竞技场关系

“规则”即 `forward`，它比用户、隧道、节点都更具体。第一版建议规则绑定采用精确匹配：

- 如果 `forwardIds` 非空，优先使用规则精确匹配。
- 规则匹配可以独立参与竞技场，多个被选中的规则共享一个限速桶。
- 第一版不建议把 `forwardIds` 再和用户/隧道/节点做复杂交集，因为 forward 本身已经包含用户、隧道和入口节点信息。

推荐行为：

```text
绑定规则：Forward 1001 / Forward 1002
竞技场：开启

结果：
Forward 1001 和 Forward 1002 共享该限速大小。
```

如果后续确实需要“规则 + 用户 + 隧道”的强交集，可以在第二阶段加入，但第一版应保持规则绑定语义足够直观。

## 运行时优先级

建议最终运行时优先级如下：

```text
规则 forward.speed_id
  >
用户-隧道 user_tunnel.speed_id
  >
用户 user.speed_limit_id
  >
隧道 tunnel.speed_id
  >
无运行时限速
```

说明：

- 规则最具体，优先级最高。
- 用户-隧道比用户更具体。
- 用户默认限速用于用户维度。
- 隧道限速作为“该隧道下规则的默认限制”，低于用户维度。
- 节点限速第一版不进入运行时优先级。

这里和上一版已有行为略有调整：现在 `user.speed_limit_id` 不只在创建规则时继承，也可以作为运行时兜底。但为了兼容旧行为，建议第一版先只在 `forwardCreate` 中按以下顺序继承：

```text
显式 speedId
  >
用户默认 speedLimitId
  >
隧道 speedId
```

同时控制面运行时先不新增 `user.speed_limit_id` 和 `tunnel.speed_id` 兜底，避免影响旧规则。等 UI 和数据稳定后，再做“运行时兜底优先级”二阶段。

## 竞技场运行时设计

### 限速桶命名

普通模式下，每个对象应该生成独立限速器名称，避免多个用户误共享。例如：

```text
speed_forward_1001
speed_user_23
speed_tunnel_7_forward_1001
```

竞技场模式下，同一个作用域必须生成同一个限速器名称。建议格式：

```text
arena_speed_{speedLimitId}_{entryNodeId}_{scopeHash}
```

示例：

```text
arena_speed_12_node_5_9f3a21c0
```

其中 `scopeHash` 由排序后的绑定快照生成，避免同一个限速规则在不同作用域下发生桶名冲突：

```text
users=2,3,4;tunnels=7;nodes=5;forwards=
```

对应 Go 侧建议方法：

```go
func BuildArenaLimiterName(speedID int64, entryNodeID int64, scope SpeedLimitBindings) string
```

命名中必须包含 `entryNodeID`。这样即使同一个限速规则被下发到多个入口节点，也会在每个入口节点内各自共享，不会让用户误以为已经形成跨节点全局 100Mbps。

### 匹配上下文

控制面生成服务配置时，需要拿到以下上下文：

```go
type SpeedLimitMatchContext struct {
    ForwardID   int64
    UserID      int64
    TunnelID    int64
    EntryNodeID int64
}
```

字段来源：

- `ForwardID`：当前转发规则 ID。
- `UserID`：当前转发规则所属用户。
- `TunnelID`：当前转发规则所属隧道。
- `EntryNodeID`：当前监听端口所在节点，建议从 `forward_port.node_id` 或等价结构取得。

### 匹配规则

第一版使用以下判定：

```text
1. 如果 forwardIds 非空：
   - 当前 ForwardID 必须在 forwardIds 内。
   - 命中后进入规则级竞技场桶。

2. 如果 forwardIds 为空：
   - userIds 非空时，UserID 必须命中。
   - tunnelIds 非空时，TunnelID 必须命中。
   - nodeIds 非空时，EntryNodeID 必须命中。
   - 所有非空层级都命中才算命中。

3. 如果 userIds/tunnelIds/nodeIds/forwardIds 全部为空：
   - 视为无效作用域，不生成竞技场限速器。
```

伪代码：

```go
func ForwardMatchesArenaScope(ctx SpeedLimitMatchContext, scope SpeedLimitBindings) bool {
    if len(scope.ForwardIDs) > 0 {
        return contains(scope.ForwardIDs, ctx.ForwardID)
    }
    if len(scope.UserIDs) > 0 && !contains(scope.UserIDs, ctx.UserID) {
        return false
    }
    if len(scope.TunnelIDs) > 0 && !contains(scope.TunnelIDs, ctx.TunnelID) {
        return false
    }
    if len(scope.NodeIDs) > 0 && !contains(scope.NodeIDs, ctx.EntryNodeID) {
        return false
    }
    return len(scope.UserIDs)+len(scope.TunnelIDs)+len(scope.NodeIDs) > 0
}
```

### 跨节点限制

当前 Agent 的限速器是本地行为。第一版竞技场共享只保证“同一个入口节点内共享”，不保证多个入口节点之间全局共享。

例如：

```text
限速规则：100Mbps
竞技场：开启
绑定用户：User A / User B / User C
入口节点：Node 1 和 Node 2 同时承载这些用户
```

第一版实际效果：

```text
Node 1 内 A/B/C 共享 100Mbps
Node 2 内 A/B/C 共享 100Mbps
```

如果要做到全局只有一个 100Mbps，需要引入：

- Redis/etcd 这类中心化令牌桶；
- 或控制面按节点切分额度，例如每个入口节点分配 50Mbps；
- 或引入专门的全局流量调度器。

这些方案都会增加延迟、复杂度和故障面，不建议放进第一版。UI 需要在竞技场开关旁提示：

```text
竞技场共享限速在第一版按入口节点分别生效，跨多个入口节点不会合并成一个全局限速池。
```

## UI 设计

### 编辑隧道

位置：

- 放在“协议过滤”模块下面。
- 放在协议过滤说明 Alert 前或后均可。建议放在协议过滤说明 Alert 前，让它属于高级配置块的一部分。

控件：

```text
隧道限速
选择规则限速的规则来限制
[ Select: 不限速 / 限速规则 A / 限速规则 B / ... ]
```

交互：

- 选择限速规则后保存隧道。
- 选择“不限速”后提交 `speedId: null`。
- 如果该限速规则被删除，编辑窗口打开时自动显示“不限速”，并提示“原限速规则不存在，已自动清空”。

### 编辑限速规则

在原有名称、速度、状态下方新增“绑定对象”区域，按以下顺序展示：

```text
用户
---
隧道
---
规则
---
节点
```

每一段使用多选选择器或可搜索列表。

在基础信息区增加开关：

```text
显示限速标签
[开关]
```

说明文案：

```text
开启后，被该限速规则影响的用户、隧道和规则会显示限速大小标签。
```

在单条限速规则的基础信息区增加“竞技场模式”子开关：

```text
竞技场模式
[开关]
```

说明文案：

```text
开启后，同一作用域内共享该限速大小。多层级同时选择时取交集。例如选择隧道 T1 和用户 A/B/C，则只有 A/B/C 使用 T1 时共享限速。
```

如果开启竞技场，同时绑定了多个入口节点或没有绑定节点但实际涉及多个入口节点，需要显示二级提示：

```text
当前共享限速按入口节点分别生效，不会跨节点合并成一个全局限速池。
```

UI 位置约束：

- 只能出现在“新增限速规则”和“编辑限速规则”的弹窗/页面内。
- 不在左侧导航、页面顶部工具栏、限速规则列表批量操作、系统设置或隧道编辑窗口里提供竞技场总开关。
- 列表中可以显示该规则是否启用竞技场，但列表项上的快速操作也只能影响当前这一条规则。

权限管控：

- 管理员：可见并可编辑用户、隧道、规则、节点四个绑定段。
- 普通用户：不可见“规则”绑定段，不允许通过前端或接口批量修改 `forward.speed_id`。
- 如果普通用户访问接口提交 `bindings.forwardIds`，后端返回 403 或忽略该字段并返回明确提示。建议返回 403，避免“看似保存成功但实际无效”。

#### 用户

含义：

- 绑定到 `user.speed_limit_id`。
- 新建规则时，该用户默认继承该限速。

操作：

- 勾选用户：把用户 `speed_limit_id` 更新为当前限速规则 ID。
- 取消用户：如果用户当前 `speed_limit_id` 是当前规则，则清空；如果用户当前绑定的是其它规则，不动。
- 如果当前限速规则开启 `showTag`，用户列表/用户卡片显示限速标签，例如 `100Mbps`。

#### 隧道

含义：

- 绑定到 `tunnel.speed_id`。
- 后续新建转发规则时，如果规则没有显式限速且用户没有默认限速，则继承隧道限速。

操作：

- 勾选隧道：把隧道 `speed_id` 更新为当前限速规则 ID。
- 取消隧道：如果隧道当前 `speed_id` 是当前规则，则清空。
- 如果当前限速规则开启 `showTag`，隧道列表/隧道卡片显示限速标签，例如 `100Mbps`。

#### 规则

含义：

- 绑定到 `forward.speed_id`。
- 影响已有转发规则，保存后需要重新下发对应转发服务。

操作：

- 勾选规则：把规则 `speed_id` 更新为当前限速规则 ID，并同步下发。
- 取消规则：如果规则当前 `speed_id` 是当前规则，则清空，并同步下发。
- 如果当前限速规则开启 `showTag`，规则列表/规则卡片显示限速标签，例如 `100Mbps`。

风险：

- 这一步会影响正在运行的转发规则，保存前需要在窗口中显示将更新的规则数量。
- 该绑定段只对管理员显示。普通用户不可见，也不可通过接口修改。

#### 节点

含义：

- 第一版仅保存 `node.speed_id` 作为默认绑定/展示，不直接参与运行时限速。

操作：

- 勾选节点：把节点 `speed_id` 更新为当前限速规则 ID。
- 取消节点：如果节点当前 `speed_id` 是当前规则，则清空。

提示文案：

```text
节点限速当前仅作为节点默认绑定，不会直接限制该节点所有流量。
```

## API 设计

### 1. 扩展现有 speed-limit list

`POST /api/v1/speed-limit/list`

每条规则增加统计：

```json
{
  "id": 12,
  "name": "10Mbps",
  "speed": 10,
  "status": 1,
  "showTag": 1,
  "arenaMode": 1,
  "bindingCounts": {
    "users": 3,
    "tunnels": 2,
    "forwards": 10,
    "nodes": 1
  }
}
```

### 2. 新增详情接口

`POST /api/v1/speed-limit/detail`

请求：

```json
{ "id": 12 }
```

返回：

```json
{
  "id": 12,
  "name": "10Mbps",
  "speed": 10,
  "status": 1,
  "showTag": 1,
  "arenaMode": 1,
  "bindings": {
    "userIds": [2, 3],
    "tunnelIds": [7],
    "forwardIds": [1001, 1002],
    "nodeIds": [5]
  },
  "options": {
    "users": [],
    "tunnels": [],
    "forwards": [],
    "nodes": []
  }
}
```

### 3. 扩展 update 接口

`POST /api/v1/speed-limit/update`

请求增加：

```json
{
  "id": 12,
  "name": "10Mbps",
  "speed": 10,
  "status": 1,
  "showTag": 1,
  "arenaMode": 1,
  "bindings": {
    "userIds": [2, 3],
    "tunnelIds": [7],
    "forwardIds": [1001, 1002],
    "nodeIds": [5]
  }
}
```

后端保存步骤：

1. 更新限速规则基础信息。
2. 计算每类对象的差异。
3. 对新增绑定对象写入当前 speed limit ID。
4. 对取消绑定对象执行“仅当当前 speed ID 等于本规则 ID 时清空”。
5. 对发生变化的 forward 规则执行重新下发。
6. 如果操作者不是管理员，拒绝 `bindings.forwardIds` 变更。
7. 如果 `arenaMode=1` 且 `bindings` 全部为空，拒绝保存并提示“竞技场模式需要至少绑定一个作用域”。

## 标签展示规则

### 展示位置

开启 `showTag` 后，以下对象如果绑定了该限速规则，需要展示标签：

- 用户列表 / 用户卡片：读取 `user.speed_limit_id`。
- 隧道列表 / 隧道卡片：读取 `tunnel.speed_id`。
- 规则列表 / 规则卡片：读取 `forward.speed_id`。

第一版不建议在节点列表显示运行时限速标签，因为节点限速暂不参与运行时。可以显示为“默认限速”或放到节点详情里，避免误导用户认为节点所有流量已被限制。

### 标签样式

建议使用小尺寸 `Chip`：

```text
100Mbps
```

样式：

- 颜色：warning 或 primary。
- 尺寸：sm。
- 位置：对象名称右侧或卡片状态行。

### 标签数据

后端可以在列表接口直接返回：

```json
{
  "speedId": 12,
  "speedLimitName": "100M",
  "speedLimitSpeed": 100,
  "speedLimitShowTag": 1
}
```

前端根据 `speedLimitShowTag === 1` 显示：

```text
100Mbps
```

这样前端不需要额外查表，也能避免已删除限速规则时展示脏标签。

## 后端实施清单

1. `model.Tunnel` 增加 `SpeedID`。
2. 如确认节点绑定第一版落库，`model.Node` 增加 `SpeedID`。
3. `ListTunnels` / `tunnelGet` / `tunnelCreate` / `tunnelUpdate` 支持 `speedId`。
4. `speedLimitList` 增加绑定数量统计。
5. 新增 `speedLimitDetail`。
6. `speedLimitUpdate` 支持 `bindings`。
7. `speedLimitCreate` / `speedLimitUpdate` 支持 `showTag`。
8. 用户、隧道、规则列表接口补充限速标签展示字段。
9. 限速规则绑定接口增加管理员权限校验：普通用户不可修改规则绑定。
10. `speedLimitCreate` / `speedLimitUpdate` 支持 `arenaMode`，并且只作为单条限速规则的字段保存。
11. 竞技场模式保存校验：绑定作用域不能为空。
12. 控制面下发时，如果命中的限速规则 `arenaMode=1`，使用共享限速桶名称；否则保持现有独立限速器名称。
13. 新增 repo 方法：
   - `ListSpeedLimitBindingSnapshot(speedID int64)`
   - `ReplaceSpeedLimitBindings(speedID int64, bindings SpeedLimitBindings)`
   - `UpdateTunnelSpeedID(tunnelID int64, speedID interface{})`
   - `UpdateNodeSpeedID(nodeID int64, speedID interface{})`
   - `ResolveArenaSpeedLimitScope(speedID int64)`
   - `BuildArenaLimiterName(speedID int64, entryNodeID int64, scope SpeedLimitBindings)`
   - `ForwardMatchesArenaScope(ctx SpeedLimitMatchContext, scope SpeedLimitBindings)`
14. 对变更的 forward 调用现有同步逻辑，失败时回滚绑定变更。

## 前端实施清单

1. `api/types.ts` 增加 `speedId` 到 Tunnel 类型。
2. `tunnel.tsx`：
   - 加载限速规则列表。
   - 表单状态增加 `speedId`。
   - 编辑弹窗“协议过滤”下方增加限速选择。
3. `limit.tsx`：
   - 加载用户、隧道、规则、节点候选列表。
   - 编辑弹窗增加绑定对象区域。
   - 顺序固定为：用户、隧道、规则、节点。
   - 基础信息区增加“显示限速标签”开关。
   - 单条限速规则基础信息区增加“竞技场模式”子开关。
   - 竞技场模式下显示交集匹配说明和跨入口节点提示。
   - 不增加任何全局竞技场开关、批量竞技场开关或隧道级竞技场总开关。
   - 普通用户不可见“规则”绑定选项卡。
   - 保存时提交 `bindings`。
4. 展示绑定数量，方便管理员知道该规则正在影响多少对象。
5. 用户、隧道、规则列表在 `showTag` 启用时展示 `100Mbps` 标签。
6. 如果开启竞技场但未选择任何作用域，前端阻止保存并提示用户选择至少一个用户、隧道、规则或节点。

## 测试计划

后端：

- 创建隧道时保存 `speedId`。
- 更新隧道时清空 `speedId`。
- 编辑限速规则绑定用户、隧道、规则、节点。
- 取消绑定时不清除已经改绑到其它限速规则的对象。
- 修改规则绑定 forward 后会触发重新下发，失败时回滚。
- 普通用户提交 `bindings.forwardIds` 被拒绝。
- `showTag=1` 时列表接口返回标签展示字段；`showTag=0` 时不显示。
- `arenaMode=1` 且绑定为空时保存失败。
- `arenaMode=1` 且绑定用户 A/B/C 时，A/B/C 在同一入口节点生成相同 limiter name。
- `arenaMode=1` 且绑定隧道 T1 与用户 A/B/C 时，只有 A/B/C 使用 T1 的流量命中同一个 limiter name。
- 其它用户使用 T1 时不命中该竞技场限速。
- User A/B/C 使用其它隧道时不命中该竞技场限速。
- 同一个限速规则在不同作用域快照下生成不同 `scopeHash`，避免桶名冲突。
- 同一个竞技场规则下发到不同入口节点时，limiter name 包含不同 `entryNodeID`，确认不会跨节点共享。

前端：

- `npm run build`。
- 编辑隧道能选择/清空限速规则。
- 编辑限速规则弹窗中四类对象按正确顺序展示。
- 普通用户不可见“规则”绑定选项卡。
- 开启“显示限速标签”后，用户、隧道、规则列表显示 `100Mbps` 标签。
- 在单条限速规则内开启“竞技场模式”后显示共享限速说明。
- 页面不存在全局竞技场开关或批量竞技场开关。
- 同时选择隧道和用户时，界面提示为交集作用域，而不是并集作用域。
- 未选择任何绑定对象时开启竞技场并保存，前端显示明确错误。
- 保存后刷新列表，绑定数量正确变化。

## 分阶段建议

### 第一阶段：稳定落地

- 隧道增加 `speedId` 字段。
- 编辑隧道支持选择限速规则。
- 限速规则编辑窗口支持绑定用户、隧道、规则、节点。
- 限速规则支持 `showTag` 和 `arenaMode`。
- 新建规则时继承顺序：显式规则 > 用户默认 > 隧道默认。
- 节点绑定只落库和展示，不参与运行时。

### 第二阶段：运行时兜底

- 控制面加入更完整优先级：

```text
规则 > 用户-隧道 > 用户 > 隧道
```

- 对已有规则重新下发时自动应用新的兜底优先级。
- 竞技场模式进入控制面下发：命中同一作用域时使用共享限速桶。
- 优先实现入口节点内共享，不做跨节点全局共享。

### 第三阶段：节点限速语义

单独确定节点限速到底是：

- 限制入口节点监听服务；
- 限制出口节点连接；
- 限制节点参与的所有转发；
- 或仅作为模板默认项。

确定后再接入运行时，避免误伤多隧道复用节点。
