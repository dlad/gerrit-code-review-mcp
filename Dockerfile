# Build stage
FROM golang:1.24.3-alpine AS builder

# Set working directory
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build cmd/gerrit-code-review-mcp.go

# Final stage
FROM alpine:latest

# Copy compiled binary
COPY --from=builder /app/gerrit-code-review-mcp /usr/local/bin/gerrit-code-review-mcp

# Set entrypoint
ENTRYPOINT ["gerrit-code-review-mcp"]