FROM cimg/base:2026.03

LABEL org.opencontainers.image.title="optimism ci-base clang"
LABEL org.opencontainers.image.description="CircleCI base image with clang, llvm-dev, and libclang-dev preinstalled for Rust bindgen jobs"
LABEL org.opencontainers.image.source="https://github.com/ethereum-optimism/optimism"

USER root

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get upgrade -y \
 && apt-get install -y --no-install-recommends \
      clang \
      llvm-dev \
      libclang-dev \
 && rm -rf /var/lib/apt/lists/*

USER circleci
