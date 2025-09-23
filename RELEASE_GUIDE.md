# 统一CICD发布指南

## 🚀 新的统一CICD流程

### 工作流合并说明
已将原来的三个workflow（CI/CD、Deploy、Release Management）合并为一个统一的 **CICD Pipeline**，具备以下功能：

- ✅ **持续集成**: 测试、构建、代码质量检查
- ✅ **持续部署**: 自动部署到staging/production环境  
- ✅ **发布管理**: 自动和手动发布控制
- ✅ **容器化**: Docker镜像构建和推送
- ✅ **安全扫描**: 代码和容器安全检查

## 🤖 智能发布系统 (新增功能)

### 基于Commit消息自动检测发布类型
现在可以通过commit消息的关键词自动决定发布类型，无需手动选择！

#### 🚀 智能检测规则
| Commit模式 | 发布类型 | 版本变化 | 示例 |
|-----------|---------|---------|------|
| `feat!:`, `BREAKING CHANGE` | **major** | 0.1.0 → 1.0.0 | `feat!: 重构API` |
| `feat:`, `feature:`, `minor:` | **minor** | 0.1.0 → 0.2.0 | `feat: 新增功能` |
| `fix:`, `bugfix:`, `patch:` | **patch** | 0.1.0 → 0.1.1 | `fix: 修复bug` |
| `alpha:`, `experimental:` | **alpha** | 0.1.0 → 0.1.1-alpha.0 | `alpha: 实验功能` |
| `beta:`, `docs:`, `refactor:` | **beta** | 0.1.0 → 0.1.1-beta.0 | `docs: 更新文档` |
| `wip:`, `[skip release]` | **跳过** | 无变化 | `wip: 开发中` |

#### ✨ 智能发布使用方法
```bash
# 1. 使用智能commit消息
git commit -m "feat: 添加用户认证功能"    # 自动创建minor版本
git commit -m "fix: 解决登录bug"        # 自动创建patch版本  
git commit -m "feat!: 重构API接口"       # 自动创建major版本
git commit -m "docs: 更新文档 [skip release]"  # 跳过发布

# 2. 推送到main分支
git push origin main

# 3. 系统自动分析commit并创建对应版本！
```

详细说明请查看：[docs/SMART_RELEASE.md](docs/SMART_RELEASE.md)

## 🎛️ 手动发布方法 (备用选项)

### 方法1: GitHub网页界面 (推荐)
1. 访问: https://github.com/Last-emo-boy/infra-core/actions
2. 选择 **"CICD Pipeline"** 工作流
3. 点击 **"Run workflow"** 按钮
4. 配置参数：
   - **release_type**: 选择发布类型 (patch/minor/major/beta/alpha)
   - **deploy_environment**: 选择部署环境 (auto/staging/production/none)
5. 点击 **"Run workflow"** 执行

### 方法2: 使用PowerShell脚本 (已更新)
```powershell
# 在项目根目录运行
./scripts/release.ps1 -ReleaseType patch    # Bug修复: 0.1.0 → 0.1.1
./scripts/release.ps1 -ReleaseType minor    # 新功能: 0.1.0 → 0.2.0  
./scripts/release.ps1 -ReleaseType major    # 重大版本: 0.x.x → 1.0.0
./scripts/release.ps1 -ReleaseType beta     # Beta测试: 0.1.0 → 0.1.1-beta.0
./scripts/release.ps1 -ReleaseType alpha    # Alpha测试: 0.1.0 → 0.1.1-alpha.0
```

### 方法3: 直接使用GitHub CLI (已更新)
```powershell
# 发布新版本
gh workflow run "CICD Pipeline" --ref main --field release_type=patch --field deploy_environment=auto

# 仅部署不发布
gh workflow run "CICD Pipeline" --ref main --field release_type=none --field deploy_environment=production
```

## 📊 版本号累计规则 (已重置)

### 当前状态: 0.1.0 ✅

#### Patch发布 (修复bug)
```
0.1.0 → 0.1.1 → 0.1.2 → 0.1.3 ...
```

#### Minor发布 (新功能)  
```
0.1.3 → 0.2.0 → 0.3.0 → 0.4.0 ...
```

#### Major发布 (生产就绪)
```
0.x.x → 1.0.0 → 2.0.0 → 3.0.0 ...
```

#### 预发布版本
```
# Alpha版本
0.1.0 → 0.1.1-alpha.0 → 0.1.1-alpha.1

# Beta版本  
0.1.1-alpha.1 → 0.1.1-beta.0 → 0.1.1-beta.1

# 自动Beta (推送main分支)
每次推送 → 0.x.y-beta.z (自动递增)
```

## 🔄 新的发布流程说明

### 智能发布 (推荐)
1. **编写有意义的commit消息** → 系统自动检测版本类型
2. **推送到main分支** → 自动测试、构建、发布、部署
3. **无需手动操作** → 完全自动化流程

### 手动发布 (备用)
- **GitHub网页界面**: Actions → CICD Pipeline → Run workflow
- **PowerShell脚本**: `./scripts/release.ps1 -ReleaseType patch`
- **GitHub CLI**: `gh workflow run "CICD Pipeline"`

### 自动触发条件
| 触发条件 | 执行操作 |
|---------|---------|
| 推送到 `main` + 智能commit | 测试 → 构建 → **智能发布** → 部署production |
| 推送到 `develop` 分支 | 测试 → 构建 → 部署staging |
| 创建Pull Request | 仅测试和代码质量检查 |
| 手动触发 | 根据参数执行相应操作 |

### 部署策略
- **develop分支** → 自动部署到 **staging**
- **main分支** → 自动部署到 **production** 
- **手动触发** → 可选择任意环境

## 🎯 实际使用流程

### 日常开发
1. **开发功能** → 推送到develop → 自动部署staging测试
2. **合并到main** → 自动生成beta预发布 + 部署production
3. **准备发布** → 手动触发对应类型的正式发布

### 发布决策
- **修复bug**: 选择 `patch`
- **新功能**: 选择 `minor` 
- **重大更新**: 选择 `major`
- **测试版本**: 选择 `alpha` 或 `beta`

## 📝 快速命令参考

```powershell
# 查看当前版本和标签状态
Get-Content VERSION
git tag -l | Sort-Object

# 查看workflow状态
gh run list --workflow="CICD Pipeline"

# 触发不同类型的发布
./scripts/release.ps1 -ReleaseType patch      # 0.1.0 → 0.1.1
./scripts/release.ps1 -ReleaseType minor      # 0.1.0 → 0.2.0
./scripts/release.ps1 -ReleaseType major      # 0.1.0 → 1.0.0
./scripts/release.ps1 -ReleaseType beta       # 0.1.0 → 0.1.1-beta.0

# 查看发布历史
gh release list --limit 10
```

## ⚡ 主要改进

### 工作流统一
- ✅ **一个workflow搞定所有**: 测试、构建、发布、部署
- ✅ **智能触发**: 根据分支和事件自动选择操作
- ✅ **灵活配置**: 手动触发时可精确控制

### 版本控制重置
- ✅ **版本号已重置**: 从0.1.0开始
- ✅ **标签已清理**: 删除v4-v7，创建v0.1.0
- ✅ **配置已优化**: 支持0.x.x预发布格式

### 部署优化
- ✅ **环境分离**: staging和production独立部署
- ✅ **自动部署**: 根据分支自动选择环境
- ✅ **手动控制**: 支持强制指定部署环境