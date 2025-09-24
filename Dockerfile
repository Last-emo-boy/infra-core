# Build stage for Go applications
FROM golang:1.24.5-alpine AS go-builder

WORKDIR /app

# Install dependencies for building with retry mechanism and multiple mirrors
RUN set -eux; \
    # Try Alpine mirrors one by one (sh-compatible syntax)
    for mirror in \
        "https://dl-cdn.alpinelinux.org/alpine" \
        "https://mirrors.tuna.tsinghua.edu.cn/alpine" \
        "https://mirrors.ustc.edu.cn/alpine" \
        "https://mirrors.aliyun.com/alpine" \
        "https://mirror.nju.edu.cn/alpine"; do \
        echo "Trying mirror: $mirror"; \
        echo "$mirror/v3.22/main" > /etc/apk/repositories && \
        echo "$mirror/v3.22/community" >> /etc/apk/repositories && \
        # Test the mirror with update
        if apk update --no-cache 2>/dev/null; then \
            echo "Successfully using mirror: $mirror"; \
            # Install packages
            if apk add --no-cache git ca-certificates; then \
                echo "Packages installed successfully with mirror: $mirror"; \
                break; \
            else \
                echo "Package installation failed with mirror: $mirror, trying next..."; \
            fi; \
        else \
            echo "Mirror $mirror failed, trying next..."; \
        fi; \
    done; \
    # Verify git is installed
    git --version

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with retry mechanism and Go proxy mirrors
RUN set -eux; \
    # Try Go module proxies one by one (sh-compatible syntax)
    for proxy in \
        "https://proxy.golang.org,direct" \
        "https://goproxy.cn,direct" \
        "https://mirrors.aliyun.com/goproxy/,direct" \
        "https://goproxy.io,direct"; do \
        echo "Trying Go proxy: $proxy"; \
        export GOPROXY="$proxy"; \
        export GOSUMDB="sum.golang.org"; \
        if go mod download && go mod verify; then \
            echo "Go modules downloaded successfully with proxy: $proxy"; \
            break; \
        else \
            echo "Go proxy $proxy failed, trying next..."; \
            go clean -modcache; \
            sleep 3; \
        fi; \
    done

# Copy source code
COPY . .

# Build applications (CGO disabled for pure Go builds, fully static binaries)
RUN set -eux; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/gate cmd/gate/main.go

RUN set -eux; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/console cmd/console/main.go

# Build stage for Node.js frontend
FROM node:20-alpine AS node-builder

WORKDIR /app/ui

# Copy package files
COPY ui/package*.json ./

# Install npm dependencies with retry mechanism and multiple registries
RUN set -eux; \
    # Configure npm for better reliability
    npm config set fetch-timeout 300000; \
    npm config set fetch-retry-mintimeout 20000; \
    npm config set fetch-retry-maxtimeout 120000; \
    # Try npm registries one by one (sh-compatible syntax)
    for registry in \
        "https://registry.npmjs.org/" \
        "https://registry.npmmirror.com/" \
        "https://mirrors.tuna.tsinghua.edu.cn/npm/" \
        "https://mirrors.ustc.edu.cn/npm/" \
        "https://registry.npm.taobao.org/"; do \
        echo "Trying npm registry: $registry"; \
        npm config set registry "$registry"; \
        if npm ci --no-audit --no-fund; then \
            echo "npm packages installed successfully with registry: $registry"; \
            break; \
        else \
            echo "npm registry $registry failed, trying next..."; \
            npm cache clean --force; \
            sleep 5; \
        fi; \
    done; \
    # Verify node_modules exists
    ls -la node_modules/ | head -5

# Copy source code and build
COPY ui/ ./
RUN npm run build

# Production stage - using scratch for minimal image size
FROM scratch AS production

# Copy CA certificates from builder
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy built applications
COPY --from=go-builder /app/bin/gate /usr/local/bin/gate
COPY --from=go-builder /app/bin/console /usr/local/bin/console

# Copy frontend build
COPY --from=node-builder /app/ui/dist /app/ui/dist

# Copy configuration files
COPY configs/ /etc/infra-core/configs/

# Set environment variables
ENV INFRA_CORE_ENV=production
ENV INFRA_CORE_DATA_DIR=/var/lib/infra-core
ENV INFRA_CORE_LOG_DIR=/var/log/infra-core

# Expose ports
EXPOSE 80 443 8082

# Default command runs console
ENTRYPOINT ["/usr/local/bin/console"]

# Alternative production stage with Alpine (if you need shell access for debugging)
FROM alpine:latest AS production-debug

# Install runtime dependencies with retry mechanism and multiple mirrors
RUN set -eux; \
    # Try Alpine mirrors one by one (sh-compatible syntax)
    for mirror in \
        "https://dl-cdn.alpinelinux.org/alpine" \
        "https://mirrors.tuna.tsinghua.edu.cn/alpine" \
        "https://mirrors.ustc.edu.cn/alpine" \
        "https://mirrors.aliyun.com/alpine" \
        "https://mirror.nju.edu.cn/alpine" \
        "https://mirrors.huaweicloud.com/alpine"; do \
        echo "Trying mirror: $mirror"; \
        echo "$mirror/latest-stable/main" > /etc/apk/repositories && \
        echo "$mirror/latest-stable/community" >> /etc/apk/repositories && \
        # Test the mirror with update
        if apk update --no-cache 2>/dev/null; then \
            echo "Successfully using mirror: $mirror"; \
            # Install packages
            if apk add --no-cache ca-certificates tzdata wget curl; then \
                echo "Packages installed successfully with mirror: $mirror"; \
                break; \
            else \
                echo "Package installation failed with mirror: $mirror, trying next..."; \
            fi; \
        else \
            echo "Mirror $mirror failed, trying next..."; \
        fi; \
    done; \
    # Verify curl is installed
    curl --version

# Create non-root user
RUN addgroup -g 1001 -S infracore && \
    adduser -S infracore -u 1001 -G infracore

# Set up directories
RUN mkdir -p /var/lib/infra-core \
    /var/log/infra-core \
    /etc/infra-core/certs \
    /app/ui/dist && \
    chown -R infracore:infracore /var/lib/infra-core \
    /var/log/infra-core \
    /etc/infra-core \
    /app

# Copy built applications
COPY --from=go-builder /app/bin/* /usr/local/bin/
COPY --from=node-builder /app/ui/dist /app/ui/dist

# Copy configuration files
COPY configs/ /etc/infra-core/configs/

# Set ownership
RUN chown -R infracore:infracore /app

# Switch to non-root user
USER infracore

# Set environment variables
ENV INFRA_CORE_ENV=production
ENV INFRA_CORE_DATA_DIR=/var/lib/infra-core
ENV INFRA_CORE_LOG_DIR=/var/log/infra-core

# Expose ports
EXPOSE 80 443 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/api/v1/health || exit 1

# Default command runs console
CMD ["console"]