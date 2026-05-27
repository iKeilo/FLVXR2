# FLVXT2 Frontend

这是 `FLVXT2` 的前端工程，基于 `rolldown-vite`、`TypeScript`、`Tailwind CSS v4` 和 `shadcn/ui`。

## 开发

```bash
pnpm install
pnpm run dev
```

## 构建

```bash
pnpm run build
```

## 说明

- 生产环境会从 `VITE_GITHUB_REPO` 读取仓库地址
- `VITE_APP_VERSION` 用于显示当前版本
- 如需调整发布链接，请同步更新 Release 工作流和安装脚本
