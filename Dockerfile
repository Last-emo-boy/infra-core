# Build stage for Go applications
FROM golang:1.24.5-alpine AS go-builder

# Build arguments for mirror selection
ARG BUILD_REGION=auto
ARG ALPINE_MIRROR=""
ARG GO_PROXY=""

WORKDIR /app

# Install dependencies with smart mirror selection
RUN set -eux; \
    echo "🐛 Debug Go Builder: BUILD_REGION=$BUILD_REGION"; \
    echo "🐛 Debug Go Builder: ALPINE_MIRROR=$ALPINE_MIRROR"; \
    echo "🐛 Debug Go Builder: GO_PROXY=$GO_PROXY"; \
    # Configure Alpine mirror based on build arguments or auto-detect
    if [ -n "$ALPINE_MIRROR" ] && [ "$ALPINE_MIRROR" != "" ]; then \
        echo "🚀 Using speed-tested optimal Alpine mirror: $ALPINE_MIRROR"; \
        echo "$ALPINE_MIRROR/v3.22/main" > /etc/apk/repositories; \
        echo "$ALPINE_MIRROR/v3.22/community" >> /etc/apk/repositories; \
    elif [ "$BUILD_REGION" = "cn" ]; then \
        echo "Using Chinese Alpine mirror"; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.22/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.22/community" >> /etc/apk/repositories; \
    elif [ "$BUILD_REGION" = "optimized" ]; then \
        echo "Using optimized configuration (mirrors selected by speed test)"; \
    fi; \
    # Install packages with fallback
    apk update --no-cache || { \
        echo "Primary mirror failed, trying fallback mirrors (prioritizing fast mirrors)..."; \
        for mirror in \
            "https://mirrors.tuna.tsinghua.edu.cn/alpine" \
            "https://mirrors.ustc.edu.cn/alpine" \
            "https://mirrors.aliyun.com/alpine" \
            "https://dl-cdn.alpinelinux.org/alpine"; do \
            echo "Trying fallback mirror: $mirror"; \
            echo "$mirror/v3.22/main" > /etc/apk/repositories; \
            echo "$mirror/v3.22/community" >> /etc/apk/repositories; \
            if apk update --no-cache 2>/dev/null; then \
                echo "Fallback mirror $mirror works"; \
                break; \
            fi; \
        done; \
    }; \
    apk add --no-cache git ca-certificates; \
    git --version

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with smart Go proxy selection
RUN set -eux; \
    # Configure Go proxy based on build arguments or region
    if [ -n "$GO_PROXY" ] && [ "$GO_PROXY" != "" ]; then \
        echo "🚀 Using speed-tested optimal Go proxy: $GO_PROXY"; \
        export GOPROXY="$GO_PROXY"; \
    elif [ "$BUILD_REGION" = "cn" ]; then \
        echo "Using Chinese Go proxy"; \
        export GOPROXY="https://goproxy.cn,direct"; \
    elif [ "$BUILD_REGION" = "optimized" ]; then \
        echo "Using optimized Go proxy (selected by speed test)"; \
        export GOPROXY="${GO_PROXY:-https://proxy.golang.org,direct}"; \
    else \
        export GOPROXY="https://proxy.golang.org,direct"; \
    fi; \
    export GOSUMDB="sum.golang.org"; \
    # Download with fallback
    go mod download && go mod verify || { \
        echo "Primary Go proxy failed, trying fallbacks..."; \
        for proxy in \
            "https://goproxy.io,direct" \
            "https://mirrors.aliyun.com/goproxy/,direct" \
            "https://proxy.golang.org,direct"; do \
            echo "Trying fallback proxy: $proxy"; \
            export GOPROXY="$proxy"; \
            if go mod download && go mod verify; then \
                echo "Fallback proxy $proxy works"; \
                break; \
            else \
                go clean -modcache; \
            fi; \
        done; \
    }

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

# Build arguments for NPM registry
ARG NPM_REGISTRY=""
ARG BUILD_REGION=""

WORKDIR /app/ui

# Copy package files
COPY ui/package*.json ./

