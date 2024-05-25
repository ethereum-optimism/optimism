FROM --platform=linux/amd64 debian:bullseye-slim as rust-build

SHELL ["/bin/bash", "-c"]

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
  apt-get install -y curl build-essential git clang lld curl

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs > rustup.sh && \
  chmod +x ./rustup.sh && \
  sh rustup.sh -y

# Install nightly toolchain
RUN source $HOME/.profile && rustup update nightly

RUN source $HOME/.profile && cargo install just
RUN source $HOME/.profile && cargo install svm-rs

# Only diff from upstream docker image is this clone instead
# of COPY. We select a specific commit to use.
COPY ./.foundryrc ./.foundryrc
COPY ./ops/scripts/install-foundry.sh ./install-foundry.sh

RUN curl -L https://foundry.paradigm.xyz | bash
RUN source $HOME/.profile && ./install-foundry.sh

RUN strip /root/.foundry/bin/forge && \
  strip /root/.foundry/bin/cast && \
  strip /root/.foundry/bin/anvil

FROM --platform=linux/amd64 ghcr.io/crytic/echidna/echidna:v2.0.4 as echidna-test

FROM --platform=linux/amd64 debian:bullseye-slim as go-build

RUN apt-get update && apt-get install -y curl ca-certificates

ENV GO_VERSION=1.21.1

# Fetch go manually, rather than using a Go base image, so we can copy the installation into the final stage
RUN curl -sL https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz -o go$GO_VERSION.linux-amd64.tar.gz && \
  tar -C /usr/local/ -xzvf go$GO_VERSION.linux-amd64.tar.gz

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

# Install the specific version of abigen from .abigenrc
COPY ./.abigenrc ./.abigenrc
RUN go install github.com/ethereum/go-ethereum/cmd/abigen@$(cat .abigenrc)

COPY ./.gethrc ./.gethrc
RUN go install github.com/ethereum/go-ethereum/cmd/geth@$(cat .gethrc)

RUN go install gotest.tools/gotestsum@latest
RUN go install github.com/vektra/mockery/v2@v2.28.1
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2

FROM --platform=linux/amd64 python:3.11.4-slim-bullseye

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
ENV PATH=/root/.cargo/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive

# Create rust directories for copying the installation into
RUN mkdir /root/.cargo && mkdir /root/.cargo/bin && mkdir /root/.rustup

# copy the go installation, but not the module cache (cache will get stale, and would add a lot of weight)
COPY --from=go-build /usr/local/go /usr/local/go
# copy tools
COPY --from=go-build /go/bin/gotestsum /go/bin/gotestsum
COPY --from=go-build /go/bin/mockery /go/bin/mockery
COPY --from=go-build /go/bin/golangci-lint /go/bin/golangci-lint
COPY --from=go-build /go/bin/abigen /usr/local/bin/abigen
COPY --from=go-build /go/bin/geth /usr/local/bin/geth

# copy the rust installation, alongside the installed toolchains
COPY --from=rust-build /root/.cargo/bin /root/.cargo/bin
COPY --from=rust-build /root/.rustup /root/.rustup
# copy tools
COPY --from=rust-build /root/.foundry/bin/forge /usr/local/bin/forge
COPY --from=rust-build /root/.foundry/bin/cast /usr/local/bin/cast
COPY --from=rust-build /root/.foundry/bin/anvil /usr/local/bin/anvil

COPY --from=echidna-test /usr/local/bin/echidna-test /usr/local/bin/echidna-test

COPY .nvmrc .nvmrc

ENV NODE_MAJOR=20
ENV SLITHER_VERSION=0.10.0

# note: python3 package in apt is python 3.9, while base image already has python 3.11
RUN /bin/sh -c set -eux; \
  apt-get update; \
  apt-get install -y --no-install-recommends bash curl openssh-client git build-essential pkg-config libssl-dev clang libclang-dev ca-certificates jq gnupg binutils-mips-linux-gnu; \
  mkdir -p /etc/apt/keyrings; \
  curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg; \
  echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list; \
  curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg; \
  chmod a+r /etc/apt/keyrings/docker.gpg; \
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null; \
  apt-get update; \
  apt-get install -y nodejs docker-ce-cli; \
  ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt; \
  npm i -g depcheck; \
  pip install slither-analyzer==$SLITHER_VERSION capstone pyelftools; \
  curl -fLSs https://raw.githubusercontent.com/CircleCI-Public/circleci-cli/master/install.sh | bash; \
  apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false; \
  rm -rf /var/lib/apt/lists/*; \
  rm -rf /root/.cache/pip; \
  rm -rf /root/.cache/npm;


RUN npm i -g pnpm && npm i -g yarn@1 && pnpm --version && yarn --version

RUN svm install 0.5.17 && \
  svm install 0.8.15 && \
  svm install 0.8.19

RUN echo "downloading and verifying Codecov uploader" && \
  curl https://keybase.io/codecovsecurity/pgp_keys.asc | gpg --no-default-keyring --keyring trustedkeys.gpg --import && \
  curl -Os "https://uploader.codecov.io/latest/linux/codecov" && \
  curl -Os "https://uploader.codecov.io/latest/linux/codecov.SHA256SUM" && \
  curl -Os "https://uploader.codecov.io/latest/linux/codecov.SHA256SUM.sig" && \
  gpgv codecov.SHA256SUM.sig codecov.SHA256SUM && \
  shasum -a 256 -c codecov.SHA256SUM || sha256sum -c codecov.SHA256SUM && \
  cp codecov /usr/local/bin/codecov && \
  chmod +x /usr/local/bin/codecov  && \
  rm codecov

# within docker use bash
SHELL ["/bin/bash", "-c"]
# set env to use bash
ENV SHELL=/bin/bash
ENV BASH=/bin/bash

ENTRYPOINT ["/bin/bash", "-c"]
