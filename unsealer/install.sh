#!/usr/bin/env sh
set -e

#
# Installs the given HashiCorp tool, verifying checksums and GPG signatures. Exits
# non-zero on failure.
#
# Usage:
#   install.sh terraform 0.11.5
#
# Requirements:
#   - gpg, with hashicorp key trusted
#   - curl
#   - sha256sum


NAME="$1"
if [ -z "$NAME" ]; then
  echo "Missing NAME"
  exit 1
fi

VERSION="$2"
if [ -z "$VERSION" ]; then
  echo "Missing VERSION"
  exit
fi

OS="$3"
if [ -z "$OS" ]; then
  OS="darwin"
fi

ARCH="$4"
if [ -z "$ARCH" ]; then
  ARCH="amd64"
fi

# Trust HashiCorp PGP key

gpg --import hashicorp.pgp

DOWNLOAD_ROOT="https://releases.hashicorp.com/${NAME}/${VERSION}/${NAME}_${VERSION}"
DOWNLOAD_ZIP="${DOWNLOAD_ROOT}_${OS}_${ARCH}.zip"
DOWNLOAD_SHA="${DOWNLOAD_ROOT}_SHA256SUMS"
DOWNLOAD_SIG="${DOWNLOAD_ROOT}_SHA256SUMS.sig"

echo "==> Installing ${NAME} v${VERSION}"

echo "--> Downloading SHASUM and SHASUM signatures"
curl -sfSO "${DOWNLOAD_SHA}"
curl -sfSO "${DOWNLOAD_SIG}"

echo "--> Verifying signatures file"
gpg --verify "${NAME}_${VERSION}_SHA256SUMS.sig" "${NAME}_${VERSION}_SHA256SUMS"

echo "--> Downloading ${NAME} v${VERSION} (${OS}/${ARCH})"
curl -sfSO "${DOWNLOAD_ZIP}"

echo "--> Validating SHA256SUM"
grep "${NAME}_${VERSION}_${OS}_${ARCH}.zip" "${NAME}_${VERSION}_SHA256SUMS" > "SHA256SUMS"
sha256sum -c "SHA256SUMS"

echo "--> Unpacking and installing"
unzip "${NAME}_${VERSION}_${OS}_${ARCH}.zip"
mv "${NAME}" "/usr/local/bin/${NAME}"
chmod +x "/usr/local/bin/${NAME}"

echo "--> Removing temporary files"
rm "${NAME}_${VERSION}_${OS}_${ARCH}.zip"
rm "${NAME}_${VERSION}_SHA256SUMS"
rm "${NAME}_${VERSION}_SHA256SUMS.sig"
rm SHA256SUMS

echo "--> Done!"