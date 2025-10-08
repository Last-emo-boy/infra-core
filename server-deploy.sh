#!/bin/bash
# InfraCore Linux Server Deployment Script
# Author: last-emo-boy
# Usage: ./server-deploy.sh [options]

# Enhanced error handling
set -euo pipefail
IFS=$'\n\t'

# Global retry configuration
readonly MAX_RETRIES=3
readonly RETRY_DELAY=5
readonly NETWORK_TIMEOUT=30
readonly DOCKER_TIMEOUT=600

# Default configuration
REPO_URL="https://github.com/last-emo-boy/infra-core.git"
BRANCH="main"
DEPLOY_DIR="/opt/infra-core"
SERVICE_USER="infracore"
ENVIRONMENT="production"
REGISTRY="ghcr.io"
IMAGE_NAME="last-emo-boy/infra-core"
BACKUP_DIR="/opt/infra-core/backups"
LOG_FILE="/var/log/infra-core/deploy.log"

# GitHub Container Registry variables (optional)
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
GITHUB_ACTOR="${GITHUB_ACTOR:-}"
BACKUP_RETENTION="${BACKUP_RETENTION:-10}"

# Mirror configuration
USE_MIRRORS="${USE_MIRRORS:-false}"
MIRROR_REGION="${MIRROR_REGION:-auto}"

# Configuration settings with defaults
NON_INTERACTIVE="${NON_INTERACTIVE:-false}"
JWT_SECRET="${JWT_SECRET:-}"
CUSTOM_DOMAIN="${CUSTOM_DOMAIN:-}"
CUSTOM_EMAIL="${CUSTOM_EMAIL:-}"
CUSTOM_HTTP_PORT="${CUSTOM_HTTP_PORT:-}"
CUSTOM_HTTPS_PORT="${CUSTOM_HTTPS_PORT:-}"
CUSTOM_API_PORT="${CUSTOM_API_PORT:-}"
CUSTOM_SSL_ENABLED="${CUSTOM_SSL_ENABLED:-}"
CUSTOM_MEMORY_LIMIT="${CUSTOM_MEMORY_LIMIT:-}"
CUSTOM_CPU_LIMIT="${CUSTOM_CPU_LIMIT:-}"
CUSTOM_BACKUP_ENABLED="${CUSTOM_BACKUP_ENABLED:-}"
CUSTOM_BACKUP_RETENTION="${CUSTOM_BACKUP_RETENTION:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Functions for colored output
# Create log directory if it doesn't exist
ensure_log_dir() {
    if [[ ! -d "$(dirname "$LOG_FILE")" ]]; then
        mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
    fi
}

log_info() {
    ensure_log_dir
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    ensure_log_dir
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    ensure_log_dir
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Universal retry function with exponential backoff
retry_with_backoff() {
    local max_attempts=${1:-$MAX_RETRIES}
    local delay=${2:-$RETRY_DELAY}
    local description="${3:-command}"
    shift 3
    
    local attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        if "$@"; then
            return 0
        else
            local exit_code=$?
            if [[ $attempt -eq $max_attempts ]]; then
                log_error "$description failed after $max_attempts attempts"
                return $exit_code
            fi
            
            log_warning "$description failed (attempt $attempt/$max_attempts), retrying in ${delay}s..."
            sleep "$delay"
            delay=$((delay * 2))  # Exponential backoff
            attempt=$((attempt + 1))
        fi
    done
}

# Enhanced network connectivity test
test_network_connectivity() {
    local test_sites=("8.8.8.8" "114.114.114.114" "1.1.1.1" "baidu.com" "qq.com")
    local timeout=${1:-$NETWORK_TIMEOUT}
    
    for site in "${test_sites[@]}"; do
        if timeout "$timeout" ping -c 1 -W 5 "$site" &>/dev/null; then
            log_success "Network connectivity verified via $site"
            return 0
        fi
    done
    
    log_error "No network connectivity detected"
    return 1
}

# Safe command execution with timeout
safe_execute() {
    local timeout_duration=${1:-300}
    local description="${2:-command}"
    shift 2
    
    if timeout "$timeout_duration" "$@"; then
        return 0
    else
        local exit_code=$?
        log_error "$description timed out after ${timeout_duration}s"
        return $exit_code
    fi
}

# Enhanced error handler
error_handler() {
    local line_number=$1
    local exit_code=$2
    local command="$BASH_COMMAND"
    
    log_error "Error on line $line_number: Command '$command' exited with code $exit_code"
    
    # Collect system information for debugging
    {
        echo "=== System Information ==="
        uname -a
        echo "=== Memory Usage ==="
        free -h
        echo "=== Disk Usage ==="
        df -h
        echo "=== Recent Logs ==="
        tail -20 "$LOG_FILE" 2>/dev/null || echo "No log file available"
    } | tee -a "$LOG_FILE" 2>/dev/null || true
    
    exit $exit_code
}

# Enhanced error handling with rollback capability
ROLLBACK_STEPS=()
DEPLOYMENT_STARTED=false

# Add step to rollback stack
add_rollback_step() {
    ROLLBACK_STEPS+=("$1")
    log_info "Added rollback step: $1"
}

# Execute rollback steps in reverse order
execute_rollback() {
    if [[ ${#ROLLBACK_STEPS[@]} -eq 0 ]]; then
        log_info "No rollback steps to execute"
        return 0
    fi
    
    log_warning "🔄 Executing rollback steps..."
    for ((i=${#ROLLBACK_STEPS[@]}-1; i>=0; i--)); do
        local step="${ROLLBACK_STEPS[i]}"
        log_info "Rolling back: $step"
        eval "$step" || log_error "Failed to rollback: $step"
    done
    ROLLBACK_STEPS=()
    log_success "Rollback completed"
}

# Enhanced error handler with automatic rollback
error_handler() {
    local line_number=$1
    local exit_code=$2
    local command="${BASH_COMMAND}"
    
    log_error "❌ Deployment failed at line $line_number (exit code: $exit_code)"
    log_error "Failed command: $command"
    
    # Auto-rollback on deployment failure
    if [[ "$DEPLOYMENT_STARTED" == "true" ]]; then
        log_warning "Deployment was in progress, initiating automatic rollback..."
        execute_rollback
    fi
    
    # Collect system information for debugging
    {
        echo "=== Error Context ==="
        echo "Line: $line_number"
        echo "Command: $command" 
        echo "Exit Code: $exit_code"
        echo "=== System Information ==="
        uname -a
        echo "=== Memory Usage ==="
        free -h
        echo "=== Disk Usage ==="
        df -h
        echo "=== Recent Logs ==="
        tail -20 "$LOG_FILE" 2>/dev/null || echo "No log file available"
    } | tee -a "$LOG_FILE" 2>/dev/null || true
    
    exit $exit_code
}

# Set up error trap
trap 'error_handler ${LINENO} $?' ERR

# Comprehensive system validation
validate_system_requirements() {
    log_step "Validating system requirements..."
    
    # Check minimum disk space (5GB)
    local available_space_kb
    local deploy_parent_dir=$(dirname "$DEPLOY_DIR")
    
    # Ensure parent directory exists for proper disk space check
    if [[ ! -d "$deploy_parent_dir" ]]; then
        mkdir -p "$deploy_parent_dir" 2>/dev/null || true
    fi
    
    # Check disk space on parent directory if deploy dir doesn't exist
    local check_dir="$DEPLOY_DIR"
    if [[ ! -d "$DEPLOY_DIR" ]]; then
        check_dir="$deploy_parent_dir"
        log_info "Deploy directory doesn't exist, checking parent: $check_dir"
    fi
    
    available_space_kb=$(df "$check_dir" 2>/dev/null | awk 'NR==2{print $4}' | tr -d '\n\r\t ' || echo "0")
    # Ensure we have a valid number
    if [[ ! "$available_space_kb" =~ ^[0-9]+$ ]]; then
        available_space_kb=0
    fi
    local available_space_gb=$((available_space_kb / 1024 / 1024))
    
    # If still getting 0, try alternative methods
    if [[ $available_space_gb -eq 0 ]]; then
        log_warning "Standard disk check failed, trying alternative methods..."
        
        # Clear any filesystem caches that might interfere
        sync 2>/dev/null || true
        
        # Try multiple fallback methods
        local methods_tried=0
        
        # Method 1: Check root filesystem
        methods_tried=$((methods_tried + 1))
        log_info "Method $methods_tried: Checking root filesystem..."
        available_space_kb=$(df / 2>/dev/null | awk 'NR==2{print $4}' | tr -d '\n\r\t ' || echo "0")
        if [[ ! "$available_space_kb" =~ ^[0-9]+$ ]]; then
            available_space_kb=0
        fi
        available_space_gb=$((available_space_kb / 1024 / 1024))
        
        if [[ $available_space_gb -eq 0 ]]; then
            # Method 2: Use df with different options
            methods_tried=$((methods_tried + 1))
            log_info "Method $methods_tried: Using df with POSIX output..."
            available_space_kb=$(df -P / 2>/dev/null | awk 'NR==2{print $4}' | tr -d '\n\r\t ' || echo "0")
            if [[ ! "$available_space_kb" =~ ^[0-9]+$ ]]; then
                available_space_kb=0
            fi
            available_space_gb=$((available_space_kb / 1024 / 1024))
        fi
        
        if [[ $available_space_gb -eq 0 ]]; then
            # Method 3: Use stat command (if available)
            methods_tried=$((methods_tried + 1))
            log_info "Method $methods_tried: Using stat command..."
            if command -v stat &>/dev/null; then
                local available_space_bytes
                available_space_bytes=$(stat -f --format="%a*%S" / 2>/dev/null | bc 2>/dev/null || echo "0")
                available_space_gb=$((available_space_bytes / 1024 / 1024 / 1024 || 0))
            fi
        fi
        
        if [[ $available_space_gb -eq 0 ]]; then
            # Method 4: Use statvfs if available
            methods_tried=$((methods_tried + 1))
            log_info "Method $methods_tried: Checking with statvfs..."
            if command -v python3 &>/dev/null; then
                available_space_gb=$(python3 -c "
import os
try:
    stat = os.statvfs('/')
    available_bytes = stat.f_bavail * stat.f_frsize
    print(int(available_bytes / (1024**3)))
except:
    print(0)
" 2>/dev/null || echo "0")
            fi
        fi
        
        if [[ $available_space_gb -gt 0 ]]; then
            log_success "Alternative disk check succeeded: ${available_space_gb}GB available (Method $methods_tried)"
        else
            log_warning "All disk check methods failed, attempting manual verification..."
            
            # Try to create a test directory to verify write access
            local test_dir="/tmp/infracore-space-test-$$"
            if mkdir -p "$test_dir" 2>/dev/null; then
                # If we can create a directory, assume we have some space
                rmdir "$test_dir" 2>/dev/null || true
                available_space_gb=10  # Assume at least 10GB if basic operations work
                log_warning "Disk check failed but write test succeeded, assuming ${available_space_gb}GB available"
            else
                log_error "Cannot determine disk space and write test failed"
            fi
        fi
    fi
    
    if [[ $available_space_gb -lt 5 ]]; then
        log_error "Insufficient disk space: ${available_space_gb}GB available, 5GB required"
        log_info "Checked directory: $check_dir"
        log_info "If this appears incorrect, try: 'df -h $check_dir' manually"
        return 1
    fi
    log_success "Disk space check passed: ${available_space_gb}GB available"
    
    # Check minimum memory (1GB)
    local total_memory_kb
    total_memory_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}' || echo "0")
    local total_memory_gb=$((total_memory_kb / 1024 / 1024))
    
    if [[ $total_memory_gb -lt 1 ]]; then
        log_error "Insufficient memory: ${total_memory_gb}GB available, 1GB required"
        return 1
    fi
    log_success "Memory check passed: ${total_memory_gb}GB available"
    
    # Check system architecture
    local arch
    arch=$(uname -m)
    if [[ ! "$arch" =~ ^(x86_64|amd64)$ ]]; then
        log_warning "Unsupported architecture: $arch (x86_64 recommended)"
    else
        log_success "Architecture check passed: $arch"
    fi
    
    # Check OS compatibility
    local os_info=""
    if [[ -f /etc/os-release ]]; then
        os_info=$(grep PRETTY_NAME /etc/os-release | cut -d'"' -f2)
        log_success "OS detected: $os_info"
        
        # Check for supported distributions
        if grep -qi "ubuntu\|debian\|centos\|rhel\|fedora\|rocky\|alma" /etc/os-release; then
            log_success "OS compatibility verified"
        else
            log_warning "Unsupported OS detected, proceeding with caution"
        fi
    else
        log_warning "Could not detect OS information"
    fi
    
    # Check required ports availability
    local required_ports=(80 443 8080 3000)
    for port in "${required_ports[@]}"; do
        if command -v ss &>/dev/null; then
            if ss -tuln | grep -q ":$port "; then
                log_warning "Port $port is already in use"
            else
                log_success "Port $port is available"
            fi
        elif command -v netstat &>/dev/null; then
            if netstat -tuln | grep -q ":$port "; then
                log_warning "Port $port is already in use"  
            else
                log_success "Port $port is available"
            fi
        fi
    done
    
    # Check kernel version for Docker compatibility
    local kernel_version
    kernel_version=$(uname -r | cut -d. -f1-2)
    local kernel_major
    kernel_major=$(echo "$kernel_version" | cut -d. -f1)
    local kernel_minor
    kernel_minor=$(echo "$kernel_version" | cut -d. -f2)
    
    if [[ $kernel_major -gt 3 ]] || [[ $kernel_major -eq 3 && $kernel_minor -ge 10 ]]; then
        log_success "Kernel version compatible: $kernel_version"
    else
        log_warning "Kernel version may not be fully compatible: $kernel_version"
    fi
    
    # Test network connectivity
    retry_with_backoff 2 5 "network connectivity test" test_network_connectivity
    
    log_success "System validation completed"
}

# Enhanced dependency verification
validate_dependencies() {
    log_step "Validating dependencies..."
    
    local required_commands=()
    
    # Always required
    required_commands+=("curl" "git" "wget" "unzip" "jq")
    
    # Docker-specific requirements
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        required_commands+=("docker" "docker-compose")
        
        # Check Docker daemon
        if command -v docker &>/dev/null; then
            if ! retry_with_backoff 3 5 "Docker daemon check" docker info; then
                log_error "Docker daemon is not running or accessible"
                return 1
            fi
            
            # Check Docker version
            local docker_version
            docker_version=$(docker --version | grep -oE '[0-9]+\.[0-9]+' | head -1)
            log_success "Docker version: $docker_version"
            
            # Check Docker Compose version
            if command -v docker-compose &>/dev/null; then
                local compose_version
                compose_version=$(docker-compose --version | grep -oE '[0-9]+\.[0-9]+' | head -1)
                log_success "Docker Compose version: $compose_version"
            fi
        fi
    else
        required_commands+=("go" "node" "npm")
    fi
    
    # Validate all required commands
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &>/dev/null; then
            log_error "Required command not found: $cmd"
            log_info "Please install $cmd and try again"
            return 1
        else
            local cmd_version=""
            case "$cmd" in
                "go") cmd_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1) ;;
                "node") cmd_version="v$(node --version | tr -d 'v')" ;;
                "npm") cmd_version="v$(npm --version)" ;;
                "git") cmd_version="v$(git --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)" ;;
                *) cmd_version="installed" ;;
            esac
            log_success "$cmd is available ($cmd_version)"
        fi
    done
    
    log_success "Dependency validation completed"
}

log_error() {
    ensure_log_dir
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    ensure_log_dir
    echo -e "${PURPLE}[STEP]${NC} $1" | tee -a "$LOG_FILE" 2>/dev/null || echo -e "${PURPLE}[STEP]${NC} $1"
}

# Show usage information
show_usage() {
    cat << EOF
InfraCore Linux Server Deployment Script

Usage: $0 [OPTIONS]

Options:
    -h, --help              Show this help message
    -b, --branch BRANCH     Git branch to deploy (default: main)
    -e, --env ENVIRONMENT   Environment to deploy (default: production)
    -d, --dir DIRECTORY     Deployment directory (default: /opt/infra-core)
    -u, --user USERNAME     Service user (default: infracore)
    -r, --repo URL          Repository URL (default: https://github.com/last-emo-boy/infra-core.git)
    --docker                Use Docker deployment (default)
    --binary                Use binary deployment
    --backup                Create backup before deployment
    --mirror [REGION]       Use mirror sources for faster builds
                           REGION: cn (China), us (US), eu (Europe), auto (detect)
                           If no region specified, uses auto-detection
    --rollback              Rollback to previous deployment
    --status                Show deployment status
    --logs                  Show service logs
    --test-mirrors          Test mirror speeds without deploying
    --test-install          Run installation verification tests
    --test-api              Run API functionality tests
    --troubleshoot          Run troubleshooting diagnostics
    --restart               Restart services
    --stop                  Stop services
    --start                 Start services
    --update                Update to latest version without full deployment
    --non-interactive       Run in non-interactive mode with default values
    --configure             Run interactive configuration setup only
    --validate-config       Validate and repair configuration files
    --upgrade               Interactive upgrade with confirmation prompt
    --uninstall             Completely uninstall InfraCore and cleanup all components

Examples:
    $0                                    # Deploy latest main branch (no mirrors)
    $0 --mirror cn                        # Deploy with Chinese mirrors
    $0 --mirror                           # Deploy with auto-detected mirrors
    $0 --branch develop --env staging     # Deploy develop branch to staging
    $0 --backup --mirror cn               # Deploy with backup and Chinese mirrors
    $0 --upgrade --mirror                 # Interactive upgrade with mirrors
    $0 --status                          # Show current status
    $0 --logs                            # Show service logs
    $0 --test-mirrors                    # Test mirror speeds
    $0 --test-install                    # Verify installation
    $0 --test-api                        # Test API functions
    $0 --troubleshoot                    # Run diagnostics
    $0 --update                          # Quick update to latest version
    $0 --uninstall                       # Completely remove InfraCore
    $0 --configure                       # Configure settings only
    NON_INTERACTIVE=true $0 --mirror     # Deploy without prompts

Environment Variables:
    JWT_SECRET             JWT secret key (auto-generated if not set)
    ACME_EMAIL             Email for SSL certificate registration
    NON_INTERACTIVE        Skip interactive prompts (true/false)
    CUSTOM_DOMAIN          Custom domain name for the service
    CUSTOM_EMAIL           Custom admin email address
    CUSTOM_HTTP_PORT       Custom HTTP port (default: 80)
    CUSTOM_HTTPS_PORT      Custom HTTPS port (default: 443)
    CUSTOM_API_PORT        Custom API port (default: 8082)

Configuration:
    The script supports interactive configuration where you can customize:
    • Domain name and admin email for SSL certificates
    • Service ports (HTTP, HTTPS, API)
    • Resource limits (memory, CPU)
    • Backup settings
    
    Use --configure to set up configuration, or set NON_INTERACTIVE=true
    to use defaults automatically.

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -b|--branch)
                BRANCH="$2"
                shift 2
                ;;
            -e|--env)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -d|--dir)
                DEPLOY_DIR="$2"
                shift 2
                ;;
            -u|--user)
                SERVICE_USER="$2"
                shift 2
                ;;
            -r|--repo)
                REPO_URL="$2"
                shift 2
                ;;
            --docker)
                DEPLOYMENT_TYPE="docker"
                shift
                ;;
            --binary)
                DEPLOYMENT_TYPE="binary"
                shift
                ;;
            --backup)
                CREATE_BACKUP=true
                shift
                ;;
            --mirror)
                USE_MIRRORS=true
                if [[ -n "$2" && ! "$2" =~ ^-- ]]; then
                    MIRROR_REGION="$2"
                    shift 2
                else
                    MIRROR_REGION="auto"
                    shift
                fi
                ;;
            --upgrade)
                ACTION="upgrade"
                shift
                ;;
            --rollback)
                ACTION="rollback"
                shift
                ;;
            --status)
                ACTION="status"
                shift
                ;;
            --logs)
                ACTION="logs"
                shift
                ;;
            --test-mirrors)
                ACTION="test-mirrors"
                shift
                ;;
            --test-install)
                ACTION="test-install"
                shift
                ;;
            --test-api)
                ACTION="test-api"
                shift
                ;;
            --troubleshoot)
                ACTION="troubleshoot"
                shift
                ;;
            --restart)
                ACTION="restart"
                shift
                ;;
            --stop)
                ACTION="stop"
                shift
                ;;
            --start)
                ACTION="start"
                shift
                ;;
            --update)
                ACTION="update"
                shift
                ;;
            --uninstall)
                ACTION="uninstall"
                shift
                ;;
            --non-interactive)
                NON_INTERACTIVE=true
                shift
                ;;
            --configure)
                ACTION="configure"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        log_info "Running as root - OK"
        return 0
    else
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check system requirements
check_requirements() {
    log_step "Checking system requirements..."
    
    # Check OS
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot determine OS version"
        exit 1
    fi
    
    . /etc/os-release
    log_info "OS: $NAME $VERSION"
    
    # Check required commands
    local required_commands=("curl" "git" "systemctl")
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        required_commands+=("docker" "docker-compose")
    else
        required_commands+=("go" "node" "npm")
    fi
    
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            log_error "Required command not found: $cmd"
            log_info "Please install $cmd and try again"
            exit 1
        else
            log_success "$cmd is available"
        fi
    done
    
    # Check Docker if needed
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        if ! docker info &> /dev/null; then
            log_error "Docker is not running"
            log_info "Please start Docker service: sudo systemctl start docker"
            exit 1
        fi
        log_success "Docker is running"
    fi
}

