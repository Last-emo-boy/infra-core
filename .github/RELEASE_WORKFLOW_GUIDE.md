# InfraCore 版本发布工作流指南

## 🚀 概述

InfraCore 使用基于 GitHub Actions 的智能版本发布系统，支持：
- **智能检测**: 基于 commit 消息自动确定发布类型
- **手动控制**: 通过 workflow_dispatch 手动触发指定类型的发布
- **标签管理**: 自动处理版本标签冲突和同步问题
- **资产构建**: 自动构建和上传发布资产

## 🔧 修复说明 - 标签冲突问题

### 问题描述
之前的发布流程中，`standard-version` 工具在 GitHub Actions 环境中容易遇到标签冲突：
```
fatal: tag 'v0.1.1-beta.2' already exists
Command failed: git tag -a v0.1.1-beta.2 -m chore(release): 0.1.1-beta.2
```

### 根本原因
1. **并发执行**: 多个 workflow 同时运行可能创建相同的标签
2. **缓存延迟**: GitHub Actions 环境中标签同步存在延迟
3. **重试机制缺失**: 失败后没有自动重试和清理机制
4. **状态不一致**: 本地和远程仓库标签状态不同步

### 修复方案

#### 1. 智能标签冲突检测与清理
```yaml
- name: Sync tags and clean up conflicts
  run: |
    echo "🔄 Syncing tags from remote..."
    git fetch --tags --force
    
    # 检查并清理可能冲突的标签
    if [ -f VERSION ]; then
      CURRENT_VERSION=$(cat VERSION)
      NEXT_BETA_TAG="v${CURRENT_VERSION%%-*}-beta.$((${CURRENT_VERSION##*beta.} + 1))"
      
      if git tag -l | grep -q "^$NEXT_BETA_TAG$"; then
        echo "⚠️ Tag $NEXT_BETA_TAG already exists, removing it..."
        git tag -d "$NEXT_BETA_TAG" || true
        git push origin ":refs/tags/$NEXT_BETA_TAG" || true
        sleep 3
      fi
    fi
```

#### 2. 重试机制
```yaml
retry_release() {
  local attempt=1
  local max_attempts=3
  local cmd="$1"
  
  while [ $attempt -le $max_attempts ]; do
    if eval "$cmd"; then
      echo "✅ Release creation succeeded on attempt $attempt"
      return 0
    else
      echo "❌ Attempt $attempt failed, cleaning up..."
      # 清理和重试逻辑
      attempt=$((attempt + 1))
    fi
  done
}
```

#### 3. 推送重试和同步
```yaml
retry_push() {
  # 带有远程同步的推送重试机制
  # 自动处理 rebase 和冲突解决
}
```

## 📋 发布类型说明

### 自动检测规则 (Smart Release)

| Commit 消息模式 | 发布类型 | 示例版本变化 | 说明 |
|----------------|----------|-------------|------|
| `feat!:`, `fix!:`, `BREAKING CHANGE` | `major` | 0.1.0 → 1.0.0 | 重大更新，包含破坏性变更 |
| `feat:`, `feature:` | `minor` | 0.1.0 → 0.2.0 | 新功能添加 |
| `fix:`, `bugfix:` | `patch` | 0.1.0 → 0.1.1 | Bug 修复 |
| `alpha:` | `alpha` | 0.1.0 → 0.1.1-alpha.0 | Alpha 测试版本 |
| `beta:` | `beta` | 0.1.0 → 0.1.1-beta.0 | Beta 测试版本 |
| `docs:`, `style:`, `refactor:` | `beta` | 0.1.0 → 0.1.1-beta.0 | 文档/重构等变更 |
| `wip:`, `draft:`, `temp:` | 跳过 | - | 工作进行中，不发布 |
| `no-release`, `skip-release` | 跳过 | - | 明确跳过发布 |

### 手动发布选项 (Manual Release)

通过 GitHub Actions 的 "Run workflow" 按钮手动触发：

