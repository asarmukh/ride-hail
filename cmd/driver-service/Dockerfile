# Stage 1: build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o driver-service ./cmd/driver-service

# Stage 2: runtime
FROM alpine:3.19

WORKDIR /app
RUN apk --no-cache add ca-certificates postgresql15-client

COPY --from=builder /app/driver-service .
COPY config.yaml /app/config.yaml

EXPOSE 3001
CMD ["./driver-service"]