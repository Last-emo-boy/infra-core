#!/bin/bash
# InfraCore Linux Server Deployment Script
# Author: last-emo-boy
# Usage: ./server-deploy.sh [options]

set -e

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

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Functions for colored output
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1" | tee -a "$LOG_FILE"
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
    --rollback              Rollback to previous deployment
    --status                Show deployment status
    --logs                  Show service logs
    --restart               Restart services
    --stop                  Stop services
    --start                 Start services
    --update                Update to latest version without full deployment

Examples:
    $0                                    # Deploy latest main branch
    $0 --branch develop --env staging     # Deploy develop branch to staging
    $0 --backup                          # Deploy with backup
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
                curl -fsSL https://get.docker.com -o get-docker.sh
                sh get-docker.sh
                systemctl enable docker
                systemctl start docker
                rm get-docker.sh
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

# Create backup
create_backup() {
    if [[ "$CREATE_BACKUP" == "true" ]]; then
        log_step "Creating backup..."
        
        local backup_name="infra-core-backup-$(date +%Y%m%d-%H%M%S)"
        local backup_path="$BACKUP_DIR/$backup_name"
        
        if [[ -d "$DEPLOY_DIR/current" ]]; then
            log_info "Backing up current deployment to: $backup_path"
            mkdir -p "$backup_path"
            
            # Backup application files
            cp -r "$DEPLOY_DIR/current" "$backup_path/"
            
            # Backup database if exists
            if [[ -f "/var/lib/infra-core/database.db" ]]; then
                cp "/var/lib/infra-core/database.db" "$backup_path/"
            fi
            
            # Backup configuration
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

# Clone or update repository
update_repository() {
    log_step "Updating repository..."
    
    local temp_dir="$DEPLOY_DIR/tmp-$(date +%s)"
    
    log_info "Cloning repository to: $temp_dir"
    git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$temp_dir"
    
    # Get commit info
    local commit_hash=$(cd "$temp_dir" && git rev-parse HEAD)
    local commit_message=$(cd "$temp_dir" && git log -1 --pretty=format:"%s")
    
    log_info "Cloned commit: $commit_hash"
    log_info "Commit message: $commit_message"
    
    # Move to deployment location
    if [[ -d "$DEPLOY_DIR/current" ]]; then
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

# Docker deployment
deploy_docker() {
    log_step "Deploying with Docker..."
    
    cd "$DEPLOY_DIR/current"
    
    # Login to GitHub Container Registry if credentials available
    if [[ -n "$GITHUB_TOKEN" ]]; then
        log_info "Logging into GitHub Container Registry..."
        echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_ACTOR" --password-stdin
    fi
    
    # Use latest image or build locally
    if [[ -n "$GITHUB_TOKEN" ]]; then
        log_info "Pulling latest image from registry..."
        docker pull "$REGISTRY/$IMAGE_NAME:latest" || {
            log_warning "Failed to pull image, building locally..."
            docker-compose -f docker-compose.yml build
        }
    else
        log_info "Building Docker images locally..."
        docker-compose -f docker-compose.yml build
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
    
    # Check requirements
    check_requirements
    
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
    
    log_success "🎉 InfraCore deployment completed successfully!"
    
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
        *)
            log_error "Unknown action: $ACTION"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"