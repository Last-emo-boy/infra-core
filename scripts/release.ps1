# Release Management Script
# 使用 GitHub CLI 手动触发发布

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("patch", "minor", "major", "alpha", "beta", "prerelease")]
    [string]$ReleaseType,
    
    [string]$Branch = "main"
)

Write-Host "🚀 Triggering release: $ReleaseType on branch: $Branch" -ForegroundColor Green

# 检查 GitHub CLI 是否安装
if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    Write-Error "GitHub CLI (gh) is not installed. Please install it first: winget install GitHub.cli"
    exit 1
}

# 检查是否已登录
gh auth status 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Error "Not authenticated with GitHub. Please run: gh auth login"
    exit 1
}

# 触发工作流
try {
    gh workflow run "Release Management" `
        --ref $Branch `
        --field release_type=$ReleaseType
    
    Write-Host "✅ Release workflow triggered successfully!" -ForegroundColor Green
    Write-Host "📝 Check progress at: https://github.com/$(gh repo view --json owner,name --jq '.owner.login + "/" + .name')/actions" -ForegroundColor Yellow
    
    # 等待几秒钟然后显示最新的工作流运行
    Start-Sleep 3
    Write-Host "`n📊 Recent workflow runs:" -ForegroundColor Cyan
    gh run list --workflow="Release Management" --limit=3
}
catch {
    Write-Error "Failed to trigger workflow: $_"
    exit 1
}

Write-Host "`n📋 Release Type Explanations:" -ForegroundColor Blue
Write-Host "  patch      - Bug fixes (0.1.0 → 0.1.1)" -ForegroundColor White
Write-Host "  minor      - New features (0.1.0 → 0.2.0)" -ForegroundColor White
Write-Host "  major      - Breaking changes (0.x.x → 1.0.0)" -ForegroundColor White
Write-Host "  alpha      - Early testing (0.1.0 → 0.1.1-alpha.0)" -ForegroundColor White
Write-Host "  beta       - Feature testing (0.1.0 → 0.1.1-beta.0)" -ForegroundColor White
Write-Host "  prerelease - General pre-release bump" -ForegroundColor White