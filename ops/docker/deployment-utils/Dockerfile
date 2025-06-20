FROM golang:1.22.7-bookworm AS go-base

RUN go install github.com/tomwright/dasel/v2/cmd/dasel@e96f281f05a1c1f01ed251f7041dfd0f49325b1a

FROM debian:12.7-slim AS base

SHELL ["/bin/bash", "-c"]

ENV PATH=/root/.cargo/bin:/root/.foundry/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive
ENV SHELL=/bin/bash

RUN apt-get update && apt-get install -y curl git jq build-essential

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs > rustup.sh && \
  chmod +x ./rustup.sh && \
  sh rustup.sh -y

RUN source $HOME/.profile && rustup update nightly
RUN curl -L https://foundry.paradigm.xyz | bash
RUN foundryup

FROM debian:12.7-slim

ENV PATH=/root/.cargo/bin:/root/.foundry/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y bash curl jq

SHELL ["/bin/bash", "-c"]

COPY --from=base /root/.foundry/bin/forge /usr/local/bin/forge
COPY --from=base /root/.foundry/bin/cast /usr/local/bin/cast
COPY --from=base /root/.foundry/bin/anvil /usr/local/bin/anvil
COPY --from=go-base /go/bin/dasel /usr/local/bin/dasel

# Install Geth for the given architecture.
RUN ARCH=$(dpkg --print-architecture) && \
  echo "Detected architecture: $ARCH" && \
  case "$ARCH" in \
      "amd64") \
          GETH_URL="https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-1.15.11-36b2371c.tar.gz" && \
          GETH_SHA="a14a4285daedf75ea04a7a298e6caa48d566a2786c93fc5e86ec2c5998c92455" && \
          GETH_DIR="geth-linux-amd64-1.15.11-36b2371c" && \
          VERIFY_SHA="true" \
          ;; \
      "arm64") \
          GETH_URL="https://gethstore.blob.core.windows.net/builds/geth-linux-arm64-1.15.11-36b2371c.tar.gz" && \
          GETH_SHA="148ec84db2268fa846ae68f6445f0c98d33e95069e40fe8c74b43ea5eb53df7b" && \
          GETH_DIR="geth-linux-arm64-1.15.11-36b2371c" && \
          VERIFY_SHA="true" \
          ;; \
      *) \
          echo "Unsupported architecture: $ARCH" && exit 1 \
          ;; \
  esac && \
  echo "Downloading: $GETH_URL" && \
  curl -L "$GETH_URL" -o geth.tar.gz && \
  echo "$GETH_SHA geth.tar.gz" | sha256sum -c - && \
  tar -xvf geth.tar.gz && \
  mv "$GETH_DIR/geth" /usr/local/bin/geth && \
  rm -rf geth.tar.gz "$GETH_DIR" && \
  chmod +x /usr/local/bin/geth