# Install system dependencies
install_dependencies() {
    log_step "Installing system dependencies..."
    
    # Update package manager
    if command -v apt-get &> /dev/null; then
        apt-get update
        apt-get install -y curl git wget unzip jq
        
        if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
            # Install Docker if not present
            if ! command -v docker &> /dev/null; then
                log_info "Installing Docker..."
                # Add Docker GPG key and repository with enhanced error handling
                apt-get install -y ca-certificates curl gnupg lsb-release
                mkdir -p /etc/apt/keyrings
                
                # Remove any existing Docker GPG key and repository
                rm -f /etc/apt/keyrings/docker.gpg
                rm -f /etc/apt/sources.list.d/docker.list
                
                # Try to get the Docker GPG key with multiple methods
                log_info "Adding Docker GPG key..."
                local gpg_success=false
                
                # Method 1: Try the official Docker GPG key
                if curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null; then
                    gpg_success=true
                    log_success "Docker GPG key added successfully"
                else
                    log_warning "Failed to add Docker GPG key via method 1, trying alternative..."
                    
                    # Method 2: Try with explicit keyserver
                    if gpg --keyserver keyserver.ubuntu.com --recv-keys 7EA0A9C3F273FCD8 2>/dev/null && 
                       gpg --export 7EA0A9C3F273FCD8 | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null; then
                        gpg_success=true
                        log_success "Docker GPG key added via keyserver"
                    else
                        log_warning "Failed to add Docker GPG key via keyserver, trying wget..."
                        
                        # Method 3: Try with wget as fallback
                        if wget -qO- https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null; then
                            gpg_success=true
                            log_success "Docker GPG key added via wget"
                        fi
                    fi
                fi
                
                if [[ "$gpg_success" == "true" ]]; then
                    # Set proper permissions
                    chmod 644 /etc/apt/keyrings/docker.gpg
                    
                    # Add Docker repository
                    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
                    
                    # Update package list with retry
                    log_info "Updating package list..."
                    local update_success=false
                    for attempt in {1..3}; do
                        if apt-get update 2>/dev/null; then
                            update_success=true
                            break
                        else
                            log_warning "Package update attempt $attempt failed, retrying..."
                            sleep 2
                        fi
                    done
                    
                    if [[ "$update_success" == "true" ]]; then
                        apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
                        systemctl enable docker
                        systemctl start docker
                        log_success "Docker installed successfully"
                    else
                        log_error "Failed to update package list after adding Docker repository"
                        return 1
                    fi
                else
                    log_error "Failed to add Docker GPG key with all methods"
                    return 1
                fi
            fi
            
            # Install Docker Compose if not present
            if ! command -v docker-compose &> /dev/null; then
                log_info "Installing Docker Compose..."
                local compose_success=false
                
                # Try to install Docker Compose with retry
                for attempt in {1..3}; do
                    if curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose 2>/dev/null; then
                        chmod +x /usr/local/bin/docker-compose
                        if /usr/local/bin/docker-compose version &>/dev/null; then
                            compose_success=true
                            log_success "Docker Compose installed successfully"
                            break
                        fi
                    fi
                    log_warning "Docker Compose installation attempt $attempt failed, retrying..."
                    sleep 2
                done
                
                if [[ "$compose_success" != "true" ]]; then
                    log_error "Failed to install Docker Compose after multiple attempts"
                    return 1
                fi
            fi
            
            # Verify Docker installation
            if ! verify_docker_installation; then
                log_error "Docker installation verification failed"
                return 1
            fi
            
            # Verify Docker installation
            if ! verify_docker_installation; then
                log_error "Docker installation verification failed"
                return 1
            fi
        fi
    elif command -v yum &> /dev/null; then
        yum update -y
        yum install -y curl git wget unzip jq
        
        if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
            if ! command -v docker &> /dev/null; then
                yum install -y docker
                systemctl enable docker
                systemctl start docker
            fi
        fi
    else
        log_warning "Unsupported package manager. Please install dependencies manually."
    fi
}

# Setup service user
setup_user() {
    log_step "Setting up service user..."
    
    if ! id "$SERVICE_USER" &>/dev/null; then
        log_info "Creating user: $SERVICE_USER"
        useradd -r -s /bin/false -d "$DEPLOY_DIR" "$SERVICE_USER"
    else
        log_info "User $SERVICE_USER already exists"
    fi
    
    # Add user to docker group if using Docker
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        usermod -aG docker "$SERVICE_USER"
    fi
}

# Setup directories
setup_directories() {
    log_step "Setting up directories..."
    
    local dirs=(
        "$DEPLOY_DIR"
        "$BACKUP_DIR" 
        "/var/log/infra-core"
        "/etc/infra-core"
        "/var/lib/infra-core"
    )
    
    # Force filesystem sync before creating directories
    sync 2>/dev/null || true
    
    for dir in "${dirs[@]}"; do
        # Check if directory exists and handle any stale mount points or filesystem issues
        if [[ -d "$dir" ]] && [[ ! -w "$dir" ]]; then
            log_warning "Directory exists but is not writable: $dir"
            # Try to fix permissions
            chmod 755 "$dir" 2>/dev/null || log_warning "Could not fix permissions for $dir"
        fi
        
        if [[ ! -d "$dir" ]]; then
            log_info "Creating directory: $dir"
            if ! mkdir -p "$dir" 2>/dev/null; then
                log_error "Failed to create directory: $dir"
                # Try creating parent directories first
                local parent_dir=$(dirname "$dir")
                if [[ ! -d "$parent_dir" ]]; then
                    log_info "Creating parent directory: $parent_dir"
                    mkdir -p "$parent_dir" 2>/dev/null || true
                fi
                # Retry creating the target directory
                if ! mkdir -p "$dir" 2>/dev/null; then
                    log_error "Still cannot create directory: $dir"
                    return 1
                fi
            fi
            log_success "Created directory: $dir"
        else
            log_info "Directory already exists: $dir"
        fi
        
        # Verify directory is accessible
        if [[ ! -d "$dir" ]] || [[ ! -w "$dir" ]]; then
            log_error "Directory is not accessible or writable: $dir"
            return 1
        fi
        
        # Set ownership and permissions
        if ! chown "$SERVICE_USER:$SERVICE_USER" "$dir" 2>/dev/null; then
            log_warning "Could not set ownership for $dir (user may not exist yet)"
        fi
        
        if ! chmod 755 "$dir" 2>/dev/null; then
            log_warning "Could not set permissions for $dir"
        fi
    done
    
    # Setup log file with improved error handling
    local log_dir=$(dirname "$LOG_FILE")
    if [[ ! -d "$log_dir" ]]; then
        mkdir -p "$log_dir" 2>/dev/null || true
    fi
    
    if ! touch "$LOG_FILE" 2>/dev/null; then
        log_warning "Could not create log file: $LOG_FILE"
        # Try alternative log location
        LOG_FILE="/tmp/infra-core-deploy.log"
        touch "$LOG_FILE" 2>/dev/null || log_warning "Could not create alternative log file"
    fi
    
    # Set log file permissions if possible
    if [[ -f "$LOG_FILE" ]]; then
        chown "$SERVICE_USER:$SERVICE_USER" "$LOG_FILE" 2>/dev/null || true
        chmod 644 "$LOG_FILE" 2>/dev/null || true
    fi
    
    # Final sync to ensure all directory operations are committed
    sync 2>/dev/null || true
    
    log_success "Directory setup completed"
}

# Enhanced backup with validation and compression
create_backup() {
    if [[ "$CREATE_BACKUP" == "true" ]]; then
        log_step "Creating comprehensive backup..."
        
        local backup_name="infra-core-backup-$(date +%Y%m%d-%H%M%S)"
        local backup_path="$BACKUP_DIR/$backup_name"
        local temp_backup_dir="/tmp/infra-backup-$$"
        
        # Check available space for backup
        local source_size=0
        if [[ -d "$DEPLOY_DIR/current" ]]; then
            source_size=$(du -sb "$DEPLOY_DIR/current" 2>/dev/null | cut -f1 | tr -d '\n\r\t ' || echo "0")
            if [[ ! "$source_size" =~ ^[0-9]+$ ]]; then
                source_size=0
            fi
        fi
        
        local available_space
        available_space=$(df "$BACKUP_DIR" | awk 'NR==2{print $4}' | head -1 | tr -d '\n\r\t ' || echo "0")
        if [[ ! "$available_space" =~ ^[0-9]+$ ]]; then
            available_space=0
        fi
        available_space=$((available_space * 1024))
        
        if [[ $source_size -gt $((available_space / 2)) ]]; then
            log_error "Insufficient space for backup. Need: $((source_size / 1024 / 1024))MB, Available: $((available_space / 1024 / 1024))MB"
            return 1
        fi
        
        if [[ -d "$DEPLOY_DIR/current" ]]; then
            log_info "Creating backup: $backup_name"
            log_info "Estimated source size: $((source_size / 1024 / 1024))MB"
            log_info "Available space: $((available_space / 1024 / 1024))MB"
            
            mkdir -p "$backup_path" "$temp_backup_dir"
            
            # Start backup monitoring in background
            monitor_backup_progress "$temp_backup_dir" &
            local monitor_pid=$!
            
            # Ensure monitor cleanup on exit
            trap "kill $monitor_pid 2>/dev/null || true" EXIT
            
            # Create manifest file
            local manifest_file="$temp_backup_dir/backup-manifest.json"
            cat > "$manifest_file" << EOF
{
    "backup_name": "$backup_name",
    "timestamp": "$(date -Iseconds)",
    "hostname": "$(hostname)",
    "user": "$(whoami)",
    "source_path": "$DEPLOY_DIR/current",
    "backup_type": "full",
    "files": []
}
EOF
            
            # Backup application files with verification
            log_info "Backing up application files..."
            if retry_with_backoff 2 3 "application backup" cp -r "$DEPLOY_DIR/current" "$temp_backup_dir/"; then
                log_success "Application files backed up successfully"
                
                # Generate file checksums with timeout
                log_info "Generating checksums..."
                if timeout 300 find "$temp_backup_dir/current" -type f -exec sha256sum {} \; > "$temp_backup_dir/checksums.sha256" 2>/dev/null; then
                    log_success "Checksums generated successfully"
                else
                    log_warning "Checksum generation timed out, continuing without checksums"
                    echo "# Checksum generation timed out" > "$temp_backup_dir/checksums.sha256"
                fi
                
                # Update manifest with file list
                log_info "Creating file manifest..."
                if timeout 60 find "$temp_backup_dir/current" -type f | jq -R . | jq -s . > "$temp_backup_dir/file_list.json" 2>/dev/null; then
                    log_success "File manifest created"
                else
                    log_warning "File manifest creation failed, using simple list"
                    find "$temp_backup_dir/current" -type f > "$temp_backup_dir/file_list.txt" 2>/dev/null || true
                fi
            else
                log_error "Failed to backup application files"
                rm -rf "$temp_backup_dir"
                return 1
            fi
            
            # Backup database if exists
            local db_path="/var/lib/infra-core/database.db"
            if [[ -f "$db_path" ]]; then
                log_info "Backing up database..."
                if retry_with_backoff 2 3 "database backup" cp "$db_path" "$temp_backup_dir/database.db"; then
                    sha256sum "$temp_backup_dir/database.db" >> "$temp_backup_dir/checksums.sha256"
                    log_success "Database backed up successfully"
                fi
            fi
            
            # Backup Docker volumes if using Docker
            if [[ "$DEPLOYMENT_TYPE" == "docker" ]] && command -v docker &>/dev/null; then
                log_info "Backing up Docker volumes..."
                local volumes_backup="$temp_backup_dir/docker_volumes"
                mkdir -p "$volumes_backup"
                
                # Check if Docker daemon is accessible
                if ! docker info &>/dev/null; then
                    log_warning "Docker daemon not accessible, skipping volume backup"
                else
                    # Get list of relevant volumes with timeout
                    log_info "Scanning for infra-core related volumes..."
                    local volume_list
                    if volume_list=$(timeout 30 docker volume ls -q 2>/dev/null | grep -E "(infra|core)" || true); then
                        if [[ -n "$volume_list" ]]; then
                            log_info "Found volumes: $(echo "$volume_list" | tr '\n' ' ')"
                            
                            # Ensure alpine image is available
                            log_info "Preparing backup environment..."
                            if ! timeout 60 docker pull alpine:latest &>/dev/null; then
                                log_warning "Failed to pull alpine image, using existing or falling back"
                            fi
                            
                            # Process each volume with individual timeout
                            echo "$volume_list" | while IFS= read -r volume; do
                                if [[ -n "$volume" ]] && [[ "$volume" != " " ]]; then
                                    log_info "Backing up volume: $volume"
                                    
                                    # Backup volume with timeout and better error handling
                                    if timeout 120 docker run --rm \
                                        --network none \
                                        -v "$volume:/source:ro" \
                                        -v "$volumes_backup:/backup" \
                                        alpine:latest \
                                        sh -c "cd /source && tar czf /backup/$volume.tar.gz . 2>/dev/null || exit 0" &>/dev/null; then
                                        log_success "Volume $volume backed up successfully"
                                    else
                                        log_warning "Failed to backup volume: $volume (continuing...)"
                                    fi
                                fi
                            done
                            log_success "Docker volumes backup completed"
                        else
                            log_info "No infra-core related volumes found"
                        fi
                    else
                        log_warning "Failed to list Docker volumes (timeout or error)"
                    fi
                fi
            fi
            
            # Backup configuration files
            log_info "Backing up configuration files..."
            local config_backup="$temp_backup_dir/config"
            mkdir -p "$config_backup"
            
            # Common config locations
            local config_files=(
                "/etc/nginx/sites-available/infra-core"
                "/etc/systemd/system/infra-core.service"
                "$DEPLOY_DIR/.env*"
                "$DEPLOY_DIR/docker-compose*.yml"
            )
            
            for config_file in "${config_files[@]}"; do
                if [[ -f "$config_file" ]]; then
                    cp "$config_file" "$config_backup/" 2>/dev/null || true
                fi
            done
            
            # Compress backup with timeout and monitoring
            log_info "Compressing backup..."
            local backup_size=$(du -sh "$temp_backup_dir" | cut -f1)
            log_info "Backup size: $backup_size"
            
            # Use timeout to prevent hanging
            if timeout 600 tar czf "$backup_path.tar.gz" -C "$temp_backup_dir" . 2>/dev/null; then
                # Verify backup integrity with timeout
                log_info "Verifying backup integrity..."
                if timeout 120 tar tzf "$backup_path.tar.gz" >/dev/null 2>&1; then
                    local final_size=$(ls -lh "$backup_path.tar.gz" | awk '{print $5}')
                    log_success "Backup created and verified: $backup_name.tar.gz ($final_size)"
                    
                    # Create symlink to latest backup
                    ln -sf "$backup_name.tar.gz" "$BACKUP_DIR/latest-backup.tar.gz"
                    
                    # Cleanup old backups (keep last 5)
                    cleanup_old_backups
                    
                    # Stop monitoring
                    kill $monitor_pid 2>/dev/null || true
                    trap - EXIT
                else
                    log_error "Backup verification failed (integrity check timed out or failed)"
                    rm -f "$backup_path.tar.gz"
                    rm -rf "$temp_backup_dir"
                    
                    # Stop monitoring
                    kill $monitor_pid 2>/dev/null || true
                    trap - EXIT
                    return 1
                fi
            else
                log_error "Backup compression failed (timeout or error)"
                rm -rf "$temp_backup_dir"
                
                # Stop monitoring
                kill $monitor_pid 2>/dev/null || true
                trap - EXIT
                return 1
            fi
            
            # Cleanup temp directory
            rm -rf "$temp_backup_dir"
            
            # Stop monitoring
            kill $monitor_pid 2>/dev/null || true
            trap - EXIT
            if [[ -d "/etc/infra-core" ]]; then
                cp -r "/etc/infra-core" "$backup_path/"
            fi
            
            # Create backup metadata
            cat > "$backup_path/metadata.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "branch": "$BRANCH",
    "environment": "$ENVIRONMENT",
    "deployment_type": "$DEPLOYMENT_TYPE",
    "commit": "$(cd "$DEPLOY_DIR/current" 2>/dev/null && git rev-parse HEAD 2>/dev/null || echo 'unknown')"
}
EOF
            
            chown -R "$SERVICE_USER:$SERVICE_USER" "$backup_path"
            log_success "Backup created: $backup_name"
            
            # Keep only last 10 backups
            cd "$BACKUP_DIR"
            ls -1t | tail -n +11 | xargs -r rm -rf
        else
            log_info "No current deployment to backup"
        fi
    fi
}

