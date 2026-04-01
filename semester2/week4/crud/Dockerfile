# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o user ./cmd/user.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /build/user .
COPY --from=builder /build/cmd/etc/user-docker.yaml ./etc/user.yaml

EXPOSE 8888

ENV TZ=Asia/Shanghai

CMD ["./user", "-f", "etc/user.yaml"]
