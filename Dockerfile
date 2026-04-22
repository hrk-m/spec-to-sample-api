# Builder stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/api ./app/main.go

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache mysql-client wget

RUN wget -qO- https://github.com/golang-migrate/migrate/releases/download/v4.18.3/migrate.linux-amd64.tar.gz \
    | tar xz -C /usr/local/bin

WORKDIR /app

COPY --from=builder /app/bin/api .
COPY --from=builder /app/db ./db
COPY entrypoint.sh .

RUN chmod +x /app/entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
