# syntax=docker/dockerfile:1.7
###############################################################################
# Stage 1 — build environment with vcpkg
#
# Builds drogon + aws-sdk-cpp(s3) once. Layer caching is preserved between
# builds by copying vcpkg.json first; the dependency tree is large but stable.
###############################################################################
FROM ubuntu:22.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
ENV VCPKG_ROOT=/opt/vcpkg \
    VCPKG_DISABLE_METRICS=1

# Base toolchain + system deps required by the vcpkg ports we use.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        cmake \
        curl \
        git \
        libssl-dev \
        libuuid1 \
        ninja-build \
        pkg-config \
        python3 \
        tar \
        unzip \
        zip \
    && rm -rf /var/lib/apt/lists/*

# Reuse the vendored vcpkg checkout in the workspace so the image can build
# without network access to fetch a separate toolchain.
COPY vcpkg "${VCPKG_ROOT}"
RUN "${VCPKG_ROOT}/scripts/bootstrap.sh" -disableMetrics

WORKDIR /src

# Copy manifest first so docker can cache the vcpkg install step as long as
# vcpkg.json is unchanged (the long, expensive build of aws-sdk-cpp).
COPY vcpkg.json ./
RUN "${VCPKG_ROOT}/vcpkg" install --triplet=x64-linux --x-install=/opt/vcpkg-installed

# Now copy the rest of the sources and build the app.
COPY CMakeLists.txt ./
COPY config.json ./
COPY config.yaml ./
COPY controllers ./controllers
COPY filters     ./filters
COPY plugins     ./plugins
COPY models      ./models
COPY views       ./views
COPY test        ./test
COPY main.cc     ./main.cc
COPY scripts     ./scripts
COPY minio       ./minio

RUN cmake -S . -B build \
        -G Ninja \
        -DCMAKE_BUILD_TYPE=Release \
        -DAWSS3_BUILD_TESTS=OFF \
        -DCMAKE_TOOLCHAIN_FILE="${VCPKG_ROOT}/scripts/buildsystems/vcpkg.cmake" \
    && cmake --build build -j

###############################################################################
# Stage 2 — minimal runtime image
###############################################################################
FROM ubuntu:22.04 AS runtime

ENV DEBIAN_FRONTEND=noninteractive \
    HTTP_HOST=0.0.0.0 \
    HTTP_PORT=8080 \
    AWS_REGION=us-east-1 \
    S3_BUCKET=test-bucket

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        libcurl4 \
        libssl3 \
        libuuid1 \
        curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /src/build/awss3         /app/awss3
COPY --from=builder /opt/vcpkg-installed     /opt/vcpkg-installed

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD curl -fsS "http://127.0.0.1:${HTTP_PORT}/health" || exit 1

ENTRYPOINT ["/app/awss3"]
