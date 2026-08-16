FROM golang:1.26.3-alpine3.22 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/indexer ./cmd/indexer

FROM alpine:3.22.1 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 10001 gorag && \
    adduser -S -D -H -u 10001 -G gorag gorag

WORKDIR /app
USER gorag:gorag

FROM runtime AS server
COPY --from=build --chown=gorag:gorag /out/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]

FROM runtime AS indexer
COPY --from=build --chown=gorag:gorag /out/indexer /app/indexer
ENTRYPOINT ["/app/indexer"]
