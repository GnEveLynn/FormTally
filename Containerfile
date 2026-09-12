FROM golang:1.27-alpine AS api-build

WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/formtally-api ./cmd/api

FROM alpine:3.22

RUN addgroup -S formtally && adduser -S -G formtally -h /nonexistent formtally
COPY --from=api-build /out/formtally-api /app/formtally-api

ENV HTTP_ADDR=0.0.0.0:8080

USER formtally
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
CMD ["/app/formtally-api"]
