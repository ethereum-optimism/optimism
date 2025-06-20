# L1 Geth Dockerfile, simplified from ops/docker/deployment-utils/Dockerfile
FROM debian:12.7-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
  curl \
  jq \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# Install Geth for the given architecture.
RUN ARCH=$(dpkg --print-architecture) && \
  echo "Detected architecture: $ARCH" && \
  case "$ARCH" in \
  "amd64") \
  GETH_URL="https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-1.15.11-36b2371c.tar.gz" && \
  GETH_SHA="a14a4285daedf75ea04a7a298e6caa48d566a2786c93fc5e86ec2c5998c92455" && \
  GETH_DIR="geth-linux-amd64-1.15.11-36b2371c" && \
  VERIFY_SHA="true" \
  ;; \
  "arm64") \
  GETH_URL="https://gethstore.blob.core.windows.net/builds/geth-linux-arm64-1.15.11-36b2371c.tar.gz" && \
  GETH_SHA="148ec84db2268fa846ae68f6445f0c98d33e95069e40fe8c74b43ea5eb53df7b" && \
  GETH_DIR="geth-linux-arm64-1.15.11-36b2371c" && \
  VERIFY_SHA="true" \
  ;; \
  *) \
  echo "Unsupported architecture: $ARCH" && exit 1 \
  ;; \
  esac && \
  echo "Downloading: $GETH_URL" && \
  curl -L "$GETH_URL" -o geth.tar.gz && \
  echo "$GETH_SHA geth.tar.gz" | sha256sum -c - && \
  tar -xvf geth.tar.gz && \
  mv "$GETH_DIR/geth" /usr/local/bin/geth && \
  rm -rf geth.tar.gz "$GETH_DIR" && \
  chmod +x /usr/local/bin/geth

# Create data directory
RUN mkdir -p /data

# Expose the RPC port
EXPOSE 8545

# Set working directory
WORKDIR /data

# Include the initialization script.
COPY l1-geth-init.sh /l1-geth-init.sh

# Run the initialization script.
RUN chmod +x /l1-geth-init.sh

# Use the initialization script as the entrypoint.
ENTRYPOINT "/l1-geth-init.sh"
