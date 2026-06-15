# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o admission-controller .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Use /app instead of /root for non-root compatibility
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/admission-controller .

# Set execute permissions and change ownership for non-root user
RUN chmod +x admission-controller && \
    chown -R 1000:1000 /app

# Create directory for certs
RUN mkdir -p /etc/webhook/certs

# Expose port
EXPOSE 8443

# Run the binary
ENTRYPOINT ["./admission-controller"]
