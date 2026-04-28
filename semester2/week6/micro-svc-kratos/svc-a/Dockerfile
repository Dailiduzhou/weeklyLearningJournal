FROM golang:1.25 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends git

COPY svc-a /src/svc-a
COPY svc-b /src/svc-b
WORKDIR /src/svc-a

RUN GOPROXY=https://goproxy.cn go mod download
RUN make build

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/svc-a/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./svc-a", "-conf", "/data/conf"]
