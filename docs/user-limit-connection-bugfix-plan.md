# 用户创建限制项 Bug 排查与改造计划

## 背景

手机端测试反馈两个问题：

1. 新增/编辑用户时，`流量限制(GB)` 和 `规则数量` 输入框无法清空。删除到最后一位时会被自动恢复为 `1`，导致只能继续输入成 `1x`、`1xx`、`1xxx`。长按全选后覆盖输入可以绕过。
2. 创建用户时希望增加“总连接数限制”。当前连接数限制只在规则/转发里配置，用户删减规则后再新增规则时需要重复设置，不方便。

本次文档只做排查与方案，不进行功能代码修改。

## 结论摘要

- 问题 1 是前端受控数字输入的归一化时机不对，不是后端校验问题。
- `flow` 和 `num` 当前在输入过程中被立刻夹到 `1..99999`，移动端删除最后一位产生空字符串时，空字符串会被 `Number("") || 0` 变成 `0`，再被 `Math.max(..., 1)` 改回 `1`。
- 问题 2 后端数据模型已有用户级 `max_connections` 字段，套餐逻辑也会写入 `user.max_connections`，但用户创建/编辑接口、用户列表返回、前端表单、转发默认继承和核心下发之间没有贯通。
- 当前真正下发到 GOST/核心配置的连接数限制来自 `forward.max_connections`，位置在 `go-backend/internal/http/handler/control_plane.go`，尚未使用用户级兜底。

## 相关代码位置

### 前端用户表单

文件：`vite-frontend/src/pages/user.tsx`

- `userForm` 状态定义包含 `flow`、`num`，但没有 `maxConnections`。
- 新增用户默认值：
  - `flow: 1000`
  - `num: 10`
- 编辑用户时从 `user.flow`、`user.num` 回填。
- 提交时直接把 `userForm` 展开到 `createUser` / `updateUser` 请求。
- 弹窗里的两个数字输入：
  - `流量限制(GB)`：`value={userForm.flow.toString()}`
  - `规则数量`：`value={userForm.num.toString()}`

触发 bug 的核心逻辑：

```tsx
const value = Math.min(
  Math.max(Number(e.target.value) || 0, 1),
  99999,
);
```

当 `e.target.value === ""` 时：

```text
Number("") -> 0
0 || 0 -> 0
Math.max(0, 1) -> 1
```

因此输入框不能处于“临时空值”状态。

### 前端 API 类型

文件：`vite-frontend/src/api/types.ts`

- `UserApiItem` 当前包含 `flow`、`num`，没有显式 `maxConnections`。
- `UserMutationPayload` 当前包含 `flow`、`num`，没有显式 `maxConnections`。
- `ForwardApiItem` 已有 `maxConnections`。

### 后端用户创建/更新

文件：`go-backend/internal/http/handler/mutations.go`

- `userCreate` 读取：
  - `flow := asInt64(req["flow"], 100)`
  - `num := asInt(req["num"], 10)`
  - 未读取 `maxConnections`
- `userUpdate` 读取：
  - `flow := asInt64(req["flow"], 100)`
  - `num := asInt(req["num"], 10)`
  - 未读取 `maxConnections`
- 更新用户后会调用：
  - `h.repo.PropagateUserFlowToTunnels(id, flow, num, expTime, flowResetTime, status)`

### 后端用户模型

文件：`go-backend/internal/store/model/model.go`

`model.User` 已有这些字段：

- `SpeedLimit`
- `MaxRules`
- `MaxConnections`
- `MaxIPAccess`

其中 `MaxConnections` 对应数据库列：

```go
MaxConnections int `gorm:"column:max_connections;default:0"`
```

但当前人工创建/编辑用户用的是旧的 `num` 字段控制规则数，并没有暴露 `max_connections`。

### 后端用户仓库方法

文件：`go-backend/internal/store/repo/repository_mutations.go`

- `CreateUser(...)` 参数不包含 `maxConnections`。
- `UpdateUserWithPassword(...)` / `UpdateUserWithoutPassword(...)` 参数不包含 `maxConnections`。
- `PropagateUserFlowToTunnels(...)` 只同步 `flow`、`num`、`exp_time`、`flow_reset_time`、`status` 到 `user_tunnel`。

套餐相关逻辑已经会写：

- `user.max_connections`
- `user.max_ip_access`
- `user.speed_limit`

说明数据库层具备承载能力，但用户管理页面未接入。

### 规则/转发连接数下发

文件：`go-backend/internal/http/handler/control_plane.go`

当前核心配置生成只看转发规则自身：

```go
if forward.MaxConnections > 0 {
  meta["maxConnections"] = forward.MaxConnections
}
```

所以用户级连接数即使存在，也不会自动影响新建/已有规则。

