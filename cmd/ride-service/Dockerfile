# Stage 1: build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o ride-service ./cmd/ride-service

# Stage 2: runtime
FROM alpine:3.19

WORKDIR /app
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/ride-service .
COPY config.yaml /app/config.yaml

EXPOSE 3000
CMD ["./ride-service"]