# Install npm dependencies with smart registry selection
RUN set -eux; \
    # Configure npm for better reliability
    npm config set fetch-timeout 300000; \
    npm config set fetch-retry-mintimeout 20000; \
    npm config set fetch-retry-maxtimeout 120000; \
    # Configure NPM registry based on build arguments or region
    if [ -n "$NPM_REGISTRY" ] && [ "$NPM_REGISTRY" != "" ]; then \
        echo "🚀 Using speed-tested optimal NPM registry: $NPM_REGISTRY"; \
        npm config set registry "$NPM_REGISTRY"; \
    elif [ "$BUILD_REGION" = "cn" ]; then \
        echo "Using Chinese NPM registry"; \
        npm config set registry "https://registry.npmmirror.com/"; \
    elif [ "$BUILD_REGION" = "optimized" ]; then \
        echo "Using optimized NPM registry (selected by speed test)"; \
        npm config set registry "${NPM_REGISTRY:-https://registry.npmjs.org/}"; \
    fi; \
    # Install with fallback
    npm ci --no-audit --no-fund || { \
        echo "Primary NPM registry failed, trying fallbacks..."; \
        for registry in \
            "https://registry.npmjs.org/" \
            "https://registry.npmmirror.com/" \
            "https://mirrors.ustc.edu.cn/npm/"; do \
            echo "Trying fallback registry: $registry"; \
            npm config set registry "$registry"; \
            npm cache clean --force; \
            if npm ci --no-audit --no-fund; then \
                echo "Fallback registry $registry works"; \
                break; \
            fi; \
        done; \
    }; \
    ls -la node_modules/ | head -3

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

# Copy configuration files to both locations for compatibility
COPY configs/ /etc/infra-core/configs/
COPY configs/ /app/configs/

# Set working directory
WORKDIR /app

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

# Build arguments for mirror selection
ARG BUILD_REGION=auto
ARG ALPINE_MIRROR=""

# Install runtime dependencies with retry mechanism and multiple mirrors
RUN set -eux; \
    echo "🐛 Debug: BUILD_REGION=$BUILD_REGION"; \
    echo "🐛 Debug: ALPINE_MIRROR=$ALPINE_MIRROR"; \
    # Configure Alpine mirror based on build arguments or auto-detect
    if [ -n "$ALPINE_MIRROR" ] && [ "$ALPINE_MIRROR" != "" ]; then \
        echo "🚀 Using speed-tested optimal Alpine mirror: $ALPINE_MIRROR"; \
        echo "$ALPINE_MIRROR/latest-stable/main" > /etc/apk/repositories; \
        echo "$ALPINE_MIRROR/latest-stable/community" >> /etc/apk/repositories; \
        if apk update --no-cache && apk add --no-cache ca-certificates tzdata wget curl; then \
            echo "✅ Packages installed successfully with optimal mirror: $ALPINE_MIRROR"; \
        else \
            echo "❌ Optimal mirror failed, falling back to multiple mirrors..."; \
            # Fallback to multiple mirrors if optimal fails - prioritize fast mirrors
            for mirror in \
                "https://mirrors.tuna.tsinghua.edu.cn/alpine" \
                "https://mirrors.ustc.edu.cn/alpine" \
                "https://mirrors.aliyun.com/alpine" \
                "https://mirror.nju.edu.cn/alpine" \
                "https://mirrors.huaweicloud.com/alpine" \
                "https://dl-cdn.alpinelinux.org/alpine"; do \
                echo "Trying fallback mirror: $mirror"; \
                echo "$mirror/latest-stable/main" > /etc/apk/repositories && \
                echo "$mirror/latest-stable/community" >> /etc/apk/repositories && \
                if apk update --no-cache 2>/dev/null && apk add --no-cache ca-certificates tzdata wget curl; then \
                    echo "✅ Packages installed successfully with fallback mirror: $mirror"; \
                    break; \
                fi; \
                echo "❌ Fallback mirror $mirror failed, trying next..."; \
            done; \
        fi; \
    elif [ "$BUILD_REGION" = "cn" ]; then \
        echo "Using Chinese Alpine mirror (region-based)"; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/latest-stable/main" > /etc/apk/repositories; \
        echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/latest-stable/community" >> /etc/apk/repositories; \
        apk update --no-cache && apk add --no-cache ca-certificates tzdata wget curl; \
    else \
        echo "Using default mirror fallback strategy (prioritizing fast mirrors)"; \
        # Try Alpine mirrors one by one - prioritize fast Chinese mirrors first
        for mirror in \
            "https://mirrors.tuna.tsinghua.edu.cn/alpine" \
            "https://mirrors.ustc.edu.cn/alpine" \
            "https://mirrors.aliyun.com/alpine" \
            "https://mirror.nju.edu.cn/alpine" \
            "https://mirrors.huaweicloud.com/alpine" \
            "https://dl-cdn.alpinelinux.org/alpine"; do \
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
    fi; \
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

# Copy configuration files to both locations
COPY configs/ /etc/infra-core/configs/
COPY configs/ /app/configs/

# Set working directory
WORKDIR /app

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