## Bug 1 修复方案：允许输入框临时为空

### 推荐方向

把表单输入态和最终提交态分开。

数字输入框在用户输入过程中允许空字符串，只有在 `blur` 或提交时才做最终校验和归一化。

### 可选实现

#### 方案 A：最小改动，表单字段改为 `number | ""`

将 `userForm.flow`、`userForm.num` 类型改为：

```ts
flow: number | "";
num: number | "";
```

输入时：

- 空字符串保留为空。
- 非数字忽略或转成空。
- 超过上限时可实时裁剪到 `99999`。
- 不在输入时强制最小值 `1`。

失焦或提交时：

- `flow === ""` 时恢复默认值或提示必填。
- `num === ""` 时恢复默认值或提示必填。
- 最终提交前统一 clamp 到 `1..99999`。

优点：改动小，用户体验直观。

风险：需要检查所有读取 `userForm.flow.toString()` 的地方，避免空字符串类型引起 TS 报错。

#### 方案 B：保留数值状态，新增 draft 输入状态

新增：

```ts
const [userFormDraft, setUserFormDraft] = useState({
  flow: "1000",
  num: "10",
});
```

输入框绑定 draft 字符串，提交时再解析到 `userForm`。

优点：数字模型保持稳定。

风险：表单状态重复，新增/编辑/关闭弹窗时要同步 draft，维护成本稍高。

### 推荐选择

采用方案 A。当前用户表单只在本组件内使用，`number | ""` 的影响面可控。

### 校验规则

- `flow`：最终值范围 `1..99999`，`99999` 继续表示不限流量。
- `num`：最终值范围 `1..99999`。
- 空值处理建议：
  - 输入过程中允许空。
  - 点击提交时如果为空，提示“请填写流量限制/规则数量”。
  - 失焦时可以恢复到上一次合法值，避免用户误以为空值能提交。

## Bug 2 改造方案：用户级总连接数限制

### 目标语义

新增用户级“总连接数限制”：

- `0` 或留空表示不限制。
- 管理员创建/编辑用户时可以设置。
- 用户新建规则时，如果规则没有单独设置连接数，则默认继承用户级限制。
- 已有规则保持原值，避免后台修改用户后悄悄改变所有规则行为。
- 核心配置下发时需要有用户级兜底，保证没有单独规则限制时仍可生效。

### 推荐字段

复用现有数据库字段：

```text
user.max_connections
```

不新增数据库列。

### 后端改造点

#### 1. 用户列表返回

文件：`go-backend/internal/store/repo/repository.go`

在 `ListUsers()` 返回 map 中增加：

```go
"maxConnections": u.MaxConnections,
```

#### 2. 用户创建

文件：`go-backend/internal/http/handler/mutations.go`

在 `userCreate` 读取：

```go
maxConnections := asInt(req["maxConnections"], 0)
```

校验：

- 小于 0 则报错或归零。
- 建议最大值 `99999`，避免异常大值进入核心配置。

传入仓库创建方法。

#### 3. 用户更新

文件：`go-backend/internal/http/handler/mutations.go`

在 `userUpdate` 读取并保存 `maxConnections`。

注意：

- 管理员手动编辑用户时，应允许将连接数改回 `0`，表示不限。
- 不建议默认把用户级连接数批量覆盖到已有 `forward.max_connections`，否则会影响已经单独配置过的规则。

#### 4. 仓库方法

文件：`go-backend/internal/store/repo/repository_mutations.go`

更新签名：

- `CreateUser(..., maxConnections int, ...)`
- `UpdateUserWithPassword(..., maxConnections int, ...)`
- `UpdateUserWithoutPassword(..., maxConnections int, ...)`

并在 `Create` / `Updates` 中写入：

```go
"max_connections": maxConnections
```

需要同步更新所有调用点，例如注册逻辑中调用 `CreateUser` 的位置。

#### 5. 核心配置下发兜底

文件：`go-backend/internal/http/handler/control_plane.go`

当前只有：

```go
if forward.MaxConnections > 0 {
  meta["maxConnections"] = forward.MaxConnections
}
```

推荐变更为：

```text
effectiveMaxConnections = forward.max_connections
if effectiveMaxConnections <= 0:
  effectiveMaxConnections = user.max_connections
```

需要让生成 control plane 服务配置时能拿到对应用户的 `MaxConnections`。

可选做法：

- 在查询 `ForwardRecord` 时 join `user.max_connections`，增加 `UserMaxConnections` 字段。
- 或在生成配置时构建 `userID -> maxConnections` 缓存。

推荐：扩展 `ForwardRecord`，因为 forward 查询已经持有 `user_id`，语义清晰。

#### 6. 新建规则默认值

文件：`vite-frontend/src/pages/forward.tsx`

