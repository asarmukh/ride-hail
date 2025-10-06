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

EXPOSE 3000
CMD ["sh", "-c", "until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do echo waiting for postgres; sleep 2; done; ./driver-service"]
