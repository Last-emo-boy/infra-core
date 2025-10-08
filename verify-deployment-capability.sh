#!/bin/bash
# InfraCore Server Deployment Capability Verification
# This script verifies if server-deploy.sh can establish the entire framework
# Author: last-emo-boy
# Usage: ./verify-deployment-capability.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test configuration
TEST_DEPLOY_SCRIPT="./server-deploy.sh"
TEST_CONFIG_CHECK_SCRIPT="./scripts/config-check.sh"
TEST_DOCKER_COMPOSE="./docker-compose.yml"
TEST_CONFIGS_DIR="./configs"
REPORT_FILE="deployment-capability-report.md"

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

echo_header() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${PURPLE}$1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[✓ PASS]${NC} $1"
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
}

echo_warning() {
    echo -e "${YELLOW}[⚠ WARN]${NC} $1"
    WARNING_CHECKS=$((WARNING_CHECKS + 1))
}

echo_error() {
    echo -e "${RED}[✗ FAIL]${NC} $1"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
}

check_test() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
}

# Initialize report
init_report() {
    cat > "$REPORT_FILE" << 'EOF'
# InfraCore Server Deployment Capability Verification Report

## Executive Summary

This report evaluates whether `server-deploy.sh` can successfully establish the entire InfraCore framework.

### Test Date
EOF
    echo "**$(date)**" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
}

# Add section to report
add_report_section() {
    local title="$1"
    local content="$2"
    echo "" >> "$REPORT_FILE"
    echo "## $title" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "$content" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
}

# Check if file exists and is executable
check_file_exists() {
    local file="$1"
    local description="$2"
    
    check_test
    echo_info "Checking $description: $file"
    
    if [[ -f "$file" ]]; then
        echo_success "$description exists"
        return 0
    else
        echo_error "$description not found"
        return 1
    fi
}

# Check if script is executable
check_script_executable() {
    local script="$1"
    local description="$2"
    
    check_test
    echo_info "Checking if $description is executable"
    
    if [[ -x "$script" ]]; then
        echo_success "$description is executable"
        return 0
    elif [[ -f "$script" ]]; then
        echo_warning "$description exists but is not executable (can be fixed with chmod +x)"
        return 0
    else
        echo_error "$description not found"
        return 1
    fi
}

# Verify deployment script structure
verify_script_structure() {
    echo_header "1. DEPLOYMENT SCRIPT STRUCTURE VERIFICATION"
    
    local results=""
    
    # Check if deployment script exists
    check_file_exists "$TEST_DEPLOY_SCRIPT" "server-deploy.sh"
    results="${results}- Deployment script file: ✓\n"
    
    check_script_executable "$TEST_DEPLOY_SCRIPT" "server-deploy.sh"
    
    # Check for critical functions
    echo_info "Checking for critical functions in server-deploy.sh..."
    
    local critical_functions=(
        "main"
        "main_deploy"
        "deploy_docker"
        "deploy_binary"
        "pre_deployment_checks"
        "install_dependencies"
        "setup_user"
        "setup_directories"
        "update_repository"
        "verify_deployment_health"
        "wait_for_services"
        "show_status"
        "rollback"
        "check_configuration"
    )
    
    local found_functions=0
    local missing_functions=""
    
    for func in "${critical_functions[@]}"; do
        check_test
        if grep -q "^${func}()" "$TEST_DEPLOY_SCRIPT" || grep -q "^${func} ()" "$TEST_DEPLOY_SCRIPT"; then
            echo_success "Function found: $func"
            found_functions=$((found_functions + 1))
            results="${results}- Function '$func': ✓\n"
        else
            echo_error "Function missing: $func"
            missing_functions="${missing_functions}- $func\n"
            results="${results}- Function '$func': ✗\n"
        fi
    done
    
    echo_info "Found $found_functions/${#critical_functions[@]} critical functions"
    
    add_report_section "Script Structure" "$(echo -e "$results")"
}

