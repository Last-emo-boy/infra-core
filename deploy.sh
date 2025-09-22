#!/bin/bash

# InfraCore Deployment Script for Linux
# Author: last-emo-boy
# Usage: ./deploy.sh [development|production]

set -e

ENVIRONMENT=${1:-production}
PROJECT_NAME="infra-core"
DOCKER_IMAGE="${PROJECT_NAME}:latest"
CONFIG_DIR="/etc/infra-core"
DATA_DIR="/var/lib/infra-core"
LOG_DIR="/var/log/infra-core"

echo "🚀 Starting InfraCore deployment (Environment: $ENVIRONMENT)"

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   echo "⚠️  This script should not be run as root. Please run as a regular user with docker permissions."
   exit 1
fi

# Check dependencies
check_dependencies() {
    echo "📋 Checking dependencies..."
    
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo "❌ Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        echo "❌ Docker daemon is not running or you don't have permission to access it."
        echo "Please start Docker or add your user to the docker group:"
        echo "sudo usermod -aG docker \$USER"
        exit 1
    fi
    
    echo "✅ All dependencies are satisfied"
}

# Create system directories
setup_directories() {
    echo "📁 Setting up directories..."
    
    sudo mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
    sudo chown -R $(id -u):$(id -g) "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
    
    echo "✅ Directories created successfully"
}

# Deploy configuration files
deploy_configs() {
    echo "⚙️  Deploying configuration files..."
    
    if [ -d "./configs" ]; then
        sudo cp -r ./configs/* "$CONFIG_DIR/"
        sudo chown -R $(id -u):$(id -g) "$CONFIG_DIR"
        echo "✅ Configuration files deployed"
    else
        echo "⚠️  No configuration directory found"
    fi
}

# Build and deploy
deploy() {
    echo "🔨 Building and deploying..."
    
    case $ENVIRONMENT in
        "development")
            echo "🛠️  Starting development environment..."
            docker-compose -f docker-compose.dev.yml down
            docker-compose -f docker-compose.dev.yml build
            docker-compose -f docker-compose.dev.yml up -d
            ;;
        "production")
            echo "🏭 Starting production environment..."
            
            # Generate JWT secret if not exists
            if [ -z "$JWT_SECRET" ]; then
                export JWT_SECRET=$(openssl rand -hex 32)
                echo "🔑 Generated JWT secret: ${JWT_SECRET:0:8}..."
            fi
            
            docker-compose down
            docker-compose build
            docker-compose up -d
            ;;
        *)
            echo "❌ Unknown environment: $ENVIRONMENT"
            echo "Usage: $0 [development|production]"
            exit 1
            ;;
    esac
}

# Health check
health_check() {
    echo "🏥 Performing health check..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f http://localhost:8082/api/v1/health &> /dev/null; then
            echo "✅ Health check passed"
            return 0
        fi
        
        echo "⏳ Attempt $attempt/$max_attempts - waiting for service..."
        sleep 2
        ((attempt++))
    done
    
    echo "❌ Health check failed after $max_attempts attempts"
    echo "📋 Checking logs..."
    docker-compose logs --tail=20
    return 1
}

# Show status
show_status() {
    echo ""
    echo "📊 Deployment Status:"
    echo "===================="
    docker-compose ps
    
    echo ""
    echo "🌐 Service URLs:"
    echo "==============="
    echo "Console: http://localhost:8082"
    echo "Frontend: http://localhost:5173 (development only)"
    echo "Gate HTTP: http://localhost:80"
    echo "Gate HTTPS: https://localhost:443"
    
    echo ""
    echo "📝 Useful Commands:"
    echo "=================="
    echo "View logs: docker-compose logs -f"
    echo "Stop services: docker-compose down"
    echo "Restart: docker-compose restart"
    echo "Update: ./deploy.sh $ENVIRONMENT"
}

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    docker system prune -f
}

# Main execution
main() {
    echo "📦 InfraCore Deployment Script"
    echo "=============================="
    
    check_dependencies
    setup_directories
    deploy_configs
    deploy
    
    if health_check; then
        show_status
        echo ""
        echo "🎉 Deployment completed successfully!"
        echo "Dashboard will be available at: http://localhost:8082"
    else
        echo "❌ Deployment failed!"
        exit 1
    fi
}

# Trap cleanup on exit
trap cleanup EXIT

# Run main function
main