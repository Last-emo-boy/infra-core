# Build stage for Go applications
FROM golang:1.24.5-alpine AS go-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build applications
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/gate cmd/gate/main.go
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/console cmd/console/main.go

# Build stage for Node.js frontend
FROM node:20-alpine AS node-builder

WORKDIR /app/ui

# Copy package files
COPY ui/package*.json ./
RUN npm ci

# Copy source code and build
COPY ui/ ./
RUN npm run build

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

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