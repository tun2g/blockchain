# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build both the node and CLI binaries
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o blockchain-node ./cmd/node
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o blockchain-cli ./cmd/cli

# Run stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates netcat-openbsd

WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/blockchain-node .
COPY --from=builder /app/blockchain-cli .

# Create data directory
RUN mkdir -p /app/data

# Set proper permissions
RUN chmod +x blockchain-node blockchain-cli

# Expose the ports
EXPOSE 50051 8080

# Default command
CMD ["./blockchain-node"] 