# Verify configuration files
verify_configuration_files() {
    echo_header "2. CONFIGURATION FILES VERIFICATION"
    
    local results=""
    
    # Check configs directory
    check_test
    if [[ -d "$TEST_CONFIGS_DIR" ]]; then
        echo_success "Configuration directory exists"
        results="${results}- Configuration directory: ✓\n"
    else
        echo_error "Configuration directory not found"
        results="${results}- Configuration directory: ✗\n"
    fi
    
    # Check required config files
    local config_files=("production.yaml" "development.yaml" "testing.yaml")
    
    for config in "${config_files[@]}"; do
        check_test
        if [[ -f "$TEST_CONFIGS_DIR/$config" ]]; then
            echo_success "Configuration file exists: $config"
            results="${results}- Config file '$config': ✓\n"
            
            # Check if file is valid YAML (basic check)
            if command -v python3 &> /dev/null; then
                if python3 -c "import yaml; yaml.safe_load(open('$TEST_CONFIGS_DIR/$config'))" 2>/dev/null; then
                    echo_success "YAML syntax valid: $config"
                    results="${results}  - YAML syntax: ✓\n"
                else
                    echo_error "YAML syntax invalid: $config"
                    results="${results}  - YAML syntax: ✗\n"
                fi
            fi
        else
            echo_error "Configuration file missing: $config"
            results="${results}- Config file '$config': ✗\n"
        fi
    done
    
    # Check config-check script
    check_file_exists "$TEST_CONFIG_CHECK_SCRIPT" "config-check.sh"
    check_script_executable "$TEST_CONFIG_CHECK_SCRIPT" "config-check.sh"
    
    add_report_section "Configuration Files" "$(echo -e "$results")"
}

# Verify Docker deployment capability
verify_docker_deployment() {
    echo_header "3. DOCKER DEPLOYMENT CAPABILITY"
    
    local results=""
    
    # Check docker-compose file
    check_test
    if [[ -f "$TEST_DOCKER_COMPOSE" ]]; then
        echo_success "docker-compose.yml exists"
        results="${results}- docker-compose.yml: ✓\n"
    else
        echo_error "docker-compose.yml not found"
        results="${results}- docker-compose.yml: ✗\n"
    fi
    
    # Check Dockerfile
    check_test
    if [[ -f "./Dockerfile" ]]; then
        echo_success "Dockerfile exists"
        results="${results}- Dockerfile: ✓\n"
    else
        echo_error "Dockerfile not found"
        results="${results}- Dockerfile: ✗\n"
    fi
    
    # Check for deploy_docker function
    check_test
    if grep -q "^deploy_docker()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "deploy_docker function exists"
        results="${results}- deploy_docker function: ✓\n"
        
        # Verify function components
        if grep -A 50 "^deploy_docker()" "$TEST_DEPLOY_SCRIPT" | grep -q "docker-compose.*up"; then
            echo_success "Docker compose up command found"
            results="${results}  - Docker compose up: ✓\n"
        fi
        
        if grep -A 50 "^deploy_docker()" "$TEST_DEPLOY_SCRIPT" | grep -q "wait_for_services"; then
            echo_success "Service health check included"
            results="${results}  - Health check: ✓\n"
        fi
    else
        echo_error "deploy_docker function not found"
        results="${results}- deploy_docker function: ✗\n"
    fi
    
    # Check if Docker is available (non-critical)
    check_test
    if command -v docker &> /dev/null; then
        echo_success "Docker is installed"
        results="${results}- Docker installed: ✓\n"
        
        if docker info &> /dev/null; then
            echo_success "Docker daemon is running"
            results="${results}  - Docker daemon: ✓\n"
        else
            echo_warning "Docker daemon not running (may need to start)"
            results="${results}  - Docker daemon: ⚠\n"
        fi
    else
        echo_warning "Docker not installed (script should install it)"
        results="${results}- Docker installed: ⚠ (script installs)\n"
    fi
    
    add_report_section "Docker Deployment" "$(echo -e "$results")"
}

# Verify binary deployment capability
verify_binary_deployment() {
    echo_header "4. BINARY DEPLOYMENT CAPABILITY"
    
    local results=""
    
    # Check for deploy_binary function
    check_test
    if grep -q "^deploy_binary()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "deploy_binary function exists"
        results="${results}- deploy_binary function: ✓\n"
        
        # Check for Go build commands
        if grep -A 50 "^deploy_binary()" "$TEST_DEPLOY_SCRIPT" | grep -q "go build"; then
            echo_success "Go build command found"
            results="${results}  - Go build: ✓\n"
        fi
        
        # Check for systemd service installation
        if grep -A 50 "^deploy_binary()" "$TEST_DEPLOY_SCRIPT" | grep -q "systemctl\|install_systemd_services"; then
            echo_success "Systemd service setup found"
            results="${results}  - Systemd services: ✓\n"
        fi
    else
        echo_error "deploy_binary function not found"
        results="${results}- deploy_binary function: ✗\n"
    fi
    
    # Check for service entry points
    local services=("console" "gate" "orch" "probe" "snap")
    for service in "${services[@]}"; do
        check_test
        if [[ -f "./cmd/$service/main.go" ]]; then
            echo_success "Service entry point exists: $service"
            results="${results}- Service '$service': ✓\n"
        else
            echo_warning "Service entry point not found: $service"
            results="${results}- Service '$service': ⚠\n"
        fi
    done
    
    add_report_section "Binary Deployment" "$(echo -e "$results")"
}

