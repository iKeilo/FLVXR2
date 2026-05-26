# 登录页右上角加监控链接 + 设置开关

## 背景

"耿璐"是登录页面(/)的内部称呼。要求在登录页右上角加一个监控链接按钮，点击后跳转到监控页面，但只显示节点监控，不显示隧道监控。同时设置页加一个开关控制显隐。

## 涉及文件及改动

### 1. 监控页面 src/pages/monitor.tsx — 支持 query param

现状: activeTab 从 localStorage("monitor-active-tab") 读取，默认 "nodes"；节点/隧道两个 tab 按钮始终显示。

改动:
- mount 时读取 window.location.search:
  - tab=nodes → 强制选中节点 tab（覆盖 localStorage）
  - hideTunnels=true → 隐藏"隧道"按钮，只显示节点监控
- 这样 /monitor?tab=nodes&hideTunnels=true 就是一个纯节点监控视图

### 2. Navbar src/components/navbar.tsx — 加监控入口按钮

现状: 右侧只有 H5 模式的设置齿轮图标。

改动: 在 NavbarContent justify="end" 区域（line 78）添加一个新图标按钮：
- 位置：在 WebView 设置齿轮之前
- 图标：使用监控/眼睛 SVG 图标
- 跳转：/monitor?tab=nodes&hideTunnels=true
- 条件渲染：仅当配置 login_monitor_link === "true" 时显示

配置读取方式：使用 getCachedConfig("login_monitor_link")，与现有 getCachedConfig("app_name") 同一模式。

### 3. 设置页 src/pages/settings.tsx — 加开关

现状: 显示设置区域只有"规则页面精简模式"开关。

改动: 在"显示设置"增加一项新开关：
- 配置 Key: login_monitor_link
- 标题: "登录页监控入口"
- 描述: "开启后，登录页右上角显示监控入口按钮"
- 状态管理、加载、保存逻辑完全复用 forwardCompactMode 的 pattern

### 4. 无需改动的部分

- 后端: 无改动，updateConfig / getConfigByName 可直接存取 login_monitor_link
- 数据库: config 表已支持任意 key-value
- Admin Layout 的 Header: 不在此处做改动，只在登录页 navbar 加

## 数据流

设置页开关 → updateConfig("login_monitor_link", "true")
       ↓
Navbar 加载时 getCachedConfig("login_monitor_link") → true
       ↓
Navbar 渲染监控图标按钮
       ↓
点击 → navigate("/monitor?tab=nodes&hideTunnels=true")
       ↓
MonitorPage 读取 query params → 只显示节点 tab，隐藏隧道按钮
