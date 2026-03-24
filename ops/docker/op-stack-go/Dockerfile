# automatically set by buildkit, can be changed with --platform flag
# see https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
# TARGETOS
# TARGETARCH
# TARGETPLATFORM
# BUILDPLATFORM

# All target images use this as base image, and add the final build results.
# It will default to the target platform.
ARG TARGET_BASE_IMAGE=alpine:3.20

# The ubuntu target base image is used for the op-challenger build with kona.
ARG UBUNTU_TARGET_BASE_IMAGE=ubuntu:24.04

# builder_foundry does not need to be built on $BUILDPLATFORM, as foundry produces static binaries.
FROM alpine:3.21 AS builder_foundry

# Build args are only-available in top-level by default, so we'll need to declare it inside the stage too
ARG TARGETARCH

RUN apk add --no-cache curl bash git jq

COPY ./op-deployer/pkg/deployer/forge/version.json /tmp/op-deployer-versions.json

# Verify forge checksums
RUN <<-EOF
    #!/bin/bash
    set -e

    FOUNDRY_VERSION="$(jq -r '.forge' /tmp/op-deployer-versions.json)"
    FOUNDRY_PLATFORM=alpine_${TARGETARCH}
    FOUNDRY_URL="https://github.com/foundry-rs/foundry/releases/download/${FOUNDRY_VERSION}/foundry_${FOUNDRY_VERSION}_${FOUNDRY_PLATFORM}.tar.gz"
    DOWNLOAD_PATH="/tmp/forge.tar.gz"

    # Download the artifact archive
    curl -fL -o ${DOWNLOAD_PATH} "${FOUNDRY_URL}"

    # Ensure the checksum matches our expectations
    ACTUAL_SHA="$(sha256sum "${DOWNLOAD_PATH}" | awk '{print $1}')"
    EXPECTED_SHA="$(jq -r --arg arch "${FOUNDRY_PLATFORM}" '.checksums[$arch]' /tmp/op-deployer-versions.json)"

    [ "${ACTUAL_SHA}" == "${EXPECTED_SHA}" ] || (echo "Forge checksum mismatch: Expected $EXPECTED_SHA, got $ACTUAL_SHA" && exit 1)

    # Extract the archive
    tar -xzf "${DOWNLOAD_PATH}" -C /usr/local/bin forge
EOF

# Display installed version for visibility
RUN forge --version

# We may be cross-building for another platform. Specify which platform we need as builder.
FROM --platform=$BUILDPLATFORM golang:1.24.13-alpine3.22 AS builder

RUN apk add --no-cache curl tar gzip make gcc musl-dev linux-headers git jq yq-go bash just

ARG TARGETARCH

# We copy the go.mod/sum first, so the `go mod download` does not have to re-run if dependencies do not change.
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN echo "go mod cache: $(go env GOMODCACHE)"
RUN echo "go build cache: $(go env GOCACHE)"

# warm-up the cache
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download

# NOTE: the Dockerfile.dockerignore file effectively describes all dependencies
COPY . /app

# We avoid copying the full .git dir into the build for just some metadata.
# Instead, specify:
# --build-arg GIT_COMMIT=$(git rev-parse HEAD)
# --build-arg GIT_DATE=$(git show -s --format='%ct')
ARG GIT_COMMIT
ARG GIT_DATE

ARG TARGETOS

# Show just version
RUN just --version

# separate docker-builds:
# - proxyd
# - any JS/TS/smart-contract builds

# Build the Go services, utilizing caches and share the many common packages.
# The "id" defaults to the value of "target", the cache will thus be reused during this build.
# "sharing" defaults to "shared", the cache will thus be available to other concurrent docker builds.

FROM --platform=$BUILDPLATFORM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon:v1.0.0 AS cannon-builder-v1-0-0
FROM --platform=$BUILDPLATFORM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon:v1.2.0 AS cannon-builder-v1-2-0
FROM --platform=$BUILDPLATFORM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon:v1.3.0 AS cannon-builder-v1-3-0
FROM --platform=$BUILDPLATFORM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon:v1.4.0 AS cannon-builder-v1-4-0
FROM --platform=$BUILDPLATFORM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon:v1.6.0 AS cannon-builder-v1-6-0

