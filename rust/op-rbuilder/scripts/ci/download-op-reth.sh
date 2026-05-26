#!/bin/bash

VERSION=${VERSION:-1.3.12}

ARCH=$(uname -m)
case $ARCH in
    "x86_64")
        if [[ "$(uname)" == "Darwin" ]]; then
            ARCH_STRING="x86_64-apple-darwin"
        else
            ARCH_STRING="x86_64-unknown-linux-gnu"
        fi
        ;;
    "arm64" | "aarch64")
        if [[ "$(uname)" == "Darwin" ]]; then
            ARCH_STRING="aarch64-apple-darwin"
        else
            ARCH_STRING="aarch64-unknown-linux-gnu"
        fi
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

FILENAME="op-reth-v${VERSION}-${ARCH_STRING}.tar.gz"
echo "Downloading ${FILENAME}"

wget -q "https://github.com/paradigmxyz/reth/releases/download/v${VERSION}/${FILENAME}"

# Extract the tar.gz file
tar -xzf "${FILENAME}"

# Make the binary executable
chmod +x op-reth

# Clean up the tar.gz file (optional)
rm "${FILENAME}"
