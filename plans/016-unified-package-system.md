# 016-unified-package-system

## 目标

参考 ForwardX 的 Plan 设计，将现有 Product（充值/流量/时长）和 SubscriptionPackage 合并为统一的「套餐」模型，一个模型管所有。

## 设计要点（参考 ForwardX Plans.tsx）

- **一个模型搞定一切**：SubscriptionPackage = Plan，不再分 Product/SubscriptionPackage，没有 recharge/traffic/time 类型
- **非订阅的附加购买项（余额充值/流量包/时长续费）直接移除**，用户仅通过购买套餐获得所有权益（和 ForwardX 一致）
- **新增全局「商城开关」**，一键关闭商城（ForwardX 有 storeStatus）
- **新增「手动分配套餐」**，管理员直接给用户分配套餐（ForwardX 有 assignPlan）
- **套餐直接关联隧道分组**（保留现有 TunnelGroup 体系，不改隧道层）
- **新增 `product` 表到 `subscription_package` 的数据迁移脚本**

## 改动清单

### 1. 后端模型

- `model/product.go`：删除 `Product` struct，SubscriptionPackage 保持不变（无 Type/Value 字段）
- `model/order.go`：Order 的 ProductType 只保留 `"package"`，移除 `"recharge"/"traffic"/"time"` 相关逻辑

### 2. 后端 Repository

- `repository.go`：从 autoMigrate 移除 `&model.Product{}`
- `repository_product.go`：删除整个文件（CreateProduct/UpdateProduct/DeleteProduct/ListProducts/GetProduct/UpdateProductOrder）
- `repository_mutations.go` 删除：`IncreaseUserBalance`、`IncreaseUserFlow`、`ExtendUserExpiry` 保留（历史数据可能需要），但不再从订单交付中调用
- `repository_mutations.go` 清理：`DeductUserBalance` 保留（支付仍需要）

### 3. 后端 Handler

- `handler/product.go`：删除 `listProducts`/`createProduct`/`updateProduct`/`deleteProduct`/`updateProductOrder` 五个 handler
- `handler/order.go`：删除 `deliverProduct` 函数，`createOrder` 中移除 recharge/traffic/time 类型处理
- `handler/payment.go`：`completePayment` 中移除 `case "recharge"/"traffic"/"time"`，只保留 `"package"` 走 `DeliverPackageToUser`
- `handler/handler.go`：移除 `/api/v1/product/*` 路由注册

### 4. 后端新加功能

- **全局商城开关**：新增 `store_enabled` 配置（可用数据库配置表或现有机制），`listPackages` 检查
- **手动分配套餐**：新增 handler `assignPackageToUser`：管理员指定用户ID + 套餐ID → 调 `DeliverPackageToUser`
- 路由：`/api/v1/package/assign`

### 5. 数据迁移

- 将现有 `product` 表数据迁移到 `subscription_package` 表：
  - recharge → 创建套餐 type=recharge，但迁移后再也不用（或者直接丢弃，由管理员重新创建套餐）
  - 考虑到现有 product 数据量可能不大，建议简单提供一个 SQL 脚本，由管理员决定是否迁移

### 6. 前端

- `api/index.ts`：删除 `getProductList/createProduct/updateProduct/deleteProduct` API 方法
- `api/types.ts`：删除 `ProductApiItem` 类型，`SubscriptionPackageApiItem` 保持不变
- `admin-products.tsx`：**参照 ForwardX Plans.tsx 重写**
  - 页面标题：套餐管理
  - 顶部卡片：商城状态开关 + 套餐总数 + 订阅记录数
  - 套餐表格：名称、价格/有效期、资源（隧道分组数）、限制（端口/流量/规则/连接/IP/限速）、状态（启用/上架）
  - 新增/编辑弹窗：名称、说明、价格、有效期、流量、端口数、限速、规则数、连接数、IP 限制、排序 + 隧道分组多选 + 启用/商城可见开关
  - 新增「商城总开关」（在顶部卡片中）
  - 新增「手动分配」按钮 + 弹窗（选用户 + 选套餐）
- `shop.tsx`：移除 Product 加载（只加载 packages），套餐卡片展示保持不变
- `my-packages.tsx`：移除 Product/recharge/traffic/time 相关显示，只展示套餐详情 + 订单记录
- `pages/index.tsx`（登录页/注册页）：检查是否还引用 Product，移除
- `admin-payment.tsx`：检查是否引用 Product，移除
- `admin-orders.tsx`：检查订单列表中的 ProductType 显示，统一为「套餐」

### 7. 菜单名称

- 侧栏：`商品` → `套餐管理`（已改）
- `admin.tsx`/`h5.tsx`：对应更新路由文本

## 实施顺序

1. 后端：模型 + repository 清理（删除 Product 相关代码）
2. 后端：交付逻辑简化（删除 recharge/traffic/time case）
3. 后端：新增手动分配 + 商城开关
4. 数据迁移脚本
5. 前端：admin-products.tsx 参照 ForwardX 重写
6. 前端：shop/my-packages/index 清理 Product 引用
7. 验证：go build + go test + npm run build

## 风险

- 现有订单中 ProductType 为 recharge/traffic/time 的历史记录无法通过新代码交付 → `completePayment` 保留向后兼容判断
- 用户余额充值功能完全移除 → 需要确保 Admin 仍能手动调整余额（已有 `IncreaseUserBalance`）
- 影响范围大：3 个后端文件删除，1 个重写，5+ 前端文件修改
