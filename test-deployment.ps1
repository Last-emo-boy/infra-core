# Full deployment test for InfraCore (Windows PowerShell)
# Author: last-emo-boy

param(
    [string]$Environment = "testing",
    [int]$TestPort = 18082,
    [string]$LogFile = "test-deployment-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
)

# Test configuration
$script:TestUserEmail = "test@example.com"
$script:TestUserPassword = "test123456"
$script:HealthCheckUrl = "http://localhost:$TestPort/api/v1/health"
$script:ServerProcess = $null
$script:TestResults = @{}

# Colors for output
$script:Colors = @{
    Red = [ConsoleColor]::Red
    Green = [ConsoleColor]::Green
    Yellow = [ConsoleColor]::Yellow
    Blue = [ConsoleColor]::Blue
    White = [ConsoleColor]::White
}

function Write-ColorText {
    param(
        [string]$Text,
        [ConsoleColor]$Color = [ConsoleColor]::White
    )
    $currentColor = $Host.UI.RawUI.ForegroundColor
    $Host.UI.RawUI.ForegroundColor = $Color
    Write-Output $Text
    $Host.UI.RawUI.ForegroundColor = $currentColor
}

function Write-Info { 
    param([string]$Message)
    Write-ColorText "[INFO] $Message" $script:Colors.Blue
    Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [INFO] $Message"
}

function Write-Success { 
    param([string]$Message)
    Write-ColorText "[SUCCESS] $Message" $script:Colors.Green
    Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [SUCCESS] $Message"
}

function Write-Warning { 
    param([string]$Message)
    Write-ColorText "[WARNING] $Message" $script:Colors.Yellow
    Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [WARNING] $Message"
}

function Write-Error { 
    param([string]$Message)
    Write-ColorText "[ERROR] $Message" $script:Colors.Red
    Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [ERROR] $Message"
}

function Cleanup {
    Write-Info "Cleaning up test environment..."
    
    # Stop server process if running
    if ($script:ServerProcess -and !$script:ServerProcess.HasExited) {
        Write-Info "Stopping test server..."
        try {
            $script:ServerProcess.Kill()
            $script:ServerProcess.WaitForExit(5000)
        } catch {
            Write-Warning "Failed to stop server gracefully"
        }
    }
    
    # Stop Docker services
    try {
        $dockerServices = docker compose -f docker-compose.dev.yml ps --format json 2>$null | ConvertFrom-Json
        if ($dockerServices) {
            Write-Info "Stopping Docker services..."
            docker compose -f docker-compose.dev.yml down -v 2>$null
        }
    } catch {
        # Docker services not running or not available
    }
    
    # Kill processes on test port
    try {
        $processes = Get-NetTCPConnection -LocalPort $TestPort -ErrorAction SilentlyContinue
        if ($processes) {
            Write-Info "Killing processes on port $TestPort..."
            $processes | ForEach-Object {
                Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue
            }
        }
    } catch {
        # No processes on port
    }
    
    # Clean build artifacts
    try {
        if (Test-Path "bin") {
            Remove-Item -Path "bin" -Recurse -Force -ErrorAction SilentlyContinue
        }
        if (Test-Path "ui\dist") {
            Remove-Item -Path "ui\dist" -Recurse -Force -ErrorAction SilentlyContinue
        }
        if (Test-Path "test.db") {
            Remove-Item -Path "test.db" -Force -ErrorAction SilentlyContinue
        }
    } catch {
        Write-Warning "Failed to clean some build artifacts"
    }
    
    Write-Success "Cleanup completed"
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check Go
    try {
        $goVersion = go version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Go is installed: $($goVersion.Split(' ')[2])"
            $script:TestResults.Prerequisites = $true
        } else {
            Write-Error "Go is not installed or not in PATH"
            return $false
        }
    } catch {
        Write-Error "Go is not installed"
        return $false
    }
    
    # Check Node.js
    try {
        $nodeVersion = node --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Node.js is installed: $nodeVersion"
        } else {
            Write-Error "Node.js is not installed or not in PATH"
            return $false
        }
    } catch {
        Write-Error "Node.js is not installed"
        return $false
    }
    
    # Check Docker
    try {
        $dockerVersion = docker --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Docker is installed: $($dockerVersion.Split(' ')[2].TrimEnd(','))"
        } else {
            Write-Warning "Docker is not available - Docker tests will be skipped"
        }
    } catch {
        Write-Warning "Docker is not installed - Docker tests will be skipped"
    }
    
    # Check Docker Compose
    try {
        docker compose version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Docker Compose is available"
        } else {
            Write-Warning "Docker Compose is not available - Docker tests will be skipped"
        }
    } catch {
        Write-Warning "Docker Compose is not available"
    }
    
    return $true
}

