# InfraCore Deployment Script for Windows
# Author: last-emo-boy
# Usage: .\deploy.ps1 [-Environment "development|production"]

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("development", "production")]
    [string]$Environment = "production"
)

$ErrorActionPreference = "Stop"

$ProjectName = "infra-core"
$DockerImage = "${ProjectName}:latest"

Write-Host "🚀 Starting InfraCore deployment (Environment: $Environment)" -ForegroundColor Green

# Check dependencies
function Test-Dependencies {
    Write-Host "📋 Checking dependencies..." -ForegroundColor Yellow
    
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Host "❌ Docker is not installed. Please install Docker Desktop first." -ForegroundColor Red
        exit 1
    }
    
    if (-not (Get-Command docker-compose -ErrorAction SilentlyContinue)) {
        Write-Host "❌ Docker Compose is not installed. Please install Docker Compose first." -ForegroundColor Red
        exit 1
    }
    
    try {
        docker info | Out-Null
    } catch {
        Write-Host "❌ Docker daemon is not running. Please start Docker Desktop." -ForegroundColor Red
        exit 1
    }
    
    Write-Host "✅ All dependencies are satisfied" -ForegroundColor Green
}

# Setup directories
function New-ProjectDirectories {
    Write-Host "📁 Setting up directories..." -ForegroundColor Yellow
    
    $directories = @("data", "log", "certs")
    foreach ($dir in $directories) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }
    
    Write-Host "✅ Directories created successfully" -ForegroundColor Green
}

# Build and deploy
function Start-Deployment {
    Write-Host "🔨 Building and deploying..." -ForegroundColor Yellow
    
    switch ($Environment) {
        "development" {
            Write-Host "🛠️  Starting development environment..." -ForegroundColor Cyan
            docker-compose -f docker-compose.dev.yml down
            docker-compose -f docker-compose.dev.yml build
            docker-compose -f docker-compose.dev.yml up -d
        }
        "production" {
            Write-Host "🏭 Starting production environment..." -ForegroundColor Cyan
            
            # Generate JWT secret if not exists
            if (-not $env:JWT_SECRET) {
                $env:JWT_SECRET = [System.Web.Security.Membership]::GeneratePassword(32, 0)
                Write-Host "🔑 Generated JWT secret: $($env:JWT_SECRET.Substring(0,8))..." -ForegroundColor Green
            }
            
            docker-compose down
            docker-compose build
            docker-compose up -d
        }
    }
}

# Health check
function Test-Health {
    Write-Host "🏥 Performing health check..." -ForegroundColor Yellow
    
    $maxAttempts = 30
    $attempt = 1
    
    while ($attempt -le $maxAttempts) {
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8082/api/v1/health" -TimeoutSec 5 -ErrorAction Stop
            if ($response.StatusCode -eq 200) {
                Write-Host "✅ Health check passed" -ForegroundColor Green
                return $true
            }
        } catch {
            # Continue trying
        }
        
        Write-Host "⏳ Attempt $attempt/$maxAttempts - waiting for service..." -ForegroundColor Yellow
        Start-Sleep -Seconds 2
        $attempt++
    }
    
    Write-Host "❌ Health check failed after $maxAttempts attempts" -ForegroundColor Red
    Write-Host "📋 Checking logs..." -ForegroundColor Yellow
    docker-compose logs --tail=20
    return $false
}

# Show status
function Show-Status {
    Write-Host ""
    Write-Host "📊 Deployment Status:" -ForegroundColor Cyan
    Write-Host "====================" -ForegroundColor Cyan
    docker-compose ps
    
    Write-Host ""
    Write-Host "🌐 Service URLs:" -ForegroundColor Cyan
    Write-Host "===============" -ForegroundColor Cyan
    Write-Host "Console: http://localhost:8082" -ForegroundColor White
    if ($Environment -eq "development") {
        Write-Host "Frontend: http://localhost:5173" -ForegroundColor White
    }
    Write-Host "Gate HTTP: http://localhost:80" -ForegroundColor White
    Write-Host "Gate HTTPS: https://localhost:443" -ForegroundColor White
    
    Write-Host ""
    Write-Host "📝 Useful Commands:" -ForegroundColor Cyan
    Write-Host "==================" -ForegroundColor Cyan
    Write-Host "View logs: docker-compose logs -f" -ForegroundColor White
    Write-Host "Stop services: docker-compose down" -ForegroundColor White
    Write-Host "Restart: docker-compose restart" -ForegroundColor White
    Write-Host "Update: .\deploy.ps1 -Environment $Environment" -ForegroundColor White
}

# Main execution
function Main {
    Write-Host "📦 InfraCore Deployment Script" -ForegroundColor Magenta
    Write-Host "==============================" -ForegroundColor Magenta
    
    Test-Dependencies
    New-ProjectDirectories
    Start-Deployment
    
    if (Test-Health) {
        Show-Status
        Write-Host ""
        Write-Host "🎉 Deployment completed successfully!" -ForegroundColor Green
        Write-Host "Dashboard will be available at: http://localhost:8082" -ForegroundColor Green
    } else {
        Write-Host "❌ Deployment failed!" -ForegroundColor Red
        exit 1
    }
}

# Run main function
try {
    Main
} catch {
    Write-Host "❌ Deployment failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    Write-Host ""
    Write-Host "🧹 Cleaning up..." -ForegroundColor Yellow
    docker system prune -f | Out-Null
}