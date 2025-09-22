#!/bin/bash
# Full deployment test for InfraCore
# Author: last-emo-boy

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_PORT=18082
TEST_ENV="testing"
HEALTH_CHECK_URL="http://localhost:${TEST_PORT}/api/v1/health"
TEST_USER_EMAIL="test@example.com"
TEST_USER_PASSWORD="test123456"

echo_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

echo_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    echo_info "Cleaning up test environment..."
    
    # Stop Docker services if running
    if docker compose -f docker-compose.dev.yml ps | grep -q "Up"; then
        echo_info "Stopping Docker services..."
        docker compose -f docker-compose.dev.yml down -v
    fi
    
    # Kill any running processes on test port
    if lsof -i :${TEST_PORT} >/dev/null 2>&1; then
        echo_info "Killing processes on port ${TEST_PORT}..."
        lsof -ti:${TEST_PORT} | xargs -r kill -9
    fi
    
    # Clean build artifacts
    echo_info "Cleaning build artifacts..."
    make clean >/dev/null 2>&1 || true
    
    echo_success "Cleanup completed"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

check_prerequisites() {
    echo_info "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        echo_error "Go is not installed"
        exit 1
    fi
    echo_success "Go $(go version | cut -d' ' -f3) is installed"
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        echo_error "Node.js is not installed"
        exit 1
    fi
    echo_success "Node.js $(node --version) is installed"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        echo_error "Docker is not installed"
        exit 1
    fi
    echo_success "Docker $(docker --version | cut -d' ' -f3 | cut -d',' -f1) is installed"
    
    # Check Docker Compose
    if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
        echo_error "Docker Compose is not available"
        exit 1
    fi
    echo_success "Docker Compose is available"
    
    # Check Make
    if ! command -v make &> /dev/null; then
        echo_warning "Make is not installed - some commands may not work"
    else
        echo_success "Make is installed"
    fi
}

test_build() {
    echo_info "Testing build process..."
    
    # Test Go build
    echo_info "Building Go backend..."
    if make build; then
        echo_success "Go backend build successful"
    else
        echo_error "Go backend build failed"
        return 1
    fi
    
    # Test UI build
    echo_info "Building React frontend..."
    if make build-ui; then
        echo_success "React frontend build successful"
    else
        echo_error "React frontend build failed"
        return 1
    fi
    
    echo_success "All builds completed successfully"
}

test_configuration() {
    echo_info "Testing configuration loading..."
    
    # Test with testing environment
    export INFRA_CORE_ENV="testing"
    export INFRA_CORE_CONSOLE_PORT="${TEST_PORT}"
    
    # Create a test config if it doesn't exist
    if [ ! -f "configs/testing.yaml" ]; then
        echo_info "Creating test configuration..."
        cat > configs/testing.yaml << EOF
database:
  path: "test.db"
console:
  port: ${TEST_PORT}
  host: "127.0.0.1"
jwt:
  secret: "test-secret-key-for-testing-only"
  expiration: "24h"
acme:
  enabled: false
logging:
  level: "debug"
  format: "json"
EOF
    fi
    
    echo_success "Configuration test completed"
}

start_test_server() {
    echo_info "Starting test server..."
    
    # Start the server in background
    INFRA_CORE_ENV=testing INFRA_CORE_CONSOLE_PORT=${TEST_PORT} ./bin/console &
    SERVER_PID=$!
    
    # Wait for server to start
    echo_info "Waiting for server to start..."
    for i in {1..30}; do
        if curl -s "${HEALTH_CHECK_URL}" > /dev/null 2>&1; then
            echo_success "Server started successfully"
            return 0
        fi
        sleep 1
    done
    
    echo_error "Server failed to start within 30 seconds"
    kill ${SERVER_PID} 2>/dev/null || true
    return 1
}

