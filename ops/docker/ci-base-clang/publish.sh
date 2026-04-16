#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Publish the ci-base-clang image to GHCR.

Usage:
  ./ops/docker/ci-base-clang/publish.sh [options]

Options:
  --image-repo <repo>    Image repository.
                         Default: ghcr.io/ethereum-optimism/ci-base
  --tag <tag>            Image tag. May be provided multiple times.
                         Default: 2026.03-clang
  --platforms <list>     Comma-separated build platforms.
                         Default: linux/amd64,linux/arm64
  --username <name>      GHCR username. Defaults to GHCR_USERNAME env var.
  --token-env <name>     Env var containing the GHCR token.
                         Default: GHCR_TOKEN
  --no-verify            Skip pulling and verifying the published image.
  --provenance           Enable build provenance attestation.
  --sbom                 Enable SBOM attestation.
  -h, --help             Show this help.

Required token scopes:
  - write:packages

Example:
  export GHCR_USERNAME=your-user
  export GHCR_TOKEN=ghp_xxx
  ./ops/docker/ci-base-clang/publish.sh \
    --tag 2026.03-clang \
    --tag 2026.03-clang-$(date -u +%Y%m%d)
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_REPO="ghcr.io/ethereum-optimism/ci-base"
PLATFORMS="linux/amd64,linux/arm64"
VERIFY=true
PROVENANCE=false
SBOM=false
USERNAME="${GHCR_USERNAME:-}"
TOKEN_ENV="GHCR_TOKEN"
TAGS=("2026.03-clang")

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image-repo)
      IMAGE_REPO="$2"
      shift 2
      ;;
    --tag)
      if [[ "${TAGS[*]}" == "2026.03-clang" ]]; then
        TAGS=()
      fi
      TAGS+=("$2")
      shift 2
      ;;
    --platforms)
      PLATFORMS="$2"
      shift 2
      ;;
    --username)
      USERNAME="$2"
      shift 2
      ;;
    --token-env)
      TOKEN_ENV="$2"
      shift 2
      ;;
    --no-verify)
      VERIFY=false
      shift
      ;;
    --provenance)
      PROVENANCE=true
      shift
      ;;
    --sbom)
      SBOM=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_cmd docker
TOKEN="${!TOKEN_ENV:-}"

if [[ -z "$USERNAME" ]]; then
  echo "error: set GHCR_USERNAME or pass --username" >&2
  exit 1
fi
if [[ -z "$TOKEN" ]]; then
  echo "error: set $TOKEN_ENV with a token that has write:packages" >&2
  exit 1
fi

echo "$TOKEN" | docker login ghcr.io -u "$USERNAME" --password-stdin

BUILD_CMD=(
  "$SCRIPT_DIR/build.sh"
  --image-repo "$IMAGE_REPO"
  --platforms "$PLATFORMS"
  --push
  --no-verify
)

if [[ "$PROVENANCE" == true ]]; then
  BUILD_CMD+=(--provenance)
fi
if [[ "$SBOM" == true ]]; then
  BUILD_CMD+=(--sbom)
fi
for tag in "${TAGS[@]}"; do
  BUILD_CMD+=(--tag "$tag")
done

"${BUILD_CMD[@]}"

if [[ "$VERIFY" == true ]]; then
  IMAGE_REF="$IMAGE_REPO:${TAGS[0]}"
  echo "Pulling and verifying $IMAGE_REF"
  docker pull "$IMAGE_REF"
  docker run --rm --entrypoint bash "$IMAGE_REF" -lc '
    set -euo pipefail
    whoami
    clang --version
    llvm-config --version
    dpkg-query -W clang llvm-dev libclang-dev
  '
fi

echo "Published ${IMAGE_REPO}:${TAGS[0]}"
