FROM rust:1.85 AS builder

WORKDIR /app

ARG BINARY="flashblocks-websocket-proxy"
ARG FEATURES

COPY . .

RUN cargo build --release --features="$FEATURES" --package=${BINARY}

FROM gcr.io/distroless/cc-debian12
WORKDIR /app

ARG BINARY="flashblocks-websocket-proxy"
COPY --from=builder /app/target/release/${BINARY} /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/flashblocks-websocket-proxy"]