test_api_endpoints() {
    echo_info "Testing API endpoints..."
    
    # Test health check
    echo_info "Testing health check endpoint..."
    if curl -s "${HEALTH_CHECK_URL}" | grep -q "healthy"; then
        echo_success "Health check endpoint working"
    else
        echo_error "Health check endpoint failed"
        return 1
    fi
    
    # Test user registration
    echo_info "Testing user registration..."
    REGISTER_RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_USER_EMAIL}\",\"password\":\"${TEST_USER_PASSWORD}\",\"username\":\"testuser\"}" \
        "http://localhost:${TEST_PORT}/api/v1/auth/register")
    
    if echo "${REGISTER_RESPONSE}" | grep -q "success\|token"; then
        echo_success "User registration working"
    else
        echo_warning "User registration may have failed (user might already exist)"
    fi
    
    # Test user login
    echo_info "Testing user login..."
    LOGIN_RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${TEST_USER_EMAIL}\",\"password\":\"${TEST_USER_PASSWORD}\"}" \
        "http://localhost:${TEST_PORT}/api/v1/auth/login")
    
    if echo "${LOGIN_RESPONSE}" | grep -q "token"; then
        echo_success "User login working"
        
        # Extract token for further tests
        TOKEN=$(echo "${LOGIN_RESPONSE}" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
        
        # Test authenticated endpoint
        echo_info "Testing authenticated endpoint..."
        PROFILE_RESPONSE=$(curl -s -H "Authorization: Bearer ${TOKEN}" \
            "http://localhost:${TEST_PORT}/api/v1/users/profile")
        
        if echo "${PROFILE_RESPONSE}" | grep -q "email\|username"; then
            echo_success "Authenticated endpoints working"
        else
            echo_error "Authenticated endpoints failed"
            return 1
        fi
    else
        echo_error "User login failed"
        return 1
    fi
    
    echo_success "API endpoint tests completed"
}

test_docker_deployment() {
    echo_info "Testing Docker deployment..."
    
    # Build and start with Docker Compose
    echo_info "Building Docker images..."
    if docker compose -f docker-compose.dev.yml build; then
        echo_success "Docker images built successfully"
    else
        echo_error "Docker image build failed"
        return 1
    fi
    
    echo_info "Starting Docker services..."
    if docker compose -f docker-compose.dev.yml up -d; then
        echo_success "Docker services started"
    else
        echo_error "Docker services failed to start"
        return 1
    fi
    
    # Wait for services to be ready
    echo_info "Waiting for Docker services to be ready..."
    for i in {1..60}; do
        if curl -s "http://localhost:8082/api/v1/health" > /dev/null 2>&1; then
            echo_success "Docker services are ready"
            break
        fi
        if [ $i -eq 60 ]; then
            echo_error "Docker services failed to become ready within 60 seconds"
            docker compose -f docker-compose.dev.yml logs
            return 1
        fi
        sleep 1
    done
    
    # Test Docker deployment health
    echo_info "Testing Docker deployment health..."
    DOCKER_HEALTH=$(curl -s "http://localhost:8082/api/v1/health")
    if echo "${DOCKER_HEALTH}" | grep -q "healthy"; then
        echo_success "Docker deployment is healthy"
    else
        echo_error "Docker deployment health check failed"
        return 1
    fi
    
    echo_success "Docker deployment test completed"
}

test_frontend_accessibility() {
    echo_info "Testing frontend accessibility..."
    
    # Test if frontend is accessible
    if curl -s "http://localhost:5173" > /dev/null 2>&1; then
        echo_success "Frontend is accessible"
    else
        echo_warning "Frontend may not be running (this is normal for API-only tests)"
    fi
    
    # Test if built assets exist
    if [ -d "ui/dist" ] && [ -f "ui/dist/index.html" ]; then
        echo_success "Frontend build artifacts exist"
    else
        echo_warning "Frontend build artifacts not found"
    fi
}

