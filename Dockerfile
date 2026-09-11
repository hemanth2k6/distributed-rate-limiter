FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o rate-limiter .

# Start a new stage from a lightweight alpine image
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/rate-limiter .

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["./rate-limiter"]
