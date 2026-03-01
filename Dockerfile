# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o salesmate ./cmd

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/salesmate /app/salesmate

# Copy templates and skills
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/skills /app/skills

# Create data directory
RUN mkdir -p /app/data /app/workspace

# Set environment variables
ENV SALESMATE_WORKSPACE=/app/workspace
ENV SALESMATE_DATA=/app/data

# Expose port
EXPOSE 18790

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:18790/health || exit 1

# Run the binary
ENTRYPOINT ["/app/salesmate"]
CMD ["gateway"]