# 阶段 1: 编译环境
FROM golang:1.23-alpine AS builder

# 设置 Go 代理，加快国内下载速度
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
# CGO_ENABLED=0 表示静态编译，不依赖系统动态库
RUN CGO_ENABLED=0 GOOS=linux go build -o library_server main.go

# 阶段 2: 运行环境
FROM alpine:latest

# 安装基础库（如时区数据、ca证书）
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从编译阶段复制二进制文件
COPY --from=builder /app/library_server .
# 复制 uploads 文件夹结构（如果需要）
COPY --from=builder /app/uploads ./uploads

# 设置时区
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 8080

CMD ["./library_server"]