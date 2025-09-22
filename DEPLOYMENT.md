# InfraCore 部署指南

本文档提供了 InfraCore 的完整部署指南，包括 GitHub Actions CI/CD 和 Linux 服务器部署。

## 🚀 快速开始

### 一键部署（推荐）

在你的 Linux 服务器上运行：

```bash
curl -fsSL https://raw.githubusercontent.com/last-emo-boy/infra-core/main/quick-deploy.sh | sudo bash
```

这个命令会：
- 自动安装所有依赖
- 下载最新代码
- 使用 Docker 部署应用
- 创建系统服务
- 进行健康检查

### 手动部署

如果你需要更多控制，可以下载完整的部署脚本：

```bash
# 下载部署脚本
wget https://raw.githubusercontent.com/last-emo-boy/infra-core/main/server-deploy.sh
chmod +x server-deploy.sh

# 查看帮助
sudo ./server-deploy.sh --help

# 部署到生产环境
sudo ./server-deploy.sh --backup

# 部署开发版本
sudo ./server-deploy.sh --branch develop --env staging
```

## 📋 部署选项

### 部署类型

1. **Docker 部署（推荐）**
   ```bash
   sudo ./server-deploy.sh --docker --backup
   ```
   - 使用 Docker 容器
   - 更好的隔离性
   - 更容易管理和扩展

2. **二进制部署**
   ```bash
   sudo ./server-deploy.sh --binary --backup
   ```
   - 直接运行可执行文件
   - 更少的资源占用
   - 更接近系统级别

### 环境配置

- **生产环境**: `--env production`（默认）
- **测试环境**: `--env staging`
- **开发环境**: `--env development`

### 常用命令

```bash
# 查看状态
sudo ./server-deploy.sh --status

# 查看日志
sudo ./server-deploy.sh --logs

# 重启服务
sudo ./server-deploy.sh --restart

# 停止服务
sudo ./server-deploy.sh --stop

# 启动服务
sudo ./server-deploy.sh --start

# 回滚到上一版本
sudo ./server-deploy.sh --rollback

# 快速更新（不完整重部署）
sudo ./server-deploy.sh --update
```

## 🔄 GitHub Actions CI/CD

### 工作流程说明

项目配置了完整的 GitHub Actions 工作流程，包括：

1. **测试阶段**
   - Go 代码测试和 lint
   - React 前端构建和类型检查
   - 代码覆盖率报告

2. **构建阶段**
   - 构建 Docker 镜像
   - 多平台支持（amd64, arm64）
   - 推送到 GitHub Container Registry

3. **安全扫描**
   - Trivy 漏洞扫描
   - 安全报告上传到 GitHub Security

4. **部署阶段**
   - 自动部署到 staging（develop 分支）
   - 自动部署到 production（main 分支）

5. **发布阶段**
   - 自动创建 GitHub Release
   - 多平台二进制文件构建
   - 部署包打包

### 分支策略

- `main` - 生产分支，触发生产部署
- `develop` - 开发分支，触发测试环境部署
- `feature/*` - 功能分支，只运行测试

### 环境变量配置

在 GitHub Repository Settings → Secrets and variables → Actions 中配置：

**必需的 Secrets:**
- `GITHUB_TOKEN` - 自动提供，用于 GHCR 认证

**可选的 Secrets（用于生产部署）:**
- `PRODUCTION_HOST` - 生产服务器 IP
- `PRODUCTION_USER` - 生产服务器用户名
- `PRODUCTION_SSH_KEY` - SSH 私钥
- `PRODUCTION_JWT_SECRET` - JWT 密钥

### 触发方式

1. **推送到 main/develop 分支**
   ```bash
   git push origin main
   ```

2. **创建 Pull Request**
   ```bash
   gh pr create --title "Feature: new awesome feature"
   ```

3. **创建 Release**
   ```bash
   gh release create v1.0.0 --title "Release v1.0.0" --notes "Release notes"
   ```

## 🔧 配置

### 环境变量

```bash
# 核心配置
export INFRA_CORE_ENV=production
export INFRA_CORE_JWT_SECRET=your-secret-key
export INFRA_CORE_CONSOLE_PORT=8082

# 数据库
export INFRA_CORE_DB_PATH=/var/lib/infra-core/database.db

# ACME/SSL
export INFRA_CORE_ACME_EMAIL=admin@example.com
export INFRA_CORE_ACME_ENABLED=true
```

