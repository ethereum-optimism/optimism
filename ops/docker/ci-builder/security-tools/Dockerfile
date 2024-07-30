# Use the previously built ci-builder image as the base
FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/ci-builder:latest as security-tools-builder

# Set NVM_DIR and ensure it exists
ENV NVM_DIR /usr/local/nvm
RUN mkdir -p $NVM_DIR

# Install nvm
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash

RUN /bin/sh -c set -eux; \
  source $NVM_DIR/nvm.sh; \
    nvm install 16; \
    nvm install 20; \
    nvm use 20; \
    nvm --version

