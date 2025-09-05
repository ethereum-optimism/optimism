# Build stage - download and extract forge
FROM alpine:3.20 AS builder
COPY pkg/deployer/forge/version.json /tmp/version.json
RUN apk add --no-cache curl jq

ARG TARGETARCH
RUN FORGE_VERSION=$(jq -r '.forge' /tmp/version.json) && \
    echo "Downloading forge version: $FORGE_VERSION for architecture: $TARGETARCH" && \
    curl -L https://github.com/foundry-rs/foundry/releases/download/${FORGE_VERSION}/foundry_${FORGE_VERSION}_alpine_${TARGETARCH}.tar.gz -o foundry.tar.gz && \
    tar -xzf foundry.tar.gz && \
    chmod +x forge

# Final runtime stage
FROM alpine:3.20
ENTRYPOINT ["/op-deployer"]

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

COPY op-deployer /op-deployer
COPY --from=builder /forge /usr/local/bin/forge
RUN forge --version

ENV FORGE_ENV=alpine

# Symlink onto the PATH
RUN ln -s /op-deployer /usr/local/bin/op-deployer
