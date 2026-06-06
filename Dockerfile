FROM golang:1.26 AS builder

WORKDIR /app

# Copy go.mod and go.sum to leverage Docker cache for dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application and store the binary in the root directory of the container
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-gateway ./cmd/gateway/main.go

# Create a minimal image for the final application
FROM alpine:latest

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /api-gateway .

COPY config.yaml .

EXPOSE 8080

CMD ["./api-gateway"]