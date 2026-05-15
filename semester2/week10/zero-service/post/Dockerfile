FROM golang:1.24-alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --no-cache tzdata

WORKDIR /build

ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .

RUN go build -ldflags="-s -w" -o /app/post post/post.go


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/post /app/post
COPY post/etc /app/etc

CMD ["./post", "-f", "etc/post.yaml"]
