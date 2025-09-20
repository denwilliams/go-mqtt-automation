# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates for Go modules
RUN apk add --no-cache git ca-certificates gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o automation-server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS and sqlite
RUN apk --no-cache add ca-certificates sqlite

# Create non-root user
RUN addgroup -S automation && adduser -S automation -G automation

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/automation-server .

# Copy static files and templates
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config ./config

# Create data directory for SQLite database
RUN mkdir -p /app/data && chown automation:automation /app/data

# Switch to non-root user
USER automation

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default configuration
ENV CONFIG_PATH=/app/config/config.yaml
ENV DB_PATH=/app/data/automation.db

# Run the application
CMD ["./automation-server", "-config", "/app/config/config.yaml"]