#!/bin/bash
# InfraCore Service Status and Port Check Script
# Usage: ./check-services.sh

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

echo "🔍 InfraCore Service Status and Port Check"
echo "=========================================="

# Check running services
echo
log_info "📊 Running Services:"

services=(console gate orchestrator probe snap)
for service in "${services[@]}"; do
    pid_file="/tmp/${service}.pid"
    port_file="/tmp/${service}.port"
    
    if [[ -f "$pid_file" ]]; then
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            port_info=""
            if [[ -f "$port_file" ]]; then
                port=$(cat "$port_file")
                port_info=" (port: $port)"
                
                # Test if port is responding
                if curl -s --max-time 3 "http://localhost:$port/health" >/dev/null 2>&1 || \
                   curl -s --max-time 3 "http://localhost:$port" >/dev/null 2>&1; then
                    port_info="$port_info ✅ responding"
                else
                    port_info="$port_info ⚠️ not responding"
                fi
            fi
            log_success "$service running (PID: $pid)$port_info"
        else
            log_error "$service stopped (stale PID file)"
            rm -f "$pid_file" "$port_file"
        fi
    else
        log_warning "$service not running"
    fi
done

# Check port usage
echo
log_info "🔌 Port Usage Summary:"

ports=(80 443 8082 9090 19090 29090 8085 18085 28085 8086 18086 28086)
for port in "${ports[@]}"; do
    if netstat -tlnp 2>/dev/null | grep -q ":$port "; then
        process=$(netstat -tlnp 2>/dev/null | grep ":$port " | awk '{print $7}' | head -1)
        case "$port" in
            80|443) service_name="Gate" ;;
            8082) service_name="Console" ;;
            9090|19090|29090) service_name="Orchestrator" ;;
            8085|18085|28085) service_name="Probe Monitor" ;;
            8086|18086|28086) service_name="Snap Service" ;;
            *) service_name="Unknown" ;;
        esac
        log_info "Port $port: $service_name ($process)"
    fi
done

# Service endpoints
echo
log_info "🌐 Service Endpoints:"
log_info "  Main Interface: http://localhost"
log_info "  Console API: http://localhost:8082"

for service in orchestrator probe snap; do
    port_file="/tmp/${service}.port"
    if [[ -f "$port_file" ]]; then
        port=$(cat "$port_file")
        case "$service" in
            orchestrator) service_name="Orchestrator" ;;
            probe) service_name="Probe Monitor" ;;
            snap) service_name="Snap Service" ;;
        esac
        log_info "  $service_name: http://localhost:$port"
    fi
done

echo
log_info "🪵 Log Inspection Shortcuts:"
if systemctl list-units --type=service 2>/dev/null | grep -q "infra-core-gate"; then
    log_info "  Gate: sudo journalctl -u infra-core-gate -f"
    log_info "  Console: sudo journalctl -u infra-core-console -f"
    log_info "  Orchestrator: sudo journalctl -u infra-core-orchestrator -f"
    log_info "  Probe: sudo journalctl -u infra-core-probe -f"
    log_info "  Snap: sudo journalctl -u infra-core-snap -f"
else
    deploy_root="${DEPLOY_ROOT:-/opt/infra-core/current}"
    log_info "  Docker Compose: (cd ${deploy_root} && docker compose logs gate console orchestrator probe snap -f)"
    log_info "  Individual logs under /var/log/infra-core/{gate,console,orchestrator,probe,snap}.log"
fi

echo
log_info "✅ Service check completed"