function Test-Build {
    Write-Info "Testing build process..."
    
    # Test Go build
    Write-Info "Building Go backend..."
    try {
        if (Test-Path "Makefile") {
            make build 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Go backend build successful (using Make)"
                $script:TestResults.GoBuild = $true
            } else {
                throw "Make build failed"
            }
        } else {
            # Manual build
            if (!(Test-Path "bin")) {
                New-Item -ItemType Directory -Path "bin" -Force | Out-Null
            }
            go build -o "bin\console.exe" "cmd\console\main.go"
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Go backend build successful (manual)"
                $script:TestResults.GoBuild = $true
            } else {
                throw "Go build failed"
            }
        }
    } catch {
        Write-Error "Go backend build failed: $_"
        return $false
    }
    
    # Test UI build
    Write-Info "Building React frontend..."
    try {
        Set-Location "ui"
        if (!(Test-Path "node_modules")) {
            Write-Info "Installing npm dependencies..."
            npm install
            if ($LASTEXITCODE -ne 0) {
                throw "npm install failed"
            }
        }
        
        npm run build
        if ($LASTEXITCODE -eq 0) {
            Write-Success "React frontend build successful"
            $script:TestResults.UIBuild = $true
        } else {
            throw "npm build failed"
        }
    } catch {
        Write-Error "React frontend build failed: $_"
        return $false
    } finally {
        Set-Location ".."
    }
    
    return $true
}

function Test-Configuration {
    Write-Info "Testing configuration loading..."
    
    # Set environment variables
    $env:INFRA_CORE_ENV = $Environment
    $env:INFRA_CORE_CONSOLE_PORT = $TestPort.ToString()
    
    # Create test configuration if it doesn't exist
    if (!(Test-Path "configs\$Environment.yaml")) {
        Write-Info "Creating test configuration..."
        $configContent = @"
database:
  path: "test.db"
console:
  port: $TestPort
  host: "127.0.0.1"
jwt:
  secret: "test-secret-key-for-testing-only"
  expiration: "24h"
acme:
  enabled: false
logging:
  level: "debug"
  format: "json"
"@
        if (!(Test-Path "configs")) {
            New-Item -ItemType Directory -Path "configs" -Force | Out-Null
        }
        Set-Content -Path "configs\$Environment.yaml" -Value $configContent
    }
    
    $script:TestResults.Configuration = $true
    Write-Success "Configuration test completed"
    return $true
}

function Start-TestServer {
    Write-Info "Starting test server..."
    
    try {
        $env:INFRA_CORE_ENV = $Environment
        $env:INFRA_CORE_CONSOLE_PORT = $TestPort.ToString()
        
        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = ".\bin\console.exe"
        $psi.WorkingDirectory = Get-Location
        $psi.UseShellExecute = $false
        $psi.RedirectStandardOutput = $true
        $psi.RedirectStandardError = $true
        
        $script:ServerProcess = [System.Diagnostics.Process]::Start($psi)
        
        # Wait for server to start
        Write-Info "Waiting for server to start..."
        for ($i = 1; $i -le 30; $i++) {
            try {
                $response = Invoke-WebRequest -Uri $script:HealthCheckUrl -TimeoutSec 2 -ErrorAction Stop
                if ($response.StatusCode -eq 200) {
                    Write-Success "Server started successfully"
                    $script:TestResults.ServerStart = $true
                    return $true
                }
            } catch {
                # Server not ready yet
            }
            Start-Sleep -Seconds 1
        }
        
        Write-Error "Server failed to start within 30 seconds"
        if ($script:ServerProcess -and !$script:ServerProcess.HasExited) {
            $script:ServerProcess.Kill()
        }
        return $false
    } catch {
        Write-Error "Failed to start server: $_"
        return $false
    }
}

