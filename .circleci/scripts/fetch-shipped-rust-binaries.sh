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
#
# If an image never appears (its build failed), this skips the run without
# paging: in CI it posts to Slack (from develop only, with no mention) and
# cancels the workflow, so the gate that pages on-call never runs. A registry
# that keeps failing for any other reason still fails the job and pages.
set -euo pipefail

TAG="${1:-${CIRCLE_SHA1:?}}"
REPO=ethereum-optimism/optimism
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

# A missing tag only means a failed image build if the image has tags for
# recent develop commits. A renamed or retired image has none, so it pages.
published_recently() {
  git log --first-parent -n 20 --format=%H "$TAG" > "$tmp/recent" \
    && crane ls "$REGISTRY/$1" > "$tmp/tags" \
    && grep -qxFf "$tmp/recent" "$tmp/tags"
}

# The images for $TAG were never published. Post to Slack and cancel the
# workflow, so the gate doesn't page. If another job in the workflow failed, or
# a step here fails, exit 1 and let the gate page instead.
skip_run() {
  printf 'Circle-Token: %s\n' "${CIRCLE_API_TOKEN:?}" \
    | curl -sf --max-time 30 -H @- "https://circleci.com/api/v2/workflow/${CIRCLE_WORKFLOW_ID:?}/job" > "$tmp/jobs"
  if jq -e '[.items[] | select(.status | test("^(failed|infrastructure_fail|timedout)$"))] | length > 0' "$tmp/jobs" > /dev/null; then
    echo "Other jobs in this workflow failed, so the gate will page." >&2
    exit 1
  fi
  text="No shipped $1 image for $TAG after $WAIT_MINUTES minutes, so the nightly shipped-binary run was skipped. Check build-images.apko.yaml for that commit: https://github.com/$REPO/commit/$TAG/checks"
  if [ "${CIRCLE_BRANCH:-}" = develop ]; then
    resp=$(printf 'Authorization: Bearer %s\n' "${SLACK_ACCESS_TOKEN:?}" \
      | curl -s --max-time 30 -H @- https://slack.com/api/chat.postMessage \
        --data-urlencode "channel=C03N11M0BBN" \
        --data-urlencode "text=$text ${CIRCLE_BUILD_URL:-}" \
        --data-urlencode "unfurl_links=false")
    echo "Slack response: $resp"
    echo "$resp" | jq -e .ok > /dev/null
  else
    echo "Would post to Slack (only develop posts): $text ${CIRCLE_BUILD_URL:-}"
  fi
  printf 'Circle-Token: %s\n' "$CIRCLE_API_TOKEN" \
    | curl -sf --max-time 30 -X POST -H @- "https://circleci.com/api/v2/workflow/${CIRCLE_WORKFLOW_ID:?}/cancel"
  echo
  # Cancelling the workflow stops this job too.
  sleep 300
  echo "The workflow is still running after the cancel request." >&2
  exit 1
}

digests=()
tmp=$(mktemp -d)
deadline=$((SECONDS + WAIT_MINUTES * 60))
for bin in "${BINARIES[@]}"; do
  until digest=$(crane digest "$REGISTRY/$bin:$TAG" 2> "$tmp/err"); do
    cat "$tmp/err" >&2
    if ((SECONDS > deadline)); then
      # The registry answers MANIFEST_UNKNOWN for a tag it doesn't have. Any
      # other error (5xx, timeout, network) could hide a real problem, so it pages.
      if grep -q MANIFEST_UNKNOWN "$tmp/err" && published_recently "$bin"; then
        echo "No $REGISTRY/$bin:$TAG after $WAIT_MINUTES minutes." >&2
        skip_run "$bin"
      fi
      echo "Could not get $REGISTRY/$bin:$TAG after $WAIT_MINUTES minutes. The registry keeps failing, or recent commits have no $bin image either." >&2
      exit 1
    fi
    echo "Waiting for $REGISTRY/$bin:$TAG"
    sleep 60
  done
  digests+=("$digest")
done

mkdir -p "$BIN_DIR"
for i in "${!BINARIES[@]}"; do
  bin=${BINARIES[$i]}
  # Pin the digest so the export, the log and the binary all match.
  ref="$REGISTRY/$bin@${digests[$i]}"
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
  # The loader and every library must come from the image, not the CI image.
  # Resolve symlinks first: an absolute link in the image's usr/lib would
  # point into the CI image.
  libs=$("$root/usr/lib/ld-linux-x86-64.so.2" --library-path "$root/usr/lib" --list "$root/usr/local/bin/$bin")
  echo "$libs"
  outside=$(
    {
      echo "$root/usr/lib/ld-linux-x86-64.so.2"
      echo "$libs" | awk '/=> /{print $3}'
    } | while read -r path; do
      [[ $(readlink -f "$path") == "$root"/* ]] || echo "$path"
    done
  )
  if [ -n "$outside" ]; then
    echo "$bin resolves libraries outside its image: $outside" >&2
    exit 1
  fi
  "$BIN_DIR/$bin" --version
done
