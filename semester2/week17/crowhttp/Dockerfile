# syntax=docker/dockerfile:1.6

# ---- Build stage ----
FROM ubuntu:24.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
ENV XMAKE_ROOT=y
ENV PATH="/root/.local/bin:${PATH}"

RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  git \
  libreadline-dev \
  libpqxx-dev \
  libhiredis-dev \
  libssl-dev \
  curl \
  ca-certificates \
  unzip \
  && rm -rf /var/lib/apt/lists/* \
  && curl -fsSL https://xmake.io/shget.text | bash \
  && xmake --version

WORKDIR /app

# Copy build manifest first to cache the dependency download layer
COPY xmake.lua ./
RUN xmake f -m release -y

# Now copy source and build
COPY src ./src
RUN xmake build -y
RUN xmake install -o /opt/crowhttp

# Collect xmake-built shared libraries (redis-plus-plus is not in apt)
RUN mkdir -p /opt/crowhttp/lib && \
  find /root/.xmake -name "libredis++*.so*" -exec cp -P {} /opt/crowhttp/lib/ \; ; \
  find /root/.xmake -name "libhiredis*.so*" -exec cp -P {} /opt/crowhttp/lib/ \; ; \
  ls -la /opt/crowhttp/lib/

# ---- Runtime stage ----
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
  libpqxx-dev \
  libhiredis-dev \
  libssl3 \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /opt/crowhttp /opt/crowhttp

# xmake-built shared libs take precedence; system libs serve as fallback
ENV LD_LIBRARY_PATH=/opt/crowhttp/lib

EXPOSE 18080

WORKDIR /opt/crowhttp/bin
ENTRYPOINT ["/opt/crowhttp/bin/crowhttp"]
