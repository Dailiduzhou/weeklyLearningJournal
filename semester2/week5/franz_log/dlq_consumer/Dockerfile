FROM golang:1.26-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o dlq_consumer .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/dlq_consumer .
CMD ["./dlq_consumer"]
