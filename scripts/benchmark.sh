#!/bin/bash

# InfraCore 性能基准测试脚本
# 使用方法: ./scripts/benchmark.sh [base_url]

set -euo pipefail

# 配置
BASE_URL="${1:-http://localhost:8082}"
CURL_FORMAT_FILE="scripts/curl-format.txt"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }

# 检查依赖
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v ab >/dev/null 2>&1; then
        log_warning "Apache Bench (ab) not found. Some tests will be skipped."
        echo "Install: sudo apt-get install apache2-utils"
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
}

# 响应时间测试
test_response_times() {
    log_info "⚡ Response Time Tests"
    echo "======================="
    
    local endpoints=(
        "/api/v1/health|Health Check"
        "/|Web Interface"
    )
    
    for endpoint_info in "${endpoints[@]}"; do
        local endpoint=$(echo "$endpoint_info" | cut -d'|' -f1)
        local name=$(echo "$endpoint_info" | cut -d'|' -f2)
        
        echo "📊 Testing: $name"
        
        if [[ -f "$CURL_FORMAT_FILE" ]]; then
            local times=$(curl -w "@$CURL_FORMAT_FILE" -o /dev/null -s "${BASE_URL}${endpoint}" 2>/dev/null || echo "Connection failed")
            if [[ "$times" != "Connection failed" ]]; then
                echo "$times"
            else
                echo "   ❌ Connection failed"
            fi
        else
            # 简单计时
            local start_time=$(date +%s%N)
            if curl -s "${BASE_URL}${endpoint}" >/dev/null 2>&1; then
                local end_time=$(date +%s%N)
                local duration=$(( (end_time - start_time) / 1000000 ))
                echo "   Response time: ${duration}ms"
            else
                echo "   ❌ Connection failed"
            fi
        fi
        echo
    done
}

# 并发测试
test_concurrent_requests() {
    if ! command -v ab >/dev/null 2>&1; then
        log_warning "Skipping concurrent tests (Apache Bench not available)"
        return
    fi
    
    log_info "🚀 Concurrent Request Tests"
    echo "============================"
    
    local test_url="${BASE_URL}/api/v1/health"
    
    echo "📊 Light Load Test (100 requests, 10 concurrent):"
    ab -n 100 -c 10 -q "$test_url" 2>/dev/null | grep -E "Requests per second|Time per request|Transfer rate" || echo "Test failed"
    echo
    
    echo "📊 Medium Load Test (1000 requests, 50 concurrent):"
    ab -n 1000 -c 50 -q "$test_url" 2>/dev/null | grep -E "Requests per second|Time per request|Transfer rate" || echo "Test failed"
    echo
}

# 资源使用测试
test_resource_usage() {
    log_info "💾 Resource Usage During Load"
    echo "=============================="
    
    if command -v docker >/dev/null 2>&1 && docker ps --filter "name=infra-core" --format "{{.Names}}" | head -1 >/dev/null 2>&1; then
        echo "📊 Container Resource Usage (before load):"
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" $(docker ps --filter "name=infra-core" --format "{{.Names}}") 2>/dev/null || echo "Unable to get stats"
        echo
        
        if command -v ab >/dev/null 2>&1; then
            echo "🔥 Running load test..."
            ab -n 500 -c 25 -q "${BASE_URL}/api/v1/health" >/dev/null 2>&1 &
            local ab_pid=$!
            
            sleep 2
            echo "📊 Container Resource Usage (during load):"
            docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" $(docker ps --filter "name=infra-core" --format "{{.Names}}") 2>/dev/null || echo "Unable to get stats"
            
            wait $ab_pid 2>/dev/null || true
            echo
        fi
    else
        echo "⚠️ Docker containers not found or not accessible"
        echo
    fi
    
    # 系统资源
    echo "🖥️ System Resource Usage:"
    echo "   CPU Load: $(uptime | awk -F'load average:' '{print $2}')"
    echo "   Memory: $(free -h | awk 'NR==2{printf "%s/%s (%.1f%%)", $3,$2,$3*100/$2}')"
    echo "   Disk: $(df -h / | awk 'NR==2{printf "%s/%s (%s)", $3,$2,$5}')"
    echo
}

# 主函数
main() {
    echo
    echo -e "${CYAN}⚡ InfraCore Performance Benchmark${NC}"
    echo -e "${CYAN}====================================${NC}"
    echo -e "Target: ${WHITE}$BASE_URL${NC}"
    echo
    
    check_dependencies
    
    # 检查服务是否可用
    if ! curl -s "$BASE_URL/api/v1/health" >/dev/null 2>&1; then
        log_error "Service not accessible at $BASE_URL"
        echo "Make sure InfraCore is running and accessible."
        exit 1
    fi
    
    test_response_times
    test_concurrent_requests
    test_resource_usage
    
    echo -e "${GREEN}🎯 Benchmark completed!${NC}"
    echo
    log_info "💡 Performance Tips:"
    echo "   • Response times under 100ms are excellent"
    echo "   • Above 500ms may indicate performance issues"
    echo "   • Monitor resource usage during peak loads"
    echo "   • Consider scaling if CPU/Memory consistently > 80%"
    echo
}

# 显示帮助
show_help() {
    echo "Usage: $0 [base_url]"
    echo
    echo "Arguments:"
    echo "  base_url    API base URL (default: http://localhost:8082)"
    echo
    echo "Examples:"
    echo "  $0                                    # Benchmark localhost"
    echo "  $0 http://api.example.com:8082        # Benchmark remote server"
    echo
}

# 处理参数
if [[ "${1:-}" == "help" || "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

main "$@"