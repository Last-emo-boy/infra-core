# InfraCore 测试和工具脚本

这个目录包含了 InfraCore 项目的各种测试、诊断和维护脚本。

## 📁 脚本列表

### 🧪 测试脚本

| 脚本 | 用途 | 使用方法 |
|------|------|----------|
| `test-installation.sh` | 验证 InfraCore 安装是否成功 | `./scripts/test-installation.sh` |
| `test-api.sh` | 测试 API 接口功能 | `./scripts/test-api.sh [base_url]` |
| `benchmark.sh` | 性能基准测试 | `./scripts/benchmark.sh [base_url]` |

### 🔧 诊断脚本

| 脚本 | 用途 | 使用方法 |
|------|------|----------|
| `troubleshoot.sh` | 故障排除和诊断 | `./scripts/troubleshoot.sh [option]` |

### 📄 配置文件

| 文件 | 用途 |
|------|------|
| `curl-format.txt` | curl 性能测试格式化模板 |

## 🚀 快速使用指南

### 1. 安装后验证

```bash
# 运行完整的安装验证
sudo ./server-deploy.sh --test-install

# 或直接运行脚本
chmod +x scripts/test-installation.sh
./scripts/test-installation.sh
```

### 2. API 功能测试

```bash
# 测试本地 API
sudo ./server-deploy.sh --test-api

# 测试远程 API
./scripts/test-api.sh http://your-domain.com:8082
```

### 3. 故障排除

```bash
# 运行完整诊断
sudo ./server-deploy.sh --troubleshoot

# 只检查特定问题
./scripts/troubleshoot.sh docker    # 只检查 Docker
./scripts/troubleshoot.sh network   # 只检查网络
./scripts/troubleshoot.sh logs      # 只查看日志
```

### 4. 性能测试

```bash
# 基本性能测试
./scripts/benchmark.sh

# 测试远程服务器
./scripts/benchmark.sh http://your-domain.com:8082
```

## 📊 测试报告示例

### 安装验证成功示例：
```
🧪 InfraCore 安装验证测试
================================
🔍 Phase 1: Environment Check
[TEST] Testing: Docker installed
✅ Docker installed: PASSED

🐳 Phase 2: Docker Services Check
✅ InfraCore services are running

🌐 Phase 3: Network Connectivity
✅ API Health Check: PASSED (HTTP 200)
✅ Web Interface: PASSED (HTTP 200)

🎉 Installation verification: EXCELLENT (95%)
Your InfraCore installation is working perfectly!
```

### API 测试成功示例：
```
🧪 InfraCore API 测试
=====================
🏥 Phase 1: Health Checks
✅ Health Check: PASSED (HTTP 200)

🔐 Phase 2: Authentication
✅ Admin Login: PASSED (HTTP 200)
✅ Token Validation: PASSED (HTTP 200)

📋 API Test Report
==================
Total Tests: 15
Passed: 14
Failed: 1

🎉 API Test Result: EXCELLENT (93%)
All critical API functions are working correctly!
```

## 🔧 故障排除指南

### 常见问题解决：

1. **端口占用**
   ```bash
   ./scripts/troubleshoot.sh network
   ```

2. **Docker 服务异常**
   ```bash
   ./scripts/troubleshoot.sh docker
   ```

3. **生成详细报告**
   ```bash
   ./scripts/troubleshoot.sh report
   ```

## 📝 脚本开发说明

### 添加新测试脚本：

1. 创建脚本文件：`scripts/your-test.sh`
2. 设置执行权限：`chmod +x scripts/your-test.sh`
3. 遵循现有脚本的日志格式
4. 更新此 README 文件

### 脚本规范：

- 使用 `set -euo pipefail` 严格模式
- 定义统一的颜色和日志函数
- 提供帮助信息 (`--help`)
- 包含错误处理和退出码
- 添加测试统计和报告

## 🤝 贡献

欢迎提交新的测试脚本和改进建议！请确保：

- 脚本具有良好的错误处理
- 包含详细的使用说明
- 遵循项目的代码规范
- 添加适当的测试用例

---

**注意**: 某些脚本可能需要 root 权限才能运行，特别是涉及 Docker 和系统资源检查的脚本。