FROM --platform=$BUILDPLATFORM builder AS cannon-builder
ARG CANNON_VERSION=v0.0.0
# Copy cannon binaries from previous versions
COPY --from=cannon-builder-v1-0-0 /usr/local/bin/cannon ./cannon/multicannon/embeds/cannon-0
COPY --from=cannon-builder-v1-3-0 /usr/local/bin/cannon-1 ./cannon/multicannon/embeds/cannon-1
COPY --from=cannon-builder-v1-4-0 /usr/local/bin/cannon-2 ./cannon/multicannon/embeds/cannon-2
COPY --from=cannon-builder-v1-2-0 /usr/local/bin/cannon-3 ./cannon/multicannon/embeds/cannon-3
COPY --from=cannon-builder-v1-3-0 /usr/local/bin/cannon-4 ./cannon/multicannon/embeds/cannon-4
COPY --from=cannon-builder-v1-4-0 /usr/local/bin/cannon-5 ./cannon/multicannon/embeds/cannon-5
COPY --from=cannon-builder-v1-4-0 /usr/local/bin/cannon-6 ./cannon/multicannon/embeds/cannon-6
COPY --from=cannon-builder-v1-6-0 /usr/local/bin/cannon-7 ./cannon/multicannon/embeds/cannon-7
# Build current binaries
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd cannon && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$CANNON_VERSION" just cannon

FROM --platform=$BUILDPLATFORM builder AS op-program-builder
ARG OP_PROGRAM_VERSION=v0.0.0
# note: we only build the host, that's all the user needs. No Go MIPS cross-build in docker
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-program && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_PROGRAM_VERSION" just op-program-host

FROM --platform=$BUILDPLATFORM builder AS op-wheel-builder
ARG OP_WHEEL_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-wheel && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_WHEEL_VERSION" just op-wheel

FROM --platform=$BUILDPLATFORM builder AS op-node-builder
ARG OP_NODE_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-node && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_NODE_VERSION" just op-node

FROM --platform=$BUILDPLATFORM builder AS op-challenger-builder
ARG OP_CHALLENGER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-challenger && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_CHALLENGER_VERSION" just op-challenger

FROM --platform=$BUILDPLATFORM builder AS op-dispute-mon-builder
ARG OP_DISPUTE_MON_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-dispute-mon && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_DISPUTE_MON_VERSION" just op-dispute-mon

FROM --platform=$BUILDPLATFORM builder AS op-batcher-builder
ARG OP_BATCHER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-batcher && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_BATCHER_VERSION" just op-batcher

FROM --platform=$BUILDPLATFORM builder AS op-proposer-builder
ARG OP_PROPOSER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-proposer && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_PROPOSER_VERSION" just op-proposer

FROM --platform=$BUILDPLATFORM builder AS op-conductor-builder
ARG OP_CONDUCTOR_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-conductor && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_CONDUCTOR_VERSION" just op-conductor

FROM --platform=$BUILDPLATFORM builder AS da-server-builder
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-alt-da && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE just da-server

FROM --platform=$BUILDPLATFORM builder AS op-supervisor-builder
ARG OP_SUPERVISOR_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-supervisor && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_SUPERVISOR_VERSION" just op-supervisor

FROM --platform=$BUILDPLATFORM builder AS op-supernode-builder
ARG OP_SUPERNODE_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-supernode && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_SUPERNODE_VERSION" just op-supernode

FROM --platform=$BUILDPLATFORM builder AS op-interop-filter-builder
ARG OP_INTEROP_FILTER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-interop-filter && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_INTEROP_FILTER_VERSION" just op-interop-filter

FROM --platform=$BUILDPLATFORM builder AS op-test-sequencer-builder
ARG OP_TEST_SEQUENCER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-test-sequencer && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_TEST_SEQUENCER_VERSION" just op-test-sequencer