# Cleanup old backups with retention policy
cleanup_old_backups() {
    log_info "Cleaning up old backups..."
    
    local max_backups=${BACKUP_RETENTION:-10}
    local backup_count
    
    if [[ -d "$BACKUP_DIR" ]]; then
        # Count existing backups
        backup_count=$(find "$BACKUP_DIR" -name "infra-core-backup-*.tar.gz" | wc -l)
        
        if [[ $backup_count -gt $max_backups ]]; then
            log_info "Found $backup_count backups, keeping latest $max_backups"
            
            # Remove oldest backups
            find "$BACKUP_DIR" -name "infra-core-backup-*.tar.gz" -printf '%T@ %p\n' | \
                sort -n | head -n -"$max_backups" | cut -d' ' -f2- | \
                xargs -r rm -f
                
            log_success "Cleaned up $((backup_count - max_backups)) old backups"
        else
            log_info "Backup count ($backup_count) within retention limit ($max_backups)"
        fi
        
        # Cleanup failed/incomplete backups
        find "$BACKUP_DIR" -name "*.tmp" -o -name "*.partial" | xargs -r rm -f
        
        # Show backup storage usage
        local backup_size
        backup_size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1 || echo "0B")
        log_info "Total backup storage: $backup_size"
    fi
}

# Validate backup before rollback
validate_backup() {
    local backup_path="$1"
    
    if [[ ! -f "$backup_path" ]]; then
        log_error "Backup file not found: $backup_path"
        return 1
    fi
    
    # Check if backup is a valid tar.gz file
    if ! tar tzf "$backup_path" >/dev/null 2>&1; then
        log_error "Backup file is corrupted or invalid: $backup_path"
        return 1
    fi
    
    # Check if backup contains required files
    local required_files=("current/" "backup-manifest.json")
    for required_file in "${required_files[@]}"; do
        if ! tar tzf "$backup_path" | grep -q "^$required_file"; then
            log_error "Backup missing required file: $required_file"
            return 1
        fi
    done
    
    # Verify checksums if available
    if tar tzf "$backup_path" | grep -q "checksums.sha256"; then
        log_info "Verifying backup checksums..."
        local temp_verify="/tmp/backup-verify-$$"
        mkdir -p "$temp_verify"
        
        if tar xzf "$backup_path" -C "$temp_verify" checksums.sha256 current/ 2>/dev/null; then
            cd "$temp_verify"
            if sha256sum -c checksums.sha256 --quiet 2>/dev/null; then
                log_success "Backup checksum verification passed"
                rm -rf "$temp_verify"
                return 0
            else
                log_warning "Some files failed checksum verification"
                rm -rf "$temp_verify"
                return 1
            fi
        else
            log_warning "Could not extract backup for verification"
            rm -rf "$temp_verify"
        fi
    fi
    
    log_success "Backup validation completed"
    return 0
}

# Enhanced rollback with backup validation and safety checks
safe_rollback() {
    local backup_name="$1"
    log_step "Performing safe rollback to backup: $backup_name"
    
    # Find backup file
    local backup_file=""
    if [[ -f "$BACKUP_DIR/$backup_name.tar.gz" ]]; then
        backup_file="$BACKUP_DIR/$backup_name.tar.gz"
    elif [[ -f "$BACKUP_DIR/$backup_name" ]]; then
        backup_file="$BACKUP_DIR/$backup_name"
    elif [[ "$backup_name" == "latest" ]] && [[ -L "$BACKUP_DIR/latest-backup.tar.gz" ]]; then
        backup_file="$BACKUP_DIR/latest-backup.tar.gz"
    else
        log_error "Backup not found: $backup_name"
        return 1
    fi
    
    # Validate backup before proceeding
    if ! validate_backup "$backup_file"; then
        log_error "Backup validation failed, aborting rollback"
        return 1
    fi
    
    # Create safety backup of current state
    log_info "Creating safety backup of current state..."
    local safety_backup="$BACKUP_DIR/pre-rollback-$(date +%Y%m%d-%H%M%S)"
    if [[ -d "$DEPLOY_DIR/current" ]]; then
        tar czf "$safety_backup.tar.gz" -C "$DEPLOY_DIR" current/ || {
            log_warning "Failed to create safety backup"
        }
    fi
    
    # Stop services gracefully
    log_info "Stopping services for rollback..."
    if ! retry_with_backoff 3 5 "service stop" stop_services; then
        log_error "Failed to stop services, aborting rollback"
        return 1
    fi
    
    # Extract backup to temporary location
    local temp_restore="/tmp/restore-$$"
    mkdir -p "$temp_restore"
    
    log_info "Extracting backup..."
    if ! tar xzf "$backup_file" -C "$temp_restore"; then
        log_error "Failed to extract backup"
        rm -rf "$temp_restore"
        return 1
    fi
    
    # Backup current deployment and restore from backup
    if [[ -d "$DEPLOY_DIR/current" ]]; then
        if [[ -d "$DEPLOY_DIR/previous" ]]; then
            rm -rf "$DEPLOY_DIR/previous"
        fi
        mv "$DEPLOY_DIR/current" "$DEPLOY_DIR/previous"
    fi
    
    # Move restored files to current location
    if [[ -d "$temp_restore/current" ]]; then
        mv "$temp_restore/current" "$DEPLOY_DIR/current"
        chown -R "$SERVICE_USER:$SERVICE_USER" "$DEPLOY_DIR/current"
        
        # Restore database if present in backup
        if [[ -f "$temp_restore/database.db" ]]; then
            log_info "Restoring database..."
            mkdir -p "/var/lib/infra-core"
            cp "$temp_restore/database.db" "/var/lib/infra-core/database.db"
            chown "$SERVICE_USER:$SERVICE_USER" "/var/lib/infra-core/database.db"
        fi
        
        # Restore Docker volumes if present
        if [[ -d "$temp_restore/docker_volumes" ]] && command -v docker &>/dev/null; then
            log_info "Restoring Docker volumes..."
            for volume_archive in "$temp_restore/docker_volumes"/*.tar.gz; do
                if [[ -f "$volume_archive" ]]; then
                    local volume_name
                    volume_name=$(basename "$volume_archive" .tar.gz)
                    
                    # Create volume if it doesn't exist
                    docker volume create "$volume_name" 2>/dev/null || true
                    
                    # Restore volume content
                    docker run --rm \
                        -v "$volume_name:/target" \
                        -v "$temp_restore/docker_volumes:/backup" \
                        alpine:latest \
                        sh -c "cd /target && tar xzf /backup/$volume_name.tar.gz" 2>/dev/null || true
                fi
            done
        fi
        
        log_success "Backup restored successfully"
    else
        log_error "Invalid backup structure"
        rm -rf "$temp_restore"
        return 1
    fi
    
    # Cleanup temp directory
    rm -rf "$temp_restore"
    
    # Start services
    log_info "Starting services after rollback..."
    if retry_with_backoff 3 10 "service start" start_services; then
        log_success "Rollback completed successfully"
        
        # Verify deployment health
        sleep 10
        if verify_deployment_health; then
            log_success "Rollback verification passed"
        else
            log_warning "Rollback completed but health check failed"
        fi
    else
        log_error "Failed to start services after rollback"
        return 1
    fi
}

# Clone or update repository with optional GitHub mirror
update_repository() {
    log_step "Updating repository..."
    
    local temp_dir="$DEPLOY_DIR/tmp-$(date +%s)"
    local clone_url="$REPO_URL"
    
    # Use GitHub mirror if mirrors are enabled and it's a GitHub repo
    if [[ "$USE_MIRRORS" == "true" && "$REPO_URL" =~ github\.com ]]; then
        local github_mirrors=(
            "https://ghfast.top/"
            "https://github.moeyy.xyz/"
            "https://gh-proxy.com/"
            "https://ghproxy.com/"
        )
        
        log_info "Mirror mode enabled, trying GitHub mirrors..."
        local clone_success=false
        
        for mirror in "${github_mirrors[@]}"; do
            local mirror_url="${mirror}${REPO_URL}"
            log_info "Trying GitHub mirror: $mirror_url"
            
            if timeout 60 git clone --depth 1 --branch "$BRANCH" "$mirror_url" "$temp_dir" 2>/dev/null; then
                log_success "Successfully cloned using mirror: $mirror"
                clone_success=true
                break
            else
                log_warning "Mirror $mirror failed, trying next..."
                rm -rf "$temp_dir" 2>/dev/null || true
            fi
        done
        
        if [[ "$clone_success" != "true" ]]; then
            log_warning "All GitHub mirrors failed, falling back to original URL"
            clone_url="$REPO_URL"
        fi
    fi
    
    # Clone with original URL if mirrors disabled or all mirrors failed
    if [[ ! -d "$temp_dir" ]]; then
        log_info "Cloning repository to: $temp_dir"
        git clone --depth 1 --branch "$BRANCH" "$clone_url" "$temp_dir"
    fi
    
    # Get commit info
    local commit_hash=$(cd "$temp_dir" && git rev-parse HEAD)
    local commit_message=$(cd "$temp_dir" && git log -1 --pretty=format:"%s")
    
    log_info "Cloned commit: $commit_hash"
    log_info "Commit message: $commit_message"
    
    # Move to deployment location
    if [[ -d "$DEPLOY_DIR/current" ]]; then
        # Remove old previous if it exists
        if [[ -d "$DEPLOY_DIR/previous" ]]; then
            rm -rf "$DEPLOY_DIR/previous"
        fi
        mv "$DEPLOY_DIR/current" "$DEPLOY_DIR/previous"
    fi
    mv "$temp_dir" "$DEPLOY_DIR/current"
    
    chown -R "$SERVICE_USER:$SERVICE_USER" "$DEPLOY_DIR/current"
    
    # Create deployment info
    cat > "$DEPLOY_DIR/current/deployment-info.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "branch": "$BRANCH",
    "commit": "$commit_hash",
    "commit_message": "$commit_message",
    "environment": "$ENVIRONMENT",
    "deployment_type": "$DEPLOYMENT_TYPE"
}
EOF
}

# Advanced network environment detection (only when mirrors are enabled)
# Compare floating point times (without bc dependency)
is_time_better() {
    local time1="$1"
    local time2="$2"
    
    # Handle special case of 999 (failure)
    if [[ "$time1" == "999" ]]; then
        return 1  # time1 is not better
    fi
    if [[ "$time2" == "999" ]]; then
        return 0  # time1 is better
    fi
    
    # Convert to integer comparison (multiply by 1000 to handle decimals)
    local int1=$(echo "$time1 * 1000" | awk '{printf "%.0f", $1}')
    local int2=$(echo "$time2 * 1000" | awk '{printf "%.0f", $1}')
    
    [[ "$int1" -lt "$int2" ]]
}

# Test mirror connectivity and speed
test_mirror_speed() {
    local mirror_url="$1"
    local test_path="$2"
    local timeout_sec="${3:-8}"
    
    # Test connectivity and measure response time
    local response_time=$(timeout "$timeout_sec" curl -s -w "%{time_total},%{http_code}" -o /dev/null "$mirror_url$test_path" 2>/dev/null || echo "999,000")
    local time_total=$(echo "$response_time" | cut -d',' -f1)
    local http_code=$(echo "$response_time" | cut -d',' -f2)
    
    # Return response time if successful (HTTP 200), otherwise return 999 (failure)
    if [[ "$http_code" == "200" ]]; then
        echo "$time_total"
    else
        echo "999"
    fi
}

# Intelligently select best mirrors based on actual speed tests
select_optimal_mirrors() {
    log_info "🧪 Testing mirror speeds to find the fastest available options..."
    
    # Alpine mirrors to test
    local alpine_mirrors=(
        "https://mirrors.tuna.tsinghua.edu.cn/alpine,清华大学"
        "https://mirrors.ustc.edu.cn/alpine,中科大"
        "https://mirrors.aliyun.com/alpine,阿里云"
        "https://mirror.sjtu.edu.cn/alpine,上海交大"
        "https://dl-cdn.alpinelinux.org/alpine,官方CDN"
        "https://uk.alpinelinux.org/alpine,英国镜像"
    )
    
    # Go proxy mirrors to test
    local go_proxies=(
        "https://goproxy.cn,七牛云"
        "https://goproxy.io,goproxy.io"
        "https://mirrors.aliyun.com/goproxy,阿里云"
        "https://proxy.golang.org,官方代理"
    )
    
    # NPM registry mirrors to test
    local npm_registries=(
        "https://registry.npmmirror.com,淘宝镜像"
        "https://registry.npm.taobao.org,淘宝镜像(旧)"
        "https://r.cnpmjs.org,cnpmjs"
        "https://registry.npmjs.org,官方源"
    )
    
    # Test Alpine mirrors
    log_info "📦 Testing Alpine package mirrors..."
    local best_alpine_url=""
    local best_alpine_name=""
    local best_alpine_time="999"
    
    for mirror_info in "${alpine_mirrors[@]}"; do
        local mirror_url=$(echo "$mirror_info" | cut -d',' -f1)
        local mirror_name=$(echo "$mirror_info" | cut -d',' -f2)
        
        log_info "Testing $mirror_name ($mirror_url)..."
        local response_time=$(test_mirror_speed "$mirror_url" "/v3.18/main/x86_64/APKINDEX.tar.gz" 8)
        
        if is_time_better "$response_time" "$best_alpine_time"; then
            best_alpine_time="$response_time"
            best_alpine_url="$mirror_url"
            best_alpine_name="$mirror_name"
        fi
        
        if [[ "$response_time" == "999" ]]; then
            log_warning "  ❌ $mirror_name: Connection failed"
        else
            log_info "  ✅ $mirror_name: ${response_time}s"
        fi
    done
    
    # Test Go proxies
    log_info "🔧 Testing Go module proxies..."
    local best_go_proxy=""
    local best_go_name=""
    local best_go_time="999"
    
    for proxy_info in "${go_proxies[@]}"; do
        local proxy_url=$(echo "$proxy_info" | cut -d',' -f1)
        local proxy_name=$(echo "$proxy_info" | cut -d',' -f2)
        
        log_info "Testing $proxy_name ($proxy_url)..."
        local response_time=$(test_mirror_speed "$proxy_url" "/github.com/gin-gonic/gin/@v/list" 6)
        
        if is_time_better "$response_time" "$best_go_time"; then
            best_go_time="$response_time"
            best_go_proxy="$proxy_url,direct"
            best_go_name="$proxy_name"
        fi
        
        if [[ "$response_time" == "999" ]]; then
            log_warning "  ❌ $proxy_name: Connection failed"
        else
            log_info "  ✅ $proxy_name: ${response_time}s"
        fi
    done
    
    # Test NPM registries
    log_info "📦 Testing NPM registries..."
    local best_npm_registry=""
    local best_npm_name=""
    local best_npm_time="999"
    
    for registry_info in "${npm_registries[@]}"; do
        local registry_url=$(echo "$registry_info" | cut -d',' -f1)
        local registry_name=$(echo "$registry_info" | cut -d',' -f2)
        
        log_info "Testing $registry_name ($registry_url)..."
        local response_time=$(test_mirror_speed "$registry_url" "/express" 6)
        
        if is_time_better "$response_time" "$best_npm_time"; then
            best_npm_time="$response_time"
            best_npm_registry="$registry_url"
            best_npm_name="$registry_name"
        fi
        
        if [[ "$response_time" == "999" ]]; then
            log_warning "  ❌ $registry_name: Connection failed"
        else
            log_info "  ✅ $registry_name: ${response_time}s"
        fi
    done
    
    # Show results and return optimal configuration
    echo "🎯 Optimal mirror selection results:"
    if [[ "$best_alpine_time" != "999" ]]; then
        echo "ALPINE_MIRROR=$best_alpine_url,ALPINE_NAME=$best_alpine_name,ALPINE_TIME=$best_alpine_time"
        log_success "Best Alpine mirror: $best_alpine_name (${best_alpine_time}s)"
    else
        echo "ALPINE_MIRROR=,ALPINE_NAME=None,ALPINE_TIME=999"
        log_error "No working Alpine mirror found!"
    fi
    
    if [[ "$best_go_time" != "999" ]]; then
        echo "GO_PROXY=$best_go_proxy,GO_NAME=$best_go_name,GO_TIME=$best_go_time"
        log_success "Best Go proxy: $best_go_name (${best_go_time}s)"
    else
        echo "GO_PROXY=https://proxy.golang.org,direct,GO_NAME=Official,GO_TIME=999"
        log_warning "No fast Go proxy found, using official proxy"
    fi
    
    if [[ "$best_npm_time" != "999" ]]; then
        echo "NPM_REGISTRY=$best_npm_registry/,NPM_NAME=$best_npm_name,NPM_TIME=$best_npm_time"
        log_success "Best NPM registry: $best_npm_name (${best_npm_time}s)"
    else
        echo "NPM_REGISTRY=https://registry.npmjs.org/,NPM_NAME=Official,NPM_TIME=999"
        log_warning "No fast NPM registry found, using official registry"
    fi
}

detect_optimal_region() {
    # If mirrors are not enabled, always return global
    if [[ "$USE_MIRRORS" != "true" ]]; then
        echo "global"
        return
    fi
    
    log_info "🌐 Detecting optimal mirror region with intelligent speed testing..."
    
    # Use intelligent mirror selection instead of simple region detection
    local mirror_results=$(select_optimal_mirrors)
    
    # Parse results
    local alpine_result=$(echo "$mirror_results" | grep "ALPINE_MIRROR=" | head -1)
    local alpine_time=$(echo "$alpine_result" | sed 's/.*ALPINE_TIME=\([^,]*\).*/\1/')
    
    # If we found fast Chinese mirrors, use "cn", otherwise use "global"  
    if [[ -n "$alpine_result" ]] && is_time_better "$alpine_time" "3.0"; then
        # Check if it's a Chinese mirror
        if echo "$alpine_result" | grep -E "(tuna|ustc|aliyun|sjtu|cn)" >/dev/null; then
            echo "cn"
            log_success "Selected Chinese region based on speed test results"
        else
            echo "global"
            log_success "Selected global region based on speed test results"
        fi
    else
        echo "global"
        log_info "Using global region as fallback"
    fi
}

