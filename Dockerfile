# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sync ./cmd/sync
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web ./cmd/web

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests and tzdata for timezone support
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy both binaries from builder
COPY --from=builder /build/sync .
COPY --from=builder /build/web .

# Default to web server, override with CMD for sync
CMD ["/app/web"]
