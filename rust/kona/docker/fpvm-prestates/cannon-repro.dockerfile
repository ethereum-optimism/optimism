################################################################
#              Build Cannon from local monorepo                #
################################################################

FROM ubuntu:22.04 AS cannon-build
SHELL ["/bin/bash", "-c"]

ARG TARGETARCH

# Install deps
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates make

ENV GO_VERSION=1.23.8

# Fetch go manually, rather than using a Go base image, so we can copy the installation into the final stage
RUN curl -sL https://go.dev/dl/go$GO_VERSION.linux-$TARGETARCH.tar.gz -o go$GO_VERSION.linux-$TARGETARCH.tar.gz && \
  tar -C /usr/local/ -xzf go$GO_VERSION.linux-$TARGETARCH.tar.gz
ENV GOPATH=/go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

# Copy monorepo source needed for the cannon build
COPY --from=monorepo go.mod go.sum /optimism/
COPY --from=monorepo cannon/ /optimism/cannon/
COPY --from=monorepo op-service/ /optimism/op-service/
COPY --from=monorepo op-preimage/ /optimism/op-preimage/

# Build cannon from local source
RUN cd /optimism/cannon && \
  make && \
  cp bin/cannon /cannon-bin

################################################################
#            Build kona-client from local source               #
################################################################

FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon-builder:v1.0.0 AS client-build
SHELL ["/bin/bash", "-c"]

ARG CLIENT_BIN
ARG KONA_CUSTOM_CONFIGS

COPY --from=custom_configs / /usr/local/kona-custom-configs

# Copy kona source from build context
COPY . /kona

ENV KONA_CUSTOM_CONFIGS=$KONA_CUSTOM_CONFIGS
ENV KONA_CUSTOM_CONFIGS_DIR=/usr/local/kona-custom-configs

# Build kona-client
RUN cd kona && \
  cargo build -Zbuild-std=core,alloc -Zjson-target-spec -p kona-client --bin $CLIENT_BIN --locked --profile release-client-lto && \
  mv ./target/mips64-unknown-none/release-client-lto/$CLIENT_BIN /kona-client-elf

################################################################
#      Create `prestate.bin.gz` + `prestate-proof.json`        #
################################################################

FROM ubuntu:22.04 AS prestate-build
SHELL ["/bin/bash", "-c"]

ARG UID=10001
ARG GID=10001

RUN groupadd --gid ${GID} app \
 && useradd  --uid ${UID} --gid ${GID} \
            --home-dir /home/app --create-home \
            --shell /usr/sbin/nologin \
            app

# Use a writable workspace owned by the non-root user
WORKDIR /work
RUN chown ${UID}:${GID} /work

# Copy cannon binary
COPY --from=cannon-build /cannon-bin /work/cannon

# Copy kona-client binary
COPY --from=client-build /kona-client-elf /work/kona-client-elf

# Make the binaries executable
RUN chmod 0555 /work/cannon /work/kona-client-elf

USER ${UID}:${GID}

# Create `prestate.bin.gz`
RUN /work/cannon load-elf \
  --path=/work/kona-client-elf \
  --out=/work/prestate.bin.gz \
  --type multithreaded64-5

# Create `prestate-proof.json`
RUN /work/cannon run \
  --proof-at "=0" \
  --stop-at "=1" \
  --input /work/prestate.bin.gz \
  --meta /work/meta.json \
  --proof-fmt "/work/%d.json" \
  --output "" && \
  mv /work/0.json /work/prestate-proof.json

################################################################
#                       Export Artifacts                       #
################################################################

FROM scratch AS export-stage

COPY --from=prestate-build /work/cannon .
COPY --from=prestate-build /work/kona-client-elf .
COPY --from=prestate-build /work/prestate.bin.gz .
COPY --from=prestate-build /work/prestate-proof.json .
COPY --from=prestate-build /work/meta.json .
