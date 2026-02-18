FROM ubuntu:22.04

ENV SHELL=/bin/bash
ENV DEBIAN_FRONTEND=noninteractive

# todo: pin `nightly` version
ENV RUST_VERSION=nightly

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

# Install Rustup and Rust
RUN curl https://sh.rustup.rs -sSf | bash -s -- -y --default-toolchain ${RUST_VERSION} --component rust-src
ENV PATH="/root/.cargo/bin:${PATH}"

# Add the special cannon build target
COPY ./mips64-unknown-none.json .

# Set up the env vars to instruct rustc to use the correct compiler and linker
# and to build correctly to support the Cannon processor
ENV CC_mips64_unknown_none=mips64-linux-gnuabi64-gcc \
  CXX_mips64_unknown_none=mips64-linux-gnuabi64-g++ \
  CARGO_TARGET_MIPS64_UNKNOWN_NONE_LINKER=mips64-linux-gnuabi64-gcc \
  RUSTFLAGS="-Clink-arg=-e_start -Cllvm-args=-mno-check-zero-division" \
  CARGO_BUILD_TARGET="/mips64-unknown-none.json" \
  RUSTUP_TOOLCHAIN=${RUST_VERSION}
