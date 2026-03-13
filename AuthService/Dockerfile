#building stage
FROM golang:1.25-alpine AS builder
LABEL authors="etternalattack"

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o auth ./cmd/auth/

#running stage
FROM alpine:latest

RUN adduser -D authuser
USER authuser

COPY --from=builder /app/auth .

EXPOSE 8000
CMD ["./auth"]