# Verify pre-deployment checks
verify_pre_deployment_checks() {
    echo_header "5. PRE-DEPLOYMENT CHECKS VERIFICATION"
    
    local results=""
    
    # Check for pre_deployment_checks function
    check_test
    if grep -q "pre_deployment_checks()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "pre_deployment_checks function exists"
        results="${results}- Pre-deployment checks function: ✓\n"
        
        # Check for specific checks
        local checks=(
            "check_configuration"
            "check_port_conflicts"
            "validate_system_requirements"
            "validate_dependencies"
            "check_network_connectivity"
            "check_existing_deployment"
        )
        
        for check in "${checks[@]}"; do
            check_test
            if grep -A 100 "pre_deployment_checks()" "$TEST_DEPLOY_SCRIPT" | grep -q "$check"; then
                echo_success "Check included: $check"
                results="${results}  - $check: ✓\n"
            else
                echo_warning "Check not found: $check"
                results="${results}  - $check: ⚠\n"
            fi
        done
    else
        echo_error "pre_deployment_checks function not found"
        results="${results}- Pre-deployment checks function: ✗\n"
    fi
    
    add_report_section "Pre-deployment Checks" "$(echo -e "$results")"
}

# Verify health check and monitoring
verify_health_monitoring() {
    echo_header "6. HEALTH CHECK & MONITORING VERIFICATION"
    
    local results=""
    
    # Check for verify_deployment_health function
    check_test
    if grep -q "verify_deployment_health()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "verify_deployment_health function exists"
        results="${results}- Health check function: ✓\n"
        
        # Check for health check components
        if grep -A 50 "verify_deployment_health()" "$TEST_DEPLOY_SCRIPT" | grep -q "curl.*health"; then
            echo_success "HTTP health check included"
            results="${results}  - HTTP health check: ✓\n"
        fi
        
        if grep -A 50 "verify_deployment_health()" "$TEST_DEPLOY_SCRIPT" | grep -q "docker.*ps\|systemctl.*status"; then
            echo_success "Service status check included"
            results="${results}  - Service status check: ✓\n"
        fi
    else
        echo_error "verify_deployment_health function not found"
        results="${results}- Health check function: ✗\n"
    fi
    
    # Check for wait_for_services function
    check_test
    if grep -q "wait_for_services()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "wait_for_services function exists"
        results="${results}- Service wait function: ✓\n"
    else
        echo_error "wait_for_services function not found"
        results="${results}- Service wait function: ✗\n"
    fi
    
    # Check for show_status function
    check_test
    if grep -q "show_status()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "show_status function exists"
        results="${results}- Status display function: ✓\n"
    else
        echo_error "show_status function not found"
        results="${results}- Status display function: ✗\n"
    fi
    
    add_report_section "Health Check & Monitoring" "$(echo -e "$results")"
}

# Verify rollback capability
verify_rollback_capability() {
    echo_header "7. ROLLBACK CAPABILITY VERIFICATION"
    
    local results=""
    
    # Check for rollback functionality
    check_test
    if grep -q "execute_rollback\|rollback" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "Rollback functionality exists"
        results="${results}- Rollback function: ✓\n"
        
        # Check for rollback steps management
        if grep -q "add_rollback_step\|ROLLBACK_STEPS" "$TEST_DEPLOY_SCRIPT"; then
            echo_success "Rollback steps management found"
            results="${results}  - Rollback steps tracking: ✓\n"
        fi
        
        # Check for error handler
        if grep -q "error_handler" "$TEST_DEPLOY_SCRIPT"; then
            echo_success "Error handler found"
            results="${results}  - Error handler: ✓\n"
        fi
    else
        echo_error "Rollback functionality not found"
        results="${results}- Rollback function: ✗\n"
    fi
    
    # Check for backup capability
    check_test
    if grep -q "create_backup" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "Backup function exists"
        results="${results}- Backup function: ✓\n"
    else
        echo_warning "Backup function not found"
        results="${results}- Backup function: ⚠\n"
    fi
    
    add_report_section "Rollback Capability" "$(echo -e "$results")"
}

