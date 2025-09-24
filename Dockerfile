# Build stage for Go applications
FROM golang:1.24.5-alpine AS go-builder

WORKDIR /app

# Install dependencies for building with retry mechanism and multiple mirrors
RUN set -eux; \
    # Update package index with retries
    for i in 1 2 3; do \
        apk update && break || { \
            echo "Retry $i: apk update failed, retrying in 5s..."; \
            sleep 5; \
        } \
    done; \
    # Install packages with retry mechanism
    for i in 1 2 3; do \
        apk add --no-cache git ca-certificates && break || { \
            echo "Retry $i: apk install failed, retrying in 5s..."; \
            sleep 5; \
        } \
    done

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with retry mechanism
RUN set -eux; \
    for i in 1 2 3; do \
        go mod download && go mod verify && break || { \
            echo "Retry $i: go mod download failed, retrying in 5s..."; \
            sleep 5; \
            go clean -modcache; \
        } \
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

# Install npm dependencies with retry mechanism
RUN set -eux; \
    # Configure npm for better reliability
    npm config set fetch-timeout 300000; \
    npm config set fetch-retry-mintimeout 20000; \
    npm config set fetch-retry-maxtimeout 120000; \
    # Install with retries
    for i in 1 2 3; do \
        npm ci && break || { \
            echo "Retry $i: npm ci failed, retrying in 10s..."; \
            sleep 10; \
            npm cache clean --force; \
        } \
    done

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

# Install runtime dependencies with retry mechanism
RUN set -eux; \
    # Update package index with retries
    for i in 1 2 3; do \
        apk update && break || { \
            echo "Retry $i: apk update failed, retrying in 5s..."; \
            sleep 5; \
        } \
    done; \
    # Install packages with retry mechanism
    for i in 1 2 3; do \
        apk add --no-cache ca-certificates tzdata wget curl && break || { \
            echo "Retry $i: apk install failed, retrying in 5s..."; \
            sleep 5; \
        } \
    done

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