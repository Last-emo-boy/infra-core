# InfraCore 🚀

A comprehensive infrastructure management platform with web-based console, reverse proxy, and multi-environment support.

**Author**: last-emo-boy  
**Version**: 1.0.0

## ✨ Features

- **🌐 Reverse Proxy Gateway** - HTTP/HTTPS proxy with ACME support
- **📊 Web Console** - React-based management interface
- **🔐 JWT Authentication** - Secure API access with role-based permissions
- **🐳 Container Management** - Service lifecycle management
- **🔧 Multi-Environment** - Development, testing, and production configurations
- **🚀 Easy Deployment** - Docker and traditional deployment options
- **📈 System Monitoring** - Real-time metrics and health checks

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Console   │    │  Reverse Proxy  │    │    Services     │
│   (React UI)    │◄──►│    (Gate)       │◄──►│   Management    │
│   Port: 5173    │    │   Port: 80/443  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Console API    │
                    │   (Backend)     │
                    │   Port: 8082    │
                    └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- **Go 1.24.5+**
- **Node.js 20+** 
- **Docker & Docker Compose** (for containerized deployment)
- **Make** (optional, for build automation)

### 1. Clone Repository

```bash
git clone https://github.com/last-emo-boy/infra-core.git
cd infra-core
```

### 2. Development Setup

```bash
# Install dependencies
make install-deps

# Start development environment
make dev

# Or manually:
# Backend
INFRA_CORE_ENV=development go run cmd/console/main.go

# Frontend (in another terminal)
cd ui && npm run dev
```

### 3. Production Deployment

#### Docker Deployment (Recommended)

```bash
# Linux/macOS
./deploy.sh production

# Windows
.\deploy.ps1 -Environment production

# Or with Make
make prod
```

#### Manual Build

```bash
# Build all components
make build-all

# Run with production config
INFRA_CORE_ENV=production ./bin/console
```

## 📁 Project Structure

```
infra-core/
├── cmd/                    # Application entry points
│   ├── console/           # Console API server
│   ├── gate/             # Reverse proxy gateway
│   └── api-test/         # API testing utility
├── pkg/                   # Shared libraries
│   ├── api/              # API handlers and middleware
│   ├── auth/             # Authentication service
│   ├── config/           # Configuration management
│   └── database/         # Database layer
├── ui/                    # React frontend
│   ├── src/
│   │   ├── components/   # React components
│   │   ├── pages/        # Page components
│   │   ├── contexts/     # React contexts
│   │   └── types/        # TypeScript types
│   └── dist/             # Built frontend assets
├── configs/               # Environment configurations
│   ├── development.yaml
│   ├── production.yaml
│   └── testing.yaml
├── docker-compose.yml     # Production Docker setup
├── docker-compose.dev.yml # Development Docker setup
├── Dockerfile             # Production Docker image
├── Dockerfile.dev         # Development Docker image
├── deploy.sh             # Linux deployment script
├── deploy.ps1            # Windows deployment script
└── Makefile              # Build automation
```

## ⚙️ Configuration

The application supports multiple environments with dedicated configuration files:

### Environment Files

- `configs/development.yaml` - Development settings
- `configs/production.yaml` - Production settings  
- `configs/testing.yaml` - Testing settings

### Environment Variables

```bash
# Core settings
INFRA_CORE_ENV=development|production|testing
INFRA_CORE_JWT_SECRET=your-secret-key
INFRA_CORE_CONSOLE_PORT=8082

# Database
INFRA_CORE_DB_PATH=/path/to/database.db

# ACME/SSL
INFRA_CORE_ACME_EMAIL=admin@example.com
INFRA_CORE_ACME_ENABLED=true
```

## 🌐 API Endpoints

### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration

### User Management
- `GET /api/v1/users/profile` - Get user profile
- `GET /api/v1/users` - List users (admin)
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user (admin)

### Service Management
- `GET /api/v1/services` - List services
- `POST /api/v1/services` - Create service
- `GET /api/v1/services/:id` - Get service details
- `PUT /api/v1/services/:id` - Update service
- `DELETE /api/v1/services/:id` - Delete service
- `POST /api/v1/services/:id/start` - Start service
- `POST /api/v1/services/:id/stop` - Stop service
- `GET /api/v1/services/:id/logs` - Get service logs

### System Information
- `GET /api/v1/system/info` - System information
- `GET /api/v1/system/metrics` - System metrics
- `GET /api/v1/system/dashboard` - Dashboard data
- `GET /api/v1/health` - Health check

## 🔧 Development

### Build Commands

```bash
# Show all available commands
make help

# Build backend only
make build

# Build frontend only
make build-ui

# Build everything
make build-all

# Run tests
make test

# Test API endpoints
make test-api

# Clean build artifacts
make clean
```

### Development Workflow

```bash
# Start development environment
make dev

# View logs
make logs

# Check service status
make status

# Restart services
make restart

# Stop services
make stop
```

### Frontend Development

```bash
cd ui

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Type checking
npm run type-check
```

## 🚀 Deployment

### Docker Deployment

1. **Linux/macOS**:
   ```bash
   ./deploy.sh production
   ```

2. **Windows**:
   ```powershell
   .\deploy.ps1 -Environment production
   ```

### Manual Deployment

1. **Build the application**:
   ```bash
   make build-all
   ```

2. **Set up configuration**:
   ```bash
   export INFRA_CORE_ENV=production
   export INFRA_CORE_JWT_SECRET=$(openssl rand -hex 32)
   ```

3. **Run the application**:
   ```bash
   ./bin/console
   ```

### Linux Server Setup

```bash
# Create system user
sudo useradd -r -s /bin/false infracore

# Create directories
sudo mkdir -p /var/lib/infra-core /var/log/infra-core /etc/infra-core
sudo chown infracore:infracore /var/lib/infra-core /var/log/infra-core

# Copy configuration
sudo cp configs/production.yaml /etc/infra-core/

# Install systemd service
sudo cp scripts/infracore.service /etc/systemd/system/
sudo systemctl enable infracore
sudo systemctl start infracore
```

## 🔒 Security

### Authentication

- JWT-based authentication with configurable expiration
- Role-based access control (admin/user roles)
- Secure password hashing with bcrypt

### HTTPS/TLS

- Automatic HTTPS certificate management via ACME
- Configurable TLS minimum version
- CORS protection for web console

### Best Practices

- Run as non-root user in production
- Use environment variables for secrets
- Enable audit logging for admin operations
- Regular security updates via Watchtower

## 📊 Monitoring

### Health Checks

- Built-in health check endpoint
- Docker health checks
- Service status monitoring

### Metrics

- System resource utilization
- Service status and logs
- API request monitoring

### Logging

- Structured JSON logging
- Configurable log levels
- Centralized log aggregation support

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Gin](https://gin-gonic.com/) - Web framework
- [React](https://reactjs.org/) - Frontend framework
- [Tailwind CSS](https://tailwindcss.com/) - CSS framework
- [Vite](https://vitejs.dev/) - Build tool
- [SQLite](https://www.sqlite.org/) - Database
- [Docker](https://www.docker.com/) - Containerization

## 📞 Support

For support, please open an issue on GitHub or contact [last-emo-boy](https://github.com/last-emo-boy).

---

Made with ❤️ by [last-emo-boy](https://github.com/last-emo-boy)