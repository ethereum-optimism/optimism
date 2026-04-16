#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Build the ci-base-clang image locally or push it with docker buildx.

Usage:
  ./ops/docker/ci-base-clang/build.sh [options]

Options:
  --image-repo <repo>    Image repository.
                         Default: ghcr.io/ethereum-optimism/ci-base
  --tag <tag>            Image tag. May be provided multiple times.
                         Default: 2026.03-clang
  --platforms <list>     Comma-separated build platforms.
                         Default: linux/amd64
  --push                 Push the image instead of loading it locally.
  --no-load              Do not load the image into the local docker daemon.
  --no-pull              Do not refresh the base image before building.
  --no-verify            Skip post-build verification.
  --provenance           Enable build provenance attestation.
  --sbom                 Enable SBOM attestation.
  -h, --help             Show this help.

Examples:
  ./ops/docker/ci-base-clang/build.sh
  ./ops/docker/ci-base-clang/build.sh --platforms linux/amd64,linux/arm64 --push
  ./ops/docker/ci-base-clang/build.sh --tag 2026.03-clang --tag 2026.03-clang-$(date -u +%Y%m%d) --push
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
PLATFORMS="linux/amd64"
PUSH=false
LOAD=true
PULL=true
VERIFY=true
PROVENANCE=false
SBOM=false
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
    --push)
      PUSH=true
      LOAD=false
      shift
      ;;
    --no-load)
      LOAD=false
      shift
      ;;
    --no-pull)
      PULL=false
      shift
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
if ! docker buildx version >/dev/null 2>&1; then
  echo "error: docker buildx is required" >&2
  exit 1
fi

if [[ "$LOAD" == true && "$PLATFORMS" == *,* ]]; then
  echo "error: --load only supports a single platform. Use --push or pass one platform." >&2
  exit 1
fi

BUILD_ARGS=(
  buildx build
  --file "$SCRIPT_DIR/Dockerfile"
  --platform "$PLATFORMS"
)

if [[ "$PULL" == true ]]; then
  BUILD_ARGS+=(--pull)
fi
if [[ "$PUSH" == true ]]; then
  BUILD_ARGS+=(--push)
elif [[ "$LOAD" == true ]]; then
  BUILD_ARGS+=(--load)
fi
if [[ "$PROVENANCE" == true ]]; then
  BUILD_ARGS+=(--provenance=true)
else
  BUILD_ARGS+=(--provenance=false)
fi
if [[ "$SBOM" == true ]]; then
  BUILD_ARGS+=(--sbom=true)
fi

for tag in "${TAGS[@]}"; do
  BUILD_ARGS+=(--tag "$IMAGE_REPO:$tag")
done

BUILD_ARGS+=("$SCRIPT_DIR")

echo "Building ${IMAGE_REPO}:${TAGS[0]}"
echo "Platforms: $PLATFORMS"
docker "${BUILD_ARGS[@]}"

if [[ "$VERIFY" == true && "$PUSH" == false && "$LOAD" == true ]]; then
  IMAGE_REF="$IMAGE_REPO:${TAGS[0]}"
  echo "Verifying $IMAGE_REF"
  docker run --rm --entrypoint bash "$IMAGE_REF" -lc '
    set -euo pipefail
    whoami
    clang --version
    llvm-config --version
    dpkg-query -W clang llvm-dev libclang-dev
  '
fi

echo "Done"