# Compile contracts in a glibc-based image (solc requires glibc, crashes on Alpine musl).
# Uses BUILDPLATFORM so it runs natively (no QEMU). Contract artifacts are platform-independent.
FROM --platform=$BUILDPLATFORM golang:1.24.13-bookworm AS op-deployer-contracts
ARG BUILDARCH
RUN apt-get update && apt-get install -y --no-install-recommends curl jq git && rm -rf /var/lib/apt/lists/*
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . /app
# Install forge for the build platform (linux/glibc variant)
RUN <<-EOF
    set -e
    FOUNDRY_VERSION="$(jq -r '.forge' op-deployer/pkg/deployer/forge/version.json)"
    FOUNDRY_PLATFORM=linux_${BUILDARCH}
    FOUNDRY_URL="https://github.com/foundry-rs/foundry/releases/download/${FOUNDRY_VERSION}/foundry_${FOUNDRY_VERSION}_${FOUNDRY_PLATFORM}.tar.gz"
    DOWNLOAD_PATH="/tmp/forge.tar.gz"
    curl -fL -o "${DOWNLOAD_PATH}" "${FOUNDRY_URL}"
    ACTUAL_SHA="$(sha256sum "${DOWNLOAD_PATH}" | awk '{print $1}')"
    EXPECTED_SHA="$(jq -r --arg arch "${FOUNDRY_PLATFORM}" '.checksums[$arch]' op-deployer/pkg/deployer/forge/version.json)"
    [ "${ACTUAL_SHA}" = "${EXPECTED_SHA}" ] || (echo "Forge checksum mismatch: Expected $EXPECTED_SHA, got $ACTUAL_SHA" && exit 1)
    tar -xzf "${DOWNLOAD_PATH}" -C /usr/local/bin forge
    rm "${DOWNLOAD_PATH}"
EOF
# Compile contracts and create artifacts tarball
RUN cd packages/contracts-bedrock && forge build --skip "/**/test/**"
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  go run ./op-deployer/pkg/deployer/artifacts/cmd/mktar \
    -base ./packages/contracts-bedrock \
    -out ./op-deployer/pkg/deployer/artifacts/forge-artifacts/artifacts.tzst \
    --exclude="*.t.sol"

FROM --platform=$BUILDPLATFORM builder AS op-deployer-builder
ARG OP_DEPLOYER_VERSION=v0.0.0
# Copy pre-compiled contract artifacts from the glibc-based contracts stage
COPY --from=op-deployer-contracts /app/op-deployer/pkg/deployer/artifacts/forge-artifacts/artifacts.tzst op-deployer/pkg/deployer/artifacts/forge-artifacts/artifacts.tzst
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  just GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_DEPLOYER_VERSION" op-deployer/build-go

FROM --platform=$BUILDPLATFORM builder AS op-dripper-builder
ARG OP_DRIPPER_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  cd op-dripper && GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_DRIPPER_VERSION" just op-dripper

FROM --platform=$BUILDPLATFORM builder AS op-faucet-builder
ARG OP_FAUCET_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_FAUCET_VERSION" \
  just op-faucet/op-faucet

FROM --platform=$BUILDPLATFORM builder AS op-interop-mon-builder
ARG OP_INTEROP_MON_VERSION=v0.0.0
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  GOOS=$TARGETOS GOARCH=$TARGETARCH GITCOMMIT=$GIT_COMMIT GITDATE=$GIT_DATE VERSION="$OP_INTEROP_MON_VERSION" \
  just op-interop-mon/op-interop-mon

# The Rust version must match rust/rust-toolchain.toml. We don't use "latest" to ensure reproducibility
FROM --platform=$BUILDPLATFORM rust:1.92 AS kona-host-builder
ARG TARGETARCH
# Install build dependencies and cross-compilation toolchains
RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  libssl-dev \
  clang \
  pkg-config \
  gcc-aarch64-linux-gnu \
  gcc-x86-64-linux-gnu \
  && rm -rf /var/lib/apt/lists/*
# Copy the entire rust workspace since kona depends on sibling crates (op-alloy, alloy-op-evm, etc.)
COPY ./rust /rust
WORKDIR /rust
# Map TARGETARCH to Rust target triple, install it, and build
RUN --mount=type=cache,target=/usr/local/cargo/registry --mount=type=cache,target=/rust/target \
  case "$TARGETARCH" in \
    amd64) RUST_TARGET=x86_64-unknown-linux-gnu ;; \
    arm64) RUST_TARGET=aarch64-unknown-linux-gnu ;; \
    *) echo "Unsupported architecture: $TARGETARCH" && exit 1 ;; \
  esac && \
  rustup target add "$RUST_TARGET" && \
  RUSTFLAGS="-C target-cpu=generic" cargo build --bin kona-host --profile release-perf --target "$RUST_TARGET" && \
  cp "/rust/target/$RUST_TARGET/release-perf/kona-host" /kona-host

FROM $TARGET_BASE_IMAGE AS cannon-target
COPY --from=cannon-builder /app/cannon/bin/cannon /usr/local/bin/
COPY --from=cannon-builder /app/cannon/multicannon/embeds/* /usr/local/bin/
CMD ["cannon"]

FROM $TARGET_BASE_IMAGE AS op-program-target
COPY --from=op-program-builder /app/op-program/bin/op-program /usr/local/bin/
CMD ["op-program"]

FROM $TARGET_BASE_IMAGE AS op-wheel-target
COPY --from=op-wheel-builder /app/op-wheel/bin/op-wheel /usr/local/bin/
CMD ["op-wheel"]

FROM $TARGET_BASE_IMAGE AS op-node-target
COPY --from=op-node-builder /app/op-node/bin/op-node /usr/local/bin/
CMD ["op-node"]

# Also produce an op-challenger loaded with kona using ubuntu
FROM $UBUNTU_TARGET_BASE_IMAGE AS op-challenger-target
RUN apt-get update && apt-get install -y --no-install-recommends musl openssl ca-certificates
COPY --from=op-challenger-builder /app/op-challenger/bin/op-challenger /usr/local/bin/
# Copy in op-program and cannon
COPY --from=op-program-builder /app/op-program/bin/op-program /usr/local/bin/
ENV OP_CHALLENGER_CANNON_SERVER=/usr/local/bin/op-program
COPY --from=cannon-builder /app/cannon/bin/cannon /usr/local/bin/
ENV OP_CHALLENGER_CANNON_BIN=/usr/local/bin/cannon
# Copy in kona-host (built from local kona/ source)
COPY --from=kona-host-builder /kona-host /usr/local/bin/kona-host
ENV OP_CHALLENGER_ASTERISC_KONA_SERVER=/usr/local/bin/kona-host
ENV OP_CHALLENGER_CANNON_KONA_SERVER=/usr/local/bin/kona-host
CMD ["op-challenger"]

FROM $TARGET_BASE_IMAGE AS op-dispute-mon-target
COPY --from=op-dispute-mon-builder /app/op-dispute-mon/bin/op-dispute-mon /usr/local/bin/
CMD ["op-dispute-mon"]

FROM $TARGET_BASE_IMAGE AS op-batcher-target
COPY --from=op-batcher-builder /app/op-batcher/bin/op-batcher /usr/local/bin/
CMD ["op-batcher"]

FROM $TARGET_BASE_IMAGE AS op-proposer-target
COPY --from=op-proposer-builder /app/op-proposer/bin/op-proposer /usr/local/bin/
CMD ["op-proposer"]

FROM $TARGET_BASE_IMAGE AS op-conductor-target
COPY --from=op-conductor-builder /app/op-conductor/bin/op-conductor /usr/local/bin/
CMD ["op-conductor"]

FROM $TARGET_BASE_IMAGE AS da-server-target
COPY --from=da-server-builder /app/op-alt-da/bin/da-server /usr/local/bin/
CMD ["da-server"]

FROM $TARGET_BASE_IMAGE AS op-supervisor-target
COPY --from=op-supervisor-builder /app/op-supervisor/bin/op-supervisor /usr/local/bin/
CMD ["op-supervisor"]

FROM $TARGET_BASE_IMAGE AS op-supernode-target
COPY --from=op-supernode-builder /app/op-supernode/bin/op-supernode /usr/local/bin/
CMD ["op-supernode"]

FROM $TARGET_BASE_IMAGE AS op-interop-filter-target
COPY --from=op-interop-filter-builder /app/op-interop-filter/bin/op-interop-filter /usr/local/bin/
CMD ["op-interop-filter"]

FROM $TARGET_BASE_IMAGE AS op-test-sequencer-target
COPY --from=op-test-sequencer-builder /app/op-test-sequencer/bin/op-test-sequencer /usr/local/bin/
CMD ["op-test-sequencer"]

FROM $TARGET_BASE_IMAGE AS op-deployer-target
RUN apk add --no-cache ca-certificates
COPY --from=builder_foundry /usr/local/bin/forge /usr/local/bin/
COPY --from=op-deployer-builder /app/op-deployer/bin/op-deployer /usr/local/bin/
ENV FORGE_ENV=alpine
CMD ["op-deployer"]

FROM $TARGET_BASE_IMAGE AS op-dripper-target
COPY --from=op-dripper-builder /app/op-dripper/bin/op-dripper /usr/local/bin/
CMD ["op-dripper"]

FROM $TARGET_BASE_IMAGE AS op-faucet-target
COPY --from=op-faucet-builder /app/op-faucet/bin/op-faucet /usr/local/bin/
CMD ["op-faucet"]

FROM $TARGET_BASE_IMAGE AS op-interop-mon-target
COPY --from=op-interop-mon-builder /app/op-interop-mon/bin/op-interop-mon /usr/local/bin/
CMD ["op-interop-mon"]
