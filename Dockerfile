# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the sync binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sync ./cmd/sync

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests and tzdata for timezone support
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/sync .

# Run the sync binary
CMD ["/app/sync"]
