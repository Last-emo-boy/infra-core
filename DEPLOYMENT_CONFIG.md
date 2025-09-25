# InfraCore 部署配置指南

## 🎯 概述

InfraCore 部署脚本现在支持智能配置管理，让你可以轻松自定义部署参数，无需手动编辑配置文件。

## 🚀 快速开始

### 交互式部署（推荐）
```bash
# 带配置引导的完整部署
sudo ./server-deploy.sh --mirror

# 脚本会自动提示你设置：
# • 域名和管理员邮箱
# • 服务端口配置  
# • SSL/TLS 证书设置
# • 资源限制（内存/CPU）
# • 备份策略
```

### 非交互式部署
```bash
# 使用默认值快速部署
sudo NON_INTERACTIVE=true ./server-deploy.sh --mirror

# 或者预设环境变量
sudo CUSTOM_DOMAIN=myapp.com CUSTOM_EMAIL=admin@myapp.com ./server-deploy.sh --mirror
```

### 仅配置设置
```bash
# 只运行配置设置，不执行部署
sudo ./server-deploy.sh --configure
```

## ⚙️ 配置选项详解

### 🌐 基础设置

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|---------|------|
| 域名 | `CUSTOM_DOMAIN` | `console.example.com` | 你的服务域名 |
| 管理员邮箱 | `CUSTOM_EMAIL` | `admin@example.com` | SSL证书申请邮箱 |

### 🚪 端口配置

| 服务 | 环境变量 | 默认值 | 说明 |
|------|----------|---------|------|
| HTTP | `CUSTOM_HTTP_PORT` | `80` | Web界面HTTP端口 |
| HTTPS | `CUSTOM_HTTPS_PORT` | `443` | Web界面HTTPS端口 |
| API | `CUSTOM_API_PORT` | `8082` | API服务端口 |

### 🔒 安全设置

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|---------|------|
| JWT密钥 | `JWT_SECRET` | 自动生成 | API认证密钥 |
| SSL启用 | `CUSTOM_SSL_ENABLED` | `true` | 自动SSL证书 |

### 📊 资源限制

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|---------|------|
| 内存限制 | `CUSTOM_MEMORY_LIMIT` | `2g` | 最大内存使用 |
| CPU限制 | `CUSTOM_CPU_LIMIT` | `2` | 最大CPU核心数 |

### 💾 备份设置

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|---------|------|
| 自动备份 | `CUSTOM_BACKUP_ENABLED` | `true` | 启用自动备份 |
| 保留天数 | `CUSTOM_BACKUP_RETENTION` | `30` | 备份保留时间 |

## 📋 部署示例

### 1. 生产环境部署
```bash
sudo CUSTOM_DOMAIN=console.mycompany.com \
     CUSTOM_EMAIL=admin@mycompany.com \
     CUSTOM_MEMORY_LIMIT=4g \
     CUSTOM_CPU_LIMIT=4 \
     ./server-deploy.sh --mirror
```

### 2. 开发环境部署
```bash
sudo CUSTOM_HTTP_PORT=8080 \
     CUSTOM_API_PORT=3000 \
     CUSTOM_SSL_ENABLED=false \
     ./server-deploy.sh --env development
```

### 3. 高可用环境部署
```bash
sudo CUSTOM_MEMORY_LIMIT=8g \
     CUSTOM_CPU_LIMIT=8 \
     CUSTOM_BACKUP_ENABLED=true \
     CUSTOM_BACKUP_RETENTION=90 \
     ./server-deploy.sh --backup --mirror
```

## 🔧 配置管理

### 查看当前配置
```bash
sudo ./server-deploy.sh --status
```

### 重新配置
```bash
# 交互式重新配置
sudo ./server-deploy.sh --configure

# 应用新配置
sudo ./server-deploy.sh --restart
```

### 配置文件位置
- 生产配置：`/opt/infra-core/current/configs/production.yaml`
- 环境变量：`/etc/infra-core/environment`
- Docker环境：`/opt/infra-core/current/.env`