| 选项 | 说明 | 用途 |
|------|------|------|
| `patch` | 补丁版本 | 紧急 bug 修复发布 |
| `minor` | 次要版本 | 功能发布 |
| `major` | 主要版本 | 重大版本发布（1.0.0） |
| `prerelease` | 预发布 | 自动检测预发布类型 |
| `beta` | Beta 版本 | Beta 测试发布 |
| `alpha` | Alpha 版本 | Alpha 测试发布 |
| `none` | 不发布 | 仅运行测试和构建 |

## 🎯 最佳实践

### 1. Commit 消息规范
使用 [Conventional Commits](https://conventionalcommits.org/) 格式：

```bash
# 功能添加
git commit -m "feat: add user authentication system"

# Bug 修复  
git commit -m "fix: resolve login timeout issue"

# 破坏性变更
git commit -m "feat!: redesign API endpoints

BREAKING CHANGE: API endpoints have been redesigned"

# 跳过发布
git commit -m "docs: update README [skip-release]"
```

### 2. 版本策略
- **开发阶段**: 使用 `beta` 预发布版本 (0.x.x-beta.x)
- **测试阶段**: 使用 `alpha` 版本进行实验性功能
- **生产就绪**: 使用 `patch`/`minor`/`major` 正式版本
- **紧急修复**: 使用手动 `patch` 发布

### 3. 分支管理
- `main` 分支: 自动触发智能发布
- `develop` 分支: 部署到 staging 环境
- feature 分支: 通过 PR 合并到 `develop`

### 4. 发布检查清单
在发布前确保：
- [ ] 所有测试通过 (Coverage ≥ 60%)
- [ ] 代码质量检查通过
- [ ] 安全扫描无严重问题
- [ ] 文档更新完整
- [ ] 变更日志生成正确

## 🔍 故障排除

### 标签冲突问题

**症状**: `fatal: tag 'vX.X.X' already exists`

**解决方案**:
1. **自动处理**: 新的 workflow 会自动检测和清理冲突标签
2. **手动清理**: 如果仍有问题，可以手动删除冲突标签：
   ```bash
   # 删除本地标签
   git tag -d v0.1.1-beta.2
   
   # 删除远程标签
   git push origin :refs/tags/v0.1.1-beta.2
   
   # 重新触发 workflow
   ```

### 推送失败问题

**症状**: `git push` 失败或超时

**解决方案**:
1. **自动重试**: workflow 包含 3 次重试机制
2. **同步检查**: 自动检测远程变更并执行 rebase
3. **手动同步**: 如需手动处理：
   ```bash
   git fetch origin main
   git rebase origin/main
   git push --follow-tags origin main
   ```

### 版本不一致问题

**症状**: VERSION 文件和 Git 标签不匹配

**解决方案**:
1. **自动验证**: workflow 会检查版本一致性
2. **手动修复**:
   ```bash
   # 检查当前状态
   cat VERSION
   git describe --tags --abbrev=0
   
   # 如有不一致，使用 git 标签为准
   git describe --tags --abbrev=0 | sed 's/^v//' > VERSION
   git add VERSION
   git commit -m "chore: sync VERSION file with git tag"
   ```

## 📊 监控和分析

### GitHub Actions 日志
- 查看详细的发布日志
- 监控重试次数和成功率
- 分析失败原因和模式

### 发布指标
- 发布频率统计
- 发布类型分布
- 失败率和原因分析

### 告警设置
- 连续发布失败告警
- 版本不一致告警
- 依赖安全漏洞告警

## 🔗 相关资源

- [Standard Version 文档](https://github.com/conventional-changelog/standard-version)
- [Conventional Commits 规范](https://conventionalcommits.org/)
- [GitHub Actions 发布工作流](https://docs.github.com/en/actions/automating-builds-and-tests/building-and-testing-nodejs)
- [Semantic Versioning](https://semver.org/)

---

**注意**: 此工作流设计为高度自动化，大多数情况下无需人工干预。如遇特殊情况，请参考故障排除部分或联系维护团队。