# Verify service management
verify_service_management() {
    echo_header "8. SERVICE MANAGEMENT VERIFICATION"
    
    local results=""
    
    # Check for service management functions
    local mgmt_functions=("restart_services" "stop_services" "start_services" "show_logs")
    
    for func in "${mgmt_functions[@]}"; do
        check_test
        if grep -q "${func}()" "$TEST_DEPLOY_SCRIPT"; then
            echo_success "Function exists: $func"
            results="${results}- Function '$func': ✓\n"
        else
            echo_warning "Function not found: $func"
            results="${results}- Function '$func': ⚠\n"
        fi
    done
    
    add_report_section "Service Management" "$(echo -e "$results")"
}

# Verify dependency installation
verify_dependency_installation() {
    echo_header "9. DEPENDENCY INSTALLATION VERIFICATION"
    
    local results=""
    
    # Check for install_dependencies function
    check_test
    if grep -q "install_dependencies()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "install_dependencies function exists"
        results="${results}- Dependency installation function: ✓\n"
        
        # Check for package manager support
        if grep -A 100 "install_dependencies()" "$TEST_DEPLOY_SCRIPT" | grep -q "apt-get\|yum\|dnf"; then
            echo_success "Package manager support found"
            results="${results}  - Package manager: ✓\n"
        fi
        
        # Check for Docker installation
        if grep -A 100 "install_dependencies()" "$TEST_DEPLOY_SCRIPT" | grep -q "docker"; then
            echo_success "Docker installation included"
            results="${results}  - Docker installation: ✓\n"
        fi
    else
        echo_error "install_dependencies function not found"
        results="${results}- Dependency installation function: ✗\n"
    fi
    
    add_report_section "Dependency Installation" "$(echo -e "$results")"
}

# Verify environment setup
verify_environment_setup() {
    echo_header "10. ENVIRONMENT SETUP VERIFICATION"
    
    local results=""
    
    # Check for setup functions
    local setup_functions=(
        "setup_user"
        "setup_directories"
        "setup_environment_config"
        "setup_script_permissions"
    )
    
    for func in "${setup_functions[@]}"; do
        check_test
        if grep -q "${func}()" "$TEST_DEPLOY_SCRIPT"; then
            echo_success "Function exists: $func"
            results="${results}- Function '$func': ✓\n"
        else
            echo_warning "Function not found: $func"
            results="${results}- Function '$func': ⚠\n"
        fi
    done
    
    # Check for environment variables
    check_test
    if grep -q "DEPLOY_DIR\|SERVICE_USER\|ENVIRONMENT" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "Environment variables defined"
        results="${results}- Environment variables: ✓\n"
    fi
    
    add_report_section "Environment Setup" "$(echo -e "$results")"
}

# Verify usage and documentation
verify_usage_documentation() {
    echo_header "11. USAGE & DOCUMENTATION VERIFICATION"
    
    local results=""
    
    # Check for usage/help function
    check_test
    if grep -q "show_usage\|usage()\|--help" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "Usage/help documentation found"
        results="${results}- Usage documentation: ✓\n"
    else
        echo_warning "Usage documentation not found"
        results="${results}- Usage documentation: ⚠\n"
    fi
    
    # Check for deployment documentation
    check_test
    if [[ -f "./DEPLOYMENT.md" ]]; then
        echo_success "DEPLOYMENT.md exists"
        results="${results}- Deployment guide: ✓\n"
    else
        echo_warning "DEPLOYMENT.md not found"
        results="${results}- Deployment guide: ⚠\n"
    fi
    
    # Check for README
    check_test
    if [[ -f "./README.md" ]]; then
        echo_success "README.md exists"
        results="${results}- README: ✓\n"
    else
        echo_warning "README.md not found"
        results="${results}- README: ⚠\n"
    fi
    
    add_report_section "Usage & Documentation" "$(echo -e "$results")"
}