## 🎨 自定义配置模板

你也可以直接编辑配置文件：

```yaml
# /opt/infra-core/current/configs/production.yaml
gate:
  host: "0.0.0.0"
  ports:
    http: 80
    https: 443
  acme:
    email: "your-email@domain.com"
    enabled: true

console:
  host: "0.0.0.0" 
  port: 8082
  cors:
    origins: ["https://your-domain.com"]
```

## 📊 部署后验证

### 1. 服务状态检查
```bash
sudo ./server-deploy.sh --status
```

### 2. 健康检查
```bash
curl -f http://localhost:8082/api/v1/health
```

### 3. 查看日志
```bash
sudo ./server-deploy.sh --logs
```

### 4. 测试功能
```bash
sudo ./server-deploy.sh --test-api
```

## 🛠️ 故障排除

### 配置问题
```bash
# 运行配置诊断
sudo ./server-deploy.sh --troubleshoot

# 重置配置为默认值
sudo ./server-deploy.sh --configure
```

### 服务问题
```bash
# 重启服务
sudo ./server-deploy.sh --restart

# 查看详细日志
docker-compose logs -f infra-core
```

### 网络问题
```bash
# 检查端口占用
sudo netstat -tlnp | grep -E ':(80|443|8082)'

# 测试网络连接
curl -v http://localhost:8082/api/v1/health
```

## 🔄 升级和迁移

### 保留配置升级
```bash
# 升级时会自动保留现有配置
sudo ./server-deploy.sh --update --mirror
```

### 迁移配置
```bash
# 备份当前配置
sudo cp /opt/infra-core/current/configs/production.yaml ~/infra-core-config-backup.yaml

# 在新服务器上恢复
sudo cp ~/infra-core-config-backup.yaml /opt/infra-core/current/configs/production.yaml
sudo ./server-deploy.sh --restart
```

## 📚 高级用法

### 环境变量优先级
1. 命令行环境变量（最高优先级）
2. 配置文件设置
3. 脚本默认值（最低优先级）

### 批量部署
```bash
#!/bin/bash
# deploy-multiple.sh

servers=("server1.com" "server2.com" "server3.com")
domain_base="myapp"
email="admin@mycompany.com"

for i in "${!servers[@]}"; do
    server="${servers[$i]}"
    domain="${domain_base}$((i+1)).mycompany.com"
    
    ssh root@$server "
        export CUSTOM_DOMAIN='$domain'
        export CUSTOM_EMAIL='$email'
        export CUSTOM_API_PORT=$((8082+$i))
        ./server-deploy.sh --mirror --non-interactive
    "
done
```

### 配置模板化
```bash
# 创建配置模板
cat > deploy-config.env << EOF
CUSTOM_DOMAIN=console.mycompany.com
CUSTOM_EMAIL=admin@mycompany.com
CUSTOM_MEMORY_LIMIT=4g
CUSTOM_CPU_LIMIT=4
CUSTOM_BACKUP_ENABLED=true
CUSTOM_BACKUP_RETENTION=60
EOF

# 使用模板部署
source deploy-config.env
sudo ./server-deploy.sh --mirror --non-interactive
```

## 💡 最佳实践

1. **生产环境**：
   - 使用真实域名和SSL证书
   - 设置合适的资源限制
   - 启用自动备份
   - 定期检查服务状态

2. **开发环境**：
   - 可以禁用SSL节省时间
   - 使用非标准端口避免冲突
   - 较低的资源限制

3. **安全考虑**：
   - 定期更换JWT密钥
   - 使用强密码的管理员邮箱
   - 定期更新系统和依赖

4. **监控和维护**：
   - 设置日志轮转
   - 监控磁盘空间
   - 定期测试备份恢复

---

💡 **提示**: 如果遇到任何问题，可以运行 `sudo ./server-deploy.sh --troubleshoot` 获取详细的诊断信息。