FROM rust:1.88 AS builder

WORKDIR /app

ARG BINARY="tdx-quote-provider"
ARG FEATURES

COPY . .

RUN apt-get update && apt-get install -y libssl3 libtss2-dev

RUN cargo build --release --features="$FEATURES" --package=${BINARY}

FROM debian:12-slim
WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y libssl3 libtss2-dev

ARG BINARY="tdx-quote-provider"
COPY --from=builder /app/target/release/${BINARY} /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/tdx-quote-provider"]