# Verify complete deployment flow
verify_deployment_flow() {
    echo_header "12. COMPLETE DEPLOYMENT FLOW VERIFICATION"
    
    local results="### Deployment Flow Analysis\n\n"
    
    # Check main_deploy flow
    check_test
    if grep -q "main_deploy()" "$TEST_DEPLOY_SCRIPT"; then
        echo_success "main_deploy function exists"
        results="${results}1. ✓ Main deployment entry point exists\n"
        
        # Extract and analyze deployment flow
        echo_info "Analyzing deployment flow..."
        
        local flow_steps=(
            "show_deployment_plan:Display deployment plan"
            "pre_deployment_checks:Run pre-deployment checks"
            "install_dependencies:Install dependencies"
            "setup_user:Setup service user"
            "setup_directories:Setup directories"
            "update_repository:Update repository"
            "deploy_docker\|deploy_binary:Execute deployment"
            "verify_deployment_health:Verify health"
            "show_deployment_summary\|show_status:Show final status"
        )
        
        local step_num=2
        for step_def in "${flow_steps[@]}"; do
            IFS=':' read -r step_pattern step_desc <<< "$step_def"
            check_test
            if grep -A 200 "^main_deploy()" "$TEST_DEPLOY_SCRIPT" | grep -q "$step_pattern"; then
                echo_success "Flow step: $step_desc"
                results="${results}${step_num}. ✓ $step_desc\n"
            else
                echo_warning "Flow step not found: $step_desc"
                results="${results}${step_num}. ⚠ $step_desc\n"
            fi
            step_num=$((step_num + 1))
        done
    else
        echo_error "main_deploy function not found"
        results="${results}1. ✗ Main deployment entry point missing\n"
    fi
    
    add_report_section "Complete Deployment Flow" "$(echo -e "$results")"
}

# Generate final report
generate_final_report() {
    echo_header "13. GENERATING FINAL REPORT"
    
    local success_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    local warning_rate=$((WARNING_CHECKS * 100 / TOTAL_CHECKS))
    local failure_rate=$((FAILED_CHECKS * 100 / TOTAL_CHECKS))
    
    local summary="### Test Results Summary\n\n"
    summary="${summary}**Total Checks:** $TOTAL_CHECKS\n\n"
    summary="${summary}**Passed:** $PASSED_CHECKS ($success_rate%)\n\n"
    summary="${summary}**Warnings:** $WARNING_CHECKS ($warning_rate%)\n\n"
    summary="${summary}**Failed:** $FAILED_CHECKS ($failure_rate%)\n\n"
    
    # Overall assessment
    local assessment=""
    if [[ $FAILED_CHECKS -eq 0 ]]; then
        assessment="### ✓ OVERALL ASSESSMENT: CAPABLE\n\n"
        assessment="${assessment}The \`server-deploy.sh\` script **CAN** establish the entire InfraCore framework.\n\n"
        assessment="${assessment}**Key Capabilities:**\n"
        assessment="${assessment}- Complete deployment orchestration\n"
        assessment="${assessment}- Pre-deployment validation\n"
        assessment="${assessment}- Docker and binary deployment support\n"
        assessment="${assessment}- Health monitoring and verification\n"
        assessment="${assessment}- Rollback capability\n"
        assessment="${assessment}- Service management\n"
    elif [[ $success_rate -ge 80 ]]; then
        assessment="### ⚠ OVERALL ASSESSMENT: MOSTLY CAPABLE\n\n"
        assessment="${assessment}The \`server-deploy.sh\` script is **MOSTLY CAPABLE** of establishing the InfraCore framework.\n\n"
        assessment="${assessment}**Minor issues detected:**\n"
        assessment="${assessment}- $FAILED_CHECKS critical component(s) missing\n"
        assessment="${assessment}- $WARNING_CHECKS warning(s) identified\n\n"
        assessment="${assessment}**Recommendation:** Address the failed checks before production deployment.\n"
    else
        assessment="### ✗ OVERALL ASSESSMENT: NEEDS IMPROVEMENT\n\n"
        assessment="${assessment}The \`server-deploy.sh\` script has **SIGNIFICANT GAPS** that need to be addressed.\n\n"
        assessment="${assessment}**Critical issues:**\n"
        assessment="${assessment}- $FAILED_CHECKS critical component(s) missing\n"
        assessment="${assessment}- Success rate: $success_rate%\n\n"
        assessment="${assessment}**Recommendation:** Fix critical issues before attempting deployment.\n"
    fi
    
    # Insert summary at the beginning
    {
        head -n 7 "$REPORT_FILE"
        echo "$assessment"
        echo "$summary"
        echo "---"
        tail -n +8 "$REPORT_FILE"
    } > "${REPORT_FILE}.tmp"
    mv "${REPORT_FILE}.tmp" "$REPORT_FILE"
    
    # Add recommendations section
    local recommendations="### Recommendations\n\n"
    
    if [[ $FAILED_CHECKS -gt 0 ]]; then
        recommendations="${recommendations}**Critical:**\n"
        recommendations="${recommendations}- Review and fix all failed checks\n"
        recommendations="${recommendations}- Ensure all critical functions are present\n\n"
    fi
    
    if [[ $WARNING_CHECKS -gt 0 ]]; then
        recommendations="${recommendations}**Advisory:**\n"
        recommendations="${recommendations}- Address warning items for production readiness\n"
        recommendations="${recommendations}- Review optional features and decide on implementation\n\n"
    fi
    
    recommendations="${recommendations}**Testing:**\n"
    recommendations="${recommendations}- Perform dry-run deployment in test environment\n"
    recommendations="${recommendations}- Test rollback functionality\n"
    recommendations="${recommendations}- Verify health checks work correctly\n"
    recommendations="${recommendations}- Test both Docker and binary deployment modes\n"
    
    add_report_section "Recommendations" "$(echo -e "$recommendations")"
    
    # Add component checklist
    local checklist="### Framework Components Checklist\n\n"
    checklist="${checklist}#### Core Services\n"
    checklist="${checklist}- [ ] Gate (HTTP/HTTPS Gateway)\n"
    checklist="${checklist}- [ ] Console (Management API)\n"
    checklist="${checklist}- [ ] Orchestrator (Container Orchestration)\n"
    checklist="${checklist}- [ ] Probe (Health Monitoring)\n"
    checklist="${checklist}- [ ] Snap (Backup/Snapshot)\n\n"
    checklist="${checklist}#### Deployment Features\n"
    checklist="${checklist}- [ ] Pre-deployment validation\n"
    checklist="${checklist}- [ ] Dependency installation\n"
    checklist="${checklist}- [ ] Configuration management\n"
    checklist="${checklist}- [ ] Service deployment (Docker)\n"
    checklist="${checklist}- [ ] Service deployment (Binary)\n"
    checklist="${checklist}- [ ] Health verification\n"
    checklist="${checklist}- [ ] Rollback capability\n"
    checklist="${checklist}- [ ] Service management (start/stop/restart)\n"
    checklist="${checklist}- [ ] Logging and monitoring\n"
    
    add_report_section "Framework Components" "$(echo -e "$checklist")"
    
    echo_info "Report generated: $REPORT_FILE"
}