### 配置文件

应用会按以下顺序加载配置：

1. `/etc/infra-core/config.yaml`
2. `./configs/{environment}.yaml`
3. 环境变量覆盖

### 端口配置

默认端口分配：
- **8082** - Console API
- **80/443** - Gate (Reverse Proxy)
- **5173** - Frontend Dev Server（开发时）

### 防火墙配置

```bash
# 允许 HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# 允许 Console API（如果需要外部访问）
sudo ufw allow 8082/tcp

# 启用防火墙
sudo ufw enable
```

## 📁 目录结构

部署后的目录结构：

```
/opt/infra-core/
├── current/              # 当前部署版本
├── previous/             # 上一个版本（用于回滚）
├── backups/              # 备份目录
│   ├── infra-core-backup-20241201-120000/
│   └── ...
└── tmp-*/               # 临时目录

/etc/infra-core/
├── config.yaml          # 主配置文件
└── environment          # 环境变量文件

/var/lib/infra-core/
├── database.db          # SQLite 数据库
└── uploads/             # 文件上传目录

/var/log/infra-core/
├── deploy.log           # 部署日志
├── console.log          # Console API 日志
└── gate.log             # Gate 日志
```

## 🔍 故障排除

### 常见问题

1. **服务无法启动**
   ```bash
   # 检查服务状态
   sudo ./server-deploy.sh --status
   
   # 查看详细日志
   sudo ./server-deploy.sh --logs
   
   # 检查配置
   sudo cat /etc/infra-core/config.yaml
   ```

2. **端口冲突**
   ```bash
   # 检查端口占用
   sudo netstat -tlnp | grep :8082
   
   # 修改配置文件中的端口
   sudo nano /etc/infra-core/config.yaml
   ```

3. **权限问题**
   ```bash
   # 检查文件权限
   ls -la /opt/infra-core/
   
   # 修复权限
   sudo chown -R infracore:infracore /opt/infra-core/
   sudo chown -R infracore:infracore /var/lib/infra-core/
   ```

4. **Docker 问题**
   ```bash
   # 检查 Docker 状态
   sudo systemctl status docker
   
   # 重启 Docker
   sudo systemctl restart docker
   
   # 清理 Docker 资源
   sudo docker system prune -f
   ```

### 日志查看

```bash
# 实时查看所有日志
sudo ./server-deploy.sh --logs

# 查看系统日志
sudo journalctl -u infra-core-* -f

# 查看 Docker 日志
sudo docker-compose -f /opt/infra-core/current/docker-compose.yml logs -f
```

### 健康检查

```bash
# API 健康检查
curl http://localhost:8082/api/v1/health

# 完整状态检查
sudo ./server-deploy.sh --status
```

## 🔄 更新和维护

### 定期更新

```bash
# 快速更新到最新版本
sudo ./server-deploy.sh --update

# 完整重新部署
sudo ./server-deploy.sh --backup
```

### 备份管理

```bash
# 手动创建备份
sudo ./server-deploy.sh --backup

# 查看备份列表
ls -la /opt/infra-core/backups/

# 恢复备份（手动）
sudo cp -r /opt/infra-core/backups/backup-name/current /opt/infra-core/
sudo ./server-deploy.sh --restart
```

### 监控建议

1. **设置系统监控**
   - CPU、内存、磁盘使用率
   - 网络连接状态
   - 服务运行状态

2. **设置应用监控**
   - API 响应时间
   - 错误率
   - 数据库连接

3. **设置日志监控**
   - 错误日志告警
   - 异常访问检测
   - 性能指标收集

### 安全建议

1. **定期更新**
   - 保持系统和应用最新
   - 定期检查安全漏洞

2. **访问控制**
   - 使用强密码
   - 配置防火墙规则
   - 限制 SSH 访问

3. **备份策略**
   - 定期自动备份
   - 异地备份存储
   - 定期恢复测试

## 📞 支持

如果遇到问题：

1. 查看本文档的故障排除部分
2. 检查 [GitHub Issues](https://github.com/last-emo-boy/infra-core/issues)
3. 创建新的 Issue 并提供详细信息：
   - 操作系统版本
   - 部署方式（Docker/Binary）
   - 错误日志
   - 复现步骤

---

Made with ❤️ by [last-emo-boy](https://github.com/last-emo-boy)