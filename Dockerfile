# --- Build stage ---
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server ./...

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server /usr/local/bin/server

EXPOSE 5000

CMD ["server"]
