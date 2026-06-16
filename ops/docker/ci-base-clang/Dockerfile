FROM cimg/base:2026.03

LABEL org.opencontainers.image.title="optimism ci-base clang"
LABEL org.opencontainers.image.description="CircleCI base image with clang, llvm-dev, libclang-dev, and sccache preinstalled for Rust bindgen jobs"
LABEL org.opencontainers.image.source="https://github.com/ethereum-optimism/optimism"

# Pinned sccache release, preinstalled so Rust CI jobs don't install it per-job.
# Checksums are the upstream .tar.gz.sha256 sidecars for v0.9.1; the build fails
# if the downloaded asset does not match (content pin, not just version pin).
ARG SCCACHE_VERSION=0.9.1
ARG SCCACHE_SHA256_amd64=911f611db54e48dc50c32232462a99848824be5ba0f6f52cc33903712cd78715
ARG SCCACHE_SHA256_arm64=a0f10f7daeeecb3439d6e903bc5dc4a4ccee67985b506d3daf742521efe45627
ARG TARGETARCH

USER root

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends \
      clang \
      llvm-dev \
      libclang-dev \
      mold \
 && apt clean \
 && rm -rf /var/lib/apt/lists/*

# Install sccache (statically linked musl build works on glibc cimg/base).
RUN set -eux; \
    case "${TARGETARCH:-amd64}" in \
      amd64) ARCH_TAG="x86_64-unknown-linux-musl";  EXPECTED_SHA256="${SCCACHE_SHA256_amd64}" ;; \
      arm64) ARCH_TAG="aarch64-unknown-linux-musl"; EXPECTED_SHA256="${SCCACHE_SHA256_arm64}" ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    ASSET="sccache-v${SCCACHE_VERSION}-${ARCH_TAG}"; \
    curl -fsSL "https://github.com/mozilla/sccache/releases/download/v${SCCACHE_VERSION}/${ASSET}.tar.gz" -o /tmp/sccache.tgz; \
    echo "${EXPECTED_SHA256}  /tmp/sccache.tgz" | sha256sum -c -; \
    tar -xzf /tmp/sccache.tgz -C /tmp; \
    install "/tmp/${ASSET}/sccache" /usr/local/bin/sccache; \
    rm -rf /tmp/sccache.tgz "/tmp/${ASSET}"; \
    sccache --version

USER circleci
