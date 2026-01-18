# Stage 1: Builder
FROM rust:1.75 AS builder

WORKDIR /app

# Copy dependency files first for better caching
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && touch src/lib.rs && echo "// placeholder" > src/lib.rs
RUN cargo fetch

# Copy source code
COPY . .

# Build the project
RUN cargo build

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install required runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN useradd -m -s /bin/bash appuser

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/target/debug/mvc_seaorm /app/
COPY --from=builder /app/templates/ /app/templates/

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

CMD ["./mvc_seaorm"]
