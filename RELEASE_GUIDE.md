# 快速发布脚本使用说明

## 🚀 手动触发发布的三种方法

### 方法1: GitHub网页界面 (推荐)
1. 访问: https://github.com/Last-emo-boy/infra-core/actions
2. 选择 "Release Management" 工作流
3. 点击 "Run workflow" 按钮
4. 选择发布类型后点击 "Run workflow"

### 方法2: 使用提供的PowerShell脚本
```powershell
# 在项目根目录运行
./scripts/release.ps1 -ReleaseType patch    # Bug修复: 0.1.0 → 0.1.1
./scripts/release.ps1 -ReleaseType minor    # 新功能: 0.1.0 → 0.2.0  
./scripts/release.ps1 -ReleaseType major    # 重大版本: 0.x.x → 1.0.0
./scripts/release.ps1 -ReleaseType beta     # Beta测试: 0.1.0 → 0.1.1-beta.0
./scripts/release.ps1 -ReleaseType alpha    # Alpha测试: 0.1.0 → 0.1.1-alpha.0
```

### 方法3: 直接使用GitHub CLI
```powershell
# 需要先安装GitHub CLI: winget install GitHub.cli
gh workflow run "Release Management" --ref main --field release_type=patch
```

## 📊 版本号累计示例

### 当前状态: 0.1.0

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

## 🎯 实际使用流程

### 日常开发
1. **开发功能** → 推送到main → 自动生成beta版本
2. **准备发布** → 手动触发对应类型的发布
3. **版本号自动累计** → 无需手动管理

### 发布决策
- **修复bug**: 选择 `patch`
- **新功能**: 选择 `minor` 
- **重大更新**: 选择 `major`
- **测试版本**: 选择 `alpha` 或 `beta`

## 📝 快速命令参考

```powershell
# 查看当前版本
Get-Content VERSION

# 查看发布历史
gh release list

# 查看工作流状态
gh run list --workflow="Release Management"

# 触发发布 (选择其中一个)
./scripts/release.ps1 -ReleaseType patch
./scripts/release.ps1 -ReleaseType minor
./scripts/release.ps1 -ReleaseType major
./scripts/release.ps1 -ReleaseType beta
```