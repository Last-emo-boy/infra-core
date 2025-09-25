#!/bin/bash

# InfraCore 故障排除脚本
# 使用方法: ./scripts/troubleshoot.sh

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

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_debug() { echo -e "${PURPLE}[DEBUG]${NC} $*"; }

# 检查是否需要 sudo
check_sudo() {
    if [[ $EUID -ne 0 ]]; then
        log_warning "This script may need sudo privileges for some operations."
        echo "If you encounter permission errors, try running: sudo $0"
        echo
    fi
}

# 收集系统信息
collect_system_info() {
    log_info "🖥️ System Information"
    echo "=========================="
    
    echo "📊 OS Information:"
    if [[ -f /etc/os-release ]]; then
        source /etc/os-release
        echo "   OS: $PRETTY_NAME"
        echo "   Version: $VERSION"
    else
        echo "   OS: $(uname -s) $(uname -r)"
    fi
    
    echo "   Architecture: $(uname -m)"
    echo "   Kernel: $(uname -r)"
    echo "   Uptime: $(uptime -p 2>/dev/null || uptime)"
    echo
    
    echo "💾  Memory Information:"
    free -h | head -2
    echo
    
    echo "💽 Disk Information:"
    df -h | grep -E '^/dev|^Filesystem'
    echo
    
    echo "🔧 CPU Information:"
    echo "   Cores: $(nproc)"
    echo "   Load Average: $(uptime | awk -F'load average:' '{print $2}')"
    echo
}

# 检查 Docker 环境
check_docker_environment() {
    log_info "🐳 Docker Environment Check"
    echo "================================"
    
    if command -v docker >/dev/null 2>&1; then
        echo "✅ Docker installed: $(docker --version)"
        
        if docker info >/dev/null 2>&1; then
            echo "✅ Docker daemon running"
            echo "   Storage Driver: $(docker info --format '{{.Driver}}')"
            echo "   Docker Root Dir: $(docker info --format '{{.DockerRootDir}}')"
        else
            log_error "❌ Docker daemon not running or not accessible"
            echo "   Try: sudo systemctl start docker"
        fi
    else
        log_error "❌ Docker not installed"
        echo "   Install Docker: https://docs.docker.com/get-docker/"
    fi
    
    if command -v docker-compose >/dev/null 2>&1; then
        echo "✅ Docker Compose installed: $(docker-compose --version)"
    else
        log_error "❌ Docker Compose not installed"
    fi
    echo
}

# 检查网络端口
check_network_ports() {
    log_info "🌐 Network Ports Check"
    echo "========================"
    
    local ports=("80" "443" "8082" "5173")
    
    for port in "${ports[@]}"; do
        local process
        if process=$(netstat -tlnp 2>/dev/null | grep ":$port " | head -1); then
            local pid=$(echo "$process" | awk '{print $7}' | cut -d'/' -f1)
            local name=$(echo "$process" | awk '{print $7}' | cut -d'/' -f2)
            echo "   Port $port: ✅ USED by $name (PID: $pid)"
        else
            echo "   Port $port: ⚪ FREE"
        fi
    done
    echo
    
    # 检查防火墙状态
    if command -v ufw >/dev/null 2>&1; then
        echo "🔥 Firewall Status (UFW):"
        ufw status 2>/dev/null || echo "   UFW not configured"
    elif command -v firewall-cmd >/dev/null 2>&1; then
        echo "🔥 Firewall Status (firewalld):"
        firewall-cmd --state 2>/dev/null || echo "   Firewalld not running"
    fi
    echo
}

# 检查 Docker 容器状态
check_docker_containers() {
    log_info "📦 Docker Containers Check"
    echo "============================"
    
    if docker ps >/dev/null 2>&1; then
        echo "🏃 Running Containers:"
        docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "infra-core|InfraCore" || echo "   No InfraCore containers found"
        echo
        
        echo "🏗️ All Containers (including stopped):"
        docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" | grep -E "infra-core|InfraCore" || echo "   No InfraCore containers found"
        echo
        
        # 检查容器资源使用
        if docker ps --filter "name=infra-core" --format "{{.Names}}" | head -1 >/dev/null 2>&1; then
            echo "📊 Container Resource Usage:"
            docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" $(docker ps --filter "name=infra-core" --format "{{.Names}}") 2>/dev/null || echo "   Unable to get container stats"
            echo
        fi
    else
        log_error "❌ Cannot access Docker containers"
    fi
}

# 检查服务配置
check_service_configuration() {
    log_info "⚙️ Service Configuration Check"
    echo "==============================="
    
    # 检查 docker-compose 文件
    local compose_files=("docker-compose.yml" "docker-compose.yaml" "docker-compose.dev.yml")
    for file in "${compose_files[@]}"; do
        if [[ -f "$file" ]]; then
            echo "✅ Found: $file"
            if docker-compose -f "$file" config >/dev/null 2>&1; then
                echo "   ✅ Configuration valid"
            else
                log_error "   ❌ Configuration invalid"
            fi
        fi
    done
    echo
    
    # 检查配置目录
    if [[ -d "configs" ]]; then
        echo "📁 Configuration files:"
        ls -la configs/ 2>/dev/null || echo "   Unable to list config files"
    fi
    echo
}

