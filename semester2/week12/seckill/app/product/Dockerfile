FROM golang:1.25-alpine AS builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache git make

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY api/ ./api/
COPY app/product/ ./app/product/

# 编译二进制文件
RUN cd app/product/cmd/product && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/product .

FROM alpine:3.21 AS runner

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/product /app/product

EXPOSE 8000 9000
VOLUME /data/conf

CMD ["./product", "-conf", "/data/conf"]
