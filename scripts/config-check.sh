#!/bin/bash
# InfraCore Configuration Management Tool
# Author: last-emo-boy
# Usage: ./config-check.sh [options]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default paths
CONFIG_DIR="./configs"
DOCKER_COMPOSE_FILE="./docker-compose.yml"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show usage
show_usage() {
    cat << EOF
InfraCore Configuration Management Tool

Usage: $0 [OPTIONS]

Options:
    -h, --help              Show this help message
    -c, --config-dir DIR    Configuration directory (default: ./configs)
    --check                 Check configuration files
    --fix                   Fix missing or broken configuration files
    --validate              Validate YAML syntax
    --docker-check          Check Docker configuration mounts
    --docker-fix            Fix Docker configuration mounts

Examples:
    $0 --check                      # Check all configuration files
    $0 --fix                        # Create missing config files
    $0 --validate                   # Validate YAML syntax
    $0 --docker-check               # Check Docker compose config mounts
    $0 -c /custom/path --check      # Check configs in custom directory
EOF
}

# Parse arguments
ACTION="check"
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -c|--config-dir)
            CONFIG_DIR="$2"
            shift 2
            ;;
        --check)
            ACTION="check"
            shift
            ;;
        --fix)
            ACTION="fix"
            shift
            ;;
        --validate)
            ACTION="validate"
            shift
            ;;
        --docker-check)
            ACTION="docker-check"
            shift
            ;;
        --docker-fix)
            ACTION="docker-fix"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Check configuration files
