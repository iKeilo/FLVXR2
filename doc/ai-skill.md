# AI Skill 使用指南

让大模型直接操作 FLVXT2 面板的技能包，支持 OpenCode、OpenClaw、Claude Code 等工具。

## 安装

### 方式 1: npm (推荐)

```bash
npm install -g @flvx/skill-api
```

postinstall 脚本会自动链接到 `~/.agents/skills/flvx-api/`。

### 方式 2: 手动链接

```bash
# 从 FLVXT2 源码
cd /path/to/flvxt2
mkdir -p ~/.agents/skills
ln -sf $(pwd)/skills/flvx-api ~/.agents/skills/

# 或从 GitHub
git clone https://github.com/iKeilo/flvxt2.git
cd flvxt2
ln -sf $(pwd)/skills/flvx-api ~/.agents/skills/
```

## 配置

设置环境变量：

```bash
export FLVX_BASE_URL="https://your-panel.example.com"
export FLVX_USERNAME="admin"
export FLVX_PASSWORD="your-password"
```

或者使用凭据文件：

```bash
mkdir -p ~/.flvxt2
cat > ~/.flvxt2/.env << 'EOF'
export FLVX_BASE_URL="https://panel.example.com"
export FLVX_USERNAME="admin"
export FLVX_PASSWORD="your-password"
EOF
chmod 600 ~/.flvxt2/.env
source ~/.flvxt2/.env
```

## 工具接入

### OpenCode

```bash
npm install -g @flvx/skill-api
export FLVX_BASE_URL="https://panel.example.com"
export FLVX_USERNAME="admin"
export FLVX_PASSWORD="your-password"
opencode
```

### OpenClaw

```bash
npm install -g @flvx/skill-api
mkdir -p ~/.openclaw/skills
ln -sf /path/to/flvxt2/skills/flvx-api ~/.openclaw/skills/flvx-api
```

### Claude Code

可以把 `skills/flvx-api/SKILL.md` 追加到你的 `CLAUDE.md`。

## 常用环境变量

- `FLVX_BASE_URL`: 面板地址
- `FLVX_USERNAME`: 用户名
- `FLVX_PASSWORD`: 密码

## 安全建议

- 使用 `~/.flvxt2/.env` 并设置 `chmod 600`
- 不要把密码写进历史命令
- Token 只在会话内临时保存
