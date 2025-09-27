#!/bin/bash
# InfraCore Services Startup Script

set -e

# Colors for logging
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Set environment
export INFRA_CORE_ENV=${INFRA_CORE_ENV:-production}
export INFRA_CORE_DATA_DIR=${INFRA_CORE_DATA_DIR:-/var/lib/infra-core}
export INFRA_CORE_LOG_DIR=${INFRA_CORE_LOG_DIR:-/var/log/infra-core}

log_info "🚀 Starting InfraCore Services"
log_info "Environment: $INFRA_CORE_ENV"

# Create necessary directories
mkdir -p "$INFRA_CORE_DATA_DIR" "$INFRA_CORE_LOG_DIR"

# Function to start a service in background
start_service() {
    local service_name=$1
    local service_binary=$2
    local service_args=${3:-""}
    
    log_info "Starting $service_name..."
    
    # Start service in background
    nohup "$service_binary" $service_args > "$INFRA_CORE_LOG_DIR/${service_name}.log" 2>&1 &
    local pid=$!
    
    # Wait a moment and check if it's still running
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        log_success "$service_name started with PID $pid"
        echo "$pid" > "/tmp/${service_name}.pid"
        return 0
    else
        log_error "$service_name failed to start"
        return 1
    fi
}

# Start services
log_info "🎯 Starting individual services..."

# Start Console API (8082)
if command -v console >/dev/null 2>&1; then
    start_service "console" "console"
elif command -v /usr/local/bin/console >/dev/null 2>&1; then
    start_service "console" "/usr/local/bin/console"
else
    log_error "Console binary not found"
fi

# Start Gate (80/443)
if command -v gate >/dev/null 2>&1; then
    start_service "gate" "gate"
elif command -v /usr/local/bin/gate >/dev/null 2>&1; then
    start_service "gate" "/usr/local/bin/gate"
else
    log_error "Gate binary not found"
fi

# Wait for services to be ready
log_info "⏳ Waiting for services to be ready..."
sleep 5

# Check service status
log_info "📊 Service Status Check:"

check_service() {
    local service_name=$1
    local port=$2
    local pid_file="/tmp/${service_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            if command -v wget >/dev/null 2>&1; then
                if wget --spider --quiet --timeout=3 "http://localhost:$port" 2>/dev/null; then
                    log_success "$service_name (PID: $pid) - ✅ Running and responding on port $port"
                else
                    log_warning "$service_name (PID: $pid) - ⚠️ Running but not responding on port $port"
                fi
            else
                log_success "$service_name (PID: $pid) - ✅ Running"
            fi
            return 0
        else
            log_error "$service_name - ❌ Process not running"
            return 1
        fi
    else
        log_error "$service_name - ❌ PID file not found"
        return 1
    fi
}

check_service "console" "8082"
check_service "gate" "80"

# Function to handle shutdown
cleanup() {
    log_info "🛑 Shutting down services..."
    
    for service in console gate; do
        pid_file="/tmp/${service}.pid"
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                log_info "Stopping $service (PID: $pid)..."
                kill -TERM "$pid" 2>/dev/null || true
                # Wait for graceful shutdown
                for i in {1..10}; do
                    if ! kill -0 "$pid" 2>/dev/null; then
                        log_success "$service stopped gracefully"
                        break
                    fi
                    sleep 1
                done
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    log_warning "Force stopping $service..."
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
            rm -f "$pid_file"
        fi
    done
    
    log_info "🎯 All services stopped"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

log_success "🎉 All services started successfully!"
log_info "📋 Service endpoints:"
log_info "  🌐 Web Interface: http://localhost:80"
log_info "  🔌 API Server: http://localhost:8082"
log_info "  📊 Health Check: http://localhost:8082/api/v1/health"

log_info "Press Ctrl+C to stop all services"

# Keep the script running and monitor services
while true; do
    sleep 30
    
    # Check if services are still running
    for service in console gate; do
        pid_file="/tmp/${service}.pid"
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if ! kill -0 "$pid" 2>/dev/null; then
                log_error "$service has stopped unexpectedly!"
                log_info "Attempting to restart $service..."
                
                # Restart the service
                case "$service" in
                    "console")
                        start_service "console" "/usr/local/bin/console" || log_error "Failed to restart console"
                        ;;
                    "gate")
                        start_service "gate" "/usr/local/bin/gate" || log_error "Failed to restart gate"
                        ;;
                esac
            fi
        fi
    done
done