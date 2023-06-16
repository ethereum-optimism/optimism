FROM debian:bullseye-20220822-slim as foundry-build

SHELL ["/bin/bash", "-c"]

WORKDIR /opt

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y curl build-essential git && \
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs > rustup.sh && \
    chmod +x ./rustup.sh && \
    ./rustup.sh -y

WORKDIR /opt/foundry

# Only diff from upstream docker image is this clone instead
# of COPY. We select a specific commit to use.
RUN git clone https://github.com/foundry-rs/foundry.git . \
    && git checkout da2392e58bb8a7fefeba46b40c4df1afad8ccd22

RUN source $HOME/.profile && \
    cargo build --release && \
    strip /opt/foundry/target/release/forge && \
    strip /opt/foundry/target/release/cast && \
    strip /opt/foundry/target/release/anvil

FROM ethereum/client-go:alltools-v1.10.25 as geth

FROM ghcr.io/crytic/echidna/echidna:v2.0.4 as echidna-test

FROM python:3.8.13-slim-bullseye

ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
ENV DEBIAN_FRONTEND=noninteractive

COPY --from=foundry-build /opt/foundry/target/release/forge /usr/local/bin/forge
COPY --from=foundry-build /opt/foundry/target/release/cast /usr/local/bin/cast
COPY --from=foundry-build /opt/foundry/target/release/anvil /usr/local/bin/anvil
COPY --from=geth /usr/local/bin/abigen /usr/local/bin/abigen
COPY --from=echidna-test /usr/local/bin/echidna-test /usr/local/bin/echidna-test
COPY check-changed.sh /usr/local/bin/check-changed

RUN apt-get update && \
  apt-get install -y bash curl openssh-client git build-essential ca-certificates jq musl gnupg coreutils && \
  curl -sL https://deb.nodesource.com/setup_16.x -o nodesource_setup.sh && \
  curl -sL https://go.dev/dl/go1.19.linux-amd64.tar.gz -o go1.19.linux-amd64.tar.gz && \
  tar -C /usr/local/ -xzvf go1.19.linux-amd64.tar.gz && \
  ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt && \
  bash nodesource_setup.sh && \
  apt-get install -y nodejs && \
  npm i -g npm@8.11.0 \
  npm i -g yarn && \
  npm i -g depcheck && \
  pip install slither-analyzer==0.9.3 && \
  go install gotest.tools/gotestsum@latest && \
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.48.0 && \
  curl -fLSs https://raw.githubusercontent.com/CircleCI-Public/circleci-cli/master/install.sh | bash && \
  chmod +x /usr/local/bin/check-changed

RUN echo "downloading solidity compilers" && \
  curl -o solc-linux-amd64-v0.5.17+commit.d19bba13 -sL https://binaries.soliditylang.org/linux-amd64/solc-linux-amd64-v0.5.17+commit.d19bba13 && \
  curl -o solc-linux-amd64-v0.8.9+commit.e5eed63a -sL https://binaries.soliditylang.org/linux-amd64/solc-linux-amd64-v0.8.9+commit.e5eed63a  && \
  curl -o solc-linux-amd64-v0.8.10+commit.fc410830 -sL https://binaries.soliditylang.org/linux-amd64/solc-linux-amd64-v0.8.10+commit.fc410830 && \
  curl -o solc-linux-amd64-v0.8.12+commit.f00d7308 -sL https://binaries.soliditylang.org/linux-amd64/solc-linux-amd64-v0.8.12+commit.f00d7308 && \
  echo "verifying checksums" && \
  (echo "c35ce7a4d3ffa5747c178b1e24c8541b2e5d8a82c1db3719eb4433a1f19e16f3 solc-linux-amd64-v0.5.17+commit.d19bba13" | sha256sum --check --status - || exit 1) && \
  (echo "f851f11fad37496baabaf8d6cb5c057ca0d9754fddb7a351ab580d7fd728cb94 solc-linux-amd64-v0.8.9+commit.e5eed63a"  | sha256sum --check --status - || exit 1) && \
  (echo "c7effacf28b9d64495f81b75228fbf4266ac0ec87e8f1adc489ddd8a4dd06d89 solc-linux-amd64-v0.8.10+commit.fc410830" | sha256sum --check --status - || exit 1) && \
  (echo "556c3ec44faf8ff6b67933fa8a8a403abe82c978d6e581dbfec4bd07360bfbf3 solc-linux-amd64-v0.8.12+commit.f00d7308" | sha256sum --check --status - || exit 1) && \
  echo "caching compilers" && \
  mkdir -p ~/.cache/hardhat-nodejs/compilers/linux-amd64 && \
  cp solc-linux-amd64-v0.5.17+commit.d19bba13 ~/.cache/hardhat-nodejs/compilers/linux-amd64/ && \
  cp solc-linux-amd64-v0.8.9+commit.e5eed63a  ~/.cache/hardhat-nodejs/compilers/linux-amd64/ && \
  cp solc-linux-amd64-v0.8.10+commit.fc410830 ~/.cache/hardhat-nodejs/compilers/linux-amd64/ && \
  cp solc-linux-amd64-v0.8.12+commit.f00d7308 ~/.cache/hardhat-nodejs/compilers/linux-amd64/ && \
  mkdir -p ~/.svm/0.5.17 && \
  cp solc-linux-amd64-v0.5.17+commit.d19bba13 ~/.svm/0.5.17/solc-0.5.17 && \
  mkdir -p ~/.svm/0.8.9  && \
  cp solc-linux-amd64-v0.8.9+commit.e5eed63a  ~/.svm/0.8.9/solc-0.8.9   && \
  mkdir -p ~/.svm/0.8.10 && \
  cp solc-linux-amd64-v0.8.10+commit.fc410830 ~/.svm/0.8.10/solc-0.8.10 && \
  mkdir -p ~/.svm/0.8.12 && \
  cp solc-linux-amd64-v0.8.12+commit.f00d7308 ~/.svm/0.8.12/solc-0.8.12 && \
  rm solc-linux-amd64-v0.5.17+commit.d19bba13 && \
  rm solc-linux-amd64-v0.8.9+commit.e5eed63a && \
  rm solc-linux-amd64-v0.8.10+commit.fc410830 && \
  rm solc-linux-amd64-v0.8.12+commit.f00d7308

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

RUN echo "downloading mockery tool" && \
  mkdir -p mockery-tmp-dir && \
  curl -o mockery-tmp-dir/mockery.tar.gz -sL https://github.com/vektra/mockery/releases/download/v2.28.1/mockery_2.28.1_Linux_x86_64.tar.gz && \
  tar -xzvf mockery-tmp-dir/mockery.tar.gz -C mockery-tmp-dir && \
  cp mockery-tmp-dir/mockery /usr/local/bin/mockery && \
  chmod +x /usr/local/bin/mockery && \
  rm -rf mockery-tmp-dir