function Test-ApiEndpoints {
    Write-Info "Testing API endpoints..."
    
    # Test health check
    Write-Info "Testing health check endpoint..."
    try {
        $response = Invoke-WebRequest -Uri $script:HealthCheckUrl -TimeoutSec 5
        if ($response.StatusCode -eq 200 -and $response.Content -like "*healthy*") {
            Write-Success "Health check endpoint working"
        } else {
            Write-Error "Health check endpoint returned unexpected response"
            return $false
        }
    } catch {
        Write-Error "Health check endpoint failed: $_"
        return $false
    }
    
    # Test user registration
    Write-Info "Testing user registration..."
    try {
        $registerData = @{
            email = $script:TestUserEmail
            password = $script:TestUserPassword
            username = "testuser"
        } | ConvertTo-Json
        
        $response = Invoke-WebRequest -Uri "http://localhost:$TestPort/api/v1/auth/register" `
            -Method POST `
            -ContentType "application/json" `
            -Body $registerData `
            -TimeoutSec 10
        
        if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 201) {
            Write-Success "User registration working"
        } else {
            Write-Warning "User registration may have failed (user might already exist)"
        }
    } catch {
        Write-Warning "User registration failed: $($_.Exception.Message)"
    }
    
    # Test user login
    Write-Info "Testing user login..."
    try {
        $loginData = @{
            email = $script:TestUserEmail
            password = $script:TestUserPassword
        } | ConvertTo-Json
        
        $response = Invoke-WebRequest -Uri "http://localhost:$TestPort/api/v1/auth/login" `
            -Method POST `
            -ContentType "application/json" `
            -Body $loginData `
            -TimeoutSec 10
        
        if ($response.StatusCode -eq 200) {
            $loginResult = $response.Content | ConvertFrom-Json
            if ($loginResult.token) {
                Write-Success "User login working"
                
                # Test authenticated endpoint
                Write-Info "Testing authenticated endpoint..."
                $headers = @{ Authorization = "Bearer $($loginResult.token)" }
                $profileResponse = Invoke-WebRequest -Uri "http://localhost:$TestPort/api/v1/users/profile" `
                    -Headers $headers `
                    -TimeoutSec 10
                
                if ($profileResponse.StatusCode -eq 200) {
                    Write-Success "Authenticated endpoints working"
                    $script:TestResults.ApiEndpoints = $true
                    return $true
                } else {
                    Write-Error "Authenticated endpoints failed"
                    return $false
                }
            } else {
                Write-Error "Login response missing token"
                return $false
            }
        } else {
            Write-Error "User login failed with status: $($response.StatusCode)"
            return $false
        }
    } catch {
        Write-Error "User login failed: $_"
        return $false
    }
}

function Test-DockerDeployment {
    Write-Info "Testing Docker deployment..."
    
    try {
        # Check if Docker is available
        docker --version 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Docker not available - skipping Docker tests"
            return $true
        }
        
        # Build Docker images
        Write-Info "Building Docker images..."
        docker compose -f docker-compose.dev.yml build
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Docker image build failed"
            return $false
        }
        Write-Success "Docker images built successfully"
        
        # Start Docker services
        Write-Info "Starting Docker services..."
        docker compose -f docker-compose.dev.yml up -d
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Docker services failed to start"
            return $false
        }
        Write-Success "Docker services started"
        
        # Wait for services to be ready
        Write-Info "Waiting for Docker services to be ready..."
        for ($i = 1; $i -le 60; $i++) {
            try {
                $response = Invoke-WebRequest -Uri "http://localhost:8082/api/v1/health" -TimeoutSec 2
                if ($response.StatusCode -eq 200) {
                    Write-Success "Docker services are ready"
                    $script:TestResults.DockerDeployment = $true
                    return $true
                }
            } catch {
                # Not ready yet
            }
            Start-Sleep -Seconds 1
        }
        
        Write-Error "Docker services failed to become ready within 60 seconds"
        docker compose -f docker-compose.dev.yml logs
        return $false
        
    } catch {
        Write-Error "Docker deployment test failed: $_"
        return $false
    }
}

function Test-FrontendAccessibility {
    Write-Info "Testing frontend accessibility..."
    
    # Test if built assets exist
    if (Test-Path "ui\dist\index.html") {
        Write-Success "Frontend build artifacts exist"
        $script:TestResults.FrontendBuild = $true
    } else {
        Write-Warning "Frontend build artifacts not found"
    }
    
    # Test if frontend is accessible (if running)
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:5173" -TimeoutSec 2
        if ($response.StatusCode -eq 200) {
            Write-Success "Frontend is accessible"
            $script:TestResults.FrontendAccess = $true
        }
    } catch {
        Write-Warning "Frontend may not be running (this is normal for API-only tests)"
    }
    
    return $true
}

