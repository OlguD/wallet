# API: statik Go binary, root olmayan kullanıcıyla çalışır.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/wallet-api ./cmd/server \
 && mkdir -p /out/data/receipts

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/wallet-api /wallet-api
# Dekont klasörü nonroot (65532) kullanıcısına ait; volume ilk oluşturulurken bu sahiplik kopyalanır.
COPY --from=build --chown=65532:65532 /out/data /data
ENV ADDR=:8080 RECEIPTS_DIR=/data/receipts MIGRATE_ON_START=true COOKIE_SECURE=true
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/wallet-api"]
