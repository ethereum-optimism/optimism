#!/usr/bin/env bash
# Build a locally verifiable bundle from one clean source revision. No upload.
set -euo pipefail
repo_root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
cd "$repo_root"
if [[ -n $(git status --porcelain) ]]; then
  echo "Commit or preserve working changes before building a versioned bundle." >&2
  exit 1
fi
revision=$(git rev-parse HEAD)
bundle_root=${1:-"$repo_root/.devnet-bin/private-interop-bundle/$revision"}
rust_profile=${RUST_PROFILE:-dev}
case "$rust_profile" in
  dev) rust_output=debug ;;
  release) rust_output=release ;;
  *) echo "RUST_PROFILE must be dev or release" >&2; exit 1 ;;
esac
if [[ -e "$bundle_root" ]]; then
  echo "Bundle destination already exists: $bundle_root" >&2
  exit 1
fi
mkdir -p "$bundle_root"
bundle_root=$(cd "$bundle_root" && pwd)
for component in op-node op-batcher op-supernode; do
  mise exec -- go build -trimpath -o "$bundle_root/$component" "./$component/cmd"
done
mise exec -- go build -trimpath -o "$bundle_root/interop-smoke" ./op-chain-ops/cmd/interop-smoke
mise exec -- go build -trimpath -o "$bundle_root/private-genesis" ./op-private-interop/cmd/genesis
(
  cd rust
  mise exec -- cargo build --locked --profile "$rust_profile" -p op-reth --bin op-reth
  cp "${CARGO_TARGET_DIR:-target}/$rust_output/op-reth" "$bundle_root/op-reth"
)
mise exec -- forge build --root packages/contracts-bedrock
# op-deployer expects forge-artifacts at the archive root.
tar --sort=name --mtime="@$(git show -s --format=%ct HEAD)" --owner=0 --group=0 --numeric-owner \
  -czf "$bundle_root/contracts.tar.gz" -C packages/contracts-bedrock forge-artifacts
if [[ $(git rev-parse HEAD) != "$revision" || -n $(git status --porcelain) ]]; then
  echo "Source changed during the build; bundle is not ready for deployment." >&2
  exit 1
fi
python3 - "$bundle_root" "$revision" "$rust_profile" <<'PY'
import hashlib, json, pathlib, platform, subprocess, sys
root, revision, profile = pathlib.Path(sys.argv[1]), sys.argv[2], sys.argv[3]
def version(args):
    return subprocess.check_output(['mise', 'exec', '--', *args], text=True).splitlines()[0]
manifest = {
    'revision': revision,
    'rust_profile': profile,
    'host': {'os': platform.system(), 'arch': platform.machine()},
    'toolchain': {'go': version(['go', 'version']), 'rust': version(['rustc', '--version']), 'forge': version(['forge', '--version'])},
    'files': {},
}
for path in sorted(root.iterdir()):
    if path.is_file():
        with path.open('rb') as source:
            digest = hashlib.file_digest(source, 'sha256').hexdigest()
        manifest['files'][path.name] = {'sha256': digest, 'bytes': path.stat().st_size}
(root / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
print('Bundle ready:', root)
PY
