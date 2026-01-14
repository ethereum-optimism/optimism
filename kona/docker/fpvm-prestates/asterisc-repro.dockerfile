################################################################
#                Build Asterisc @ `ASTERISC_TAG`               #
################################################################

FROM ubuntu:22.04 AS asterisc-build
SHELL ["/bin/bash", "-c"]

ARG TARGETARCH
ARG ASTERISC_TAG

# Install deps
RUN apt-get update && apt-get install -y --no-install-recommends git curl ca-certificates make

ENV GO_VERSION=1.22.7

# Fetch go manually, rather than using a Go base image, so we can copy the installation into the final stage
RUN curl -sL https://go.dev/dl/go$GO_VERSION.linux-$TARGETARCH.tar.gz -o go$GO_VERSION.linux-$TARGETARCH.tar.gz && \
  tar -C /usr/local/ -xzf go$GO_VERSION.linux-$TARGETARCH.tar.gz
ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

# Clone and build Asterisc @ `ASTERISC_TAG`
RUN git clone https://github.com/ethereum-optimism/asterisc && \
  cd asterisc && \
  git checkout $ASTERISC_TAG && \
  make && \
  cp rvgo/bin/asterisc /asterisc-bin

################################################################
#               Build kona-client @ `CLIENT_TAG`               #
################################################################

FROM ghcr.io/op-rs/kona/asterisc-builder:0.3.0 AS client-build
SHELL ["/bin/bash", "-c"]

ARG CLIENT_BIN
ARG CLIENT_TAG

# Install deps
RUN apt-get update && apt-get install -y --no-install-recommends git

# Clone kona at the specified tag
RUN git clone https://github.com/op-rs/kona

# Build kona-client on the selected tag
RUN cd kona && \
  git checkout $CLIENT_TAG && \
  cargo build -Zbuild-std=core,alloc -p kona-client --bin $CLIENT_BIN --locked --profile release-client-lto && \
  mv ./target/riscv64imac-unknown-none-elf/release-client-lto/$CLIENT_BIN /kona-client-elf

################################################################
#      Create `prestate.bin.gz` + `prestate-proof.json`        #
################################################################

FROM ubuntu:22.04 AS prestate-build
SHELL ["/bin/bash", "-c"]

# Set env
ENV ASTERISC_BIN_PATH="/asterisc"
ENV CLIENT_BIN_PATH="/kona-client-elf"
ENV PRESTATE_OUT_PATH="/prestate.bin.gz"
ENV PROOF_OUT_PATH="/prestate-proof.json"

# Copy asterisc binary
COPY --from=asterisc-build /asterisc-bin $ASTERISC_BIN_PATH

# Copy kona-client binary
COPY --from=client-build /kona-client-elf $CLIENT_BIN_PATH

# Create `prestate.bin.gz`
RUN $ASTERISC_BIN_PATH load-elf \
  --path=$CLIENT_BIN_PATH \
  --out=$PRESTATE_OUT_PATH

# Create `prestate-proof.json`
RUN $ASTERISC_BIN_PATH run \
  --proof-at "=0" \
  --stop-at "=1" \
  --input $PRESTATE_OUT_PATH \
  --meta ./meta.json \
  --proof-fmt "./%d.json" \
  --output "" && \
  mv 0.json $PROOF_OUT_PATH

################################################################
#                       Export Artifacts                       #
################################################################

FROM scratch AS export-stage

COPY --from=prestate-build /asterisc .
COPY --from=prestate-build /kona-client-elf .
COPY --from=prestate-build /prestate.bin.gz .
COPY --from=prestate-build /prestate-proof.json .
COPY --from=prestate-build /meta.json .
