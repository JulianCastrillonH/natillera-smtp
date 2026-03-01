# ---- Build ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Limitar paralelismo del compilador para reducir uso de RAM en free tier
ENV GOMAXPROCS=1
ENV GOGC=50

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -p 1 -ldflags="-s -w" -o server ./cmd/server

# ---- Runtime ----
FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
