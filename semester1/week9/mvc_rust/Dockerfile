# syntax=docker/dockerfile:1
# ---------- 构建阶段 ----------
# 工具链要和 Cargo.toml / Cargo.lock 对得上：
#   * 本项目 Cargo.toml 用的是 edition = "2024"，它需要 Cargo >= 1.85
#   * Cargo.lock 里 base64ct / globset / home / ignore 也声明了 edition 2024
# 用旧镜像（比如 1.83）会在解析 manifest 时就报错：
#   error: failed to parse manifest at `/app/Cargo.toml`
#   Caused by: feature `edition2024` is required
# 这里钉在 1.98，和本机验证过的工具链一致；想永远跟最新稳定版可以改成 rust:1-bookworm。
FROM rust:1.98-bookworm AS builder
WORKDIR /app

# 先只拷 Cargo.toml / Cargo.lock，用占位 main.rs 把依赖编译一遍。
# 这样以后只改 src/ 下的代码时，依赖层还能命中缓存，不用重新编译 actix-web 全家桶。
# --locked：严格按 Cargo.lock 的版本来，保证镜像里编的和本机跑过的是同一套依赖。
COPY Cargo.toml Cargo.lock ./
RUN mkdir -p src \
    && echo 'fn main() {}' > src/main.rs \
    && cargo build --release --locked \
    && rm -rf src

# 再拷真实源码和模板
COPY src ./src
COPY templates ./templates
# 占位 main.rs 和真实 main.rs 时间戳可能相同，touch 一下确保重新编译
RUN touch src/main.rs && cargo build --release --locked

# ---------- 运行阶段 ----------
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -s /bin/bash appuser

WORKDIR /app
COPY --from=builder /app/target/release/mvc_rust ./mvc_rust
# Tera::new("templates/**/*") 是相对工作目录找模板的，必须带上
COPY --from=builder /app/templates ./templates
RUN chown -R appuser:appuser /app

USER appuser
EXPOSE 8080

# 容器内探活：登录页能返回 200 就算健康
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD curl -fsS http://127.0.0.1:8080/api/users/login >/dev/null || exit 1

CMD ["./mvc_rust"]
