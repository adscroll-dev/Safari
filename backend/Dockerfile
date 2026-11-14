# VYRO backend — multi-stage Docker build (Go 1.23, no paid services)
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /vyro-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 vyro
USER vyro
WORKDIR /app
COPY --from=build /vyro-server /app/vyro-server
EXPOSE 8080
ENV PORT=8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/vyro-server"]
