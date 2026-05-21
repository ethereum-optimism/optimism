#
# Base container (with sccache and cargo-chef)
#
# - https://github.com/mozilla/sccache
# - https://github.com/LukeMathWalker/cargo-chef
#
# Based on https://depot.dev/blog/rust-dockerfile-best-practices
#
ARG FEATURES
ARG RBUILDER_BIN="op-rbuilder"

FROM rust:1.94 AS base
ARG TARGETPLATFORM

RUN apt-get update \
    && apt-get install -y clang libclang-dev libtss2-dev

RUN rustup component add clippy rustfmt


RUN set -eux; \
    case "$TARGETPLATFORM" in \
      "linux/amd64")  ARCH_TAG="x86_64-unknown-linux-musl" ;; \
      "linux/arm64")  ARCH_TAG="aarch64-unknown-linux-musl" ;; \
      *) \
        echo "Unsupported platform: $TARGETPLATFORM"; \
        exit 1 \
        ;; \
    esac; \
    wget -O /tmp/sccache.tar.gz \
      "https://github.com/mozilla/sccache/releases/download/v0.8.2/sccache-v0.8.2-${ARCH_TAG}.tar.gz"; \
    tar -xf /tmp/sccache.tar.gz -C /tmp; \
    mv /tmp/sccache-v0.8.2-${ARCH_TAG}/sccache /usr/local/bin/sccache; \
    chmod +x /usr/local/bin/sccache; \
    rm -rf /tmp/sccache.tar.gz /tmp/sccache-v0.8.2-${ARCH_TAG}

RUN cargo install cargo-chef --version ^0.1 --locked


ENV CARGO_HOME=/usr/local/cargo
ENV RUSTC_WRAPPER=sccache
ENV SCCACHE_DIR=/sccache

#
# op-rbuilder/Cargo.toml has path dependencies to sibling crates inside the
# parent rust/ workspace (../op-reth/..., ../op-revm/, ../op-alloy/...,
# ../alloy-op-evm/, ../alloy-op-hardforks/, ../rollup-boost/...). We pull
# those siblings in as a named BuildKit context so `docker buildx build`
# from inside rust/op-rbuilder/ still resolves them. From rust/op-rbuilder
# the parent dir `..` is the rust/ workspace, so:
#
#   docker buildx build \
#       --build-context monorepo-rust=.. \
#       ...
#
# Inside the image the final layout is flat under /workspace/:
#   /workspace/op-rbuilder/       <-- the build context (".")
#   /workspace/op-reth/           <-- from --build-context monorepo-rust
#   /workspace/op-revm/           <-- from --build-context monorepo-rust
#   /workspace/op-alloy/          <-- from --build-context monorepo-rust
#   /workspace/alloy-op-evm/      <-- from --build-context monorepo-rust
#   /workspace/rollup-boost/      <-- from --build-context monorepo-rust
#   ...
#
# which matches `../<crate>/...` in op-rbuilder/Cargo.toml.

#
# Planner container (running "cargo chef prepare")
#
FROM base AS planner
WORKDIR /workspace/op-rbuilder
COPY --from=monorepo-rust . /workspace/
COPY . .
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=$SCCACHE_DIR,sharing=locked \
    cargo chef prepare --recipe-path recipe.json

#
# Builder container (running "cargo chef cook" and "cargo build --release")
#
FROM base AS builder
WORKDIR /workspace/op-rbuilder
COPY --from=monorepo-rust . /workspace/
COPY --from=planner /workspace/op-rbuilder/recipe.json recipe.json
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=$SCCACHE_DIR,sharing=locked \
    cargo chef cook --release --recipe-path recipe.json
COPY . .


FROM builder AS rbuilder
ARG RBUILDER_BIN
ARG FEATURES
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=$SCCACHE_DIR,sharing=locked \
    cargo build --release --features="$FEATURES" --package=${RBUILDER_BIN}

#
# Reproducible builder container (deterministic source-date-epoch, no caching, no incremental builds)
#
FROM base AS rbuilder-reproducible
ARG RBUILDER_BIN
ARG FEATURES
ARG TARGETPLATFORM

ARG CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_RUSTFLAGS="-C target-feature=+crt-static -C link-arg=-static-libgcc -C link-arg=-Wl,--build-id=none -C metadata=target --remap-path-prefix=/workspace=."

ARG CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_RUSTFLAGS="-C target-feature=+crt-static -C link-arg=-static-libgcc -C link-arg=-Wl,--build-id=none -C metadata=target --remap-path-prefix=/workspace=."

WORKDIR /workspace/op-rbuilder
COPY --from=monorepo-rust . /workspace/
COPY . .
RUN case "$TARGETPLATFORM" in \
      "linux/amd64")  ARCH_TAG="x86_64-unknown-linux-gnu" ;; \
      "linux/arm64")  ARCH_TAG="aarch64-unknown-linux-gnu" ;; \
      *) \
        echo "Unsupported platform: $TARGETPLATFORM"; \
        exit 1 \
        ;; \
    esac; \
    SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct) \
    RUSTFLAGS="-C target-feature=+crt-static -C link-arg=-static-libgcc -C link-arg=-Wl,--build-id=none -C metadata='' --remap-path-prefix=/workspace=." \
    CARGO_INCREMENTAL=0 \
    LC_ALL=C \
    TZ=UTC \
    CFLAGS="-D__TIME__=\"\" -D__DATE__=\"\"" \
    CXXFLAGS="-D__TIME__=\"\" -D__DATE__=\"\"" \
    cargo build --release --locked --features="$FEATURES" --package=${RBUILDER_BIN} --target "${ARCH_TAG}"

# Runtime container for rbuilder
FROM gcr.io/distroless/cc-debian13:latest@sha256:8b5d1db6d2253036a53cb8362d3e3fa82a7caf84c247772c46a023166c64e977 AS rbuilder-runtime
ARG RBUILDER_BIN
WORKDIR /app
COPY --from=rbuilder /workspace/op-rbuilder/target/release/${RBUILDER_BIN} /app/rbuilder
ENTRYPOINT ["/app/rbuilder"]

# Reproducible runtime container for rbuilder
FROM gcr.io/distroless/cc-debian13:latest@sha256:8b5d1db6d2253036a53cb8362d3e3fa82a7caf84c247772c46a023166c64e977 AS rbuilder-reproducible-runtime
ARG RBUILDER_BIN
WORKDIR /app
COPY --from=rbuilder-reproducible /workspace/op-rbuilder/target/*/release/${RBUILDER_BIN} /app/rbuilder
ENTRYPOINT ["/app/rbuilder"]
