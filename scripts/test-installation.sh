#!/bin/bash

# InfraCore 安装验证测试脚本
# 使用方法: ./scripts/test-installation.sh

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_test() { echo -e "${PURPLE}[TEST]${NC} $*"; }

# 测试函数
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_test "Testing: $test_name"
    
    if eval "$test_command" >/dev/null 2>&1; then
        log_success "✅ $test_name: PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "❌ $test_name: FAILED"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 带输出的测试函数
run_test_with_output() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_test "Testing: $test_name"
    
    local output
    if output=$(eval "$test_command" 2>&1); then
        log_success "✅ $test_name: PASSED"
        echo "   Output: $output"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "❌ $test_name: FAILED"
        echo "   Error: $output"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# HTTP 测试函数
test_http_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_test "Testing HTTP: $name"
    
    local response
    if response=$(curl -s -w "HTTP_STATUS:%{http_code}" "$url" 2>/dev/null); then
        local http_status=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
        local body=$(echo "$response" | sed 's/HTTP_STATUS:[0-9]*$//')
        
        if [[ "$http_status" == "$expected_status" ]]; then
            log_success "✅ $name: PASSED (HTTP $http_status)"
            if [[ -n "$body" && ${#body} -lt 200 ]]; then
                echo "   Response: $body"
            fi
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        else
            log_error "❌ $name: FAILED (Expected HTTP $expected_status, got HTTP $http_status)"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            return 1
        fi
    else
        log_error "❌ $name: FAILED (Connection error)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 主函数
main() {
    echo
    echo -e "${CYAN}🧪 InfraCore 安装验证测试${NC}"
    echo -e "${CYAN}================================${NC}"
    echo
    
    # 1. 基础环境检查
    log_info "🔍 Phase 1: Environment Check"
    run_test "Docker installed" "docker --version"
    run_test "Docker Compose installed" "docker-compose --version"
    run_test "curl installed" "curl --version"
    
    # 2. Docker 服务状态检查
    echo
    log_info "🐳 Phase 2: Docker Services Check"
    run_test "Docker daemon running" "docker info"
    
    if docker-compose ps >/dev/null 2>&1; then
        run_test_with_output "Docker Compose services status" "docker-compose ps --format table"
        
        # 检查具体服务
        if docker-compose ps | grep -q "infra-core.*Up"; then
            log_success "InfraCore services are running"
        else
            log_warning "InfraCore services may not be fully started"
        fi
    else
        log_warning "Docker Compose project not found or not running"
    fi
    
    # 3. 网络连接测试
    echo
    log_info "🌐 Phase 3: Network Connectivity"
    test_http_endpoint "API Health Check" "http://localhost:8082/api/v1/health" "200"
    test_http_endpoint "Web Interface" "http://localhost" "200"
    
    # 如果开发模式，测试 UI 服务器
    if curl -s http://localhost:5173 >/dev/null 2>&1; then
        test_http_endpoint "UI Dev Server" "http://localhost:5173" "200"
    fi
    
    # 4. API 功能测试
    echo
    log_info "🔐 Phase 4: API Functionality"
    
    # 测试登录接口
    local login_response
    login_response=$(curl -s -X POST http://localhost:8082/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        -w "HTTP_STATUS:%{http_code}" 2>/dev/null || echo "HTTP_STATUS:000")
    
    local login_status=$(echo "$login_response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
    if [[ "$login_status" == "200" ]]; then
        log_success "✅ Admin login: PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        
        # 提取 JWT token
        local token
        token=$(echo "$login_response" | sed 's/HTTP_STATUS:[0-9]*$//' | \
                grep -o '"token":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "")
        
        if [[ -n "$token" ]]; then
            # 测试需要认证的接口
            test_http_endpoint "User Profile" "http://localhost:8082/api/v1/users/profile" "200"
            test_http_endpoint "System Info" "http://localhost:8082/api/v1/system/info" "200"
        fi
    else
        log_error "❌ Admin login: FAILED (HTTP $login_status)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # 5. 系统资源检查
    echo
    log_info "📊 Phase 5: System Resources"
    
    # 检查磁盘空间
    local disk_usage
    disk_usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ "$disk_usage" -lt 90 ]]; then
        log_success "✅ Disk usage: $disk_usage% (OK)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        log_warning "⚠️ Disk usage: $disk_usage% (High)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # 检查内存使用
    local mem_usage
    mem_usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    if [[ "$mem_usage" -lt 90 ]]; then
        log_success "✅ Memory usage: $mem_usage% (OK)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        log_warning "⚠️ Memory usage: $mem_usage% (High)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # 6. 安全检查
    echo
    log_info "🔒 Phase 6: Security Check"
    
    # 检查端口安全
    run_test "No unnecessary ports exposed" "! netstat -tlnp | grep -E ':22.*0.0.0.0|:3306.*0.0.0.0|:5432.*0.0.0.0'"
    
    # 检查 Docker 容器安全
    if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "healthy"; then
        log_success "✅ Docker health checks: PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        log_warning "⚠️ Some containers may not have health checks"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # 7. 测试报告
    echo
    echo -e "${CYAN}📋 Test Report${NC}"
    echo -e "${CYAN}===============${NC}"
    echo -e "Total Tests: ${WHITE}$TOTAL_TESTS${NC}"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    echo
    
    local success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    if [[ $success_rate -ge 90 ]]; then
        echo -e "${GREEN}🎉 Installation verification: EXCELLENT ($success_rate%)${NC}"
        echo -e "${GREEN}Your InfraCore installation is working perfectly!${NC}"
    elif [[ $success_rate -ge 75 ]]; then
        echo -e "${YELLOW}⚠️ Installation verification: GOOD ($success_rate%)${NC}"
        echo -e "${YELLOW}Your InfraCore installation is mostly working, but some issues need attention.${NC}"
    else
        echo -e "${RED}❌ Installation verification: POOR ($success_rate%)${NC}"
        echo -e "${RED}Your InfraCore installation has significant issues that need to be resolved.${NC}"
    fi
    
    echo
    log_info "💡 Quick access URLs:"
    echo "   Web Interface: http://localhost"
    echo "   API Server: http://localhost:8082"
    echo "   Health Check: http://localhost:8082/api/v1/health"
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        echo
        log_info "🔧 For troubleshooting help, run:"
        echo "   sudo ./server-deploy.sh --status"
        echo "   docker-compose logs --tail=50"
        echo "   ./scripts/troubleshoot.sh"
        exit 1
    fi
    
    exit 0
}

# 运行主函数
main "$@"