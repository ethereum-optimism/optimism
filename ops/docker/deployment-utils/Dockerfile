FROM golang:1.26.4-bookworm AS go-base

# go install fetches modules with no built-in retry; loop so a transient module proxy drop doesn't flake CI.
RUN n=0; until go install github.com/tomwright/dasel/v2/cmd/dasel@master; do n=$((n+1)); [ "$n" -ge 5 ] && exit 1; echo "go install retry $n/5 in 10s" >&2; sleep 10; done

FROM debian:12.7-slim AS base

SHELL ["/bin/bash", "-c"]

ENV PATH=/root/.cargo/bin:/root/.foundry/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive
ENV SHELL=/bin/bash

RUN apt-get -o Acquire::Retries=8 update && apt-get -o Acquire::Retries=8 install -y curl git jq build-essential

RUN curl --proto '=https' --tlsv1.2 -sSf --retry 5 --retry-all-errors --retry-delay 2 https://sh.rustup.rs > rustup.sh && \
  chmod +x ./rustup.sh && \
  sh rustup.sh -y

RUN source $HOME/.profile && rustup update nightly
RUN curl -L --retry 5 --retry-all-errors --retry-delay 2 https://foundry.paradigm.xyz | bash
RUN foundryup

FROM debian:12.7-slim

ENV PATH=/root/.cargo/bin:/root/.foundry/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get -o Acquire::Retries=8 update && apt-get -o Acquire::Retries=8 install -y bash curl jq

SHELL ["/bin/bash", "-c"]

COPY --from=base /root/.foundry/bin/forge /usr/local/bin/forge
COPY --from=base /root/.foundry/bin/cast /usr/local/bin/cast
COPY --from=base /root/.foundry/bin/anvil /usr/local/bin/anvil
COPY --from=go-base /go/bin/dasel /usr/local/bin/dasel
