#!/bin/sh
# InfraCore Services Startup Script
# Compatible with POSIX sh (ash in Alpine)

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

# Function to check if port is available
is_port_available() {
    local port=$1
    ! netstat -tlnp 2>/dev/null | grep -q ":$port "
}

# Function to find available port with fallbacks
find_available_port() {
    local service_name=$1
    local primary_port=$2
    local fallback1=${3:-$((primary_port + 10000))}
    local fallback2=${4:-$((primary_port + 20000))}
    
    if is_port_available "$primary_port"; then
        echo "$primary_port"
        return 0
    fi
    
    log_warning "$service_name: Primary port $primary_port is occupied, trying fallback..."
    
    if is_port_available "$fallback1"; then
        log_info "$service_name: Using fallback port $fallback1"
        echo "$fallback1"
        return 0
    fi
    
    log_warning "$service_name: Fallback port $fallback1 is occupied, trying secondary fallback..."
    
    if is_port_available "$fallback2"; then
        log_info "$service_name: Using secondary fallback port $fallback2" 
        echo "$fallback2"
        return 0
    fi
    
    log_error "$service_name: All configured ports are occupied ($primary_port, $fallback1, $fallback2)"
    return 1
}

# Function to start a service in background with port fallback
start_service() {
    local service_name=$1
    local service_binary=$2
    local service_port=${3:-""}
    local service_args=${4:-""}
    
    log_info "Starting $service_name..."
    
    # If port is specified, handle port fallback
    if [ -n "$service_port" ]; then
        local available_port
        case "$service_name" in
            "orchestrator")
                available_port=$(find_available_port "$service_name" "$service_port" "${ORCHESTRATOR_FALLBACK_PORT:-19090}" "${ORCHESTRATOR_FALLBACK2_PORT:-29090}")
                ;;
            "probe")
                available_port=$(find_available_port "$service_name" "$service_port" "${PROBE_FALLBACK_PORT:-18085}" "${PROBE_FALLBACK2_PORT:-28085}")
                ;;
            "snap")
                available_port=$(find_available_port "$service_name" "$service_port" "${SNAP_FALLBACK_PORT:-18086}" "${SNAP_FALLBACK2_PORT:-28086}")
                ;;
            *)
                available_port=$(find_available_port "$service_name" "$service_port")
                ;;
        esac
        
        if [ -z "$available_port" ]; then
            log_error "$service_name: No available ports, skipping service startup"
            return 1
        fi
        
        # Update service args with available port
        service_args="--port=$available_port $service_args"
        
        # Store the actual port for later reference
        echo "$available_port" > "/tmp/${service_name}.port"
    fi
    
    # Start service in background
    nohup "$service_binary" $service_args > "$INFRA_CORE_LOG_DIR/${service_name}.log" 2>&1 &
    local pid=$!
    
    # Wait a moment and check if it's still running
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        local port_info=""
        if [ -f "/tmp/${service_name}.port" ]; then
            port_info=" on port $(cat "/tmp/${service_name}.port")"
        fi
        log_success "$service_name started with PID $pid$port_info"
        echo "$pid" > "/tmp/${service_name}.pid"
        return 0
    else
        log_error "$service_name failed to start"
        return 1
    fi
}

# Start services
log_info "🎯 Starting individual services..."

# Core Services (always start)

# Start Console API (8082)
log_info "Starting core service: Console API"
if command -v console >/dev/null 2>&1; then
    start_service "console" "console" "" "--config=/app/configs/production.yaml"
elif command -v /usr/local/bin/console >/dev/null 2>&1; then
    start_service "console" "/usr/local/bin/console" "" "--config=/app/configs/production.yaml"
else
    log_error "Console binary not found"
fi

# Start Gate (80/443)
log_info "Starting core service: Gate"
if command -v gate >/dev/null 2>&1; then
    start_service "gate" "gate"
elif command -v /usr/local/bin/gate >/dev/null 2>&1; then
    start_service "gate" "/usr/local/bin/gate"
else
    log_error "Gate binary not found"
fi

# All Services (enabled by default with port fallback)

# Start Orchestrator (9090) - with port fallback
if [ "${ENABLE_ORCHESTRATOR:-true}" = "true" ]; then
    log_info "Starting service: Orchestrator (with port fallback)"
    orch_port="${INFRA_CORE_ORCH_PORT:-9090}"
    if command -v orchestrator >/dev/null 2>&1; then
        start_service "orchestrator" "orchestrator" "$orch_port" "--config=/app/configs/production.yaml"
    elif command -v /usr/local/bin/orchestrator >/dev/null 2>&1; then
        start_service "orchestrator" "/usr/local/bin/orchestrator" "$orch_port" "--config=/app/configs/production.yaml"
    else
        log_warning "Orchestrator enabled but binary not found - will be built in next deployment"
    fi
else
    log_info "Orchestrator service disabled (ENABLE_ORCHESTRATOR=false)"
fi

