FROM ubuntu:22.04

ENV SHELL=/bin/bash
ENV DEBIAN_FRONTEND noninteractive

# todo: pin `nightly` version
ENV RUST_VERSION nightly

RUN apt-get update && apt-get install --assume-yes --no-install-recommends \
  ca-certificates \
  build-essential \
  curl \
  g++-riscv64-linux-gnu \
  libc6-dev-riscv64-cross \
  binutils-riscv64-linux-gnu \
  llvm \
  clang \
  make \
  cmake \
  git

# Install Rustup and Rust
RUN curl https://sh.rustup.rs -sSf | bash -s -- -y --default-toolchain ${RUST_VERSION} --component rust-src
ENV PATH="/root/.cargo/bin:${PATH}"

# Set up the env vars to instruct rustc to use the correct compiler and linker
# and to build correctly to support the Cannon processor
ENV CC_riscv64_unknown_none_elf=riscv64-linux-gnu-gcc \
  CXX_riscv64_unknown_none_elf=riscv64-linux-gnu-g++ \
  CARGO_TARGET_RISCV64_UNKNOWN_NONE_ELF_LINKER=riscv64-linux-gnu-gcc \
  RUSTFLAGS="-Clink-arg=-e_start -Ctarget-feature=-c,-zicsr,-zifencei,-zicntr,zihpm" \
  CARGO_BUILD_TARGET="riscv64imac-unknown-none-elf" \
  RUSTUP_TOOLCHAIN=${RUST_VERSION}
