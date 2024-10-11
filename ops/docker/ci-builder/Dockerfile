# Copy docker buildx in order to generate the absolute prestate
# in the CI pipeline for reproducible fault proof builds
FROM --platform=linux/amd64 docker as buildx
COPY --from=docker/buildx-bin /buildx /usr/libexec/docker/cli-plugins/docker-buildx
RUN docker buildx version

FROM --platform=linux/amd64 debian:bullseye-slim as rust-build

SHELL ["/bin/bash", "-c"]

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
  apt-get install -y build-essential git clang lld curl jq

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs > rustup.sh && \
  chmod +x ./rustup.sh && \
  sh rustup.sh -y

# Install nightly toolchain
RUN source $HOME/.profile && rustup update nightly

RUN source $HOME/.profile && cargo install just
RUN source $HOME/.profile && cargo install svm-rs

# Only diff from upstream docker image is this clone instead
# of COPY. We select a specific commit to use.
COPY ./versions.json ./versions.json
COPY ./ops/scripts/install-foundry.sh ./install-foundry.sh

RUN curl -L https://foundry.paradigm.xyz | bash
RUN source $HOME/.profile && ./install-foundry.sh

RUN strip /root/.foundry/bin/forge && \
  strip /root/.foundry/bin/cast && \
  strip /root/.foundry/bin/anvil && \
  strip /root/.cargo/bin/svm && \
  strip /root/.cargo/bin/just

FROM --platform=linux/amd64 debian:bullseye-slim as go-build

RUN apt-get update && apt-get install -y curl ca-certificates jq binutils

ENV GO_VERSION=1.22.7

# Fetch go manually, rather than using a Go base image, so we can copy the installation into the final stage
RUN curl -sL https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz -o go$GO_VERSION.linux-amd64.tar.gz && \
  tar -C /usr/local/ -xzvf go$GO_VERSION.linux-amd64.tar.gz

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

# Install the specific version of abigen and geth from version control
COPY ./versions.json ./versions.json
RUN go install github.com/ethereum/go-ethereum/cmd/abigen@$(jq -r .abigen < versions.json)
RUN go install github.com/ethereum/go-ethereum/cmd/geth@$(jq -r .geth < versions.json)

RUN go install gotest.tools/gotestsum@v1.12.0
RUN go install github.com/vektra/mockery/v2@v2.46.0
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
RUN go install github.com/mikefarah/yq/v4@v4.44.3

# Strip binaries to reduce size
RUN strip /go/bin/gotestsum && \
  strip /go/bin/mockery && \
  strip /go/bin/golangci-lint && \
  strip /go/bin/abigen && \
  strip /go/bin/geth && \
  strip /go/bin/yq

FROM --platform=linux/amd64 debian:bullseye-slim as base-builder

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
ENV PATH=/root/.cargo/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive

# copy the go installation, but not the module cache (cache will get stale, and would add a lot of weight)
COPY --from=go-build /usr/local/go /usr/local/go

# copy tools
COPY --from=go-build /go/bin/gotestsum /go/bin/gotestsum
COPY --from=go-build /go/bin/mockery /go/bin/mockery
COPY --from=go-build /go/bin/golangci-lint /go/bin/golangci-lint
COPY --from=go-build /go/bin/abigen /usr/local/bin/abigen
COPY --from=go-build /go/bin/geth /usr/local/bin/geth
COPY --from=go-build /go/bin/yq /go/bin/yq

# copy tools
COPY --from=rust-build /root/.foundry/bin/forge /usr/local/bin/forge
COPY --from=rust-build /root/.foundry/bin/cast /usr/local/bin/cast
COPY --from=rust-build /root/.foundry/bin/anvil /usr/local/bin/anvil
COPY --from=rust-build /root/.cargo/bin/svm /usr/local/bin/svm
COPY --from=rust-build /root/.cargo/bin/just /usr/local/bin/just

COPY ./versions.json ./versions.json

RUN /bin/sh -c set -eux; \
  apt-get update; \
  apt-get install -y --no-install-recommends bash curl openssh-client git build-essential ca-certificates jq gnupg binutils-mips-linux-gnu python3 python3-pip python3-setuptools; \
  mkdir -p /etc/apt/keyrings; \
  curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg; \
  chmod a+r /etc/apt/keyrings/docker.gpg; \
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null; \
  apt-get update; \
  apt-get install -y docker-ce-cli; \
  ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt; \
  pip install capstone pyelftools; \
  pip install semgrep==$(jq -r .semgrep < versions.json); \
  curl -fLSs https://raw.githubusercontent.com/CircleCI-Public/circleci-cli/master/install.sh | bash; \
  apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false; \
  rm -rf /var/lib/apt/lists/*; \
  rm -rf /root/.cache/pip;

RUN svm install 0.8.25 && \
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

# Copy docker buildx
COPY --from=buildx /usr/libexec/docker/cli-plugins/docker-buildx /usr/libexec/docker/cli-plugins/docker-buildx

# within docker use bash
SHELL ["/bin/bash", "-c"]

# set env to use bash
ENV SHELL=/bin/bash
ENV BASH=/bin/bash

ENTRYPOINT ["/bin/bash", "-c"]

FROM base-builder as rust-builder

# Install clang & lld
RUN apt-get update && apt-get install -y clang lld

# Copy the rust installation, alongside the installed toolchains
COPY --from=rust-build /root/.cargo /root/.cargo
COPY --from=rust-build /root/.rustup /root/.rustup

# copy the rust installation, alongside the installed toolchains
COPY --from=rust-build /root/.cargo/bin /root/.cargo/bin
COPY --from=rust-build /root/.rustup /root/.rustup
