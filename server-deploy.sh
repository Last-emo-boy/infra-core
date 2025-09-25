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

# Set up error trap
trap 'error_handler ${LINENO} $?' ERR

# Comprehensive system validation
validate_system_requirements() {
    log_step "Validating system requirements..."
    
    # Check minimum disk space (5GB)
    local available_space_kb
    available_space_kb=$(df "$DEPLOY_DIR" 2>/dev/null | awk 'NR==2{print $4}' || echo "0")
    local available_space_gb=$((available_space_kb / 1024 / 1024))
    
    if [[ $available_space_gb -lt 5 ]]; then
        log_error "Insufficient disk space: ${available_space_gb}GB available, 5GB required"
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
    --restart               Restart services
    --stop                  Stop services
    --start                 Start services
    --update                Update to latest version without full deployment
    --upgrade               Interactive upgrade with confirmation prompt

Examples:
    $0                                    # Deploy latest main branch (no mirrors)
    $0 --mirror cn                        # Deploy with Chinese mirrors
    $0 --mirror                           # Deploy with auto-detected mirrors
    $0 --branch develop --env staging     # Deploy develop branch to staging
    $0 --backup --mirror cn               # Deploy with backup and Chinese mirrors
    $0 --upgrade --mirror                 # Interactive upgrade with mirrors
    $0 --status                          # Show current status
    $0 --logs                            # Show service logs
    $0 --update                          # Quick update to latest version

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
                # Add Docker GPG key and repository
                apt-get install -y ca-certificates curl gnupg lsb-release
                mkdir -p /etc/apt/keyrings
                curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
                echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
                apt-get update
                apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
                systemctl enable docker
                systemctl start docker
            fi
            
            # Install Docker Compose if not present
            if ! command -v docker-compose &> /dev/null; then
                log_info "Installing Docker Compose..."
                curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
                chmod +x /usr/local/bin/docker-compose
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
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_info "Creating directory: $dir"
            mkdir -p "$dir"
        fi
        chown "$SERVICE_USER:$SERVICE_USER" "$dir"
        chmod 755 "$dir"
    done
    
    # Setup log file
    touch "$LOG_FILE"
    chown "$SERVICE_USER:$SERVICE_USER" "$LOG_FILE"
    chmod 644 "$LOG_FILE"
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
            source_size=$(du -sb "$DEPLOY_DIR/current" 2>/dev/null | cut -f1 || echo "0")
        fi
        
        local available_space
        available_space=$(df "$BACKUP_DIR" | awk 'NR==2{print $4}' | head -1)
        available_space=$((available_space * 1024))
        
        if [[ $source_size -gt $((available_space / 2)) ]]; then
            log_error "Insufficient space for backup. Need: $((source_size / 1024 / 1024))MB, Available: $((available_space / 1024 / 1024))MB"
            return 1
        fi
        
        if [[ -d "$DEPLOY_DIR/current" ]]; then
            log_info "Creating backup: $backup_name"
            mkdir -p "$backup_path" "$temp_backup_dir"
            
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
                
                # Generate file checksums
                log_info "Generating checksums..."
                find "$temp_backup_dir/current" -type f -exec sha256sum {} \; > "$temp_backup_dir/checksums.sha256"
                
                # Update manifest with file list
                find "$temp_backup_dir/current" -type f | jq -R . | jq -s . > "$temp_backup_dir/file_list.json"
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
                
                # List and backup relevant volumes
                docker volume ls -q | grep -E "(infra|core)" | while read -r volume; do
                    if [[ -n "$volume" ]]; then
                        log_info "Backing up volume: $volume"
                        docker run --rm \
                            -v "$volume:/source:ro" \
                            -v "$volumes_backup:/backup" \
                            alpine:latest \
                            tar czf "/backup/$volume.tar.gz" -C /source . 2>/dev/null || true
                    fi
                done
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
            
            # Compress backup
            log_info "Compressing backup..."
            if tar czf "$backup_path.tar.gz" -C "$temp_backup_dir" .; then
                # Verify backup integrity
                log_info "Verifying backup integrity..."
                if tar tzf "$backup_path.tar.gz" >/dev/null 2>&1; then
                    log_success "Backup created and verified: $backup_name.tar.gz"
                    
                    # Create symlink to latest backup
                    ln -sf "$backup_name.tar.gz" "$BACKUP_DIR/latest-backup.tar.gz"
                    
                    # Cleanup old backups (keep last 5)
                    cleanup_old_backups
                else
                    log_error "Backup verification failed"
                    rm -f "$backup_path.tar.gz"
                    rm -rf "$temp_backup_dir"
                    return 1
                fi
            else
                log_error "Failed to compress backup"
                rm -rf "$temp_backup_dir"
                return 1
            fi
            
            # Cleanup temp directory
            rm -rf "$temp_backup_dir"
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
detect_optimal_region() {
    # If mirrors are not enabled, always return global
    if [[ "$USE_MIRRORS" != "true" ]]; then
        echo "global"
        return
    fi
    
    local china_sites=("baidu.com" "qq.com" "taobao.com" "163.com")
    local global_sites=("google.com" "github.com" "cloudflare.com")
    local china_score=0
    local global_score=0
    
    log_info "Detecting optimal mirror region..."
    
    # Test China sites
    for site in "${china_sites[@]}"; do
        if timeout 5 ping -c 1 -W 2 "$site" &>/dev/null; then
            china_score=$((china_score + 1))
            log_info "✓ Chinese site $site accessible"
        fi
    done
    
    # Test global sites  
    for site in "${global_sites[@]}"; do
        if timeout 5 ping -c 1 -W 2 "$site" &>/dev/null; then
            global_score=$((global_score + 1))
            log_info "✓ Global site $site accessible"
        fi
    done
    
    # Speed test for mirror selection
    local mirror_speeds=()
    
    # Test Alpine mirror speeds
    log_info "Testing mirror speeds..."
    for mirror in "mirrors.tuna.tsinghua.edu.cn" "mirrors.ustc.edu.cn" "dl-cdn.alpinelinux.org"; do
        local speed=$(timeout 10 curl -s -w "%{time_total}" -o /dev/null "https://$mirror/alpine/v3.18/main/x86_64/APKINDEX.tar.gz" 2>/dev/null || echo "999")
        mirror_speeds+=("$mirror:$speed")
        log_info "Mirror $mirror response time: ${speed}s"
    done
    
    # Determine best region
    if [[ $china_score -ge 2 ]] && [[ $china_score -gt $global_score ]]; then
        echo "cn"
        log_success "Selected Chinese mirrors (china_score=$china_score, global_score=$global_score)"
    else
        echo "global" 
        log_success "Selected global mirrors (china_score=$china_score, global_score=$global_score)"
    fi
}

# Build Docker images with optional mirror optimization  
build_with_mirrors() {
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
    
    case "$region" in
        "cn")
            log_info "Configuring build for Chinese mirrors..."
            cat > "$compose_override" << 'EOF'
version: '3.8'
services:
  infra-core:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        - BUILD_REGION=cn
        - ALPINE_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/alpine
        - GO_PROXY=https://goproxy.cn,direct
        - NPM_REGISTRY=https://registry.npmmirror.com/
EOF
            ;;
        *)
            log_info "Using global configuration with automatic fallback..."
            cat > "$compose_override" << 'EOF'