generate_test_report() {
    echo_info "Generating test report..."
    
    REPORT_FILE="test_report_$(date +%Y%m%d_%H%M%S).txt"
    
    cat > "${REPORT_FILE}" << EOF
InfraCore Deployment Test Report
Generated: $(date)
Test Environment: ${TEST_ENV}
Test Port: ${TEST_PORT}

========================================
Test Results Summary
========================================

Prerequisites Check: ${PREREQ_STATUS:-UNKNOWN}
Build Process: ${BUILD_STATUS:-UNKNOWN}
Configuration: ${CONFIG_STATUS:-UNKNOWN}
API Endpoints: ${API_STATUS:-UNKNOWN}
Docker Deployment: ${DOCKER_STATUS:-UNKNOWN}
Frontend Accessibility: ${FRONTEND_STATUS:-UNKNOWN}

========================================
System Information
========================================

OS: $(uname -a)
Go Version: $(go version)
Node Version: $(node --version)
Docker Version: $(docker --version)

========================================
Docker Services Status
========================================

$(docker compose -f docker-compose.dev.yml ps 2>/dev/null || echo "Docker services not running")

========================================
Recommendations
========================================

EOF

    if [ "${BUILD_STATUS}" = "PASSED" ] && [ "${API_STATUS}" = "PASSED" ]; then
        echo "✅ Core functionality is working correctly" >> "${REPORT_FILE}"
        echo "✅ Ready for production deployment" >> "${REPORT_FILE}"
    else
        echo "❌ Issues detected - review test output before deployment" >> "${REPORT_FILE}"
    fi
    
    if [ "${DOCKER_STATUS}" = "PASSED" ]; then
        echo "✅ Docker deployment is working" >> "${REPORT_FILE}"
        echo "✅ Can use Docker for production deployment" >> "${REPORT_FILE}"
    else
        echo "⚠️  Docker deployment issues - consider manual deployment" >> "${REPORT_FILE}"
    fi
    
    echo_success "Test report generated: ${REPORT_FILE}"
}

main() {
    echo_info "Starting InfraCore Deployment Test"
    echo_info "=================================="
    
    # Initialize status variables
    PREREQ_STATUS="FAILED"
    BUILD_STATUS="FAILED"
    CONFIG_STATUS="FAILED"
    API_STATUS="FAILED"
    DOCKER_STATUS="FAILED"
    FRONTEND_STATUS="FAILED"
    
    # Run tests
    if check_prerequisites; then
        PREREQ_STATUS="PASSED"
    else
        echo_error "Prerequisites check failed"
        exit 1
    fi
    
    if test_configuration; then
        CONFIG_STATUS="PASSED"
    fi
    
    if test_build; then
        BUILD_STATUS="PASSED"
    else
        echo_error "Build test failed"
        exit 1
    fi
    
    # Start server for API tests
    if start_test_server; then
        if test_api_endpoints; then
            API_STATUS="PASSED"
        fi
        
        # Kill the test server
        kill ${SERVER_PID} 2>/dev/null || true
        sleep 2
    fi
    
    # Test Docker deployment
    if test_docker_deployment; then
        DOCKER_STATUS="PASSED"
    fi
    
    test_frontend_accessibility
    FRONTEND_STATUS="CHECKED"
    
    # Generate report
    generate_test_report
    
    echo_info "=================================="
    echo_info "Test Summary:"
    echo "Prerequisites: ${PREREQ_STATUS}"
    echo "Build: ${BUILD_STATUS}"
    echo "Configuration: ${CONFIG_STATUS}"
    echo "API: ${API_STATUS}"
    echo "Docker: ${DOCKER_STATUS}"
    echo "Frontend: ${FRONTEND_STATUS}"
    
    if [ "${BUILD_STATUS}" = "PASSED" ] && [ "${API_STATUS}" = "PASSED" ]; then
        echo_success "🎉 All critical tests passed! Ready for deployment."
        exit 0
    else
        echo_error "❌ Some tests failed. Review the output before deploying."
        exit 1
    fi
}

# Run the main function
main "$@"