check_configs() {
    log_info "🔍 Checking configuration files in: $CONFIG_DIR"
    
    if [[ ! -d "$CONFIG_DIR" ]]; then
        log_error "Configuration directory not found: $CONFIG_DIR"
        return 1
    fi
    
    local required_configs=("production.yaml" "development.yaml" "testing.yaml")
    local missing_configs=()
    local broken_configs=()
    
    for config_file in "${required_configs[@]}"; do
        local config_path="$CONFIG_DIR/$config_file"
        
        if [[ ! -f "$config_path" ]]; then
            log_warning "Missing: $config_file"
            missing_configs+=("$config_file")
        else
            log_success "Found: $config_file"
            
            # Check YAML syntax
            if command -v python3 >/dev/null 2>&1; then
                if ! python3 -c "import yaml; yaml.safe_load(open('$config_path'))" 2>/dev/null; then
                    log_warning "YAML syntax error in: $config_file"
                    broken_configs+=("$config_file")
                fi
            elif command -v yq >/dev/null 2>&1; then
                if ! yq eval . "$config_path" >/dev/null 2>&1; then
                    log_warning "YAML syntax error in: $config_file"
                    broken_configs+=("$config_file")
                fi
            fi
        fi
    done
    
    # Summary
    echo
    log_info "📋 Configuration Summary:"
    log_info "  Total required: ${#required_configs[@]}"
    log_info "  Found: $((${#required_configs[@]} - ${#missing_configs[@]}))"
    log_info "  Missing: ${#missing_configs[@]}"
    log_info "  Broken: ${#broken_configs[@]}"
    
    if [[ ${#missing_configs[@]} -gt 0 ]]; then
        echo
        log_warning "Missing configuration files:"
        for config in "${missing_configs[@]}"; do
            echo "  - $config"
        done
        log_info "Run with --fix to create missing files"
    fi
    
    if [[ ${#broken_configs[@]} -gt 0 ]]; then
        echo
        log_warning "Broken configuration files:"
        for config in "${broken_configs[@]}"; do
            echo "  - $config"
        done
        log_info "Run with --fix to repair broken files"
    fi
    
    if [[ ${#missing_configs[@]} -eq 0 && ${#broken_configs[@]} -eq 0 ]]; then
        echo
        log_success "✅ All configuration files are present and valid!"
    fi
}

# Fix configuration files
fix_configs() {
    log_info "🔧 Fixing configuration files..."
    
    # Create directory if missing
    if [[ ! -d "$CONFIG_DIR" ]]; then
        log_info "Creating configuration directory: $CONFIG_DIR"
        mkdir -p "$CONFIG_DIR"
    fi
    
    # Create missing files
    local configs=("production.yaml" "development.yaml" "testing.yaml")
    
    for config_file in "${configs[@]}"; do
        local config_path="$CONFIG_DIR/$config_file"
        
        if [[ ! -f "$config_path" ]] || ! python3 -c "import yaml; yaml.safe_load(open('$config_path'))" 2>/dev/null; then
            log_info "Creating/fixing: $config_file"
            
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
            fi
        else
            log_success "OK: $config_file"
        fi
    done
    
    log_success "Configuration files fixed!"
}

# Validate YAML syntax
validate_configs() {
    log_info "📝 Validating YAML syntax..."
    
    if ! command -v python3 >/dev/null 2>&1; then
        log_warning "Python3 not found, skipping YAML validation"
        return 0
    fi
    
    local config_files=("$CONFIG_DIR"/*.yaml "$CONFIG_DIR"/*.yml)
    local valid_count=0
    local invalid_count=0
    
    for config_file in "${config_files[@]}"; do
        if [[ -f "$config_file" ]]; then
            local filename=$(basename "$config_file")
            if python3 -c "import yaml; yaml.safe_load(open('$config_file'))" 2>/dev/null; then
                log_success "Valid YAML: $filename"
                ((valid_count++))
            else
                log_error "Invalid YAML: $filename"
                python3 -c "import yaml; yaml.safe_load(open('$config_file'))" 2>&1 | head -3
                ((invalid_count++))
            fi
        fi
    done
    
    echo
    log_info "Validation Summary:"
    log_info "  Valid files: $valid_count"
    log_info "  Invalid files: $invalid_count"
    
    if [[ $invalid_count -eq 0 ]]; then
        log_success "✅ All YAML files are valid!"
    else
        log_warning "⚠️  Some YAML files have syntax errors"
    fi
}

# Check Docker configuration mounts
check_docker_config() {
    log_info "🐳 Checking Docker configuration mounts..."
    
    if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
        log_error "Docker Compose file not found: $DOCKER_COMPOSE_FILE"
        return 1
    fi
    
    # Check for config mount
    if grep -q "./configs:/app/configs" "$DOCKER_COMPOSE_FILE"; then
        log_success "✅ Configuration mount found in docker-compose.yml"
    else
        log_warning "⚠️  Configuration mount not found in docker-compose.yml"
        log_info "Expected: './configs:/app/configs:ro'"
        log_info "Run with --docker-fix to add the mount"
    fi
    
    # Check if configs directory exists
    if [[ -d "./configs" ]]; then
        log_success "✅ Configuration directory exists"
    else
        log_warning "⚠️  Configuration directory not found: ./configs"
    fi
}

# Fix Docker configuration mounts
fix_docker_config() {
    log_info "🔧 Fixing Docker configuration mounts..."
    
    if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
        log_error "Docker Compose file not found: $DOCKER_COMPOSE_FILE"
        return 1
    fi
    
    # Backup original file
    local backup_file="$DOCKER_COMPOSE_FILE.backup-$(date +%Y%m%d-%H%M%S)"
    cp "$DOCKER_COMPOSE_FILE" "$backup_file"
    log_info "Backed up docker-compose.yml to: $backup_file"
    
    # Add config mount if not exists
    if ! grep -q "./configs:/app/configs" "$DOCKER_COMPOSE_FILE"; then
        # Find volumes section and add config mount
        sed -i '/- \.\/certs:\/etc\/infra-core\/certs/a\      - ./configs:/app/configs:ro  # Mount configs as read-only' "$DOCKER_COMPOSE_FILE"
        log_success "✅ Added configuration mount to docker-compose.yml"
    else
        log_success "✅ Configuration mount already exists"
    fi
    
    # Create configs directory if missing
    if [[ ! -d "./configs" ]]; then
        mkdir -p "./configs"
        log_success "✅ Created configuration directory"
    fi
}

# Configuration templates
create_production_config() {
    cat > "$1" << 'EOF'
# Production Environment Configuration
gate:
  host: "0.0.0.0"
  ports:
    http: 80
    https: 443
  logs:
    level: "info"
    console: false
    file: "/var/log/infra-core/gate.log"
  acme:
    directory_url: "https://acme-v02.api.letsencrypt.org/directory"
    email: "admin@example.com"
    cache_dir: "/etc/infra-core/certs"
    challenge_type: "http-01"
    enabled: true

console:
  host: "0.0.0.0"
  port: 8082
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
      secret: ""  # Must be set via environment variable
      expires_hours: 8
    session:
      timeout_minutes: 30
  cors:
    enabled: true
    origins: ["https://console.example.com"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
EOF
}

create_development_config() {
    cat > "$1" << 'EOF'
# Development Environment Configuration
gate:
  host: "0.0.0.0"
  ports:
    http: 8080
    https: 8443
  logs:
    level: "debug"
    console: true
    file: "/var/log/infra-core/gate.log"
  acme:
    enabled: false

console:
  host: "0.0.0.0"
  port: 3000
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
    origins: ["http://localhost:3000", "http://localhost:5173"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
EOF
}

create_testing_config() {
    cat > "$1" << 'EOF'
# Testing Environment Configuration
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
      secret: "test-secret-key"
      expires_hours: 1
    session:
      timeout_minutes: 30
  cors:
    enabled: true
    origins: ["*"]
    methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    headers: ["Content-Type", "Authorization"]
EOF
}

# Main execution
main() {
    case "$ACTION" in
        "check")
            check_configs
            ;;
        "fix")
            fix_configs
            ;;
        "validate")
            validate_configs
            ;;
        "docker-check")
            check_docker_config
            ;;
        "docker-fix")
            fix_docker_config
            ;;
        *)
            log_error "Unknown action: $ACTION"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"