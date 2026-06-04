FROM cimg/base:2026.03

LABEL org.opencontainers.image.title="optimism ci-base clang"
LABEL org.opencontainers.image.description="CircleCI base image with clang, llvm-dev, libclang-dev, and sccache preinstalled for Rust bindgen jobs"
LABEL org.opencontainers.image.source="https://github.com/ethereum-optimism/optimism"

# Pinned sccache release, preinstalled so Rust CI jobs don't install it per-job.
ARG SCCACHE_VERSION=0.9.1
ARG TARGETARCH

USER root

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends \
      build-essential \
      clang \
      llvm \
      llvm-dev \
      libclang-dev \
      libsqlite3-dev \
      libtss2-dev \
      pkg-config \
 && rm -rf /var/lib/apt/lists/*

# Install sccache (statically linked musl build works on glibc cimg/base).
RUN set -eux; \
    case "${TARGETARCH:-amd64}" in \
      amd64) ARCH_TAG="x86_64-unknown-linux-musl" ;; \
      arm64) ARCH_TAG="aarch64-unknown-linux-musl" ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    ASSET="sccache-v${SCCACHE_VERSION}-${ARCH_TAG}"; \
    curl -fsSL "https://github.com/mozilla/sccache/releases/download/v${SCCACHE_VERSION}/${ASSET}.tar.gz" -o /tmp/sccache.tgz; \
    tar -xzf /tmp/sccache.tgz -C /tmp; \
    install "/tmp/${ASSET}/sccache" /usr/local/bin/sccache; \
    rm -rf /tmp/sccache.tgz "/tmp/${ASSET}"; \
    sccache --version

USER circleci
