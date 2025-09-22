#!/bin/bash
# InfraCore One-Click Deployment Script
# Author: last-emo-boy
# Usage: curl -fsSL https://raw.githubusercontent.com/last-emo-boy/infra-core/main/quick-deploy.sh | sudo bash

set -e

# Configuration
REPO_URL="https://github.com/last-emo-boy/infra-core.git"
DEPLOY_DIR="/opt/infra-core"
SCRIPT_URL="https://raw.githubusercontent.com/last-emo-boy/infra-core/main/server-deploy.sh"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🚀 InfraCore One-Click Deployment${NC}"
echo "=================================="

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}❌ This script must be run as root${NC}"
    echo "Usage: curl -fsSL https://raw.githubusercontent.com/last-emo-boy/infra-core/main/quick-deploy.sh | sudo bash"
    exit 1
fi

# Download the full deployment script
echo -e "${BLUE}📥 Downloading deployment script...${NC}"
curl -fsSL "$SCRIPT_URL" -o /tmp/server-deploy.sh
chmod +x /tmp/server-deploy.sh

# Run the deployment
echo -e "${BLUE}🔄 Starting deployment...${NC}"
/tmp/server-deploy.sh --backup

# Cleanup
rm -f /tmp/server-deploy.sh

echo -e "${GREEN}✅ Deployment completed!${NC}"
echo ""
echo "🌐 Access your application at: http://$(curl -s ifconfig.me):8082"
echo "📋 Use 'sudo /opt/infra-core/current/server-deploy.sh --status' to check status"
echo "📜 Use 'sudo /opt/infra-core/current/server-deploy.sh --logs' to view logs"