# Build Docker images with optional mirror optimization  
build_docker_images() {
    log_info "Mirror settings: USE_MIRRORS=$USE_MIRRORS, MIRROR_REGION=$MIRROR_REGION"
    
    if [[ "$USE_MIRRORS" == "true" ]]; then
        log_step "Building Docker images with mirror optimization..."
        
        # Detect optimal region with enhanced logic
        local region
        if [[ "$MIRROR_REGION" != "auto" ]]; then
            region="$MIRROR_REGION"
            log_info "Using user-specified mirror region: $region"
        else
            region=$(retry_with_backoff 2 3 "network detection" detect_optimal_region) || region="global"
            
            # Ensure we have a valid region
            if [[ -z "$region" ]]; then
                region="global"
                log_warning "Region detection failed, using global mirrors"
            fi
        fi
        
        # Create temporary docker-compose override for build args
        local compose_override="docker-compose.build-override.yml"
        
        # Debug: Show current directory and available files
        log_info "Current directory: $(pwd)"
        log_info "Available compose files: $(ls -la *.yml *.yaml 2>/dev/null || echo 'None found')"
    
    # Get intelligent mirror recommendations
    log_info "🎯 Getting optimal mirror configuration..."
    local mirror_results=$(select_optimal_mirrors)
    
    # Parse the optimal mirror results
    local alpine_line=$(echo "$mirror_results" | grep "ALPINE_MIRROR=" | head -1)
    local go_line=$(echo "$mirror_results" | grep "GO_PROXY=" | head -1)  
    local npm_line=$(echo "$mirror_results" | grep "NPM_REGISTRY=" | head -1)
    
    local optimal_alpine=$(echo "$alpine_line" | sed 's/ALPINE_MIRROR=\([^,]*\).*/\1/')
    local optimal_go=$(echo "$go_line" | sed 's/GO_PROXY=\([^,]*\).*/\1/')
    local optimal_npm=$(echo "$npm_line" | sed 's/NPM_REGISTRY=\([^,]*\).*/\1/')
    
    # Generate optimized docker-compose override based on test results
    log_info "📝 Generating optimized build configuration..."
    cat > "$compose_override" << EOF
version: '3.8'
services:
  infra-core:
    build:
      context: .
      dockerfile: Dockerfile
      target: production-debug
      args:
        - BUILD_REGION=optimized
        - ALPINE_MIRROR=${optimal_alpine}
        - GO_PROXY=${optimal_go}
        - NPM_REGISTRY=${optimal_npm}
EOF

    # Show what we selected
    local alpine_name=$(echo "$alpine_line" | sed 's/.*ALPINE_NAME=\([^,]*\).*/\1/')
    local go_name=$(echo "$go_line" | sed 's/.*GO_NAME=\([^,]*\).*/\1/')
    local npm_name=$(echo "$npm_line" | sed 's/.*NPM_NAME=\([^,]*\).*/\1/')
    
    log_success "🚀 Optimized configuration selected:"
    log_info "  • Alpine packages: $alpine_name ($optimal_alpine)"  
    log_info "  • Go modules: $go_name ($optimal_go)"
    log_info "  • NPM packages: $npm_name ($optimal_npm)"
    
    # Debug: Verify the override file was created and show its contents
    if [[ -f "$compose_override" ]]; then
        log_info "Created build override file: $compose_override"
        log_info "Override file contents:"
        cat "$compose_override" | sed 's/^/  /'
    else
        log_warning "Failed to create override file: $compose_override"
    fi
    
    # Enhanced Docker build with retry and optimization
    log_info "Starting robust Docker build process..."
    
    # Pre-build optimizations
    log_info "Optimizing Docker build environment..."
    
    # Enable buildkit for better performance and caching
    export DOCKER_BUILDKIT=1
    export COMPOSE_DOCKER_CLI_BUILD=1
    
    # Clean up build cache if needed (on build failures)
    local cleanup_cache=false
    
    # Build strategies to try in order
    local build_strategies=(
        "optimized_with_cache"
        "optimized_no_cache" 
        "standard_with_cache"
        "standard_no_cache"
        "fallback_minimal"
    )
    
    local build_success=false
    
    for strategy in "${build_strategies[@]}"; do
        log_info "Trying build strategy: $strategy"
        
        local build_cmd=""
        local timeout_duration="$DOCKER_TIMEOUT"
        
        case "$strategy" in
            "optimized_with_cache")
                if [[ -f "$compose_override" ]]; then
                    build_cmd="docker-compose -f docker-compose.yml -f '$compose_override' build --parallel"
                else
                    build_cmd="docker-compose -f docker-compose.yml build --parallel"
                fi
                timeout_duration=900
                ;;
            "optimized_no_cache")
                if [[ -f "$compose_override" ]]; then
                    build_cmd="docker-compose -f docker-compose.yml -f '$compose_override' build --no-cache --parallel"
                else
                    build_cmd="docker-compose -f docker-compose.yml build --no-cache --parallel"
                fi
                timeout_duration=1200
                ;;
            "standard_with_cache")
                if [[ -f "$compose_override" ]]; then
                    build_cmd="docker-compose -f docker-compose.yml -f '$compose_override' build --parallel"
                else
                    build_cmd="docker-compose -f docker-compose.yml build --parallel"
                fi
                timeout_duration=900
                ;;
            "standard_no_cache")
                if [[ -f "$compose_override" ]]; then
                    build_cmd="docker-compose -f docker-compose.yml -f '$compose_override' build --no-cache --parallel"
                else
                    build_cmd="docker-compose -f docker-compose.yml build --no-cache --parallel"
                fi
                timeout_duration=1200
                ;;
            "fallback_minimal")
                # Clear build cache and try minimal build
                log_warning "Attempting fallback build with cache cleanup..."
                docker builder prune -f 2>/dev/null || true
                docker system prune -f 2>/dev/null || true
                if [[ -f "$compose_override" ]]; then
                    build_cmd="docker-compose -f docker-compose.yml -f '$compose_override' build --no-cache"
                else
                    build_cmd="docker-compose -f docker-compose.yml build --no-cache"
                fi
                timeout_duration=1800
                ;;
        esac
        
        # Execute build command directly
        log_info "Executing: $build_cmd"
        
        if eval "timeout $timeout_duration $build_cmd"; then
            log_success "Docker build completed successfully with strategy: $strategy"
            
            # Simple verification - check if docker-compose can see the services
            if docker-compose -f docker-compose.yml config >/dev/null 2>&1; then
                log_success "Docker compose configuration verified"
                build_success=true
                break
            else
                log_error "Docker compose configuration verification failed"
            fi
        else
            log_warning "Build strategy '$strategy' failed, trying next..."
        fi
        
        # Clean up partial builds on failure
        docker-compose -f docker-compose.yml down --remove-orphans 2>/dev/null || true
        
        # For failed builds with cache, clean cache before next attempt
        if [[ "$strategy" =~ "cache" ]]; then
            docker builder prune -f 2>/dev/null || true
        fi
    done
    
    if [[ "$build_success" != "true" ]]; then
        log_error "All Docker build strategies failed"
        
        # Diagnostic information
        log_error "=== Build Diagnostics ==="
        docker version 2>/dev/null || log_error "Docker version check failed"
        docker info 2>/dev/null || log_error "Docker info check failed" 
        df -h 2>/dev/null || log_error "Disk space check failed"
        
        return 1
    fi
    
    # Post-build optimizations
    log_info "Optimizing built images..."
    
    # Remove intermediate/dangling images
    docker image prune -f 2>/dev/null || true
    
    # Show final image sizes
    log_info "Final image sizes:"
    docker-compose -f docker-compose.yml config --services | while read -r service; do
        local image_name
        image_name=$(docker-compose -f docker-compose.yml config | grep -A 10 "^  $service:" | grep "image:" | awk '{print $2}' | head -1)
        if [[ -n "$image_name" ]]; then
            docker images "$image_name" --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | tail -n +2
        fi
    done
    
        # Cleanup temporary file
        rm -f "$compose_override"
    else
        log_step "Building Docker images with standard configuration..."
        log_info "Mirror optimization disabled. Use --mirror to enable faster builds."
        
        # Build with standard Docker Compose
        if timeout "$DOCKER_TIMEOUT" docker-compose -f docker-compose.yml build; then
            log_success "Docker images built successfully with standard configuration"
        else
            log_error "Docker build failed with standard configuration"
            return 1
        fi
    fi
}

# Comprehensive deployment health verification
verify_deployment_health() {
    log_step "Verifying deployment health..."
    
    local health_score=0
    local max_score=8
    
    # Wait for services to initialize
    sleep 10
    
    # Check Docker containers if using Docker
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]] && command -v docker &>/dev/null; then
        log_info "Checking Docker container health..."
        
        if docker-compose -f docker-compose.yml ps | grep -q "Up"; then
            health_score=$((health_score + 2))
            log_success "Docker containers are running"
        else
            log_error "Docker containers are not running properly"
        fi
    fi
    
    # Check HTTP endpoints
    local endpoints=("http://localhost:8080/health" "http://localhost:3000" "http://localhost:80")
    local healthy_endpoints=0
    
    for endpoint in "${endpoints[@]}"; do
        if curl -s --connect-timeout 5 --max-time 10 "$endpoint" >/dev/null 2>&1; then
            healthy_endpoints=$((healthy_endpoints + 1))
            log_success "Endpoint $endpoint is responding"
        fi
    done
    
    if [[ $healthy_endpoints -gt 0 ]]; then
        health_score=$((health_score + 2))
    fi
    
    # Check disk space
    local disk_usage
    disk_usage=$(df "$DEPLOY_DIR" | awk 'NR==2{print $(NF-1)}' | tr -d '%')
    
    if [[ $disk_usage -lt 90 ]]; then
        health_score=$((health_score + 1))
        log_success "Disk usage is healthy: ${disk_usage}%"
    else
        log_warning "Disk usage is high: ${disk_usage}%"
    fi
    
    # Check memory usage
    local memory_usage
    memory_usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    
    if [[ $memory_usage -lt 85 ]]; then
        health_score=$((health_score + 1))
        log_success "Memory usage is healthy: ${memory_usage}%"
    else
        log_warning "Memory usage is high: ${memory_usage}%"
    fi
    
    # Check file ownership
    if [[ -d "$DEPLOY_DIR/current" ]] && [[ "$(stat -c %U "$DEPLOY_DIR/current")" == "$SERVICE_USER" ]]; then
        health_score=$((health_score + 1))
        log_success "File ownership is correct"
    fi
    
    # Check log files for recent errors
    local error_count=0
    if [[ -f "$LOG_FILE" ]]; then
        error_count=$(tail -20 "$LOG_FILE" | grep -i error | wc -l 2>/dev/null | tr -d ' \n' || echo "0")
        error_count=${error_count//[^0-9]/}
        error_count=${error_count:-0}
    fi
    
    if [[ $error_count -eq 0 ]]; then
        health_score=$((health_score + 1))
        log_success "No recent errors in logs"
    fi
    
    # Calculate health percentage
    local health_percentage=$((health_score * 100 / max_score))
    
    log_info "Health Score: $health_score/$max_score ($health_percentage%)"
    
    if [[ $health_percentage -ge 75 ]]; then
        log_success "Deployment health check PASSED"
        return 0
    else
        log_warning "Deployment health check FAILED"
        return 1
    fi
}

# Docker deployment
deploy_docker() {
    log_step "Deploying with Docker..."
    
    cd "$DEPLOY_DIR/current"
    
    # Login to GitHub Container Registry if credentials available
    if [[ -n "${GITHUB_TOKEN:-}" ]] && [[ -n "${GITHUB_ACTOR:-}" ]]; then
        log_info "Logging into GitHub Container Registry..."
        echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_ACTOR" --password-stdin
    fi
    
    # Use latest image or build locally
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        log_info "Pulling latest image from registry..."
        docker pull "$REGISTRY/$IMAGE_NAME:latest" || {
            log_warning "Failed to pull image, building locally..."
            build_docker_images
        }
    else
        log_info "Building Docker images locally..."
        build_docker_images
    fi
    
    # Setup environment configuration
    setup_environment_config
    
    # Stop existing services
    docker-compose -f docker-compose.yml down || true
    
    # Start services
    log_info "Starting Docker services..."
    docker-compose -f docker-compose.yml up -d
    
    # Wait for services to be ready
    wait_for_services
}

# Binary deployment
deploy_binary() {
    log_step "Deploying with binaries..."
    
    cd "$DEPLOY_DIR/current"
    
    # Install Go if not present
    if ! command -v go &> /dev/null; then
        log_info "Installing Go..."
        local go_version="1.24.5"
        wget "https://golang.org/dl/go${go_version}.linux-amd64.tar.gz"
        rm -rf /usr/local/go
        tar -C /usr/local -xzf "go${go_version}.linux-amd64.tar.gz"
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        export PATH=$PATH:/usr/local/go/bin
        rm "go${go_version}.linux-amd64.tar.gz"
    fi
    
    # Install Node.js if not present
    if ! command -v node &> /dev/null; then
        log_info "Installing Node.js..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
        apt-get install -y nodejs
    fi
    
    # Build application
    log_info "Building Go backend..."
    sudo -u "$SERVICE_USER" go mod download
    sudo -u "$SERVICE_USER" go build -ldflags="-s -w" -o bin/console cmd/console/main.go
    sudo -u "$SERVICE_USER" go build -ldflags="-s -w" -o bin/gate cmd/gate/main.go
    
    log_info "Building React frontend..."
    cd ui
    sudo -u "$SERVICE_USER" npm ci
    sudo -u "$SERVICE_USER" npm run build
    cd ..
    
    # Setup environment configuration
    setup_environment_config
    
    # Install systemd services
    install_systemd_services
    
    # Start services using systemd or start-services script
    if command -v systemctl >/dev/null 2>&1; then
        log_info "Starting services with systemd..."
        systemctl daemon-reload
        systemctl enable infra-core-console
        systemctl enable infra-core-gate
        systemctl restart infra-core-console
        systemctl restart infra-core-gate
    elif [[ -f "$DEPLOY_DIR/current/scripts/start-services.sh" ]]; then
        log_info "Starting services with start-services script..."
        chmod +x "$DEPLOY_DIR/current/scripts/start-services.sh"
        cd "$DEPLOY_DIR/current"
        
        # Start in background for binary mode
        nohup bash scripts/start-services.sh > "$LOG_DIR/services.log" 2>&1 &
        echo $! > /tmp/infra-core-services.pid
    else
        log_error "No service management system available"
        return 1
    fi
    
    # Wait for services to be ready
    wait_for_services
}

# Validate and sanitize port input
validate_port() {
    local port_input="$1"
    local default_port="$2"
    local port_name="$3"
    
    # Take only the first line/word from input
    port_input=$(echo "$port_input" | head -n1 | awk '{print $1}')
    
    # Validate port number
    if [[ -n "$port_input" ]] && ! [[ "$port_input" =~ ^[0-9]+$ ]] || [[ "$port_input" -lt 1 || "$port_input" -gt 65535 ]]; then
        log_warning "Invalid $port_name port '$port_input'. Using default: $default_port"
        echo "$default_port"
    else
        echo "${port_input:-$default_port}"
    fi
}

# Interactive configuration setup
setup_interactive_config() {
    log_step "🔧 Interactive Configuration Setup"
    
    local config_changed=false
    
    # Check if running in non-interactive mode
    if [[ "$NON_INTERACTIVE" == "true" ]]; then
        log_info "Non-interactive mode: using default values"
        setup_default_config
        return
    fi
    
    log_info "We'll now configure essential settings for your InfraCore deployment."
    log_info "Press Enter to use default values shown in [brackets]."
    echo
    
    # 1. Domain configuration
    log_info "🌐 Domain and Email Configuration"
    local current_domain=$(grep -o "console\.[^\"]*" "$DEPLOY_DIR/current/configs/production.yaml" 2>/dev/null | head -1 || echo "your-domain.com")
    local current_email=$(grep -o "admin@[^\"]*" "$DEPLOY_DIR/current/configs/production.yaml" 2>/dev/null | head -1 || echo "admin@your-domain.com")
    
    read -p "🔗 Enter your domain name [$current_domain]: " domain_input
    CUSTOM_DOMAIN="${domain_input:-$current_domain}"
    
    read -p "📧 Enter admin email for SSL certificates [$current_email]: " email_input
    CUSTOM_EMAIL="${email_input:-$current_email}"
    
    # 2. Security configuration
    echo
    log_info "🔐 Security Configuration"
    
    if [[ -z "${JWT_SECRET:-}" ]]; then
        log_warning "JWT secret not set. Generating a secure random secret..."
        JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)
        log_success "JWT secret generated automatically"
    else
        log_info "Using existing JWT secret"
    fi
    
    # 3. Service ports configuration
    echo
    log_info "🚪 Service Ports Configuration"
    local current_http_port=$(grep -A2 "ports:" "$DEPLOY_DIR/current/configs/production.yaml" | grep "http:" | awk '{print $2}' || echo "80")
    local current_https_port=$(grep -A3 "ports:" "$DEPLOY_DIR/current/configs/production.yaml" | grep "https:" | awk '{print $2}' || echo "443")
    local current_api_port=$(grep "port:" "$DEPLOY_DIR/current/configs/production.yaml" | awk '{print $2}' || echo "8082")
    
    read -p "🌐 HTTP port [$current_http_port]: " http_port_input
    CUSTOM_HTTP_PORT=$(validate_port "$http_port_input" "$current_http_port" "HTTP")
    
    read -p "🔒 HTTPS port [$current_https_port]: " https_port_input
    CUSTOM_HTTPS_PORT=$(validate_port "$https_port_input" "$current_https_port" "HTTPS")
    
    read -p "🔌 API port [$current_api_port]: " api_port_input
    CUSTOM_API_PORT=$(validate_port "$api_port_input" "$current_api_port" "API")
    
    # 4. SSL/TLS configuration
    echo
    log_info "🔒 SSL/TLS Configuration"
    local ssl_enabled="true"
    read -p "Enable automatic SSL certificates with Let's Encrypt? [Y/n]: " ssl_input
    if [[ "$ssl_input" =~ ^[Nn]$ ]]; then
        ssl_enabled="false"
        log_warning "SSL disabled. You'll need to configure certificates manually."
    fi
    CUSTOM_SSL_ENABLED="$ssl_enabled"
    
    # 5. Resource limits
    echo
    log_info "📊 Resource Configuration"
    read -p "🧠 Maximum memory usage (GB) [2]: " memory_input
    CUSTOM_MEMORY_LIMIT="${memory_input:-2}g"
    
    read -p "⚡ CPU cores limit [2]: " cpu_input
    CUSTOM_CPU_LIMIT="${cpu_input:-2}"
    
    # 6. Backup configuration
    echo
    log_info "💾 Backup Configuration"
    read -p "🔄 Enable automatic backups? [Y/n]: " backup_input
    CUSTOM_BACKUP_ENABLED="true"
    if [[ "$backup_input" =~ ^[Nn]$ ]]; then
        CUSTOM_BACKUP_ENABLED="false"
    fi
    
    if [[ "$CUSTOM_BACKUP_ENABLED" == "true" ]]; then
        read -p "📅 Backup retention days [30]: " retention_input
        CUSTOM_BACKUP_RETENTION="${retention_input:-30}"
    fi
    
    # Summary
    echo
    log_success "🎯 Configuration Summary:"
    log_info "  Domain: $CUSTOM_DOMAIN"
    log_info "  Admin Email: $CUSTOM_EMAIL"
    log_info "  HTTP Port: $CUSTOM_HTTP_PORT"
    log_info "  HTTPS Port: $CUSTOM_HTTPS_PORT"
    log_info "  API Port: $CUSTOM_API_PORT"
    log_info "  SSL Enabled: $CUSTOM_SSL_ENABLED"
    log_info "  Memory Limit: $CUSTOM_MEMORY_LIMIT"
    log_info "  CPU Limit: $CUSTOM_CPU_LIMIT"
    log_info "  Backup Enabled: $CUSTOM_BACKUP_ENABLED"
    [[ "$CUSTOM_BACKUP_ENABLED" == "true" ]] && log_info "  Backup Retention: $CUSTOM_BACKUP_RETENTION days"
    
    echo
    read -p "✅ Proceed with this configuration? [Y/n]: " confirm
    if [[ "$confirm" =~ ^[Nn]$ ]]; then
        log_info "Configuration cancelled. Re-run with different values."
        exit 0
    fi
    
    config_changed=true
    apply_custom_config
}

