#
# Base container (with sccache and cargo-chef)
#
# - https://github.com/mozilla/sccache
# - https://github.com/LukeMathWalker/cargo-chef
#
# Based on https://depot.dev/blog/rust-dockerfile-best-practices
#
FROM rust:1.95 AS base

ARG FEATURES
ARG RELEASE=true

RUN cargo install sccache --version ^0.9 --locked
RUN cargo install cargo-chef --version ^0.1 --locked

RUN apt-get update \
    && apt-get install -y clang libclang-dev

ENV CARGO_HOME=/usr/local/cargo
ENV RUSTC_WRAPPER=sccache
ENV SCCACHE_DIR=/sccache

#
# rollup-boost/Cargo.toml has path dependencies to sibling crates inside the
# parent rust/ workspace (../op-reth/crates/payload, ../op-alloy/crates/*).
# We pull those siblings in as a named BuildKit context so `docker buildx
# build` from inside rust/rollup-boost/ still resolves them. From
# rust/rollup-boost the parent dir `..` is the rust/ workspace, so:
#
#   docker buildx build \
#       --build-context monorepo-rust=.. \
#       ...
#
# Inside the image the final layout is flat under /workspace/:
#   /workspace/rollup-boost/   <-- the build context (".")
#   /workspace/op-reth/        <-- from --build-context monorepo-rust
#   /workspace/op-alloy/       <-- from --build-context monorepo-rust
#   ...
#
# which matches `../<crate>/...` in rollup-boost/Cargo.toml.

#
# Planner container (running "cargo chef prepare")
#
FROM base AS planner
WORKDIR /workspace/rollup-boost

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
WORKDIR /workspace/rollup-boost
# Default binary filename
ARG SERVICE_NAME="rollup-boost"
COPY --from=monorepo-rust . /workspace/
# op-reth's chainspec build.rs regenerates its gitignored superchain archive from
# the submodule; provide the subset it reads at /superchain-registry.
COPY --from=superchain-registry superchain /superchain-registry/superchain
COPY --from=planner /workspace/rollup-boost/recipe.json recipe.json

RUN --mount=type=cache,target=$SCCACHE_DIR,sharing=locked \
    PROFILE_FLAG=$([ "$RELEASE" = "true" ] && echo "--release" || echo "") && \
    cargo chef cook $PROFILE_FLAG --locked --recipe-path recipe.json

COPY . .

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=$SCCACHE_DIR,sharing=locked \
    PROFILE_FLAG=$([ "$RELEASE" = "true" ] && echo "--release" || echo "") && \
    TARGET_DIR=$([ "$RELEASE" = "true" ] && echo "release" || echo "debug") && \
    cargo build $PROFILE_FLAG --locked --features="$FEATURES" --package=${SERVICE_NAME} && \
    cp "target/${TARGET_DIR}/${SERVICE_NAME}" /tmp/final_binary && \
    test -n "$('/tmp/final_binary' --help 2>&1)"

#
# Runtime container
#
FROM chainguard/wolfi-base:latest@sha256:02dab76bd852a70556b5b2002195c8a5fdab77d323c433bf6642aab080489795
RUN n=0; until apk add --no-cache ca-certificates-bundle ca-certificates busybox glibc libgcc libstdc++ openssl; do n=$((n+1)); [ "$n" -ge 15 ] && exit 1; echo "apk add retry $n/15 in 20s" >&2; sleep 20; done
WORKDIR /app

ARG SERVICE_NAME
# Copy binary with its proper service name
COPY --from=builder /tmp/final_binary /usr/local/bin/${SERVICE_NAME}
# Also copy as a fixed entrypoint name
COPY --from=builder /tmp/final_binary /usr/local/bin/entrypoint

ENTRYPOINT ["/usr/local/bin/entrypoint"]