# 检查日志文件
check_logs() {
    log_info "📋 Recent Logs Check"
    echo "====================="
    
    if docker-compose ps >/dev/null 2>&1; then
        echo "🐳 Docker Compose Logs (last 20 lines):"
        echo "----------------------------------------"
        docker-compose logs --tail=20 2>/dev/null || echo "   Unable to get compose logs"
        echo
        
        # 检查各个服务的日志
        local services=("console" "gate" "ui")
        for service in "${services[@]}"; do
            if docker-compose ps "$service" >/dev/null 2>&1; then
                echo "📋 $service service logs (last 10 lines):"
                echo "----------------------------------------"
                docker-compose logs --tail=10 "$service" 2>/dev/null || echo "   Unable to get $service logs"
                echo
            fi
        done
    fi
    
    # 检查系统日志
    if command -v journalctl >/dev/null 2>&1; then
        echo "📋 System logs related to Docker (last 10 lines):"
        echo "--------------------------------------------------"
        journalctl -u docker --no-pager -n 10 2>/dev/null || echo "   Unable to get system logs"
        echo
    fi
}

# 网络连接测试
test_network_connectivity() {
    log_info "🌐 Network Connectivity Test"
    echo "============================="
    
    local endpoints=(
        "localhost:8082|API Server"
        "localhost:80|Web Interface"  
        "localhost:5173|UI Dev Server"
    )
    
    for endpoint in "${endpoints[@]}"; do
        local url=$(echo "$endpoint" | cut -d'|' -f1)
        local name=$(echo "$endpoint" | cut -d'|' -f2)
        
        if curl -s --max-time 5 "http://$url" >/dev/null 2>&1; then
            echo "   ✅ $name ($url): Accessible"
        else
            echo "   ❌ $name ($url): Not accessible"
        fi
    done
    echo
    
    # DNS 测试
    echo "🔍 DNS Resolution Test:"
    local dns_targets=("github.com" "docker.io" "registry.npmjs.org")
    for target in "${dns_targets[@]}"; do
        if nslookup "$target" >/dev/null 2>&1; then
            echo "   ✅ $target: Resolvable"
        else
            echo "   ❌ $target: Resolution failed"
        fi
    done
    echo
}

# 生成修复建议
generate_fix_suggestions() {
    log_info "🔧 Common Fix Suggestions"
    echo "=========================="
    
    echo "1. 🐳 Docker Issues:"
    echo "   sudo systemctl start docker"
    echo "   sudo systemctl enable docker"
    echo "   sudo usermod -aG docker \$USER  # Then logout/login"
    echo
    
    echo "2. 🔌 Port Conflicts:"
    echo "   sudo systemctl stop apache2 nginx"
    echo "   sudo netstat -tlnp | grep ':80 '"
    echo "   sudo lsof -i :80"
    echo
    
    echo "3. 🚀 Service Restart:"
    echo "   docker-compose down && docker-compose up -d"
    echo "   docker-compose restart"
    echo "   docker-compose up -d --force-recreate"
    echo
    
    echo "4. 🧹 Clean Docker Environment:"
    echo "   docker system prune -f"
    echo "   docker-compose down -v"
    echo "   docker volume prune -f"
    echo
    
    echo "5. 📊 Check Service Status:"
    echo "   sudo ./server-deploy.sh --status"
    echo "   docker-compose ps"
    echo "   docker-compose logs -f"
    echo
    
    echo "6. 🔄 Complete Rebuild:"
    echo "   docker-compose down"
    echo "   docker-compose build --no-cache"
    echo "   docker-compose up -d"
    echo
}

# 收集故障排除报告
generate_debug_report() {
    local report_file="infracore-debug-$(date +%Y%m%d-%H%M%S).txt"
    
    log_info "📊 Generating debug report: $report_file"
    
    {
        echo "InfraCore Debug Report"
        echo "Generated at: $(date)"
        echo "========================================"
        echo
        
        collect_system_info
        check_docker_environment
        check_network_ports
        check_docker_containers
        check_service_configuration
        
        echo "🔧 Environment Variables:"
        echo "========================"
        env | grep -E "DOCKER|COMPOSE|INFRA" | sort || echo "No relevant env vars found"
        echo
        
        echo "📁 Current Directory Contents:"
        echo "=============================="
        ls -la
        echo
        
    } > "$report_file"
    
    log_success "Debug report saved to: $report_file"
    echo "You can share this file when seeking help."
}

# 主函数
main() {
    echo
    echo -e "${CYAN}🔧 InfraCore Troubleshooting Tool${NC}"
    echo -e "${CYAN}===================================${NC}"
    echo
    
    check_sudo
    
    # 解析命令行参数
    case "${1:-all}" in
        "system")
            collect_system_info
            ;;
        "docker")
            check_docker_environment
            check_docker_containers
            ;;
        "network")
            check_network_ports
            test_network_connectivity
            ;;
        "logs")
            check_logs
            ;;
        "config")
            check_service_configuration
            ;;
        "fixes")
            generate_fix_suggestions
            ;;
        "report")
            generate_debug_report
            ;;
        "all"|*)
            collect_system_info
            check_docker_environment
            check_network_ports
            check_docker_containers
            check_service_configuration
            check_logs
            test_network_connectivity
            generate_fix_suggestions
            
            echo
            log_info "💡 To generate a detailed debug report, run:"
            echo "   $0 report"
            ;;
    esac
    
    echo
    echo -e "${GREEN}🎯 Troubleshooting completed!${NC}"
    echo
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [option]"
    echo
    echo "Options:"
    echo "  all     - Run all checks (default)"
    echo "  system  - System information only"
    echo "  docker  - Docker environment check"
    echo "  network - Network and port check"
    echo "  logs    - Show recent logs"
    echo "  config  - Check configuration files"
    echo "  fixes   - Show common fix suggestions"
    echo "  report  - Generate detailed debug report"
    echo "  help    - Show this help message"
    echo
}

# 处理参数
if [[ "${1:-}" == "help" || "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    show_help
    exit 0
fi

# 运行主函数
main "$@"