# Build stage
FROM golang:1.25-alpine AS builder

# Build argument for environment selection (default: local)
ARG ENV=local

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Re-declare ARG for runtime stage
ARG ENV=local

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/server .

# Copy the environment-specific config file
COPY --from=builder /app/deploy/${ENV}.env ./.env

# Expose port
EXPOSE 8080

# Source env file and run the application
# Using sh to load env vars from file
CMD sh -c 'export $(cat .env | grep -v "^#" | xargs) && ./server'