version: '3.8'
services:
  infra-core:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        - BUILD_REGION=global
EOF
            ;;
    esac
    
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
                build_cmd="docker-compose -f docker-compose.yml build --parallel"
                timeout_duration=900
                ;;
            "standard_no_cache")
                build_cmd="docker-compose -f docker-compose.yml build --no-cache --parallel" 
                timeout_duration=1200
                ;;
            "fallback_minimal")
                # Clear build cache and try minimal build
                log_warning "Attempting fallback build with cache cleanup..."
                docker builder prune -f 2>/dev/null || true
                docker system prune -f 2>/dev/null || true
                build_cmd="docker-compose -f docker-compose.yml build --no-cache"
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
        error_count=$(tail -20 "$LOG_FILE" | grep -i error | wc -l || echo "0")
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
            log_warning "Failed to pull image, building locally with mirror optimization..."
            build_with_mirrors
        }
    else
        log_info "Building Docker images locally with mirror optimization..."
        build_with_mirrors
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
    
    # Start services
    systemctl daemon-reload
    systemctl enable infra-core-console
    systemctl enable infra-core-gate
    systemctl restart infra-core-console
    systemctl restart infra-core-gate
    
    # Wait for services to be ready
    wait_for_services
}

# Setup environment configuration
setup_environment_config() {
    log_step "Setting up environment configuration..."
    
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
    
    # Set environment variables
    cat > "/etc/infra-core/environment" << EOF
INFRA_CORE_ENV=$ENVIRONMENT
INFRA_CORE_CONFIG_PATH=/etc/infra-core/config.yaml
INFRA_CORE_DATA_DIR=/var/lib/infra-core
INFRA_CORE_LOG_DIR=/var/log/infra-core
EOF
    
    chown "$SERVICE_USER:$SERVICE_USER" "/etc/infra-core/environment"
    chmod 600 "/etc/infra-core/environment"
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

# Main deployment function
main_deploy() {
    log_step "Starting InfraCore deployment..."
    
    # Set default deployment type
    DEPLOYMENT_TYPE=${DEPLOYMENT_TYPE:-"docker"}
    
    log_info "Deployment configuration:"
    log_info "  Repository: $REPO_URL"
    log_info "  Branch: $BRANCH"
    log_info "  Environment: $ENVIRONMENT"
    log_info "  Deployment Type: $DEPLOYMENT_TYPE"
    log_info "  Deploy Directory: $DEPLOY_DIR"
    log_info "  Service User: $SERVICE_USER"
    
    # Enhanced system validation
    validate_system_requirements
    validate_dependencies
    
    # Install dependencies
    install_dependencies
    
    # Setup user and directories
    setup_user
    setup_directories
    
    # Create backup if requested
    create_backup
    
    # Update repository
    update_repository
    
    # Deploy based on type
    if [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then
        deploy_docker
    else
        deploy_binary
    fi
    
    # Post-deployment validation
    if verify_deployment_health; then
        log_success "🎉 InfraCore deployment completed successfully and health check passed!"
    else
        log_warning "Deployment completed but health check failed. Please review the system."
    fi
    
    # Show final status
    show_status
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
        *)
            log_error "Unknown action: $ACTION"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"