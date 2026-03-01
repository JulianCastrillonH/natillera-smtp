# ---- Build ----
# golang:alpine usa musl libc → binario más pequeño y build más liviano
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .

# GOFLAGS=-trimpath reduce tamaño; -p 1 limita paralelismo para free tier
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -p 1 \
    -ldflags="-s -w" \
    -o server \
    ./cmd/server

# ---- Runtime ----
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