# Apply custom configuration
apply_custom_config() {
    log_info "📝 Applying custom configuration..."
    
    # Create custom production config
    local config_file="$DEPLOY_DIR/current/configs/production.yaml"
    local backup_config="$config_file.backup-$(date +%Y%m%d-%H%M%S)"
    
    if [[ -f "$config_file" ]]; then
        cp "$config_file" "$backup_config"
        log_info "Original config backed up to: $backup_config"
    fi
    
    # Generate production configuration with custom values
    cat > "$config_file" << EOF
# Production Environment Configuration - Generated $(date)
gate:
  host: "0.0.0.0"
  ports:
    http: $CUSTOM_HTTP_PORT
    https: $CUSTOM_HTTPS_PORT
  logs:
    level: "info"
    console: false
    file: "/var/log/infra-core/gate.log"
  acme:
    directory_url: "https://acme-v02.api.letsencrypt.org/directory"
    email: "$CUSTOM_EMAIL"
    cache_dir: "/etc/infra-core/certs"
    challenge_type: "http-01"
    enabled: $CUSTOM_SSL_ENABLED
  resources:
    memory_limit: "$CUSTOM_MEMORY_LIMIT"
    cpu_limit: "$CUSTOM_CPU_LIMIT"

console:
  host: "0.0.0.0"
  port: $CUSTOM_API_PORT
  logs:
    level: "info"
    console: false
    file: "/var/log/infra-core/console.log"
  database:
    path: "/var/lib/infra-core/console.db"
    wal_mode: true
    timeout: "30s"
  auth:
    jwt:
      secret: ""  # Set via environment variable
      expires_hours: 8
    session:
      timeout_minutes: 30
  cors:
    enabled: true
    origins: ["https://$CUSTOM_DOMAIN", "http://localhost:3000"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
  backup:
    enabled: $CUSTOM_BACKUP_ENABLED
    retention_days: ${CUSTOM_BACKUP_RETENTION:-30}
    path: "/var/lib/infra-core/backups"
EOF
    
    # Update docker-compose with custom ports
    update_docker_compose_ports
    
    log_success "Custom configuration applied successfully"
}

# Setup default configuration for non-interactive mode
setup_default_config() {
    log_info "🔧 Setting up default configuration..."
    
    # Set default values
    CUSTOM_DOMAIN="${CUSTOM_DOMAIN:-console.example.com}"
    CUSTOM_EMAIL="${CUSTOM_EMAIL:-admin@example.com}"
    CUSTOM_HTTP_PORT="${CUSTOM_HTTP_PORT:-80}"
    CUSTOM_HTTPS_PORT="${CUSTOM_HTTPS_PORT:-443}"
    CUSTOM_API_PORT="${CUSTOM_API_PORT:-8082}"
    CUSTOM_SSL_ENABLED="${CUSTOM_SSL_ENABLED:-true}"
    CUSTOM_MEMORY_LIMIT="${CUSTOM_MEMORY_LIMIT:-2g}"
    CUSTOM_CPU_LIMIT="${CUSTOM_CPU_LIMIT:-2}"
    CUSTOM_BACKUP_ENABLED="${CUSTOM_BACKUP_ENABLED:-true}"
    CUSTOM_BACKUP_RETENTION="${CUSTOM_BACKUP_RETENTION:-30}"
    
    # Generate JWT secret if not provided
    if [[ -z "${JWT_SECRET:-}" ]]; then
        JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)
        log_info "Generated JWT secret automatically"
    fi
    
    log_info "Using default configuration values"
    apply_custom_config
}

