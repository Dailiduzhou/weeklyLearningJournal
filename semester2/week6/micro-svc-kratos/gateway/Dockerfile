FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/gateway/

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /app /app
COPY configs /data/conf

EXPOSE 8080

CMD ["/app", "-conf", "/data/conf", "-addr", ":8080", "-etcd", "etcd:2379"]