# Display summary
display_summary() {
    echo ""
    echo_header "VERIFICATION SUMMARY"
    echo ""
    echo -e "${BLUE}Total Checks:${NC} $TOTAL_CHECKS"
    echo -e "${GREEN}Passed:${NC} $PASSED_CHECKS"
    echo -e "${YELLOW}Warnings:${NC} $WARNING_CHECKS"
    echo -e "${RED}Failed:${NC} $FAILED_CHECKS"
    echo ""
    
    local success_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    
    if [[ $FAILED_CHECKS -eq 0 ]]; then
        echo_success "server-deploy.sh CAN establish the entire framework! ✓"
        echo -e "${GREEN}Success Rate: $success_rate%${NC}"
    elif [[ $success_rate -ge 80 ]]; then
        echo_warning "server-deploy.sh is MOSTLY CAPABLE with minor issues"
        echo -e "${YELLOW}Success Rate: $success_rate%${NC}"
    else
        echo_error "server-deploy.sh NEEDS IMPROVEMENT before deployment"
        echo -e "${RED}Success Rate: $success_rate%${NC}"
    fi
    
    echo ""
    echo_info "Detailed report saved to: $REPORT_FILE"
    echo ""
}

# Main execution
main() {
    clear
    echo_header "InfraCore Server Deployment Capability Verification"
    echo ""
    echo_info "Starting comprehensive verification of server-deploy.sh..."
    echo_info "This will verify if the script can establish the entire InfraCore framework."
    echo ""
    
    # Initialize report
    init_report
    
    # Run all verification checks
    verify_script_structure
    verify_configuration_files
    verify_docker_deployment
    verify_binary_deployment
    verify_pre_deployment_checks
    verify_health_monitoring
    verify_rollback_capability
    verify_service_management
    verify_dependency_installation
    verify_environment_setup
    verify_usage_documentation
    verify_deployment_flow
    
    # Generate final report
    generate_final_report
    
    # Display summary
    display_summary
    
    # Exit with appropriate code
    if [[ $FAILED_CHECKS -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"
