# 🚀 InfraCore 一键部署使用说明

## 快速部署到 Linux 服务器

### 方法一：一键部署（推荐）

在你的 Linux 服务器上运行：

```bash
curl -fsSL https://raw.githubusercontent.com/last-emo-boy/infra-core/main/quick-deploy.sh | sudo bash
```

### 方法二：下载脚本部署

```bash
# 下载部署脚本
wget https://raw.githubusercontent.com/last-emo-boy/infra-core/main/server-deploy.sh
chmod +x server-deploy.sh

# 运行部署
sudo ./server-deploy.sh --backup
```

## 🔧 GitHub Actions 自动部署

### 1. 设置 GitHub Secrets

在你的 GitHub 仓库中，进入 `Settings > Secrets and variables > Actions`，添加以下 secrets：

```
SERVER_HOST=your-server-ip
SERVER_USER=root
SERVER_SSH_KEY=your-private-ssh-key
GITHUB_TOKEN=自动提供
```

可选 secrets：
```
SERVER_PORT=22
SLACK_WEBHOOK=your-slack-webhook-url
```

### 2. 触发自动部署

- **推送到 main 分支**：自动部署到生产环境
- **推送到 develop 分支**：自动部署到测试环境
- **手动触发**：在 GitHub Actions 页面点击 "Run workflow"

## 📋 常用命令

部署完成后，在服务器上可以使用：

```bash
# 查看状态
sudo /opt/infra-core/current/server-deploy.sh --status

# 查看日志
sudo /opt/infra-core/current/server-deploy.sh --logs

# 重启服务
sudo /opt/infra-core/current/server-deploy.sh --restart

# 更新到最新版本
sudo /opt/infra-core/current/server-deploy.sh --update

# 回滚到上一版本
sudo /opt/infra-core/current/server-deploy.sh --rollback
```

## 🌐 访问应用

部署完成后，访问：

- **Web Console**: `http://your-server-ip:8082`
- **API Health Check**: `http://your-server-ip:8082/api/v1/health`

## 📞 获取帮助

如有问题，请查看：
- [完整部署文档](DEPLOYMENT.md)
- [项目 README](README.md)
- [GitHub Issues](https://github.com/last-emo-boy/infra-core/issues)

---

Made with ❤️ by [last-emo-boy](https://github.com/last-emo-boy)