新增规则时，如果当前登录/选定用户有 `maxConnections > 0`：

- 表单 `maxConnections` 默认填用户级限制。
- 仍允许规则单独覆盖。
- 若留空或填 0，后端下发时仍可由用户级兜底生效。

这一点对管理员代用户创建规则、用户自己创建规则两种路径都要核对。

### 前端改造点

#### 1. 类型补充

文件：`vite-frontend/src/api/types.ts`

- `UserApiItem.maxConnections?: number`
- `UserMutationPayload.maxConnections?: number`

#### 2. 用户表单状态补充

文件：`vite-frontend/src/pages/user.tsx`

新增：

```ts
maxConnections: number | "";
```

默认值：

```ts
maxConnections: 0
```

编辑回填：

```ts
maxConnections: user.maxConnections ?? 0
```

#### 3. 用户创建/编辑弹窗增加输入项

位置建议放在 `规则数量` 旁边或其下方。

文案：

- Label：`总连接数限制`
- Placeholder：`不限制`
- Description：`留空或 0 表示不限制；规则未单独设置时使用此限制`

输入规则：

- 允许清空。
- `0` 表示不限。
- 建议范围 `0..99999`。

#### 4. 用户列表/卡片展示

可选展示，建议在用户详情或卡片限制区域增加一行：

```text
连接数：不限 / N
```

如果当前界面已经很拥挤，可先只在编辑弹窗展示，不影响功能。

## 兼容策略

- 老数据库：`user.max_connections` 已在 GORM model 中，AutoMigrate 会补列；无需新增 SQL migration。
- 老用户：默认 `0`，表示不限，不改变现有行为。
- 老规则：已有 `forward.max_connections` 保持优先级；用户级限制只作为兜底。
- 套餐用户：套餐已经写入 `user.max_connections`，新逻辑会让该字段真正参与下发。
- 免费/授权逻辑：本次功能属于用户限制能力，暂不绑定商业授权，除非后续明确要求。

## 建议优先级

1. 先修复数字输入无法清空，这是确定性前端 bug，影响手机端创建用户。
2. 再打通用户级 `maxConnections` 的 CRUD。
3. 最后处理核心配置兜底与新建规则默认继承。

## 测试计划

### 前端手工测试

- 手机端新增用户：
  - 删除 `流量限制(GB)` 最后一位后输入框能保持空。
  - 删除 `规则数量` 最后一位后输入框能保持空。
  - 空值提交能看到明确提示，或失焦恢复到合法值。
  - 输入 `123` 不会变成 `1123`。
  - 长按全选覆盖仍正常。
- 桌面端新增/编辑用户：
  - `flow=99999` 表示不限仍正常。
  - `num=1`、`num=99999` 正常。
  - `总连接数限制=0/空` 表示不限。
  - `总连接数限制=50` 能保存并编辑回填。

### 后端接口测试

- `POST /api/v1/user/create`
  - 带 `maxConnections: 50` 后数据库 `user.max_connections=50`。
  - 带 `maxConnections: 0` 后表示不限。
  - 负数应被拒绝或归零，按最终实现保持一致。
- `POST /api/v1/user/update`
  - 可以从 `50` 改成 `0`。
  - 不影响密码留空更新逻辑。
- `POST /api/v1/user/list`
  - 返回 `maxConnections`。

### 下发配置测试

- 创建用户级 `maxConnections=50`。
- 新建规则时不填规则级连接数。
- 下发后检查生成服务 metadata 中有：

```json
{
  "maxConnections": 50
}
```

- 如果规则级设置 `maxConnections=20`，应优先下发 `20`。
- 如果用户级和规则级都是 `0`，不应写入 `maxConnections`。

### 回归测试

- 用户删除/新增规则不应丢失用户级连接数。
- 套餐购买/后台发放套餐后，`pkg.maxConnections` 应能成为用户级默认限制。
- 规则管理原有 `ConnectionLimitField` 行为保持：留空代表规则级不限制。

## 待确认问题

1. 用户级总连接数限制是“该用户所有规则总和”还是“每条规则默认限制”？
   - 推荐第一阶段实现为“每条规则默认/兜底限制”，因为当前核心配置的 `maxConnections` 是服务级 metadata。
   - 如果要做“用户所有规则总连接数总和限制”，需要在 Agent/核心层增加跨服务聚合限制，目前不是简单字段下发能完成。
2. 管理员编辑用户级连接数后，是否要批量同步到已有规则？
   - 推荐默认不批量同步，只作为后续新规则默认值和下发兜底。
   - 可追加一个“同步到已有规则”的按钮，但不建议默认执行。
3. 空输入失焦时是恢复旧值还是保持空直到提交提示？
   - 推荐保持空，提交时提示；手机端体验最直观。
