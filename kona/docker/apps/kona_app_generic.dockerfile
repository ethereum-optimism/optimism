ARG REPO_LOCATION

################################
#   Dependency Installation    #
#            Stage             #
################################
FROM ubuntu:22.04 AS dep-setup-stage
SHELL ["/bin/bash", "-c"]

# Install deps
RUN apt-get update && apt-get install -y --no-install-recommends \
  build-essential \
  git \
  curl \
  ca-certificates \
  libssl-dev \
  clang \
  pkg-config

# Install rust
ENV RUST_VERSION=1.88
RUN curl https://sh.rustup.rs -sSf | bash -s -- -y --default-toolchain ${RUST_VERSION} --profile minimal
ENV PATH="/root/.cargo/bin:${PATH}"

# Install cargo-binstall
RUN curl -L --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/cargo-bins/cargo-binstall/main/install-from-binstall-release.sh | bash

RUN cargo binstall cargo-chef -y

################################
#    Local Repo Setup Stage    #
################################
FROM dep-setup-stage AS app-local-setup-stage

# Copy in the local repository
COPY . /kona

################################
#   Remote Repo Setup Stage    #
################################
FROM dep-setup-stage AS app-remote-setup-stage
SHELL ["/bin/bash", "-c"]

ARG TAG
ARG REPOSITORY

# Clone kona at the specified tag
RUN git clone https://github.com/${REPOSITORY} && \
  cd kona && \
  git checkout "${TAG}"

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
COPY --from=app-setup kona .
RUN cargo chef prepare --recipe-path recipe.json

FROM build-entrypoint AS builder
# Since we only copy recipe.json, if the dependencies don't change, this step and the next one will be cached.
COPY --from=planner /app/recipe.json recipe.json

# Build dependencies - this is the caching Docker layer!
RUN RUSTFLAGS="-C target-cpu=generic" cargo chef cook --bin "${BIN_TARGET}" --profile "${BUILD_PROFILE}" --recipe-path recipe.json

# Build application. This step will systematically trigger a cache invalidation if the source code changes.
COPY --from=app-setup kona .
# Build the application binary on the selected tag. Since we build the external dependencies in the previous step,
# this step will reuse the target directory from the previous step.
RUN RUSTFLAGS="-C target-cpu=generic" cargo build --bin "${BIN_TARGET}" --profile "${BUILD_PROFILE}"

# Export stage
FROM ubuntu:22.04 AS export-stage
SHELL ["/bin/bash", "-c"]

ARG BIN_TARGET
ARG BUILD_PROFILE

# Install ca-certificates and libssl-dev for TLS support.
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  libssl-dev

# Copy in the binary from the build image.
COPY --from=builder "app/target/${BUILD_PROFILE}/${BIN_TARGET}" "/usr/local/bin/${BIN_TARGET}"

# Copy in the entrypoint script.
COPY ./docker/apps/entrypoint.sh /entrypoint.sh

# Export the binary name to the environment.
ENV BIN_TARGET="${BIN_TARGET}"
ENTRYPOINT [ "/entrypoint.sh" ]
