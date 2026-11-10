#!/usr/bin/env bash
# Exports op-reth, kona-node, kona-host and op-zk-proposer from the images that
# build-images.apko.yaml publishes for a develop commit (maxperf, Wolfi rustc),
# so op-acceptance-tests can run the binaries we ship instead of CI's own build.
#
# Each binary goes where the acceptance tests expect CI's build
# (rust/target/release/<name>) as a launcher that runs the image's binary with
# the image's own glibc loader and libraries, so it doesn't depend on the CI
# image's libraries.
#
# usage: fetch-shipped-rust-binaries.sh [TAG]
#   TAG is a full commit SHA and defaults to CIRCLE_SHA1. Every develop commit
#   gets images tagged with its SHA, usually 30 to 40 minutes after it lands, so
#   this waits for them.
set -euo pipefail

TAG="${1:-${CIRCLE_SHA1:?}}"
REGISTRY=us-docker.pkg.dev/oplabs-tools-artifacts/images
BINARIES=(op-reth kona-node kona-host op-zk-proposer)
WAIT_MINUTES=90
EXPORT_DIR=.circleci-cache/shipped-rust-binaries
BIN_DIR=rust/target/release

CRANE_DIR=$(mktemp -d)
GOBIN="$CRANE_DIR" go install github.com/google/go-containerregistry/cmd/crane@v0.22.1
# crane has no overall deadline, so a stalled transfer would otherwise hang the job.
crane() {
  timeout 10m "$CRANE_DIR/crane" "$@"
}

deadline=$((SECONDS + WAIT_MINUTES * 60))
for bin in "${BINARIES[@]}"; do
  until crane digest "$REGISTRY/$bin:$TAG" > /dev/null; do
    if ((SECONDS > deadline)); then
      echo "No $REGISTRY/$bin:$TAG after $WAIT_MINUTES minutes. Did build-images.apko.yaml fail for $TAG, or is the registry unreachable?" >&2
      exit 1
    fi
    echo "Waiting for $REGISTRY/$bin:$TAG"
    sleep 60
  done
done

mkdir -p "$BIN_DIR"
for bin in "${BINARIES[@]}"; do
  # Pin the digest so the export, the log and the binary all match.
  ref="$REGISTRY/$bin@$(crane digest "$REGISTRY/$bin:$TAG")"
  revision=$(crane manifest "$ref" | jq -r '.annotations["org.opencontainers.image.revision"]')
  if [ "$revision" != "$TAG" ]; then
    echo "$ref was built from $revision, not $TAG" >&2
    exit 1
  fi
  root="$PWD/$EXPORT_DIR/$bin"
  mkdir -p "$root"
  crane export --platform linux/amd64 "$ref" - | tar -xf - -C "$root" "usr/local/bin/$bin" usr/lib

  cat > "$BIN_DIR/$bin" << EOF
#!/bin/sh
# $bin from $ref
root="\$(dirname "\$0")/../../../$EXPORT_DIR/$bin"
exec "\$root/usr/lib/ld-linux-x86-64.so.2" --library-path "\$root/usr/lib" --argv0 "\$0" "\$root/usr/local/bin/$bin" "\$@"
EOF
  chmod +x "$BIN_DIR/$bin"

  echo "== $bin"
  echo "image: $ref"
  echo "revision: $revision"
  # Every library must come from the image, not the CI image.
  libs=$("$root/usr/lib/ld-linux-x86-64.so.2" --library-path "$root/usr/lib" --list "$root/usr/local/bin/$bin")
  echo "$libs"
  outside=$(echo "$libs" | grep '=> ' | grep -v "=> $root/" || true)
  if [ -n "$outside" ]; then
    echo "$bin resolves libraries outside its image: $outside" >&2
    exit 1
  fi
  "$BIN_DIR/$bin" --version
done
