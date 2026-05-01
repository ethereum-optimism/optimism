################################################################
#   Reproducible kona prestate build — thin environment wrapper #
#                                                               #
#   Build logic lives in rust/justfile.                         #
#   This Dockerfile provides a fixed environment and calls it.  #
################################################################

################################################################
#              Build Cannon from local monorepo                #
################################################################

FROM golang:1.24.13-alpine3.22 AS cannon-build

RUN apk add --no-cache bash just

COPY go.mod go.sum /app/
COPY cannon/ /app/cannon/
COPY op-service/ /app/op-service/
COPY op-preimage/ /app/op-preimage/
COPY justfiles/ /app/justfiles/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /app/cannon && just cannon

################################################################
#    Build kona-client ELF + generate prestate                 #
################################################################

FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/cannon-builder:v2.0.0 AS kona-build-env
SHELL ["/bin/bash", "-c"]

ARG VARIANT=kona-client

# --- Layer 1: mise + pinned toolchains ---
# mise's rust plugin bootstraps rustup at a pinned version and installs
# every rust entry from mise.toml (stable 1.94 + the dated nightly).
# All artifacts come from pinned sources — no `curl | sh` from an
# unpinned URL in the reproducible build path.
COPY ops/scripts/install_mise.sh /tmp/install_mise.sh
RUN chmod +x /tmp/install_mise.sh && /tmp/install_mise.sh
ENV PATH="/root/.local/bin:${PATH}"

# Pin rustup's CARGO_HOME and RUSTUP_HOME to standard locations so the
# cargo registry cache mount below lands where cargo actually reads it.
ENV CARGO_HOME=/root/.cargo
ENV RUSTUP_HOME=/root/.rustup

COPY mise.toml /app/mise.toml
COPY rust/rust-toolchain.toml /app/rust/rust-toolchain.toml
WORKDIR /app
RUN mise trust && mise install rust go just jq

# Rustup's cargo/rustc proxies go ahead of mise shims on PATH. Any
# cargo call then resolves to the rustup proxy directly and respects
# RUSTUP_TOOLCHAIN (set by build-kona-client-elf). If it went through
# the mise shim instead, mise would re-set RUSTUP_TOOLCHAIN to the
# active rust from mise.toml (stable 1.94) and build the wrong
# toolchain into the prestate.
ENV PATH="/root/.cargo/bin:/root/.local/share/mise/shims:${PATH}"
ENV MISE_GLOBAL_CONFIG_FILE="/app/mise.toml"

# --- Layer 2: add rust-src to the dated nightly (needed for -Zbuild-std) ---
# mise.toml's nightly entry only lists rustfmt; just install-nightly adds
# rust-src on top via the rustup that mise just bootstrapped.
COPY rust/justfile /app/rust/justfile
RUN cd /app/rust && just install-nightly

# --- Layer 3: Rust workspace source ---
COPY rust/Cargo.toml rust/Cargo.lock /app/rust/
COPY rust/.cargo/ /app/rust/.cargo/
COPY rust/kona/ /app/rust/kona/
COPY rust/op-alloy/ /app/rust/op-alloy/
COPY rust/alloy-op-evm/ /app/rust/alloy-op-evm/
COPY rust/alloy-op-hardforks/ /app/rust/alloy-op-hardforks/
COPY rust/op-revm/ /app/rust/op-revm/
# op-reth is a workspace member but not a kona-client dependency.
# We need its Cargo.toml files so the workspace resolves.
COPY rust/op-reth/ /app/rust/op-reth/

# kona-hardforks build.rs walks ancestors of CARGO_MANIFEST_DIR for
# op-core/nuts/bundles. Stage the bundles at /app/op-core so the walk
# succeeds inside the rust/-scoped build.
COPY op-core/nuts/bundles /app/op-core/nuts/bundles

# --- Layer 4: Custom chain configs ---
# Always copy from the named build context. The justfile stages an empty
# temp dir when KONA_CUSTOM_CONFIGS_DIR is unset, so this COPY succeeds in
# both modes without conditional Dockerfile logic.
ARG KONA_CUSTOM_CONFIGS=false
COPY --from=kona-custom-configs / /usr/local/kona-custom-configs
ENV KONA_CUSTOM_CONFIGS=${KONA_CUSTOM_CONFIGS}

# --- Layer 5: Build kona-client ELF ---
# build-kona-client-elf sets CARGO_BUILD_TARGET to the corrected target spec
# from the source tree, overriding cannon-builder's baked-in spec.
RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/app/rust/target \
    cd /app/rust && \
    if [ "$KONA_CUSTOM_CONFIGS" = "true" ]; then \
      export KONA_CUSTOM_CONFIGS_DIR=/usr/local/kona-custom-configs; \
    fi && \
    just build-kona-client-elf ${VARIANT} && \
    cp /app/rust/target/mips64-unknown-none/release-client-lto/${VARIANT} /app/kona-elf

################################################################
#   Generate prestate                                          #
################################################################

FROM kona-build-env AS prestate-build

COPY --from=cannon-build /app/cannon/bin/cannon /app/cannon
RUN /app/cannon load-elf \
      --path=/app/kona-elf \
      --out=/app/prestate.bin.gz \
      --type multithreaded64-5 && \
    /app/cannon run \
      --proof-at "=0" \
      --stop-at "=1" \
      --input /app/prestate.bin.gz \
      --meta /app/meta.json \
      --proof-fmt "/app/%d.json" \
      --output "" && \
    mv /app/0.json /app/prestate-proof.json

################################################################
#                       Export Artifacts                       #
################################################################

FROM scratch AS export-stage

COPY --from=prestate-build /app/prestate.bin.gz .
COPY --from=prestate-build /app/prestate-proof.json .
COPY --from=prestate-build /app/meta.json .