# Start Probe Monitor (8085) - with port fallback
if [ "${ENABLE_PROBE_MONITOR:-true}" = "true" ]; then
    log_info "Starting service: Probe Monitor (with port fallback)"
    probe_port="${INFRA_CORE_PROBE_PORT:-8085}"
    if command -v probe >/dev/null 2>&1; then
        start_service "probe" "probe" "$probe_port" "--config=/app/configs/production.yaml"
    elif command -v /usr/local/bin/probe >/dev/null 2>&1; then
        start_service "probe" "/usr/local/bin/probe" "$probe_port" "--config=/app/configs/production.yaml"
    else
        log_warning "Probe Monitor enabled but binary not found - will be built in next deployment"
    fi
else
    log_info "Probe Monitor service disabled (ENABLE_PROBE_MONITOR=false)"
fi

# Start Snap Service (8086) - with port fallback
if [ "${ENABLE_SNAP_SERVICE:-true}" = "true" ]; then
    log_info "Starting service: Snap Service (with port fallback)"
    snap_port="${INFRA_CORE_SNAP_PORT:-8086}"
    if command -v snap >/dev/null 2>&1; then
        start_service "snap" "snap" "$snap_port" "--config=/app/configs/production.yaml"
    elif command -v /usr/local/bin/snap >/dev/null 2>&1; then
        start_service "snap" "/usr/local/bin/snap" "$snap_port" "--config=/app/configs/production.yaml"
    else
        log_warning "Snap Service enabled but binary not found - will be built in next deployment"
    fi
else
    log_info "Snap Service disabled (ENABLE_SNAP_SERVICE=false)"
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

# Check core services
check_service "console" "8082"
check_service "gate" "80"

# Check additional services with dynamic ports
for service in orchestrator probe snap; do
    if [ -f "/tmp/${service}.pid" ]; then
        service_port="unknown"
        if [ -f "/tmp/${service}.port" ]; then
            service_port=$(cat "/tmp/${service}.port")
        fi
        check_service "$service" "$service_port"
    fi
done

# Function to handle shutdown
cleanup() {
    log_info "🛑 Shutting down services..."
    
    for service in console gate orchestrator probe snap; do
        pid_file="/tmp/${service}.pid"
        port_file="/tmp/${service}.port"
        
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                local port_info=""
                if [ -f "$port_file" ]; then
                    port_info=" (port: $(cat "$port_file"))"
                fi
                log_info "Stopping $service (PID: $pid)$port_info..."
                kill -TERM "$pid" 2>/dev/null || true
                
                # Wait for graceful shutdown
                i=1
                while [ $i -le 10 ]; do
                    if ! kill -0 "$pid" 2>/dev/null; then
                        log_success "$service stopped gracefully"
                        break
                    fi
                    sleep 1
                    i=$((i + 1))
                done
                
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    log_warning "Force stopping $service..."
                    kill -KILL "$pid" 2>/dev/null || true
                fi
            fi
            rm -f "$pid_file"
            rm -f "$port_file"
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
log_info "  🔌 Console API: http://localhost:8082"
log_info "  📊 Health Check: http://localhost:8082/api/v1/health"

# Display additional service endpoints with dynamic ports
for service in orchestrator probe snap; do
    if [ -f "/tmp/${service}.pid" ] && [ -f "/tmp/${service}.port" ]; then
        service_port=$(cat "/tmp/${service}.port")
        case "$service" in
            "orchestrator")
                log_info "  🎯 Orchestrator: http://localhost:$service_port"
                ;;
            "probe")
                log_info "  🔍 Probe Monitor: http://localhost:$service_port"
                ;;
            "snap")
                log_info "  💾 Snap Service: http://localhost:$service_port"
                ;;
        esac
    fi
done

log_info "Press Ctrl+C to stop all services"

# Keep the script running and monitor services
while true; do
    sleep 30
    
    # Check if services are still running
    for service in console gate orchestrator probe snap; do
        pid_file="/tmp/${service}.pid"
        port_file="/tmp/${service}.port"
        
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if ! kill -0 "$pid" 2>/dev/null; then
                log_error "$service has stopped unexpectedly!"
                log_info "Attempting to restart $service..."
                
                # Restart the service with port information
                service_port=""
                if [ -f "$port_file" ]; then
                    service_port=$(cat "$port_file")
                fi
                
                case "$service" in
                    "console")
                        start_service "console" "/usr/local/bin/console" "" "--config=/app/configs/production.yaml" || log_error "Failed to restart console"
                        ;;
                    "gate")
                        start_service "gate" "/usr/local/bin/gate" || log_error "Failed to restart gate"
                        ;;
                    "orchestrator")
                        start_service "orchestrator" "/usr/local/bin/orchestrator" "$service_port" "--config=/app/configs/production.yaml" || log_error "Failed to restart orchestrator"
                        ;;
                    "probe")
                        start_service "probe" "/usr/local/bin/probe" "$service_port" "--config=/app/configs/production.yaml" || log_error "Failed to restart probe"
                        ;;
                    "snap")
                        start_service "snap" "/usr/local/bin/snap" "$service_port" "--config=/app/configs/production.yaml" || log_error "Failed to restart snap"
                        ;;
                esac
            fi
        fi
    done
done