# Check if port is available
check_port_available() {
    local port=$1
    
    # Validate port number
    if [[ ! "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 ]] || [[ "$port" -gt 65535 ]]; then
        log_error "Invalid port number: $port"
        return 1
    fi
    
    # Check if port is in use (check both IPv4 and IPv6)
    if netstat -tlnp 2>/dev/null | grep -E ":${port}[[:space:]]" >/dev/null; then
        return 1  # Port is in use
    elif ss -tlnp 2>/dev/null | grep -E ":${port}[[:space:]]" >/dev/null; then
        return 1  # Port is in use (alternative check)
    else
        return 0  # Port is available
    fi
}

# Update docker-compose ports
update_docker_compose_ports() {
    local compose_file="$DEPLOY_DIR/current/docker-compose.yml"
    local backup_compose="$compose_file.backup-$(date +%Y%m%d-%H%M%S)"
    
    if [[ -f "$compose_file" ]]; then
        cp "$compose_file" "$backup_compose"
        
        # Check for port conflicts first
        log_info "Checking for port conflicts..."
        local ports_to_check=("$CUSTOM_HTTP_PORT" "$CUSTOM_HTTPS_PORT" "$CUSTOM_API_PORT")
        local port_names=("HTTP" "HTTPS" "API")
        local conflicts=()
        
        for i in "${!ports_to_check[@]}"; do
            local port="${ports_to_check[$i]}"
            local name="${port_names[$i]}"
            if ! check_port_available "$port"; then
                conflicts+=("${name}:${port}")
                log_warning "${name} port ${port} is already in use"
            fi
        done
        
        if [[ ${#conflicts[@]} -gt 0 ]]; then
            log_error "Port conflicts detected: ${conflicts[*]}"
            log_info "Please stop the conflicting services or choose different ports"
            log_info "To see what's using these ports: sudo netstat -tlnp | grep -E ':(${CUSTOM_HTTP_PORT}|${CUSTOM_HTTPS_PORT}|${CUSTOM_API_PORT})'"
            return 1
        fi
        
        # Update ports in docker-compose.yml (simple validation - ports should be numbers)
        if [[ ! "$CUSTOM_HTTP_PORT" =~ ^[0-9]+$ ]] || [[ ! "$CUSTOM_HTTPS_PORT" =~ ^[0-9]+$ ]] || [[ ! "$CUSTOM_API_PORT" =~ ^[0-9]+$ ]]; then
            log_error "Invalid port numbers provided"
            return 1
        fi
        
        # Update ports in docker-compose.yml
        sed -i "s/\"80:80\"/\"${CUSTOM_HTTP_PORT}:80\"/g" "$compose_file"
        sed -i "s/\"443:443\"/\"${CUSTOM_HTTPS_PORT}:443\"/g" "$compose_file"
        sed -i "s/\"8082:8082\"/\"${CUSTOM_API_PORT}:8082\"/g" "$compose_file"
        
        # Add resource limits
        if ! grep -q "deploy:" "$compose_file"; then
            sed -i '/container_name: infra-core/a\    deploy:\n      resources:\n        limits:\n          memory: '${CUSTOM_MEMORY_LIMIT}'\n          cpus: "'${CUSTOM_CPU_LIMIT}'"' "$compose_file"
        fi
        
        log_info "Docker Compose configuration updated"
    fi
}

# Setup environment configuration
setup_environment_config() {
    log_step "Setting up environment configuration..."
    
    # Run interactive config if not already done
    if [[ -z "$CUSTOM_DOMAIN" ]]; then
        setup_interactive_config
    fi
    
    local config_source="$DEPLOY_DIR/current/configs/$ENVIRONMENT.yaml"
    local config_target="/etc/infra-core/config.yaml"
    
    if [[ -f "$config_source" ]]; then
        log_info "Copying configuration: $config_source -> $config_target"
        cp "$config_source" "$config_target"
        chown "$SERVICE_USER:$SERVICE_USER" "$config_target"
        chmod 600 "$config_target"
    else
        log_error "Configuration file not found: $config_source"
        exit 1
    fi
    
    # Set environment variables with custom values
    cat > "/etc/infra-core/environment" << EOF
INFRA_CORE_ENV=$ENVIRONMENT
INFRA_CORE_CONFIG_PATH=/etc/infra-core/config.yaml
INFRA_CORE_DATA_DIR=/var/lib/infra-core
INFRA_CORE_LOG_DIR=/var/log/infra-core
INFRA_CORE_JWT_SECRET=$JWT_SECRET
INFRA_CORE_ACME_EMAIL=$CUSTOM_EMAIL
EOF
    
    # Create .env file for Docker Compose
    cat > "$DEPLOY_DIR/current/.env" << EOF
JWT_SECRET=$JWT_SECRET
ACME_EMAIL=$CUSTOM_EMAIL
INFRA_CORE_ENV=$ENVIRONMENT
CUSTOM_DOMAIN=$CUSTOM_DOMAIN
CUSTOM_HTTP_PORT=$CUSTOM_HTTP_PORT
CUSTOM_HTTPS_PORT=$CUSTOM_HTTPS_PORT
CUSTOM_API_PORT=$CUSTOM_API_PORT
EOF
    
    chown "$SERVICE_USER:$SERVICE_USER" "/etc/infra-core/environment"
    chmod 600 "/etc/infra-core/environment"
    
    chown "$SERVICE_USER:$SERVICE_USER" "$DEPLOY_DIR/current/.env"
    chmod 600 "$DEPLOY_DIR/current/.env"
    
    log_success "Environment configuration completed"
    
    # Validate configuration files
    validate_configuration_files
    
    # Show configuration summary
    show_config_summary
}

# Validate and ensure configuration files exist
validate_configuration_files() {
    log_step "📋 Validating configuration files..."
    
    local config_dir="$DEPLOY_DIR/current/configs"
    local required_configs=("production.yaml" "development.yaml" "testing.yaml")
    local missing_configs=()
    
    # Check if configs directory exists
    if [[ ! -d "$config_dir" ]]; then
        log_warning "Configuration directory not found: $config_dir"
        log_info "Creating configuration directory..."
        mkdir -p "$config_dir"
    fi
    
    # Check each required configuration file
    for config_file in "${required_configs[@]}"; do
        local config_path="$config_dir/$config_file"
        if [[ ! -f "$config_path" ]]; then
            log_warning "Missing configuration file: $config_file"
            missing_configs+=("$config_file")
        else
            log_info "✅ Found: $config_file"
            # Validate YAML syntax
            if command -v python3 >/dev/null 2>&1; then
                if ! python3 -c "import yaml; yaml.safe_load(open('$config_path'))" 2>/dev/null; then
                    log_warning "⚠️  YAML syntax error in: $config_file"
                fi
            fi
        fi
    done
    
    # Create missing configuration files from templates
    if [[ ${#missing_configs[@]} -gt 0 ]]; then
        log_info "Creating missing configuration files..."
        create_default_config_files "${missing_configs[@]}"
    fi
    
    # Ensure configuration files are accessible in Docker
    ensure_docker_config_access
    
    log_success "Configuration validation completed"
}

# Create default configuration files
create_default_config_files() {
    local missing_configs=("$@")
    local config_dir="$DEPLOY_DIR/current/configs"
    
    for config_file in "${missing_configs[@]}"; do
        local config_path="$config_dir/$config_file"
        log_info "Creating default configuration: $config_file"
        
        case "$config_file" in
            "production.yaml")
                create_production_config "$config_path"
                ;;
            "development.yaml")
                create_development_config "$config_path"
                ;;
            "testing.yaml")
                create_testing_config "$config_path"
                ;;
        esac
        
        if [[ -f "$config_path" ]]; then
            log_success "Created: $config_file"
            chmod 644 "$config_path"
        else
            log_error "Failed to create: $config_file"
        fi
    done
}

# Create production configuration
create_production_config() {
    local config_path="$1"
    
    cat > "$config_path" << EOF
# Production Environment Configuration - Auto-generated $(date)
gate:
  host: "0.0.0.0"
  ports:
    http: ${CUSTOM_HTTP_PORT:-80}
    https: ${CUSTOM_HTTPS_PORT:-443}
  logs:
    level: "info"
    console: false
    file: "/var/log/infra-core/gate.log"
  acme:
    directory_url: "https://acme-v02.api.letsencrypt.org/directory"
    email: "${CUSTOM_EMAIL:-admin@example.com}"
    cache_dir: "/etc/infra-core/certs"
    challenge_type: "http-01"
    enabled: ${CUSTOM_SSL_ENABLED:-true}
  resources:
    memory_limit: "${CUSTOM_MEMORY_LIMIT:-2g}"
    cpu_limit: "${CUSTOM_CPU_LIMIT:-2}"

console:
  host: "0.0.0.0"
  port: ${CUSTOM_API_PORT:-8082}
  logs:
    level: "info"
    console: false
    file: "/var/log/infra-core/console.log"
  database:
    path: "/var/lib/infra-core/console.db"
    wal_mode: true
    timeout: "30s"
  auth:
    jwt:
      secret: ""  # Set via environment variable
      expires_hours: 8
    session:
      timeout_minutes: 30
  cors:
    enabled: true
    origins: ["https://${CUSTOM_DOMAIN:-localhost}", "http://localhost:3000"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
  backup:
    enabled: ${CUSTOM_BACKUP_ENABLED:-true}
    retention_days: ${CUSTOM_BACKUP_RETENTION:-30}
    path: "/var/lib/infra-core/backups"
EOF
}

# Create development configuration  
create_development_config() {
    local config_path="$1"
    
    cat > "$config_path" << EOF
# Development Environment Configuration - Auto-generated $(date)
gate:
  host: "0.0.0.0"
  ports:
    http: ${CUSTOM_HTTP_PORT:-8080}
    https: ${CUSTOM_HTTPS_PORT:-8443}
  logs:
    level: "debug"
    console: true
    file: "/var/log/infra-core/gate.log"
  acme:
    enabled: false  # Disabled in development

console:
  host: "0.0.0.0"
  port: ${CUSTOM_API_PORT:-3000}
  logs:
    level: "debug"
    console: true
    file: "/var/log/infra-core/console.log"
  database:
    path: "/var/lib/infra-core/console-dev.db"
    wal_mode: true
    timeout: "30s"
  auth:
    jwt:
      secret: "dev-secret-key"
      expires_hours: 24
    session:
      timeout_minutes: 120
  cors:
    enabled: true
    origins: ["http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
  backup:
    enabled: false
EOF
}

# Create testing configuration
create_testing_config() {
    local config_path="$1"
    
    cat > "$config_path" << EOF
# Testing Environment Configuration - Auto-generated $(date)
gate:
  host: "127.0.0.1"
  ports:
    http: 18080
    https: 18443
  logs:
    level: "debug"
    console: true
    file: "/tmp/infra-core-test/gate.log"
  acme:
    enabled: false

console:
  host: "127.0.0.1"
  port: 18082
  logs:
    level: "debug"
    console: true
    file: "/tmp/infra-core-test/console.log"
  database:
    path: "/tmp/infra-core-test/console-test.db"
    wal_mode: false
    timeout: "5s"
  auth:
    jwt:
      secret: "test-secret-key-do-not-use-in-production"
      expires_hours: 1
    session:
      timeout_minutes: 30
  cors:
    enabled: true
    origins: ["*"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
  backup:
    enabled: false
EOF
}

# Ensure Docker containers can access configuration files
ensure_docker_config_access() {
    log_info "🐳 Ensuring Docker configuration access..."
    
    local config_dir="$DEPLOY_DIR/current/configs"
    
    # Set proper permissions for configuration files
    find "$config_dir" -name "*.yaml" -exec chmod 644 {} \;
    find "$config_dir" -name "*.yml" -exec chmod 644 {} \;
    
    # Ensure directory is readable
    chmod 755 "$config_dir"
    
    # Test Docker bind mount path
    if [[ -f "$DEPLOY_DIR/current/docker-compose.yml" ]]; then
        # Check if config mount is in docker-compose
        if ! grep -q "./configs:/app/configs" "$DEPLOY_DIR/current/docker-compose.yml"; then
            log_warning "Configuration mount not found in docker-compose.yml"
            log_info "Adding configuration volume mount..."
            add_config_mount_to_compose
        else
            log_success "Configuration mount verified in docker-compose.yml"
        fi
    fi
    
    log_success "Docker configuration access configured"
}

# Add configuration mount to docker-compose.yml
add_config_mount_to_compose() {
    local compose_file="$DEPLOY_DIR/current/docker-compose.yml"
    local backup_file="$compose_file.backup-config-$(date +%Y%m%d-%H%M%S)"
    
    # Backup original file
    cp "$compose_file" "$backup_file"
    log_info "Backed up docker-compose.yml to: $backup_file"
    
    # Add configuration mount if not exists
    if ! grep -q "./configs:/app/configs" "$compose_file"; then
        # Find the volumes section and add config mount
        sed -i '/- \.\/certs:\/etc\/infra-core\/certs/a\      - ./configs:/app/configs:ro  # Mount configs as read-only' "$compose_file"
        log_success "Added configuration mount to docker-compose.yml"
    fi
}

# Show configuration summary
show_config_summary() {
    log_info "📋 Current Configuration:"
    echo "  🌐 Domain: ${CUSTOM_DOMAIN:-'Not set'}"
    echo "  📧 Admin Email: ${CUSTOM_EMAIL:-'Not set'}" 
    echo "  🚪 HTTP Port: ${CUSTOM_HTTP_PORT:-'80'}"
    echo "  🔒 HTTPS Port: ${CUSTOM_HTTPS_PORT:-'443'}"
    echo "  🔌 API Port: ${CUSTOM_API_PORT:-'8082'}"
    echo "  🔐 SSL Enabled: ${CUSTOM_SSL_ENABLED:-'true'}"
    echo "  🧠 Memory Limit: ${CUSTOM_MEMORY_LIMIT:-'2g'}"
    echo "  ⚡ CPU Limit: ${CUSTOM_CPU_LIMIT:-'2'}"
    echo "  💾 Backup Enabled: ${CUSTOM_BACKUP_ENABLED:-'true'}"
    [[ "${CUSTOM_BACKUP_ENABLED:-true}" == "true" ]] && echo "  📅 Backup Retention: ${CUSTOM_BACKUP_RETENTION:-'30'} days"
    echo "  🔑 JWT Secret: ${JWT_SECRET:+Set (hidden)} ${JWT_SECRET:-Not set}"
}

# Monitor backup progress
monitor_backup_progress() {
    local backup_dir="$1"
    local start_time=$(date +%s)
    
    while [[ -d "$backup_dir" ]]; do
        sleep 10
        
        if [[ -d "$backup_dir" ]]; then
            local current_size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1 || echo "0")
            local elapsed=$(($(date +%s) - start_time))
            
            if [[ $elapsed -gt 0 ]]; then
                log_info "📊 Backup progress: $current_size processed (${elapsed}s elapsed)"
            fi
            
            # Safety timeout - kill if backup takes too long
            if [[ $elapsed -gt 1800 ]]; then  # 30 minutes
                log_error "Backup process timeout (30 minutes), terminating..."
                break
            fi
        fi
    done
}

# Monitor backup progress
monitor_backup_progress() {
    local backup_dir="$1"
    local start_time=$(date +%s)
    
    while [[ -d "$backup_dir" ]]; do
        sleep 10
        
        if [[ -d "$backup_dir" ]]; then
            local current_size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1 || echo "0")
            local elapsed=$(($(date +%s) - start_time))
            
            if [[ $elapsed -gt 0 ]] && [[ $((elapsed % 30)) -eq 0 ]]; then
                log_info "📊 Backup progress: $current_size processed (${elapsed}s elapsed)"
            fi
            
            # Safety timeout - kill if backup takes too long
            if [[ $elapsed -gt 1800 ]]; then  # 30 minutes
                log_error "Backup process timeout (30 minutes), terminating..."
                break
            fi
        fi
    done
}

# Clean up problematic Docker GPG keys
cleanup_docker_gpg_keys() {
    log_info "Cleaning up Docker GPG configuration..."
    
    # Remove potentially corrupted Docker GPG key and repository
    rm -f /etc/apt/keyrings/docker.gpg 2>/dev/null || true
    rm -f /etc/apt/sources.list.d/docker.list 2>/dev/null || true
    
    # Clean APT cache
    apt-get clean 2>/dev/null || true
    rm -rf /var/lib/apt/lists/* 2>/dev/null || true
    
    log_success "Docker GPG cleanup completed"
}

# Verify Docker installation and functionality
verify_docker_installation() {
    log_info "Verifying Docker installation..."
    
    # Check if Docker command exists
    if ! command -v docker &>/dev/null; then
        log_error "Docker command not found"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info &>/dev/null; then
        log_warning "Docker daemon not running, attempting to start..."
        systemctl start docker
        sleep 3
        
        if ! docker info &>/dev/null; then
            log_error "Failed to start Docker daemon"
            return 1
        fi
    fi
    
    # Test Docker with a simple container
    log_info "Testing Docker functionality..."
    if timeout 30 docker run --rm hello-world &>/dev/null; then
        log_success "Docker is working correctly"
        return 0
    else
        log_error "Docker test container failed"
        return 1
    fi
}

# Comprehensive pre-deployment checks
pre_deployment_checks() {
    log_step "🔍 Running pre-deployment checks..."
    
    # Clean up any problematic GPG keys before installation
    cleanup_docker_gpg_keys
    
    local checks_passed=0
    local total_checks=6
    
    # 1. Configuration files check
    log_info "1/6 Checking configuration files..."
    if check_configuration; then
        checks_passed=$((checks_passed + 1))
        log_success "✅ Configuration check passed"
    else
        log_error "❌ Configuration check failed"
    fi
    
    # 2. Port availability check  
    log_info "2/6 Checking port availability..."
    if check_port_conflicts; then
        checks_passed=$((checks_passed + 1))
        log_success "✅ Port availability check passed"
    else
        log_error "❌ Port conflicts detected"
    fi
    
    # 3. System resources check
    log_info "3/6 Checking system resources..."
    if validate_system_requirements; then
        checks_passed=$((checks_passed + 1))
        log_success "✅ System requirements check passed"
    else
        log_error "❌ System requirements check failed"
    fi
    
    # 4. Dependencies check
    log_info "4/6 Checking dependencies..."
    if validate_dependencies; then
        checks_passed=$((checks_passed + 1))
        log_success "✅ Dependencies check passed"
    else
        log_error "❌ Dependencies check failed"
    fi
    
    # 5. Network connectivity check
    log_info "5/6 Checking network connectivity..."
    if check_network_connectivity; then
        checks_passed=$((checks_passed + 1))
        log_success "✅ Network connectivity check passed"
    else
        log_warning "⚠️ Network connectivity check failed (non-critical)"
    fi
    
    # 6. Existing deployment check
    log_info "6/6 Checking existing deployment..."
    if check_existing_deployment; then
        checks_passed=$((checks_passed + 1))
        log_success "✅ Existing deployment check passed"
    else
        log_warning "⚠️ Existing deployment issues detected"
    fi
    
    # Summary
    log_info "Pre-deployment checks completed: $checks_passed/$total_checks passed"
    
    if [[ $checks_passed -lt 4 ]]; then
        log_error "Critical pre-deployment checks failed. Deployment cannot continue."
        log_info "Please resolve the issues above and try again."
        exit 1
    elif [[ $checks_passed -lt $total_checks ]]; then
        if [[ "$NON_INTERACTIVE" != "true" ]]; then
            log_warning "Some checks failed. Continue with deployment? [y/N]"
            read -r -n 1 continue_deploy
            echo
            if [[ ! "$continue_deploy" =~ ^[Yy]$ ]]; then
                log_info "Deployment cancelled by user"
                exit 0
            fi
        else
            log_warning "Some checks failed, but continuing in non-interactive mode"
        fi
    fi
    
    log_success "Pre-deployment checks completed successfully"
}

# Check for port conflicts with fallback support
check_port_conflicts() {
    log_info "🔍 Checking port conflicts with automatic fallback..."
    
    local required_ports=("80" "443" "8082" "9090" "8085" "8086")
    local conflicts=()
    local fallback_info=()
    
    for port in "${required_ports[@]}"; do
        if netstat -tlnp 2>/dev/null | grep -q ":${port} "; then
            local process=$(netstat -tlnp 2>/dev/null | grep ":${port} " | awk '{print $7}' | head -1)
            conflicts+=("${port}:${process}")
            
            # Add fallback information for optional services
            case "$port" in
                9090)
                    fallback_info+=("Orchestrator will fallback to port 19090 or 29090")
                    ;;
                8085) 
                    fallback_info+=("Probe Monitor will fallback to port 18085 or 28085")
                    ;;
                8086)
                    fallback_info+=("Snap Service will fallback to port 18086 or 28086")
                    ;;
                80|443|8082)
                    fallback_info+=("Critical service port $port - manual intervention may be needed")
                    ;;
            esac
        fi
    done
    
    if [[ ${#conflicts[@]} -gt 0 ]]; then
        log_warning "Port conflicts detected:"
        for conflict in "${conflicts[@]}"; do
            log_warning "  Port ${conflict%:*} is used by ${conflict#*:}"
        done
        
        if [[ ${#fallback_info[@]} -gt 0 ]]; then
            log_info "Fallback configuration:"
            for info in "${fallback_info[@]}"; do
                log_info "  • $info"
            done
        fi
        return 1
    fi
    
    log_success "All ports are available"
    return 0
}

# Check network connectivity
check_network_connectivity() {
    local test_urls=("github.com" "docker.io" "registry.npmjs.org")
    local failed_count=0
    
    for url in "${test_urls[@]}"; do
        if ! curl -s --max-time 5 "$url" >/dev/null 2>&1; then
            log_warning "Cannot reach $url"
            failed_count=$((failed_count + 1))
        fi
    done
    
    if [[ $failed_count -eq ${#test_urls[@]} ]]; then
        log_error "No network connectivity detected"
        return 1
    elif [[ $failed_count -gt 0 ]]; then
        log_warning "Some network connectivity issues detected"
        return 0
    fi
    
    return 0
}

# Check existing deployment state
check_existing_deployment() {
    if [[ -d "$DEPLOY_DIR/current" ]]; then
        log_info "Found existing deployment at $DEPLOY_DIR/current"
        
        # Check if services are running
        if command -v docker-compose >/dev/null 2>&1; then
            local running_containers
            running_containers=$(docker-compose -f "$DEPLOY_DIR/current/docker-compose.yml" ps -q 2>/dev/null | wc -l)
            if [[ $running_containers -gt 0 ]]; then
                log_info "Found $running_containers running containers"
                add_rollback_step "cd '$DEPLOY_DIR/current' && docker-compose down"
            fi
        fi
        
        # Prepare backup rollback
        add_rollback_step "mv '$DEPLOY_DIR/current' '$DEPLOY_DIR/rollback-$(date +%Y%m%d-%H%M%S)'"
    fi
    
    return 0
}

# Check configuration files using external tool or built-in validation
check_configuration() {
    log_info "🔍 Checking configuration files..."
    
    # Use config-check script if available
    local config_check_script="$DEPLOY_DIR/current/scripts/config-check.sh"
    if [[ -f "$config_check_script" ]]; then
        log_info "Using configuration check tool..."
        chmod +x "$config_check_script"
        cd "$DEPLOY_DIR/current"
        if ! "$config_check_script" --check; then
            log_warning "Configuration issues detected"
            if [[ "$NON_INTERACTIVE" != "true" ]]; then
                read -p "Fix configuration files automatically? (y/N): " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    "$config_check_script" --fix
                    "$config_check_script" --validate
                    log_success "Configuration files fixed"
                    return 0
                else
                    log_warning "Proceeding with existing configuration"
                    return 1
                fi
            else
                log_info "Running in non-interactive mode, auto-fixing configuration..."
                "$config_check_script" --fix
                "$config_check_script" --validate
                return 0
            fi
        else
            log_success "All configuration files are valid"
            return 0
        fi
        
        # Check Docker configuration mounts
        if ! "$config_check_script" --docker-check; then
            log_warning "Docker configuration mount issues detected"
            if [[ "$NON_INTERACTIVE" != "true" ]]; then
                read -p "Fix Docker configuration mounts? (y/N): " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    "$config_check_script" --docker-fix
                    log_success "Docker configuration mounts fixed"
                    return 0
                fi
            else
                log_info "Running in non-interactive mode, auto-fixing Docker mounts..."
                "$config_check_script" --docker-fix
                return 0
            fi
        fi
    else
        # Fallback to built-in validation
        log_info "Config check tool not found, using built-in validation..."
        validate_configuration_files
        create_default_config_files
        ensure_docker_config_access
        return 0
    fi
    
    return 1
}

# Install systemd services
install_systemd_services() {
    log_step "Installing systemd services..."
    
    # Console service
    cat > /etc/systemd/system/infra-core-console.service << EOF
[Unit]
Description=InfraCore Console API Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$DEPLOY_DIR/current
ExecStart=$DEPLOY_DIR/current/bin/console
EnvironmentFile=/etc/infra-core/environment
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=infra-core-console

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/infra-core /var/log/infra-core
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

    # Gate service
    cat > /etc/systemd/system/infra-core-gate.service << EOF
[Unit]
Description=InfraCore Reverse Proxy Gateway
After=network.target infra-core-console.service
Wants=network.target
Requires=infra-core-console.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$DEPLOY_DIR/current
ExecStart=$DEPLOY_DIR/current/bin/gate
EnvironmentFile=/etc/infra-core/environment
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=infra-core-gate

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/infra-core /var/log/infra-core
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
}

# Wait for services to be ready
wait_for_services() {
    log_step "Waiting for services to be ready..."
    
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log_info "Health check attempt $attempt/$max_attempts..."
        
        if curl -s http://localhost:8082/api/v1/health &> /dev/null; then
            log_success "Services are healthy!"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "Services failed to become healthy within $max_attempts attempts"
    log_info "Checking service status..."
    show_status
    return 1
}

# Show deployment status
show_status() {
    log_step "Deployment Status"
    
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        echo "Docker Services:"
        docker-compose -f "$DEPLOY_DIR/current/docker-compose.yml" ps
        echo
        echo "Docker Images:"
        docker images | grep infra-core
    else
        echo "Systemd Services:"
        systemctl status infra-core-console --no-pager -l
        echo
        systemctl status infra-core-gate --no-pager -l
    fi
    
    echo
    echo "Deployment Info:"
    if [[ -f "$DEPLOY_DIR/current/deployment-info.json" ]]; then
        cat "$DEPLOY_DIR/current/deployment-info.json" | jq .
    fi
    
    echo
    echo "Health Check:"
    if curl -s http://localhost:8082/api/v1/health; then
        echo -e "\n${GREEN}✓ Services are healthy${NC}"
    else
        echo -e "\n${RED}✗ Services are not responding${NC}"
    fi
}

# Show service logs
show_logs() {
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        docker-compose -f "$DEPLOY_DIR/current/docker-compose.yml" logs -f --tail=100
    else
        journalctl -u infra-core-console -u infra-core-gate -f -n 100
    fi
}

# Restart services
restart_services() {
    log_step "Restarting services..."
    
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        cd "$DEPLOY_DIR/current"
        docker-compose -f docker-compose.yml restart
    else
        systemctl restart infra-core-console infra-core-gate
    fi
    
    wait_for_services
}

# Stop services
stop_services() {
    log_step "Stopping services..."
    
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        cd "$DEPLOY_DIR/current"
        docker-compose -f docker-compose.yml down
    else
        systemctl stop infra-core-console infra-core-gate
    fi
}

# Start services
start_services() {
    log_step "Starting services..."
    
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        cd "$DEPLOY_DIR/current"
        docker-compose -f docker-compose.yml up -d
    else
        systemctl start infra-core-console infra-core-gate
    fi
    
    wait_for_services
}

# Rollback to previous deployment
rollback() {
    log_step "Rolling back to previous deployment..."
    
    if [[ ! -d "$DEPLOY_DIR/previous" ]]; then
        log_error "No previous deployment found"
        exit 1
    fi
    
    # Stop current services
    stop_services
    
    # Swap directories
    mv "$DEPLOY_DIR/current" "$DEPLOY_DIR/rollback-temp"
    mv "$DEPLOY_DIR/previous" "$DEPLOY_DIR/current"
    mv "$DEPLOY_DIR/rollback-temp" "$DEPLOY_DIR/previous"
    
    # Start services with previous deployment
    start_services
    
    log_success "Rollback completed successfully"
}

# Quick update without full deployment
quick_update() {
    log_step "Performing quick update..."
    
    # Create backup
    CREATE_BACKUP=true
    create_backup
    
    # Update repository
    update_repository
    
    # Restart services with new code
    restart_services
    
    log_success "Quick update completed successfully"
}

# Interactive upgrade with user confirmation
upgrade() {
    log_step "Interactive upgrade process..."
    
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║                          INFRACORE UPGRADE CONFIRMATION                       ║"
    echo "╠══════════════════════════════════════════════════════════════════════════════╣"
    echo "║ This will perform the following actions:                                      ║"
    echo "║ • Create a backup of current deployment                                       ║"
    echo "║ • Update repository to latest version                                         ║"
    echo "║ • Rebuild and restart all services                                            ║"
    echo "║ • Verify deployment health                                                     ║"
    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    echo
    
    # Show current status
    log_info "Current deployment information:"
    if [ -d "$DEPLOY_DIR" ]; then
        if [ -d "$DEPLOY_DIR/.git" ]; then
            cd "$DEPLOY_DIR"
            local current_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "Unknown")
            local current_branch=$(git branch --show-current 2>/dev/null || echo "Unknown")
            log_info "  Current commit: $current_commit"
            log_info "  Current branch: $current_branch"
        fi
        local deploy_size=$(du -sh "$DEPLOY_DIR" 2>/dev/null | cut -f1 || echo "Unknown")
        log_info "  Deploy directory size: $deploy_size"
    else
        log_info "  No existing deployment found"
    fi
    
    # Show what will be updated to
    log_info "Target deployment information:"
    log_info "  Repository: $REPO_URL"
    log_info "  Branch: $BRANCH"
    log_info "  Environment: $ENVIRONMENT"
    if [ "$USE_MIRRORS" = true ]; then
        log_info "  Using mirrors: Yes (Region: $MIRROR_REGION)"
    else
        log_info "  Using mirrors: No"
    fi
    
    echo
    echo "⚠️  WARNING: This operation will temporarily stop your services during the upgrade."
    echo "💾 A backup will be created automatically before proceeding."
    echo
    
    # Interactive confirmation
    local confirmation=""
    while [[ ! "$confirmation" =~ ^[YyNn]$ ]]; do
        echo -n "Do you want to proceed with the upgrade? [y/N]: "
        read -r confirmation
        confirmation=${confirmation:-n}  # Default to 'n' if empty
    done
    
    if [[ "$confirmation" =~ ^[Nn]$ ]]; then
        log_info "Upgrade cancelled by user"
        exit 0
    fi
    
    log_success "User confirmed upgrade. Starting deployment..."
    echo
    
    # Perform the actual upgrade (same as main deployment)
    main_deploy
}

# Comprehensive uninstall function with safety checks
uninstall_infracore() {
    log_step "InfraCore Uninstallation Process"
    
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║                        ⚠️  INFRACORE UNINSTALL WARNING ⚠️                    ║"
    echo "╠══════════════════════════════════════════════════════════════════════════════╣"
    echo "║ This will PERMANENTLY REMOVE the following components:                        ║"
    echo "║                                                                              ║"
    echo "║ 🗂️  All deployment files and application data                                ║"
    echo "║ 🐳 All Docker containers, images, and volumes                               ║"
    echo "║ 🔧 All systemd services and configuration files                             ║"
    echo "║ 👤 Service user account (if created by this script)                        ║"
    echo "║ 📂 All log files and backups                                                ║"
    echo "║ 🌐 SSL certificates and ACME configurations                                 ║"
    echo "║                                                                              ║"
    echo "║ ⛔ THIS ACTION CANNOT BE UNDONE WITHOUT A BACKUP! ⛔                        ║"
    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    echo
    
    # Show current deployment information
    local deployment_found=false
    
    log_info "Current InfraCore installation status:"
    
    # Check deployment directory
    if [[ -d "$DEPLOY_DIR" ]]; then
        deployment_found=true
        local deploy_size=$(du -sh "$DEPLOY_DIR" 2>/dev/null | cut -f1 || echo "Unknown")
        log_info "  📁 Deployment directory: $DEPLOY_DIR ($deploy_size)"
        
        # Check for git information
        if [[ -d "$DEPLOY_DIR/current/.git" ]]; then
            cd "$DEPLOY_DIR/current"
            local current_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "Unknown")
            local current_branch=$(git branch --show-current 2>/dev/null || echo "Unknown")
            log_info "     Branch: $current_branch, Commit: $current_commit"
        fi
    else
        log_info "  📁 Deployment directory: Not found"
    fi
    
    # Check Docker containers
    if command -v docker &>/dev/null; then
        local containers=$(docker ps -a --filter "name=infra" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null | tail -n +2)
        if [[ -n "$containers" ]]; then
            deployment_found=true
            log_info "  🐳 Docker containers found:"
            echo "$containers" | sed 's/^/     /'
        else
            log_info "  🐳 Docker containers: None found"
        fi
        
        # Check Docker images
        local images=$(docker images | grep -E "(infra|core)" | awk '{print $1":"$2"\t"$7$8}' 2>/dev/null)
        if [[ -n "$images" ]]; then
            log_info "  🖼️  Docker images found:"
            echo "$images" | sed 's/^/     /'
        fi
        
        # Check Docker volumes
        local volumes=$(docker volume ls | grep -E "(infra|core)" | awk '{print $2}' 2>/dev/null)
        if [[ -n "$volumes" ]]; then
            log_info "  💾 Docker volumes found:"
            echo "$volumes" | sed 's/^/     /'
        fi
    fi
    
    # Check systemd services
    local services_found=""
    for service in "infra-core-console" "infra-core-gate"; do
        if systemctl list-unit-files "$service.service" &>/dev/null; then
            local status=$(systemctl is-active "$service" 2>/dev/null || echo "inactive")
            services_found+="     $service.service ($status)\n"
            deployment_found=true
        fi
    done
    
    if [[ -n "$services_found" ]]; then
        log_info "  🔧 Systemd services found:"
        echo -e "$services_found"
    else
        log_info "  🔧 Systemd services: None found"
    fi
    
    # Check service user
    if id "$SERVICE_USER" &>/dev/null; then
        deployment_found=true
        log_info "  👤 Service user: $SERVICE_USER (exists)"
    else
        log_info "  👤 Service user: $SERVICE_USER (not found)"
    fi
    
    # Check backup directory
    if [[ -d "$BACKUP_DIR" ]]; then
        local backup_count=$(find "$BACKUP_DIR" -name "*.tar.gz" 2>/dev/null | wc -l 2>/dev/null | tr -d ' \n' || echo "0")
        backup_count=${backup_count//[^0-9]/}
        backup_count=${backup_count:-0}
        local backup_size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1 || echo "Unknown")
        if [[ $backup_count -gt 0 ]]; then
            log_info "  💾 Backups found: $backup_count files ($backup_size)"
        else
            log_info "  💾 Backup directory exists but empty"
        fi
    else
        log_info "  💾 Backup directory: Not found"
    fi
    
    # If no deployment found, offer to continue anyway
    if [[ "$deployment_found" != "true" ]]; then
        log_warning "No InfraCore installation detected!"
        echo
        echo -n "No deployment found. Continue with cleanup anyway? [y/N]: "
        read -r force_cleanup
        if [[ ! "$force_cleanup" =~ ^[Yy]$ ]]; then
            log_info "Uninstall cancelled - no installation found"
            exit 0
        fi
    fi
    
    echo
    echo "🛡️  BACKUP RECOMMENDATION:"
    echo "   Before proceeding, you should create a final backup of your data."
    echo "   This includes databases, configuration files, and any custom modifications."
    echo
    
    # Backup option
    local create_final_backup=""
    while [[ ! "$create_final_backup" =~ ^[YyNn]$ ]]; do
        echo -n "Create a final backup before uninstalling? [Y/n]: "
        read -r create_final_backup
        create_final_backup=${create_final_backup:-y}  # Default to 'y'
    done
    
    if [[ "$create_final_backup" =~ ^[Yy]$ ]]; then
        log_info "Creating final backup before uninstall..."
        CREATE_BACKUP=true
        create_backup || log_warning "Backup creation failed, but continuing with uninstall"
    fi
    
    echo
    echo "🔴 FINAL CONFIRMATION:"
    echo "   Type 'DELETE EVERYTHING' (exactly) to confirm permanent removal:"
    echo -n "   > "
    read -r final_confirmation
    
    if [[ "$final_confirmation" != "DELETE EVERYTHING" ]]; then
        log_info "Uninstall cancelled - confirmation phrase not matched"
        exit 0
    fi
    
    log_warning "Starting permanent uninstallation process..."
    echo
    
    # Perform the actual uninstall
    perform_complete_uninstall
}

# Perform complete uninstall of all components
perform_complete_uninstall() {
    log_step "Executing comprehensive uninstall..."
    
    local uninstall_errors=0
    
    # 1. Stop all services gracefully
    log_info "🛑 Stopping all InfraCore services..."
    
    # Stop Docker services
    if command -v docker &>/dev/null && [[ -f "$DEPLOY_DIR/current/docker-compose.yml" ]]; then
        cd "$DEPLOY_DIR/current" 2>/dev/null
        if docker-compose -f docker-compose.yml down --remove-orphans 2>/dev/null; then
            log_success "Docker services stopped successfully"
        else
            log_warning "Failed to stop Docker services gracefully"
            uninstall_errors=$((uninstall_errors + 1))
        fi
    fi
    
    # Stop systemd services
    for service in "infra-core-console" "infra-core-gate"; do
        if systemctl is-active "$service" &>/dev/null; then
            if systemctl stop "$service" 2>/dev/null; then
                log_success "Stopped $service"
            else
                log_warning "Failed to stop $service"
                uninstall_errors=$((uninstall_errors + 1))
            fi
        fi
        
        if systemctl is-enabled "$service" &>/dev/null; then
            if systemctl disable "$service" 2>/dev/null; then
                log_success "Disabled $service"
            else
                log_warning "Failed to disable $service"
            fi
        fi
    done
    
    # 2. Remove Docker components
    log_info "🐳 Cleaning up Docker components..."
    
    if command -v docker &>/dev/null; then
        # Remove containers
        local containers=$(docker ps -a --filter "name=infra" --format "{{.ID}}" 2>/dev/null)
        if [[ -n "$containers" ]]; then
            echo "$containers" | xargs -r docker rm -f 2>/dev/null && log_success "Removed InfraCore containers" || log_warning "Failed to remove some containers"
        fi
        
        # Remove images
        local images=$(docker images --filter "reference=*infra*" --filter "reference=*core*" --format "{{.ID}}" 2>/dev/null)
        if [[ -n "$images" ]]; then
            echo "$images" | xargs -r docker rmi -f 2>/dev/null && log_success "Removed InfraCore images" || log_warning "Failed to remove some images"
        fi
        
        # Remove volumes
        local volumes=$(docker volume ls --filter "name=infra" --filter "name=core" --format "{{.Name}}" 2>/dev/null)
        if [[ -n "$volumes" ]]; then
            echo "$volumes" | xargs -r docker volume rm -f 2>/dev/null && log_success "Removed InfraCore volumes" || log_warning "Failed to remove some volumes"
        fi
        
        # Clean up unused Docker resources
        docker system prune -f 2>/dev/null && log_success "Cleaned up unused Docker resources"
    fi
    
    # 3. Remove systemd service files
    log_info "🔧 Removing systemd service files..."
    
    local service_files=(
        "/etc/systemd/system/infra-core-console.service"
        "/etc/systemd/system/infra-core-gate.service"
    )
    
    for service_file in "${service_files[@]}"; do
        if [[ -f "$service_file" ]]; then
            if rm -f "$service_file" 2>/dev/null; then
                log_success "Removed $service_file"
            else
                log_warning "Failed to remove $service_file"
                uninstall_errors=$((uninstall_errors + 1))
            fi
        fi
    done
    
    systemctl daemon-reload 2>/dev/null
    
    # 4. Remove all deployment files and directories
    log_info "🗂️  Removing deployment files and directories..."
    
    local directories_to_remove=(
        "$DEPLOY_DIR"
        "/etc/infra-core"
        "/var/lib/infra-core"
        "/var/log/infra-core"
    )
    
    for dir in "${directories_to_remove[@]}"; do
        if [[ -d "$dir" ]]; then
            if rm -rf "$dir" 2>/dev/null; then
                log_success "Removed directory: $dir"
            else
                log_warning "Failed to remove directory: $dir"
                uninstall_errors=$((uninstall_errors + 1))
            fi
        fi
    done
    
    # 5. Remove backups (with additional confirmation)
    if [[ -d "$BACKUP_DIR" ]]; then
        local backup_count=$(find "$BACKUP_DIR" -name "*.tar.gz" 2>/dev/null | wc -l 2>/dev/null | tr -d ' \n' || echo "0")
        backup_count=${backup_count//[^0-9]/}
        backup_count=${backup_count:-0}
        if [[ $backup_count -gt 0 ]]; then
            log_warning "Found $backup_count backup files in $BACKUP_DIR"
            echo -n "Remove backup files too? [y/N]: "
            read -r remove_backups
            if [[ "$remove_backups" =~ ^[Yy]$ ]]; then
                if rm -rf "$BACKUP_DIR" 2>/dev/null; then
                    log_success "Removed backup directory: $BACKUP_DIR"
                else
                    log_warning "Failed to remove backup directory: $BACKUP_DIR"
                fi
            else
                log_info "Backup directory preserved: $BACKUP_DIR"
            fi
        fi
    fi
    
    # 6. Remove service user (with caution)
    log_info "👤 Handling service user account..."
    
    if id "$SERVICE_USER" &>/dev/null; then
        # Check if user has other processes or files
        local user_processes=$(ps -u "$SERVICE_USER" --no-headers 2>/dev/null | wc -l 2>/dev/null | tr -d ' \n' || echo "0")
        local user_files=$(find / -user "$SERVICE_USER" -not -path "/proc/*" -not -path "/sys/*" 2>/dev/null | wc -l 2>/dev/null | tr -d ' \n' || echo "0")
        
        # Ensure variables are numeric
        user_processes=${user_processes//[^0-9]/}
        user_files=${user_files//[^0-9]/}
        user_processes=${user_processes:-0}
        user_files=${user_files:-0}
        
        if [[ $user_processes -gt 0 ]]; then
            log_warning "User $SERVICE_USER has running processes ($user_processes). Skipping user removal."
        elif [[ $user_files -gt 0 ]]; then
            log_warning "User $SERVICE_USER owns files outside deployment ($user_files). Skipping user removal."
        else
            echo -n "Remove service user '$SERVICE_USER'? [Y/n]: "
            read -r remove_user
            remove_user=${remove_user:-y}
            
            if [[ "$remove_user" =~ ^[Yy]$ ]]; then
                if userdel "$SERVICE_USER" 2>/dev/null; then
                    log_success "Removed service user: $SERVICE_USER"
                else
                    log_warning "Failed to remove service user: $SERVICE_USER"
                fi
            else
                log_info "Service user preserved: $SERVICE_USER"
            fi
        fi
    fi
    
    # 7. Clean up any remaining configuration fragments
    log_info "🧹 Cleaning up remaining configuration fragments..."
    
    # Remove nginx configurations if they exist
    local nginx_configs=(
        "/etc/nginx/sites-available/infra-core"
        "/etc/nginx/sites-enabled/infra-core"
    )
    
    for nginx_config in "${nginx_configs[@]}"; do
        if [[ -f "$nginx_config" ]]; then
            rm -f "$nginx_config" 2>/dev/null && log_success "Removed Nginx config: $nginx_config"
        fi
    done
    
    # Remove any cron jobs
    if command -v crontab &>/dev/null && crontab -u "$SERVICE_USER" -l 2>/dev/null | grep -q "infra-core"; then
        log_info "Found InfraCore cron jobs, removing..."
        crontab -u "$SERVICE_USER" -r 2>/dev/null && log_success "Removed cron jobs"
    fi
    
    # Clean up any remaining processes
    local remaining_processes=$(pgrep -f "infra.*core" 2>/dev/null)
    if [[ -n "$remaining_processes" ]]; then
        log_warning "Found remaining InfraCore processes, terminating..."
        echo "$remaining_processes" | xargs -r kill -TERM 2>/dev/null
        sleep 3
        echo "$remaining_processes" | xargs -r kill -KILL 2>/dev/null || true
    fi
    
    # 8. Final verification
    log_step "Performing final verification..."
    
    local remaining_issues=0
    
    # Check for remaining files
    local remaining_files=$(find /opt /etc /var -name "*infra*core*" -o -name "*infracore*" 2>/dev/null | grep -v "/proc/" | grep -v "/sys/" | head -10)
    if [[ -n "$remaining_files" ]]; then
        log_warning "Some files may still remain:"
        echo "$remaining_files" | sed 's/^/  /'
        remaining_issues=$((remaining_issues + 1))
    fi
    
    # Check for remaining Docker resources
    if command -v docker &>/dev/null; then
        local remaining_docker=$(docker ps -a --filter "name=infra" --format "{{.Names}}" 2>/dev/null)
        if [[ -n "$remaining_docker" ]]; then
            log_warning "Some Docker containers may still remain: $remaining_docker"
            remaining_issues=$((remaining_issues + 1))
        fi
    fi
    
    # Check for remaining services
    for service in "infra-core-console" "infra-core-gate"; do
        if systemctl list-unit-files "$service.service" &>/dev/null; then
            log_warning "Service file may still exist: $service.service"
            remaining_issues=$((remaining_issues + 1))
        fi
    done
    
    # Final summary
    echo
    log_step "Uninstall Summary"
    
    if [[ $uninstall_errors -eq 0 && $remaining_issues -eq 0 ]]; then
        log_success "🎉 InfraCore has been completely uninstalled!"
        log_success "✅ All components removed successfully"
        log_info "   • All services stopped and removed"
        log_info "   • All Docker resources cleaned up"
        log_info "   • All deployment files deleted"
        log_info "   • System configuration restored"
    elif [[ $uninstall_errors -gt 0 || $remaining_issues -gt 0 ]]; then
        log_warning "⚠️  Uninstall completed with some issues:"
        if [[ $uninstall_errors -gt 0 ]]; then
            log_warning "   • $uninstall_errors errors occurred during uninstall"
        fi
        if [[ $remaining_issues -gt 0 ]]; then
            log_warning "   • $remaining_issues potential remaining components detected"
        fi
        log_info "   • Most components were successfully removed"
        log_info "   • You may need to manually clean up remaining items"
    fi
    
    # Force filesystem sync and clear caches to ensure changes are reflected
    log_info "🔄 Synchronizing filesystem changes..."
    sync 2>/dev/null || true
    
    # Clear filesystem caches to ensure accurate disk space reporting
    if [[ -w /proc/sys/vm/drop_caches ]]; then
        echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
        log_success "Filesystem caches cleared"
    fi
    
    # Give the system a moment to fully process the changes
    sleep 2
    
    echo
    log_info "If you plan to reinstall InfraCore in the future:"
    log_info "  • Use the same deployment script with standard options"
    log_info "  • Backups are preserved (if not explicitly removed)"
    log_info "  • System dependencies (Docker, etc.) are still available"
    log_info "  • Wait a few seconds before reinstalling to ensure filesystem sync"
    
    log_info "Thank you for using InfraCore! 🚀"
}

# Main deployment function with enhanced orchestration
main_deploy() {
    log_step "🚀 Starting InfraCore deployment..."
    DEPLOYMENT_STARTED=true
    DEPLOYMENT_START_TIME=$(date +%s)
    DEPLOYMENT_ID="deploy-$(date +%s)-$$"
    
    # Set default deployment type
    DEPLOYMENT_TYPE=${DEPLOYMENT_TYPE:-"docker"}
    
    # Display deployment plan
    show_deployment_plan
    
    # Run comprehensive pre-deployment checks
    pre_deployment_checks
    
    # Install dependencies
    monitor_deployment_progress "Installing dependencies"
    install_dependencies
    add_rollback_step "log_info 'Dependencies rollback not implemented'"
    
    # Setup user and directories
    monitor_deployment_progress "Setting up user and directories"
    setup_user
    setup_directories
    add_rollback_step "userdel -f '$SERVICE_USER' 2>/dev/null || true"
    
    # Create backup if requested
    if [[ "$CREATE_BACKUP" == "true" ]]; then
        monitor_deployment_progress "Creating backup"
        create_backup
    fi
    
    # Update repository
    monitor_deployment_progress "Updating repository"
    update_repository
    add_rollback_step "rm -rf '$DEPLOY_DIR/current'"
    
    # Set up script permissions
    monitor_deployment_progress "Setting up script permissions"
    setup_script_permissions
    
    # Deploy based on type
    monitor_deployment_progress "Deploying application ($DEPLOYMENT_TYPE)"
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        deploy_docker
        add_rollback_step "cd '$DEPLOY_DIR/current' && docker-compose down && docker system prune -f"
    else
        deploy_binary
        add_rollback_step "systemctl stop infra-core-* 2>/dev/null || true"
    fi
    
    # Post-deployment validation
    log_step "🏥 Running health checks..."
    if verify_deployment_health; then
        DEPLOYMENT_STARTED=false  # Successful deployment, disable auto-rollback
        log_success "🎉 InfraCore deployment completed successfully and health check passed!"
        show_deployment_summary
        
        # Clear rollback steps on success
        ROLLBACK_STEPS=()
    else
        log_error "❌ Deployment failed health checks"
        log_info "Initiating automatic rollback..."
        execute_rollback
        exit 1
    fi
    
    # Stop monitoring and generate reports
    stop_resource_monitoring
    
    # Generate deployment metrics
    generate_deployment_metrics
    
    # Perform post-deployment analysis
    perform_post_deployment_analysis
    
    # Create status dashboard
    create_deployment_status_dashboard
    
    # Cleanup temporary files
    cleanup_deployment_temp_files
    
    # Show final status
    show_status
}

# Cleanup temporary files and resources
cleanup_deployment_temp_files() {
    log_step "🧹 Cleaning up temporary files..."
    
    # Remove temporary monitoring files
    rm -f "/tmp/deployment-progress-$DEPLOYMENT_ID" 2>/dev/null || true
    rm -f "/tmp/monitoring-$DEPLOYMENT_ID.pid" 2>/dev/null || true
    rm -f "/tmp/deployment-resources-$DEPLOYMENT_ID.log" 2>/dev/null || true
    
    # Archive deployment logs if they exist
    if [[ -f "$LOG_FILE" ]]; then
        mkdir -p "$DEPLOY_DIR/logs" 2>/dev/null || true
        cp "$LOG_FILE" "$DEPLOY_DIR/logs/deployment-$DEPLOYMENT_ID.log" 2>/dev/null || true
    fi
    
    log_info "✅ Cleanup completed"
}

# Set up permissions for all scripts
setup_script_permissions() {
    log_info "🔐 Setting up script permissions..."
    
    local scripts_dir="$DEPLOY_DIR/current/scripts"
    
    if [[ -d "$scripts_dir" ]]; then
        # Make all shell scripts executable
        find "$scripts_dir" -name "*.sh" -type f -exec chmod +x {} \;
        
        # Set proper ownership
        chown -R "$SERVICE_USER:$SERVICE_USER" "$scripts_dir" 2>/dev/null || true
        
        log_success "Script permissions configured"
        
        # Log available scripts
        log_info "Available scripts:"
        for script in "$scripts_dir"/*.sh; do
            if [[ -f "$script" ]]; then
                local script_name=$(basename "$script")
                log_info "  ✅ $script_name"
            fi
        done
    else
        log_warning "Scripts directory not found: $scripts_dir"
    fi
}

# Generate deployment metrics and report
generate_deployment_metrics() {
    local metrics_file="$DEPLOY_DIR/deployment-metrics.json"
    local deployment_end_time=$(date +%s)
    local deployment_duration=$((deployment_end_time - DEPLOYMENT_START_TIME))
    
    cat > "$metrics_file" <<EOF
{
    "deployment_id": "$DEPLOYMENT_ID",
    "timestamp": "$(date -Iseconds)",
    "duration_seconds": $deployment_duration,
    "environment": "$ENVIRONMENT",
    "deployment_type": "$DEPLOYMENT_TYPE",
    "repository": "$REPO_URL",
    "branch": "$BRANCH",
    "deploy_directory": "$DEPLOY_DIR",
    "service_user": "$SERVICE_USER",
    "backup_created": ${CREATE_BACKUP:-false},
    "rollback_steps": ${#ROLLBACK_STEPS[@]},
    "success": true,
    "health_check_passed": true,
    "system_info": {
        "hostname": "$(hostname)",
        "os": "$(uname -s)",
        "kernel": "$(uname -r)",
        "architecture": "$(uname -m)",
        "cpu_cores": $(nproc),
        "memory_gb": $(free -g | awk '/^Mem:/{print $2}'),
        "disk_usage": "$(df -h $DEPLOY_DIR | tail -1 | awk '{print $5}')"
    }
}
EOF
    
    log_info "📊 Deployment metrics saved to: $metrics_file"
    log_info "⏱️  Total deployment time: $(format_duration $deployment_duration)"
}

# Format duration in human readable format
format_duration() {
    local seconds=$1
    local minutes=$((seconds / 60))
    local remaining_seconds=$((seconds % 60))
    
    if [[ $minutes -gt 0 ]]; then
        echo "${minutes}m ${remaining_seconds}s"
    else
        echo "${seconds}s"
    fi
}

# Show deployment plan to user
show_deployment_plan() {
    echo
    log_info "📋 Deployment Plan:"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "  🎯 Target: $REPO_URL ($BRANCH)"
    log_info "  📁 Directory: $DEPLOY_DIR"
    log_info "  🏷️  Environment: $ENVIRONMENT"
    log_info "  🚀 Type: $DEPLOYMENT_TYPE"
    log_info "  👤 User: $SERVICE_USER"
    log_info "  � Backup: $([ "$CREATE_BACKUP" == "true" ] && echo "✅ Enabled" || echo "❌ Disabled")"
    log_info "  🌐 Mirrors: $([ "$USE_MIRRORS" == "true" ] && echo "✅ Enabled ($MIRROR_REGION)" || echo "❌ Disabled")"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ "$NON_INTERACTIVE" != "true" ]]; then
        echo
        read -p "Continue with this deployment plan? [Y/n]: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            log_info "Deployment cancelled by user"
            exit 0
        fi
    fi
}

# Real-time deployment monitoring
monitor_deployment_progress() {
    local step_name="$1"
    local progress_file="/tmp/deployment-progress-$DEPLOYMENT_ID"
    
    echo "$(date -Iseconds): Starting $step_name" >> "$progress_file"
    log_info "📊 Progress: $step_name"
    
    # Start resource monitoring in background
    if [[ ! -f "/tmp/monitoring-$DEPLOYMENT_ID.pid" ]]; then
        start_resource_monitoring &
        echo $! > "/tmp/monitoring-$DEPLOYMENT_ID.pid"
    fi
}

# Start resource monitoring
start_resource_monitoring() {
    local monitoring_file="/tmp/deployment-resources-$DEPLOYMENT_ID.log"
    
    while [[ -f "/tmp/monitoring-$DEPLOYMENT_ID.pid" ]]; do
        {
            echo "$(date -Iseconds),$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ','),$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}'),$(df "$DEPLOY_DIR" | tail -1 | awk '{print $(NF-1)}' | sed 's/%//')"
        } >> "$monitoring_file" 2>/dev/null
        sleep 5
    done
}

# Stop resource monitoring
stop_resource_monitoring() {
    local pid_file="/tmp/monitoring-$DEPLOYMENT_ID.pid"
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        kill "$pid" 2>/dev/null || true
        rm -f "$pid_file"
        
        # Generate resource usage report
        generate_resource_report
    fi
}

# Generate resource usage report
generate_resource_report() {
    local monitoring_file="/tmp/deployment-resources-$DEPLOYMENT_ID.log"
    local report_file="$DEPLOY_DIR/deployment-resource-report.txt"
    
    if [[ -f "$monitoring_file" ]]; then
        {
            echo "Resource Usage During Deployment"
            echo "================================"
            echo "Deployment ID: $DEPLOYMENT_ID"
            echo "Timestamp: $(date)"
            echo ""
            echo "Load Average (1min), Memory Usage (%), Disk Usage (%)"
            echo "----------------------------------------------------"
            cat "$monitoring_file"
        } > "$report_file"
        
        log_info "📈 Resource usage report saved to: $report_file"
        rm -f "$monitoring_file"
    fi
}

# Performance analysis after deployment
perform_post_deployment_analysis() {
    log_step "📊 Performing post-deployment analysis..."
    
    local analysis_file="$DEPLOY_DIR/post-deployment-analysis.json"
    local current_time=$(date -Iseconds)
    
    # Collect system metrics
    local cpu_info=$(lscpu | grep "CPU(s):" | head -1 | awk '{print $2}')
    local memory_total=$(free -m | grep Mem | awk '{print $2}')
    local memory_used=$(free -m | grep Mem | awk '{print $3}')
    local disk_total=$(df -h "$DEPLOY_DIR" | tail -1 | awk '{print $2}')
    local disk_used=$(df -h "$DEPLOY_DIR" | tail -1 | awk '{print $3}')
    local load_avg=$(uptime | awk -F'load average:' '{print $2}' | xargs)
    
    # Docker specific metrics
    local container_count=0
    local image_count=0
    if command -v docker >/dev/null 2>&1; then
        container_count=$(docker ps -q | wc -l)
        image_count=$(docker images -q | wc -l)
    fi
    
    # Generate analysis report
    cat > "$analysis_file" <<EOF
{
    "analysis_timestamp": "$current_time",
    "deployment_id": "$DEPLOYMENT_ID",
    "system_metrics": {
        "cpu_cores": $cpu_info,
        "memory_total_mb": $memory_total,
        "memory_used_mb": $memory_used,
        "memory_usage_percent": $(echo "scale=2; $memory_used * 100 / $memory_total" | bc -l),
        "disk_total": "$disk_total",
        "disk_used": "$disk_used",
        "load_average": "$load_avg"
    },
    "docker_metrics": {
        "running_containers": $container_count,
        "total_images": $image_count
    },
    "deployment_health": "$(verify_deployment_health >/dev/null 2>&1 && echo "healthy" || echo "unhealthy")",
    "recommendations": $(generate_optimization_recommendations)
}
EOF
    
    log_info "📊 Post-deployment analysis saved to: $analysis_file"
    display_analysis_summary "$analysis_file"
}

# Generate optimization recommendations
generate_optimization_recommendations() {
    local recommendations='[]'
    
    # Check memory usage
    local memory_usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    if [[ $memory_usage -gt 80 ]]; then
        recommendations=$(echo "$recommendations" | jq '. += ["Consider adding more memory or optimizing memory usage"]')
    fi
    
    # Check disk usage
    local disk_usage=$(df "$DEPLOY_DIR" | tail -1 | awk '{print $(NF-1)}' | sed 's/%//')
    if [[ $disk_usage -gt 85 ]]; then
        recommendations=$(echo "$recommendations" | jq '. += ["Consider cleaning up old deployments or expanding disk space"]')
    fi
    
    # Check CPU load
    local load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',')
    local cpu_cores=$(nproc)
    if (( $(echo "$load_avg > $cpu_cores" | bc -l) )); then
        recommendations=$(echo "$recommendations" | jq '. += ["High CPU load detected, consider optimizing or adding more CPU cores"]')
    fi
    
    echo "$recommendations"
}

# Display analysis summary
display_analysis_summary() {
    local analysis_file="$1"
    
    if [[ -f "$analysis_file" ]] && command -v jq >/dev/null 2>&1; then
        echo
        log_info "📊 Post-Deployment Analysis Summary:"
        log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        local memory_usage=$(jq -r '.system_metrics.memory_usage_percent' "$analysis_file")
        local container_count=$(jq -r '.docker_metrics.running_containers' "$analysis_file")
        local health=$(jq -r '.deployment_health' "$analysis_file")
        
        log_info "  💾 Memory Usage: ${memory_usage}%"
        log_info "  🐳 Running Containers: $container_count"
        log_info "  🏥 Health Status: $health"
        
        local recommendations=$(jq -r '.recommendations[]' "$analysis_file" 2>/dev/null)
        if [[ -n "$recommendations" ]]; then
            log_info "  💡 Recommendations:"
            echo "$recommendations" | while IFS= read -r rec; do
                log_info "    • $rec"
            done
        else
            log_info "  ✅ No optimization recommendations at this time"
        fi
        
        log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    fi
}

# Create deployment status dashboard
create_deployment_status_dashboard() {
    local dashboard_file="$DEPLOY_DIR/deployment-status.html"
    
    cat > "$dashboard_file" <<'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>InfraCore Deployment Status</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .status-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .status-card { background: #f8f9fa; padding: 15px; border-radius: 8px; border-left: 4px solid #28a745; }
        .status-card.warning { border-left-color: #ffc107; }
        .status-card.error { border-left-color: #dc3545; }
        .metric { margin: 10px 0; }
        .metric-label { font-weight: bold; color: #495057; }
        .metric-value { color: #007bff; }
        .refresh-btn { background: #007bff; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
    </style>
    <script>
        function refreshStatus() {
            location.reload();
        }
        setInterval(refreshStatus, 30000); // Auto-refresh every 30 seconds
    </script>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 InfraCore Deployment Status</h1>
            <button class="refresh-btn" onclick="refreshStatus()">Refresh Status</button>
            <p>Last Updated: <span id="timestamp">TIMESTAMP_PLACEHOLDER</span></p>
        </div>
        
        <div class="status-grid">
            <div class="status-card">
                <h3>📊 System Metrics</h3>
                <div class="metric">
                    <span class="metric-label">CPU Usage:</span>
                    <span class="metric-value">CPU_USAGE_PLACEHOLDER</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Memory Usage:</span>
                    <span class="metric-value">MEMORY_USAGE_PLACEHOLDER</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Disk Usage:</span>
                    <span class="metric-value">DISK_USAGE_PLACEHOLDER</span>
                </div>
            </div>
            
            <div class="status-card">
                <h3>🐳 Docker Status</h3>
                <div class="metric">
                    <span class="metric-label">Running Containers:</span>
                    <span class="metric-value">CONTAINER_COUNT_PLACEHOLDER</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Docker Images:</span>
                    <span class="metric-value">IMAGE_COUNT_PLACEHOLDER</span>
                </div>
            </div>
            
            <div class="status-card">
                <h3>🌐 Service Endpoints</h3>
                <div class="metric">
                    <span class="metric-label">Main Service:</span>
                    <span class="metric-value">SERVICE_STATUS_PLACEHOLDER</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Health Check:</span>
                    <span class="metric-value">HEALTH_STATUS_PLACEHOLDER</span>
                </div>
            </div>
            
            <div class="status-card">
                <h3>📈 Deployment Info</h3>
                <div class="metric">
                    <span class="metric-label">Deployment ID:</span>
                    <span class="metric-value">DEPLOYMENT_ID_PLACEHOLDER</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Environment:</span>
                    <span class="metric-value">ENVIRONMENT_PLACEHOLDER</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Deploy Time:</span>
                    <span class="metric-value">DEPLOY_TIME_PLACEHOLDER</span>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
EOF
    
    # Update placeholders with actual values
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1 | cut -d',' -f1 | xargs)
    local memory_usage=$(free | grep Mem | awk '{printf("%.1f", $3/$2 * 100.0)}')
    local disk_usage=$(df "$DEPLOY_DIR" | tail -1 | awk '{print $(NF-1)}')
    local container_count=$(docker ps -q 2>/dev/null | wc -l || echo "N/A")
    local image_count=$(docker images -q 2>/dev/null | wc -l || echo "N/A")
    
    sed -i "s/TIMESTAMP_PLACEHOLDER/$(date)/" "$dashboard_file"
    sed -i "s/CPU_USAGE_PLACEHOLDER/${cpu_usage}%/" "$dashboard_file"
    sed -i "s/MEMORY_USAGE_PLACEHOLDER/${memory_usage}%/" "$dashboard_file"
    sed -i "s/DISK_USAGE_PLACEHOLDER/${disk_usage}/" "$dashboard_file"
    sed -i "s/CONTAINER_COUNT_PLACEHOLDER/$container_count/" "$dashboard_file"
    sed -i "s/IMAGE_COUNT_PLACEHOLDER/$image_count/" "$dashboard_file"
    sed -i "s/SERVICE_STATUS_PLACEHOLDER/$(verify_deployment_health >/dev/null 2>&1 && echo "🟢 Running" || echo "🔴 Issues")/" "$dashboard_file"
    sed -i "s/HEALTH_STATUS_PLACEHOLDER/$(verify_deployment_health >/dev/null 2>&1 && echo "✅ Healthy" || echo "⚠️ Unhealthy")/" "$dashboard_file"
    sed -i "s/DEPLOYMENT_ID_PLACEHOLDER/$DEPLOYMENT_ID/" "$dashboard_file"
    sed -i "s/ENVIRONMENT_PLACEHOLDER/$ENVIRONMENT/" "$dashboard_file"
    sed -i "s/DEPLOY_TIME_PLACEHOLDER/$(format_duration $(($(date +%s) - DEPLOYMENT_START_TIME)))/" "$dashboard_file"
    
    log_info "📊 Status dashboard created: $dashboard_file"
    
    # Also create a simple status endpoint
    if command -v python3 >/dev/null 2>&1; then
        create_status_api_endpoint
    fi
}

# Create a simple status API endpoint
create_status_api_endpoint() {
    local api_script="$DEPLOY_DIR/status-api.py"
    
    cat > "$api_script" <<'EOF'
#!/usr/bin/env python3
import json
import subprocess
import http.server
import socketserver
from datetime import datetime

def get_system_status():
    try:
        # Get system metrics
        cpu_result = subprocess.run(['top', '-bn1'], capture_output=True, text=True)
        cpu_usage = "N/A"
        for line in cpu_result.stdout.split('\n'):
            if 'Cpu(s)' in line:
                cpu_usage = line.split('%')[0].split()[-1]
                break
        
        memory_result = subprocess.run(['free'], capture_output=True, text=True)
        memory_lines = memory_result.stdout.split('\n')[1].split()
        memory_usage = round(int(memory_lines[2]) / int(memory_lines[1]) * 100, 1)
        
        # Get Docker status
        docker_ps = subprocess.run(['docker', 'ps', '-q'], capture_output=True, text=True)
        container_count = len([l for l in docker_ps.stdout.strip().split('\n') if l])
        
        return {
            'timestamp': datetime.now().isoformat(),
            'cpu_usage': f"{cpu_usage}%",
            'memory_usage': f"{memory_usage}%",
            'running_containers': container_count,
            'status': 'healthy'
        }
    except Exception as e:
        return {
            'timestamp': datetime.now().isoformat(),
            'error': str(e),
            'status': 'error'
        }

class StatusHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/status':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            status = get_system_status()
            self.wfile.write(json.dumps(status).encode())
        else:
            super().do_GET()

if __name__ == "__main__":
    PORT = 8087
    with socketserver.TCPServer(("", PORT), StatusHandler) as httpd:
        print(f"Status API server running on port {PORT}")
        httpd.serve_forever()
EOF
    
    chmod +x "$api_script"
    log_info "🔗 Status API created: $api_script (run with python3 to start)"
}

# Show deployment completion summary
show_deployment_summary() {
    echo
    log_success "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_success "🚀 InfraCore is now running successfully!"
    log_success "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Get server IP
    local server_ip=$(curl -s -4 icanhazip.com 2>/dev/null || curl -s -4 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}' || echo "localhost")
    
    echo
    log_info "🌐 Access your InfraCore instance:"
    if [[ "${CUSTOM_SSL_ENABLED:-true}" == "true" && -n "${CUSTOM_DOMAIN}" ]]; then
        log_info "  🔒 Web Interface: https://${CUSTOM_DOMAIN}"
        log_info "  🔌 API Endpoint:  https://${CUSTOM_DOMAIN}/api/v1"
    else
        log_info "  🌐 Web Interface: http://${server_ip}:${CUSTOM_HTTP_PORT:-80}"
        log_info "  🔌 API Endpoint:  http://${server_ip}:${CUSTOM_API_PORT:-8082}/api/v1"
    fi
    
    echo
    log_info "📋 Quick Commands:"
    log_info "  📊 Check Status:     sudo $0 --status"
    log_info "  📝 View Logs:        sudo $0 --logs"
    log_info "  🔄 Restart Services: sudo $0 --restart"
    log_info "  ⚙️  Reconfigure:     sudo $0 --configure"
    log_info "  🛠️  Troubleshoot:     sudo $0 --troubleshoot"
    
    echo
    log_info "📚 Documentation and Support:"
    log_info "  📖 GitHub: https://github.com/last-emo-boy/infra-core"
    log_info "  🐛 Issues: https://github.com/last-emo-boy/infra-core/issues"
    
    # Show current configuration
    show_config_summary
    
    echo
    log_success "✨ Enjoy your InfraCore deployment!"
    log_success "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Main execution
main() {
    # Initialize variables
    DEPLOYMENT_TYPE="docker"
    CREATE_BACKUP=false
    ACTION="deploy"
    
    # Parse arguments
    parse_args "$@"
    
    # Ensure basic log directory exists early
    mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
    
    # Check root access
    check_root
    
    # Check configuration files
    check_configuration
    
    # Welcome message for interactive deployments
    if [[ "$ACTION" == "deploy" && "$NON_INTERACTIVE" != "true" ]]; then
        echo
        log_info "🎉 Welcome to InfraCore Deployment!"
        log_info "This script will help you deploy and configure InfraCore with ease."
        echo
    fi
    
    # Execute based on action
    case "$ACTION" in
        "deploy")
            main_deploy
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs
            ;;
        "restart")
            restart_services
            ;;
        "stop")
            stop_services
            ;;
        "start")
            start_services
            ;;
        "rollback")
            rollback
            ;;
        "update")
            quick_update
            ;;
        "upgrade")
            upgrade
            ;;
        "configure")
            log_step "🔧 Running interactive configuration setup..."
            cd "$DEPLOY_DIR/current" || { log_error "Deployment directory not found"; exit 1; }
            setup_interactive_config
            log_success "Configuration completed successfully!"
            log_info "Run './server-deploy.sh --restart' to apply the new configuration"
            ;;
        "validate-config")
            log_step "📋 Validating configuration files..."
            cd "$DEPLOY_DIR/current" || { log_error "Deployment directory not found"; exit 1; }
            validate_configuration_files
            show_config_summary
            log_success "Configuration validation completed!"
            ;;
        "test-mirrors")
            log_step "🧪 Testing mirror speeds..."
            select_optimal_mirrors
            log_success "Mirror speed test completed!"
            ;;
        "test-install")
            log_step "✅ Running installation verification tests..."
            if [[ -f "$DEPLOY_DIR/current/scripts/test-installation.sh" ]]; then
                chmod +x "$DEPLOY_DIR/current/scripts/test-installation.sh"
                cd "$DEPLOY_DIR/current" && bash scripts/test-installation.sh
            else
                log_error "Test script not found: $DEPLOY_DIR/current/scripts/test-installation.sh"
                exit 1
            fi
            ;;
        "test-api")
            log_step "🧪 Running API functionality tests..."
            if [[ -f "$DEPLOY_DIR/current/scripts/test-api.sh" ]]; then
                chmod +x "$DEPLOY_DIR/current/scripts/test-api.sh"
                cd "$DEPLOY_DIR/current" && bash scripts/test-api.sh
            else
                log_error "Test script not found: $DEPLOY_DIR/current/scripts/test-api.sh"
                exit 1
            fi
            ;;
        "troubleshoot")
            log_step "🔧 Running troubleshooting diagnostics..."
            if [[ -f "$DEPLOY_DIR/current/scripts/troubleshoot.sh" ]]; then
                chmod +x "$DEPLOY_DIR/current/scripts/troubleshoot.sh"
                cd "$DEPLOY_DIR/current" && bash scripts/troubleshoot.sh
            else
                log_error "Troubleshoot script not found: $DEPLOY_DIR/current/scripts/troubleshoot.sh"
                exit 1
            fi
            ;;
        "uninstall")
            uninstall_infracore
            ;;
        *)
            log_error "Unknown action: $ACTION"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"