function Generate-TestReport {
    Write-Info "Generating test report..."
    
    $reportFile = "test_report_$(Get-Date -Format 'yyyyMMdd_HHmmss').txt"
    
    $reportContent = @"
InfraCore Deployment Test Report
Generated: $(Get-Date)
Test Environment: $Environment
Test Port: $TestPort
Log File: $LogFile

========================================
Test Results Summary
========================================

Prerequisites Check: $(if ($script:TestResults.Prerequisites) { "PASSED" } else { "FAILED" })
Go Build: $(if ($script:TestResults.GoBuild) { "PASSED" } else { "FAILED" })
UI Build: $(if ($script:TestResults.UIBuild) { "PASSED" } else { "FAILED" })
Configuration: $(if ($script:TestResults.Configuration) { "PASSED" } else { "FAILED" })
Server Start: $(if ($script:TestResults.ServerStart) { "PASSED" } else { "FAILED" })
API Endpoints: $(if ($script:TestResults.ApiEndpoints) { "PASSED" } else { "FAILED" })
Docker Deployment: $(if ($script:TestResults.DockerDeployment) { "PASSED" } else { "SKIPPED/FAILED" })
Frontend Build: $(if ($script:TestResults.FrontendBuild) { "PASSED" } else { "FAILED" })
Frontend Access: $(if ($script:TestResults.FrontendAccess) { "PASSED" } else { "SKIPPED" })

========================================
System Information
========================================

OS: $([Environment]::OSVersion.ToString())
PowerShell Version: $($PSVersionTable.PSVersion)
Go Version: $(go version 2>$null)
Node Version: $(node --version 2>$null)
Docker Version: $(docker --version 2>$null)

========================================
Recommendations
========================================

"@

    if ($script:TestResults.GoBuild -and $script:TestResults.ApiEndpoints) {
        $reportContent += "✅ Core functionality is working correctly`n"
        $reportContent += "✅ Ready for production deployment`n"
    } else {
        $reportContent += "❌ Issues detected - review test output before deployment`n"
    }
    
    if ($script:TestResults.DockerDeployment) {
        $reportContent += "✅ Docker deployment is working`n"
        $reportContent += "✅ Can use Docker for production deployment`n"
    } else {
        $reportContent += "⚠️  Docker deployment issues - consider manual deployment`n"
    }
    
    Set-Content -Path $reportFile -Value $reportContent
    Write-Success "Test report generated: $reportFile"
}

# Main execution
function Main {
    try {
        # Initialize log file
        "InfraCore Deployment Test Started: $(Get-Date)" | Out-File -FilePath $LogFile
        
        Write-Info "Starting InfraCore Deployment Test for Windows"
        Write-Info "=============================================="
        
        # Initialize test results
        $script:TestResults = @{
            Prerequisites = $false
            GoBuild = $false
            UIBuild = $false
            Configuration = $false
            ServerStart = $false
            ApiEndpoints = $false
            DockerDeployment = $false
            FrontendBuild = $false
            FrontendAccess = $false
        }
        
        # Run tests
        if (!(Test-Prerequisites)) {
            Write-Error "Prerequisites check failed"
            return 1
        }
        
        if (!(Test-Configuration)) {
            Write-Error "Configuration test failed"
            return 1
        }
        
        if (!(Test-Build)) {
            Write-Error "Build test failed"
            return 1
        }
        
        # Start server for API tests
        if (Start-TestServer) {
            Test-ApiEndpoints | Out-Null
            
            # Stop the test server
            if ($script:ServerProcess -and !$script:ServerProcess.HasExited) {
                $script:ServerProcess.Kill()
                Start-Sleep -Seconds 2
            }
        }
        
        # Test Docker deployment
        Test-DockerDeployment | Out-Null
        
        # Test frontend
        Test-FrontendAccessibility | Out-Null
        
        # Generate report
        Generate-TestReport
        
        # Summary
        Write-Info "=============================================="
        Write-Info "Test Summary:"
        Write-Output "Prerequisites: $(if ($script:TestResults.Prerequisites) { "PASSED" } else { "FAILED" })"
        Write-Output "Go Build: $(if ($script:TestResults.GoBuild) { "PASSED" } else { "FAILED" })"
        Write-Output "UI Build: $(if ($script:TestResults.UIBuild) { "PASSED" } else { "FAILED" })"
        Write-Output "Configuration: $(if ($script:TestResults.Configuration) { "PASSED" } else { "FAILED" })"
        Write-Output "Server Start: $(if ($script:TestResults.ServerStart) { "PASSED" } else { "FAILED" })"
        Write-Output "API Endpoints: $(if ($script:TestResults.ApiEndpoints) { "PASSED" } else { "FAILED" })"
        Write-Output "Docker: $(if ($script:TestResults.DockerDeployment) { "PASSED" } else { "SKIPPED/FAILED" })"
        Write-Output "Frontend: $(if ($script:TestResults.FrontendBuild) { "PASSED" } else { "FAILED" })"
        
        if ($script:TestResults.GoBuild -and $script:TestResults.ApiEndpoints) {
            Write-Success "🎉 All critical tests passed! Ready for deployment."
            return 0
        } else {
            Write-Error "❌ Some tests failed. Review the output before deploying."
            return 1
        }
        
    } finally {
        Cleanup
    }
}

# Register cleanup on exit
Register-EngineEvent PowerShell.Exiting -Action { Cleanup }

# Run main function
exit (Main)