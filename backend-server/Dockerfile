FROM golang:1.26 AS builder
WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY backend-server/ ./backend-server/

RUN CGO_ENABLED=0 GOOS=linux go build -o /backend-server ./backend-server/main.go

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /backend-server .

EXPOSE 8081 8082 8083 8084 8085

CMD ["./backend-server"]

