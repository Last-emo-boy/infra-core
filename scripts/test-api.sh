#!/bin/bash

# InfraCore API 测试脚本
# 使用方法: ./scripts/test-api.sh [base_url]

set -euo pipefail

# 默认配置
BASE_URL="${1:-http://localhost:8082}"
DEFAULT_USER="admin"
DEFAULT_PASS="admin123"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 全局变量
JWT_TOKEN=""
TEST_COUNT=0
PASS_COUNT=0
FAIL_COUNT=0

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_test() { echo -e "${PURPLE}[TEST]${NC} $*"; }

# API 测试函数
test_api() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="${4:-}"
    local expected_status="${5:-200}"
    local auth_required="${6:-false}"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    log_test "$name"
    
    # 构建 curl 命令
    local curl_cmd="curl -s -w '\nHTTP_STATUS:%{http_code}\n'"
    
    # 添加认证头
    if [[ "$auth_required" == "true" && -n "$JWT_TOKEN" ]]; then
        curl_cmd="$curl_cmd -H 'Authorization: Bearer $JWT_TOKEN'"
    fi
    
    # 添加请求方法和数据
    curl_cmd="$curl_cmd -X $method"
    if [[ -n "$data" ]]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    # 添加 URL
    curl_cmd="$curl_cmd '$BASE_URL$endpoint'"
    
    # 执行请求
    local response
    if response=$(eval "$curl_cmd" 2>/dev/null); then
        local body=$(echo "$response" | sed '$d')
        local status=$(echo "$response" | tail -1 | grep -o '[0-9]*')
        
        if [[ "$status" == "$expected_status" ]]; then
            log_success "✅ $name: PASSED (HTTP $status)"
            if [[ -n "$body" && ${#body} -lt 300 ]]; then
                echo "   Response: $body"
            fi
            PASS_COUNT=$((PASS_COUNT + 1))
            
            # 如果是登录请求，提取 token
            if [[ "$endpoint" == "/api/v1/auth/login" && "$status" == "200" ]]; then
                JWT_TOKEN=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "")
                if [[ -n "$JWT_TOKEN" ]]; then
                    log_info "   JWT Token extracted successfully"
                fi
            fi
            
            return 0
        else
            log_error "❌ $name: FAILED (Expected HTTP $expected_status, got HTTP $status)"
            if [[ -n "$body" ]]; then
                echo "   Response: $body"
            fi
            FAIL_COUNT=$((FAIL_COUNT + 1))
            return 1
        fi
    else
        log_error "❌ $name: FAILED (Connection error)"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return 1
    fi
}

# 主测试流程
main() {
    echo
    echo -e "${CYAN}🧪 InfraCore API 测试${NC}"
    echo -e "${CYAN}=====================${NC}"
    echo -e "Base URL: ${WHITE}$BASE_URL${NC}"
    echo
    
    # 1. 基础健康检查
    log_info "🏥 Phase 1: Health Checks"
    test_api "Health Check" "GET" "/api/v1/health" "" "200"
    
    # 2. 认证测试
    echo
    log_info "🔐 Phase 2: Authentication"
    test_api "Admin Login" "POST" "/api/v1/auth/login" "{\"username\":\"$DEFAULT_USER\",\"password\":\"$DEFAULT_PASS\"}" "200"
    
    if [[ -n "$JWT_TOKEN" ]]; then
        test_api "Token Validation" "GET" "/api/v1/users/profile" "" "200" "true"
        test_api "Token Refresh" "POST" "/api/v1/auth/refresh" "" "200" "true"
    else
        log_warning "⚠️ JWT token not available, skipping authenticated tests"
    fi
    
    # 3. 用户管理测试
    echo
    log_info "👥 Phase 3: User Management"
    if [[ -n "$JWT_TOKEN" ]]; then
        test_api "Get User Profile" "GET" "/api/v1/users/profile" "" "200" "true"
        test_api "Update User Profile" "PUT" "/api/v1/users/profile" "{\"display_name\":\"Test Admin\"}" "200" "true"
        test_api "List Users" "GET" "/api/v1/users" "" "200" "true"
    else
        log_warning "⚠️ Skipping user tests (no authentication)"
    fi
    
    # 4. 系统信息测试
    echo
    log_info "📊 Phase 4: System Information"
    if [[ -n "$JWT_TOKEN" ]]; then
        test_api "System Info" "GET" "/api/v1/system/info" "" "200" "true"
        test_api "System Metrics" "GET" "/api/v1/system/metrics" "" "200" "true"
        test_api "Dashboard Data" "GET" "/api/v1/system/dashboard" "" "200" "true"
    else
        log_warning "⚠️ Skipping system tests (no authentication)"
    fi
    
    # 5. 服务管理测试
    echo  
    log_info "🐳 Phase 5: Service Management"
    if [[ -n "$JWT_TOKEN" ]]; then
        test_api "List Services" "GET" "/api/v1/services" "" "200" "true"
        # 注意：这里不测试创建/删除服务，因为那会影响系统
    else
        log_warning "⚠️ Skipping service tests (no authentication)"
    fi
    
    # 6. 错误处理测试
    echo
    log_info "❌ Phase 6: Error Handling"
    test_api "Invalid Endpoint" "GET" "/api/v1/nonexistent" "" "404"
    test_api "Invalid Login" "POST" "/api/v1/auth/login" "{\"username\":\"invalid\",\"password\":\"wrong\"}" "401"
    test_api "Unauthorized Access" "GET" "/api/v1/users/profile" "" "401"
    
    # 7. 性能测试（简单）
    echo
    log_info "⚡ Phase 7: Performance Test"
    local start_time=$(date +%s%N)
    test_api "Response Time Test" "GET" "/api/v1/health" "" "200"
    local end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $duration -lt 1000 ]]; then
        log_success "✅ Response time: ${duration}ms (Excellent)"
    elif [[ $duration -lt 2000 ]]; then
        log_success "✅ Response time: ${duration}ms (Good)"
    else
        log_warning "⚠️ Response time: ${duration}ms (Slow)"
    fi
    
    # 8. 测试报告
    echo
    echo -e "${CYAN}📋 API Test Report${NC}"
    echo -e "${CYAN}==================${NC}"
    echo -e "Total Tests: ${WHITE}$TEST_COUNT${NC}"
    echo -e "Passed: ${GREEN}$PASS_COUNT${NC}"
    echo -e "Failed: ${RED}$FAIL_COUNT${NC}"
    echo
    
    local success_rate=$((PASS_COUNT * 100 / TEST_COUNT))
    if [[ $success_rate -ge 90 ]]; then
        echo -e "${GREEN}🎉 API Test Result: EXCELLENT ($success_rate%)${NC}"
        echo -e "${GREEN}All critical API functions are working correctly!${NC}"
    elif [[ $success_rate -ge 75 ]]; then
        echo -e "${YELLOW}⚠️ API Test Result: GOOD ($success_rate%)${NC}"
        echo -e "${YELLOW}Most API functions are working, but some issues were found.${NC}"
    else
        echo -e "${RED}❌ API Test Result: POOR ($success_rate%)${NC}"
        echo -e "${RED}Significant API issues detected. Please check the logs.${NC}"
    fi
    
    echo
    log_info "🔗 API Documentation:"
    echo "   Swagger UI: $BASE_URL/docs (if available)"
    echo "   Health Check: $BASE_URL/api/v1/health"
    echo "   Base URL: $BASE_URL"
    
    if [[ $FAIL_COUNT -gt 0 ]]; then
        exit 1
    fi
    exit 0
}

# 显示帮助
show_help() {
    echo "Usage: $0 [base_url]"
    echo
    echo "Arguments:"
    echo "  base_url    API base URL (default: http://localhost:8082)"
    echo
    echo "Examples:"
    echo "  $0                                    # Test localhost"
    echo "  $0 http://api.example.com:8082        # Test remote server"
    echo "  $0 https://your-domain.com            # Test production"
    echo
    echo "Environment Variables:"
    echo "  API_USER    Username for login (default: admin)"
    echo "  API_PASS    Password for login (default: admin123)"
}

# 检查依赖
check_dependencies() {
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required but not installed."
        echo "Install curl: sudo apt-get install curl"
        exit 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        log_warning "jq is not installed. JSON responses will be shown as raw text."
        echo "Install jq for better JSON formatting: sudo apt-get install jq"
    fi
}

# 处理参数
if [[ "${1:-}" == "help" || "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

# 使用环境变量覆盖默认值
DEFAULT_USER="${API_USER:-$DEFAULT_USER}"
DEFAULT_PASS="${API_PASS:-$DEFAULT_PASS}"

# 检查依赖并运行
check_dependencies
main "$@"