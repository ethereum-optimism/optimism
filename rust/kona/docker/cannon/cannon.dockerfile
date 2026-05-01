FROM ubuntu:22.04

ENV SHELL=/bin/bash
ENV DEBIAN_FRONTEND=noninteractive

# MIPS64 cross-compilation toolchain — the only thing that must be frozen
# in a Docker image because apt package versions are not deterministically pinnable.
# Everything else (Rust, Go, mise, just) is installed on top from pinned version sources.
RUN apt-get update && apt-get install --assume-yes --no-install-recommends \
  ca-certificates \
  build-essential \
  curl \
  g++-mips64-linux-gnuabi64 \
  libc6-dev-mips64-cross \
  binutils-mips64-linux-gnuabi64 \
  llvm \
  clang \
  make \
  cmake \
  git
