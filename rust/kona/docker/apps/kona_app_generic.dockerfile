ARG REPO_LOCATION

################################
#   Dependency Installation    #
#            Stage             #
################################
FROM ubuntu:22.04 AS dep-setup-stage
SHELL ["/bin/bash", "-c"]

# Install deps
RUN apt-get -o Acquire::Retries=8 update && apt-get -o Acquire::Retries=8 install -y --no-install-recommends \
  build-essential \
  git \
  curl \
  ca-certificates \
  libssl-dev \
  clang \
  pkg-config

# Install rust
ENV RUST_VERSION=1.94
RUN curl https://sh.rustup.rs -sSf --retry 5 --retry-all-errors --retry-delay 2 | bash -s -- -y --default-toolchain ${RUST_VERSION} --profile minimal
ENV PATH="/root/.cargo/bin:${PATH}"

# Install cargo-binstall
RUN curl -L --proto '=https' --tlsv1.2 -sSf --retry 5 --retry-all-errors --retry-delay 2 https://raw.githubusercontent.com/cargo-bins/cargo-binstall/main/install-from-binstall-release.sh | bash

RUN cargo binstall cargo-chef cargo-auditable -y

################################
#    Local Repo Setup Stage    #
################################
FROM dep-setup-stage AS app-local-setup-stage

# Copy in the local workspace repository
COPY . /workspace

# Pull in the NUT bundle JSONs from an additional named build context. The
# kona-hardforks build.rs walks ancestors of CARGO_MANIFEST_DIR looking for an
# op-core/ sibling; placing the bundles at /workspace/op-core/nuts/bundles
# satisfies that walk without widening the primary rust/ context.
COPY --from=nuts-bundles / /workspace/op-core/nuts/bundles

################################
#   Remote Repo Setup Stage    #
################################
FROM dep-setup-stage AS app-remote-setup-stage
SHELL ["/bin/bash", "-c"]

ARG TAG
ARG REPOSITORY

# Clone kona at the specified tag. op-core is preserved alongside rust so the
# kona-hardforks build.rs ancestor walk finds the NUT bundles.
RUN git clone https://github.com/${REPOSITORY} repo && \
  cd repo && \
  git checkout "${TAG}" && \
  mv rust /workspace && \
  mv op-core /workspace/op-core

################################
#       App Build Stage        #
################################
FROM app-${REPO_LOCATION}-setup-stage AS app-setup

# We need a separate entrypoint to take advantage of docker's cache.
# If we didn't do this, the full build would be triggered every time the source code changes.
FROM dep-setup-stage AS build-entrypoint
ARG BIN_TARGET
ARG BUILD_PROFILE

WORKDIR /app

FROM build-entrypoint AS planner
# Triggers a cache invalidation if `app-setup` is modified.
COPY --from=app-setup /workspace .
RUN cargo chef prepare --recipe-path recipe.json

FROM build-entrypoint AS builder
# Since we only copy recipe.json, if the dependencies don't change, this step and the next one will be cached.
COPY --from=planner /app/recipe.json recipe.json

# Build dependencies - this is the caching Docker layer!
RUN RUSTFLAGS="-C target-cpu=generic" cargo chef cook --bin "${BIN_TARGET}" --locked --profile "${BUILD_PROFILE}" --recipe-path recipe.json

# Build metadata for the version string, read at compile time via `option_env!`
# in kona-node (kona/bin/node/src/version.rs). Declared here — after the
# `cargo chef cook` dependency layer — so that a new commit invalidates only the
# app build below, not the cached dependency layer. Only the kona-node bake
# target passes these; other apps built from this shared Dockerfile leave them
# empty (and don't read them). GIT_VERSION is the release tag and the source of
# truth for the reported version.
ARG GIT_VERSION=""
ENV GIT_VERSION=$GIT_VERSION
ARG GIT_COMMIT=""
ENV GIT_COMMIT=$GIT_COMMIT
ARG GIT_DATE=""
ENV GIT_DATE=$GIT_DATE
ENV BUILD_PROFILE=$BUILD_PROFILE

# Build application. This step will systematically trigger a cache invalidation if the source code changes.
COPY --from=app-setup /workspace .
# Build the application binary on the selected tag. Since we build the external dependencies in the previous step,
# this step will reuse the target directory from the previous step.
RUN RUSTFLAGS="-C target-cpu=generic" cargo auditable build --bin "${BIN_TARGET}" --locked --profile "${BUILD_PROFILE}"

# Export stage
FROM chainguard/wolfi-base:latest AS export-stage

ARG BIN_TARGET
ARG BUILD_PROFILE

# Fixed non-root user/group for runtime
ARG UID=10001
ARG GID=10001

# Install ca-certificates, openssl, libstdc++ for TLS + C++ runtime support.
# apk has no built-in download retry; loop so a transient CDN drop doesn't flake CI (~5 min budget).
RUN n=0; until apk add --no-cache ca-certificates openssl libstdc++ bash shadow; do n=$((n+1)); [ "$n" -ge 15 ] && exit 1; echo "apk add retry $n/15 in 20s" >&2; sleep 20; done

RUN update-ca-certificates

# Create non-root runtime user
RUN groupadd --gid ${GID} app \
 && useradd  --uid ${UID} --gid ${GID} \
            --home-dir /home/app --create-home \
            --shell /usr/sbin/nologin \
            app

# Copy in the binary from the build image.
COPY --from=builder "/app/target/${BUILD_PROFILE}/${BIN_TARGET}" "/usr/local/bin/${BIN_TARGET}"

# Copy in the entrypoint script.
COPY ./kona/docker/apps/entrypoint.sh /entrypoint.sh

# Ensure the entrypoint and binary are executable and readable by the non-root user
RUN chmod 0555 "/usr/local/bin/${BIN_TARGET}" \
 && chmod 0555 /entrypoint.sh

# Export the binary name to the environment.
ENV BIN_TARGET="${BIN_TARGET}"

# Drop privileges
USER ${UID}:${GID}

ENTRYPOINT [ "/entrypoint.sh" ]
