FROM golang:1.26.0-alpine AS builder

WORKDIR /app

# Кэшируем зависимости отдельным слоем — не пересобираются, если go.mod/go.sum не менялись
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/graphs-service ./cmd

# ---- Run stage ----
FROM alpine:3.20

# ca-certificates — нужен для исходящих HTTPS-соединений (если понадобится, например, внешние API)
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/graphs-service .

EXPOSE 8080

CMD ["./graphs-service"]