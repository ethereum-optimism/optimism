# OP Stack Dockerfile, simplified from ops/docker/op-stack-go/Dockerfile

# Build arguments
ARG TARGET_BASE_IMAGE=ubuntu:22.04
ARG TARGETOS
ARG TARGETARCH

# Base builder image
FROM golang:1.22.7-alpine3.20 AS builder

RUN apk add --no-cache curl tar gzip make gcc musl-dev linux-headers git jq bash

# Install mise for toolchain management
RUN curl https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise sh

# Install yq
RUN case "$TARGETARCH" in \
  "amd64") YQ_ARCH="amd64" ;; \
  "arm64") YQ_ARCH="arm64" ;; \
  *) YQ_ARCH="amd64" ;; \
  esac && \
  wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_$YQ_ARCH -O /usr/local/bin/yq && \
  chmod +x /usr/local/bin/yq

# Install versioned toolchain
COPY ./mise.toml .
RUN mise trust && mise install -v -y just && cp $(mise which just) /usr/local/bin/just && just --version

# Copy and download Go dependencies
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download

# Copy source code
COPY . /app

# Build arguments for git metadata
ARG GIT_COMMIT
ARG GIT_DATE

# Rust builder for Espresso crypto libraries
FROM rust:1.84.1-alpine3.20 AS rust-builder
# TODO: Check the hash of the Espresso GO library when switch to the new one.
# <https://app.asana.com/1/1208976916964769/project/1209392461754458/task/1210755885228839?focus=true>
ARG ESPRESSO_NETWORK_GO_VER=0.0.34
RUN apk add perl make openssl-dev musl-dev gcc
ADD https://github.com/EspressoSystems/espresso-network-go/archive/refs/tags/v$ESPRESSO_NETWORK_GO_VER.tar.gz /source.tgz
RUN tar -oxzf /source.tgz
WORKDIR /espresso-network-go-$ESPRESSO_NETWORK_GO_VER
RUN --mount=type=cache,target=/usr/local/cargo/registry \
  --mount=type=cache,target=/usr/local/cargo/git/db \
  --mount=type=cache,target=/espresso-network-go/verification/rust/target \
  cargo build --release  --locked  --manifest-path ./verification/rust/Cargo.toml
RUN mkdir -p /libespresso
RUN cp ./verification/rust/target/release/libespresso_crypto_helper.a \
  /libespresso/libespresso_crypto_helper-aarch64-unknown-linux-gnu.a
RUN cp ./verification/rust/target/release/libespresso_crypto_helper.a \
  /libespresso/libespresso_crypto_helper-x86_64-unknown-linux-gnu.a

# CGO builder for components that need Espresso crypto linking
FROM alpine:3.20 AS op-cgo-builder
# Install dependencies
RUN apk add musl-dev gcc go g++ curl tar gzip make gcc linux-headers git jq bash yq
# Install just from mise
COPY ./mise.toml .
RUN curl -L https://github.com/casey/just/releases/download/$(yq '.tools.just' mise.toml)/just-$(yq '.tools.just' mise.toml)-x86_64-unknown-linux-musl.tar.gz | \
  tar xz -C /usr/local/bin just
# Go sources
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
# Copy rust libs for dynamic linking
COPY --from=rust-builder /libespresso/* /lib
# Warm-up the cache
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download
COPY . /app

# Build op-node
FROM op-cgo-builder AS op-node-builder
ARG OP_NODE_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build cd op-node && \
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -a -ldflags '-extldflags "-static"' \
  -o bin/op-node ./cmd/main.go

# Build op-batcher
FROM op-cgo-builder AS op-batcher-builder
ARG OP_BATCHER_VERSION=v0.0.0
WORKDIR /app/op-batcher
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_BATCHER_VERSION"
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build just op-batcher

# Build op-proposer
FROM builder AS op-proposer-builder
ARG OP_PROPOSER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build cd op-proposer && make op-proposer  \
  GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_PROPOSER_VERSION"

# Final runtime images
FROM $TARGET_BASE_IMAGE AS op-node-target
RUN apt-get update && apt-get install -y gcc && rm -rf /var/lib/apt/lists/*
ENV AZTEC_SRS_PATH /aztec/kzg10-aztec20-srs-1048584.bin
ADD "https://github.com/EspressoSystems/ark-srs/releases/download/v0.2.0/kzg10-aztec20-srs-1048584.bin" /aztec/kzg10-aztec20-srs-1048584.bin
COPY --from=op-node-builder /app/op-node/bin/op-node /usr/local/bin/

# Create config directory
RUN mkdir -p /config

CMD ["op-node"]

FROM $TARGET_BASE_IMAGE AS op-batcher-target
RUN apt-get update && apt-get install -y gcc && rm -rf /var/lib/apt/lists/*
ENV AZTEC_SRS_PATH /aztec/kzg10-aztec20-srs-1048584.bin
ADD "https://github.com/EspressoSystems/ark-srs/releases/download/v0.2.0/kzg10-aztec20-srs-1048584.bin" /aztec/kzg10-aztec20-srs-1048584.bin
COPY --from=op-batcher-builder /app/op-batcher/bin/op-batcher /usr/local/bin/
CMD ["op-batcher"]

FROM $TARGET_BASE_IMAGE AS op-proposer-target
COPY --from=op-proposer-builder /app/op-proposer/bin/op-proposer /usr/local/bin/
CMD ["op-proposer"]
