FROM --platform=linux/amd64 ubuntu:22.04

ENV SHELL=/bin/bash
ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install --assume-yes --no-install-recommends \
  ca-certificates \
  build-essential \
  curl \
  g++-mips-linux-gnu \
  libc6-dev-mips-cross \
  binutils-mips-linux-gnu \
  llvm \
  clang \
  python3 \
  python3-pip \ 
  xxd

RUN python3 -m pip install capstone pyelftools
