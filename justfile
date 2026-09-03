import 'justfiles/git.just'

BEDROCK_TAGS_REMOTE := env('BEDROCK_TAGS_REMOTE', 'origin')
OP_STACK_GO_BUILDER := env('OP_STACK_GO_BUILDER', 'us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest')
PYTHON := env('PYTHON', 'python3')

TEST_TIMEOUT := env('TEST_TIMEOUT', '10m')

# Go test runs cover every package in the module except these, which run in
# dedicated CI jobs or can't run in the standard go-tests environment:
#   op-acceptance-tests             dedicated acceptance-test job (needs a running devnet)
#   cannon                          dedicated cannon job (slow MIPS emulation tests)
#   rust                            rust-e2e pipeline (needs prebuilt Rust binaries)
#   op-deployer/pkg/deployer/forge  fails when forge is on PATH (ethereum-optimism/optimism#21200)
# See the list-test-packages recipe, which expands `go list ./...` minus these.
EXCLUDED_TEST_PKGS := "op-acceptance-tests cannon rust op-deployer/pkg/deployer/forge"

# Fault-proof packages run in the default go-tests job and again in a dedicated
# job with Cannon enabled, so they keep a separate list.
FRAUD_PROOF_TEST_PKGS := "./op-e2e/faultproofs/..."

# Lists all available targets.
help:
  @just --list

# Install the repo's git hooks (core.hooksPath -> .githooks). Idempotent; run once per clone.
install-git-hooks:
  git config core.hooksPath .githooks
  @echo "Installed git hooks: core.hooksPath -> .githooks"

# Initializes/updates the superchain-registry submodule — the single canonical SR
# commit pin. Scoped to ONLY this submodule (never a bare `git submodule update`).
# With no ref it initializes at the pinned commit (shallow) and leaves an
# already-present checkout untouched; with a ref (tag or commit sha) it moves the
# submodule there.
[script('bash')]
update-superchain-registry-submodule ref="":
  set -euo pipefail
  # Check out the submodule at the pinned (gitlink) commit, initializing it if
  # absent and resetting a stale or dirty working tree (--force).
  git submodule update --init --force --depth 1 -- superchain-registry
  if [ -n "{{ref}}" ]; then
    # Move it to the requested ref and stage the new gitlink so the subsequent
    # (no-ref) syncs treat that as the pinned commit instead of resetting it.
    git -C superchain-registry fetch --depth 1 origin "{{ref}}"
    git -C superchain-registry checkout --detach FETCH_HEAD
    git add superchain-registry
  fi

# Builds op-core/superchain/superchain-configs.zip (gitignored) from the
# superchain-registry submodule. Lightweight; this is the recipe the Go build/test
# targets depend on. Verify mode: skips work if the existing zip already matches the
# committed .sha256, otherwise regenerates and asserts it still matches (failing on drift).
build-superchain-go: update-superchain-registry-submodule
  bash op-core/superchain/sync-superchain.sh

# Regenerates op-core/superchain/superchain-configs.zip AND rewrites its committed
# .sha256 from the submodule — build-superchain-go in refresh mode. Sibling of
# sync-superchain-rust; both are run by sync-superchain when bumping the registry.
sync-superchain-go:
  @OP_CORE_SYNC_SUPERCHAIN=1 just build-superchain-go

# Regenerates the committed Rust artifacts from the submodule: kona's etc/*.json
# (via KONA_SYNC_SUPERCHAIN) and op-reth's superchain-configs.tar.sha256 +
# chain_specs.rs (via OP_RETH_SYNC_SUPERCHAIN; the tar itself is gitignored).
sync-superchain-rust: update-superchain-registry-submodule
  cd rust && KONA_SYNC_SUPERCHAIN=true cargo build -p kona-registry
  cd rust && OP_RETH_SYNC_SUPERCHAIN=1 cargo build -p reth-optimism-chainspec --features superchain-configs

# One-command superchain-registry sync. With a ref (tag or commit sha) it moves the
# submodule there first, then regenerates every dependent artifact (Go + Rust), so
# the submodule pointer and the committed artifacts can never drift out of sync.
sync-superchain ref="": (update-superchain-registry-submodule ref) sync-superchain-go sync-superchain-rust

# Builds Go components and contracts-bedrock.
build: build-go build-contracts

# Builds main Go components.
build-go: submodules build-superchain-go op-node op-proposer op-batcher op-challenger op-dispute-mon cannon

# Builds contracts-bedrock.
build-contracts:
  cd packages/contracts-bedrock && just build

# Builds the custom linter.
build-customlint:
  cd linter && just build

# Lints Go code with specific linters.
lint-go: build-customlint build-superchain-go
  ./linter/bin/op-golangci-lint run ./...
  go mod tidy -diff

# Lints Go code with specific linters and fixes reported issues.
lint-go-fix: build-customlint build-superchain-go
  ./linter/bin/op-golangci-lint run ./... --fix

# Checks that op-geth version in go.mod is valid.
check-op-geth-version:
  go run ./ops/scripts/check-op-geth-version

# Builds Docker images for Go components using buildx.
[script('bash')]
golang-docker: update-superchain-registry-submodule
  set -euo pipefail
  GIT_COMMIT=$(git rev-parse HEAD) \
  GIT_DATE=$(git show -s --format='%ct') \
  IMAGE_TAGS=$(git rev-parse HEAD),latest \
  docker buildx bake \
      --progress plain \
      --load \
      -f docker-bake.hcl \
      op-node op-batcher op-proposer op-challenger op-dispute-mon

# Builds selected Docker image targets using buildx.
[private]
[script('bash')]
docker-bake targets: update-superchain-registry-submodule
  set -euo pipefail
  GIT_COMMIT=$(git rev-parse HEAD)
  GIT_DATE=$(git show -s --format='%ct')
  IMAGE_TAGS=${IMAGE_TAGS:-$GIT_COMMIT,latest}
  read -ra bake_targets <<< "{{targets}}"
  GIT_COMMIT="$GIT_COMMIT" \
  GIT_DATE="$GIT_DATE" \
  IMAGE_TAGS="$IMAGE_TAGS" \
  docker buildx bake \
      --progress plain \
      --load \
      -f docker-bake.hcl \
      "${bake_targets[@]}"

# Builds Docker image for op-node using buildx.
op-node-docker: (docker-bake "op-node")

# Builds Docker image for op-batcher using buildx.
op-batcher-docker: (docker-bake "op-batcher")

# Builds the requested local Docker images for op-node and op-batcher.
op-stack-go-requested-docker: (docker-bake "op-node op-batcher")

# Removes the Docker buildx builder.
docker-builder-clean:
  docker buildx rm buildx-build

# Creates a Docker buildx builder.
docker-builder:
  docker buildx create \
    --driver=docker-container --name=buildx-build --bootstrap --use

# Computes GIT_VERSION for all images and outputs JSON.
[script('bash')]
compute-git-versions:
  GIT_COMMIT=$(git rev-parse HEAD) ./ops/scripts/compute-git-versions.sh

# Builds cross-platform Docker image for op-node.
[script('bash')]
cross-op-node:
  set -euo pipefail
  GITCOMMIT=$(git rev-parse HEAD)
  tags=$(git tag --points-at "$GITCOMMIT" | grep '^op-node/' | sed 's/op-node\///' | sort -V)
  preferred_tag=$(echo "$tags" | grep -v -- '-rc' | tail -n 1)
  if [ -z "$preferred_tag" ]; then
      if [ -z "$tags" ]; then
          GIT_VERSION="untagged"
      else
          GIT_VERSION=$(echo "$tags" | tail -n 1)
      fi
  else
      GIT_VERSION="$preferred_tag"
  fi
  GIT_COMMIT="$GITCOMMIT" \
  GIT_DATE=$(git show -s --format='%ct') \
  IMAGE_TAGS=$(git rev-parse HEAD),latest \
  PLATFORMS="linux/arm64" \
  GIT_VERSION="$GIT_VERSION" \
  docker buildx bake \
      --progress plain \
      --builder=buildx-build \
      --load \
      --no-cache \
      -f docker-bake.hcl \
      op-node

# Builds Docker image for Bedrock contracts.
[script('bash')]
contracts-bedrock-docker:
  set -euo pipefail
  IMAGE_TAGS=$(git rev-parse HEAD),latest \
  docker buildx bake \
      --progress plain \
      --load \
      -f docker-bake.hcl \
      contracts-bedrock

# Updates git submodules.
submodules:
  git submodule update --init --recursive

# Builds op-node binary.
op-node:
  just ./op-node/op-node

# Generates mocks for op-node.
generate-mocks-op-node:
  cd op-node && just generate-mocks

# Generates mocks for op-service.
generate-mocks-op-service:
  cd op-service && just generate-mocks

# Builds op-batcher binary.
op-batcher:
  just ./op-batcher/op-batcher

# Builds op-proposer binary.
op-proposer:
  just ./op-proposer/op-proposer

# Builds op-challenger binary.
op-challenger:
  cd op-challenger && just op-challenger

# Builds op-dispute-mon binary.
op-dispute-mon:
  cd op-dispute-mon && just op-dispute-mon

# Builds op-supernode binary.
op-supernode:
  just ./op-supernode/op-supernode

# Builds op-interop-filter binary.
op-interop-filter:
  just ./op-interop-filter/op-interop-filter

# Builds cannon binary.
cannon:
  cd cannon && just cannon

# Builds the reproducible kona prestates (all variants).
reproducible-prestate-kona:
  cd rust && just build-kona-reproducible-prestate

# Builds the reproducible kona prestates and prints their hashes.
[script('bash')]
reproducible-prestate:
  set -euo pipefail
  (cd rust && just build-kona-reproducible-prestate)
  (cd rust && just output-kona-prestate-hash)

# Builds the kona prestates, natively when the MIPS64 cross-linker is installed and via Docker otherwise.
cannon-prestates:
  cd rust && just build-kona-prestates-auto

# Verifies the reproducibility of released cannon prestates against the
# superchain-registry standard prestates. Only kona-client/v* releases are
# rebuilt and verified; op-program prestates remain in the registry but are no
# longer re-validated.
verify-reproducibility:
  rm -rf ops/prestate-reproducibility/temp/states
  ./ops/prestate-reproducibility/build-prestates.sh
  env GO111MODULE=on go run ./ops/prestate-reproducibility/prestates/verify/verify.go --input ops/prestate-reproducibility/temp/states/versions.json

# Cleans up unused dependencies in Go modules.
# Bypasses the Go module proxy for freshly released versions.
# See https://proxy.golang.org/ for more info.
mod-tidy: build-superchain-go
  GOPRIVATE="github.com/ethereum-optimism" go mod tidy

# Removes all generated files under bin/.
clean:
  rm -rf ./bin
  cd packages/contracts-bedrock/ && forge clean

# Completely clean the project directory.
nuke: clean
  git clean -Xdf

# Runs unit tests for individual components.
test-unit:
  cd op-node && just test
  cd op-proposer && just test
  cd op-batcher && just test
  cd op-e2e && just test
  cd packages/contracts-bedrock && just test

# Runs semgrep on the entire monorepo.
semgrep:
  semgrep scan --config .semgrep/rules/ --error .

# Runs semgrep CI checks against develop baseline.
[script('bash')]
semgrep-ci:
  set -euo pipefail
  DEV_REF=$(git rev-parse develop)
  SEMGREP_REPO_NAME=ethereum-optimism/optimism semgrep ci --baseline-commit="$DEV_REF"

# Makes pre-test setup.
make-pre-test:
  cd op-e2e && just pre-test

# Lists the Go packages under test: every package in the module except those in
# EXCLUDED_TEST_PKGS (which run in dedicated CI jobs or can't run in this environment).
[script('bash')]
list-test-packages:
  set -euo pipefail
  go list -e ./... | grep -vE "^github.com/ethereum-optimism/optimism/($(echo '{{EXCLUDED_TEST_PKGS}}' | tr ' ' '|'))(/|$)"

# Runs comprehensive Go tests across all packages.
[script('bash')]
go-tests: cannon build-contracts make-pre-test build-superchain-go
  set -euo pipefail
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="false"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  go test -parallel="$PARALLEL" -timeout={{TEST_TIMEOUT}} $(just list-test-packages)

# Runs comprehensive Go tests with -short flag.
[script('bash')]
go-tests-short: cannon build-contracts make-pre-test build-superchain-go
  set -euo pipefail
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="false"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  go test -short -parallel="$PARALLEL" -timeout={{TEST_TIMEOUT}} $(just list-test-packages)

# Internal: runs Go tests with gotestsum for CI.
[script('bash')]
_go-tests-ci-internal go_test_flags="": build-superchain-go
  set -euo pipefail
  (cd cannon && just diff-hello-elf)
  echo "Setting up test directories..."
  mkdir -p ./tmp/test-results ./tmp/testlogs
  echo "Running Go tests with gotestsum..."
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="false"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  export OP_TESTLOG_FILE_LOGGER_OUTDIR=$(realpath ./tmp/testlogs)
  source ./ops/scripts/source-ci-archive-rpcs.sh
  export NAT_INTEROP_LOADTEST_TARGET=10
  export NAT_INTEROP_LOADTEST_TIMEOUT=30s
  ALL_PACKAGES="$(just list-test-packages | tr '\n' ' ')"
  if [ -n "${CIRCLE_NODE_TOTAL:-}" ] && [ "$CIRCLE_NODE_TOTAL" -gt 1 ]; then
      NODE_INDEX=${CIRCLE_NODE_INDEX:-0}
      NODE_TOTAL=${CIRCLE_NODE_TOTAL:-1}
      if [ -z "${ALL_PACKAGES// /}" ]; then
          echo "ERROR: list-test-packages produced no packages" >&2
          exit 1
      fi
      # Split by historical timing instead of round-robin: timing-balanced
      # packing keyed on the JUnit classname (= Go import path) that gotestsum
      # uploads via store_test_results. list-test-packages already emits one
      # concrete import path per line; `circleci tests split` reads
      # CIRCLE_NODE_TOTAL/INDEX itself and falls back to name-based splitting
      # for packages with no timing data yet.
      PARALLEL_PACKAGES=$(printf '%s\n' $ALL_PACKAGES \
          | circleci tests split --split-by=timings --timings-type=classname \
          | tr '\n' ' ')
      # An empty share for one node is a legitimate timing-bucketing outcome;
      # the packages run on the other nodes. Only an empty package list
      # (checked above) means the job would silently test nothing.
      if [ -z "${PARALLEL_PACKAGES// /}" ]; then
          echo "No packages assigned to node $NODE_INDEX/$NODE_TOTAL, skipping."
          exit 0
      fi
      echo "Node $NODE_INDEX/$NODE_TOTAL running packages: $PARALLEL_PACKAGES"
      ./ops/scripts/gotestsum-split.sh --format=standard-verbose \
          --junitfile=./tmp/test-results/results-"$NODE_INDEX".xml \
          --jsonfile=./tmp/testlogs/log-"$NODE_INDEX".json \
          --rerun-fails=3 \
          --rerun-fails-max-failures=50 \
          --packages="$PARALLEL_PACKAGES" \
          -- -p=4 -parallel="$PARALLEL" {{go_test_flags}} -timeout={{TEST_TIMEOUT}} -tags="ci"
  else
      ./ops/scripts/gotestsum-split.sh --format=standard-verbose \
          --junitfile=./tmp/test-results/results.xml \
          --jsonfile=./tmp/testlogs/log.json \
          --rerun-fails=3 \
          --rerun-fails-max-failures=50 \
          --packages="$ALL_PACKAGES" \
          -- -p=4 -parallel="$PARALLEL" {{go_test_flags}} -timeout={{TEST_TIMEOUT}} -tags="ci"
  fi

# Runs short Go tests with gotestsum for CI.
go-tests-short-ci:
  just _go-tests-ci-internal "-short"

# Runs comprehensive Go tests with gotestsum for CI.
go-tests-ci:
  just _go-tests-ci-internal ""

# Runs fraud proofs Go tests with gotestsum for CI.
[script('bash')]
go-tests-fraud-proofs-ci:
  set -euo pipefail
  echo "Setting up test directories..."
  mkdir -p ./tmp/test-results ./tmp/testlogs
  echo "Running Go tests with gotestsum..."
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="true"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  export OP_TESTLOG_FILE_LOGGER_OUTDIR=$(realpath ./tmp/testlogs)
  source ./ops/scripts/source-ci-archive-rpcs.sh
  export NAT_INTEROP_LOADTEST_TARGET=10
  export NAT_INTEROP_LOADTEST_TIMEOUT=30s
  ./ops/scripts/gotestsum-split.sh --format=standard-verbose \
      --junitfile=./tmp/test-results/results.xml \
      --jsonfile=./tmp/testlogs/log.json \
      --rerun-fails=3 \
      --rerun-fails-max-failures=50 \
      --packages="{{FRAUD_PROOF_TEST_PKGS}}" \
      -- -parallel="$PARALLEL" -timeout={{TEST_TIMEOUT}}

# Runs comprehensive Go tests (alias for go-tests).
test: go-tests

# Updates the Geth version used in the project.
update-op-geth:
  ./ops/scripts/update-op-geth.py

# Build all Rust binaries (release) for sysgo tests.
# Every binary needs an explicit `-p`: a bare `--bin` only resolves against the
# `default-members` of the workspace, and op-reth-sdm-fixture is a plain member.
build-rust-release:
  cd rust && cargo build --release -p kona-node --bin kona-node -p kona-host --bin kona-host -p op-reth --bin op-reth -p op-reth-sdm-fixture --bin op-reth-sdm-fixture

# Checks that locked NUT bundles have not been modified.
check-nut-locks:
  go run ./ops/scripts/check-nut-locks

# Checks that committed NUT pre-fork states regenerate without drift. Assumes required build artifacts are present.
[script('bash')]
_check-nut-prefork-states:
  set -euo pipefail
  shopt -s nullglob
  for state in op-core/nuts/state/*_state.json; do
    fork="$(basename "$state" _state.json)"
    if [ "$fork" = "jovian" ]; then
      continue
    fi
    just _nut-prefork-state-for "$fork"
  done
  git diff --exit-code -- op-core/nuts/state/*_state.json

# Checks that committed NUT pre-fork states regenerate without drift.
check-nut-prefork-states: build-contracts build-superchain-go
  just _check-nut-prefork-states

# Snapshots current-upgrade-bundle.json as a fork's NUT bundle and updates the lock file.
nut-snapshot-for fork:
  go run ./ops/scripts/nut-snapshot-for {{fork}}

# Verifies a fork's NUT bundle was correctly built from its recorded commit.
nut-provenance-verify fork:
  go run ./ops/scripts/nut-provenance-verify {{fork}}

# Generates op-core/nuts/state/<fork>_state.json (predecessor state + frozen <fork> bundle).
_nut-prefork-state-for fork:
  OP_E2E_GEN_PREFORK_STATE={{fork}} go test -count=1 -run TestGenerateForkState ./rust/kona/tests/proofs/

# Generates op-core/nuts/state/<fork>_state.json (predecessor state + frozen <fork> bundle).
nut-prefork-state-for fork: build-contracts build-superchain-go
  just _nut-prefork-state-for {{fork}}

# Checks that TODO comments have corresponding issues.
todo-checker:
  ./ops/scripts/todo-checker.sh

# Runs semgrep tests.
semgrep-test:
  semgrep scan --test --config .semgrep/rules/ .semgrep/tests/

# Runs shellcheck.
shellcheck:
  find . -type f -name '*.sh' -not -path '*/node_modules/*' -not -path './packages/contracts-bedrock/lib/*' -not -path './packages/contracts-bedrock/kout*/*' -not -path './docs/public-docs/*' -exec sh -c 'echo "Checking $1"; shellcheck "$1"' _ {} \;
  find . -type f -name '*.sh' -not -path '*/node_modules/*' -not -path './packages/contracts-bedrock/lib/*' -not -path './packages/contracts-bedrock/kout*/*' -not -path './docs/public-docs/*' -exec shfmt --diff {} \;

# Format shell scripts with shfmt.
shfmt-fix:
  find . -type f -name '*.sh' -not -path '*/node_modules/*' -not -path './packages/contracts-bedrock/lib/*' -not -path './packages/contracts-bedrock/kout*/*' -not -path './docs/public-docs/*' -exec shfmt --write {} \;

# Generates a table of contents for the README.md file.
toc:
  md_toc -p github README.md

latest-versions:
  ./ops/scripts/latest-versions.sh

# Usage:
#   just update-op-geth-ref 2f0528b
#   just update-op-geth-ref v1.101602.4
#   just update-op-geth-ref optimism
[script('bash')]
update-op-geth-ref ref:
    set -euo pipefail
    ref="{{ref}}"
    if [ -z "$ref" ]; then echo "error: provide a hash/tag/branch"; exit 1; fi
    tmpl=$(printf "\173\173.Version\175\175")
    ver=$(go list -m -f "$tmpl" github.com/ethereum-optimism/op-geth@"$ref")
    if [ -z "$ver" ]; then echo "error: couldn't resolve $ref"; exit 1; fi
    go mod edit -replace=github.com/ethereum/go-ethereum=github.com/ethereum-optimism/op-geth@"$ver"
    go mod tidy
    echo "Updated op-geth to $ver"

# Prints the latest stable semver tag for a component (excludes pre-releases).
latest-tag component:
    @git tag -l '{{ component }}/v*' --sort=-v:refname | grep -E '^[^/]+/v[0-9]+\.[0-9]+\.[0-9]+$' | head -1

# Prints the latest RC tag for a component.
latest-rc-tag component:
    @git tag -l '{{ component }}/v*' --sort=-v:refname | grep -E '^[^/]+/v[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$' | head -1

# Prints the repository paths that ship in a component — the single source of
# truth for "what belongs to <component>", defined next to the code it describes.
# Release notes read it, and so does the proofs release tooling, so a component's
# paths are never restated anywhere else.
#
# Output is one `<label><TAB><path>` pair per line. Consecutive lines sharing a
# label form a group: a set of paths that ship as one unit — a bundled binary's
# dependency closure, or the shared infrastructure every Go component links.
# Consumers that section their output group by label, keeping a closure from
# swamping the component's own changes; consumers that only need the path set
# read the second column. Emitting both in one stream means a component's paths
# and their grouping cannot disagree.
#
# A directory is written as a plain prefix ("cannon/"), not a glob, because the
# two disagree: as a git pathspec, `cannon/**/*` skips files sitting directly in
# cannon/ (README.md, Dockerfile.diff) since `**/` needs an intervening
# directory. Consumers needing a glob append `**/*` themselves — release-notes
# does, for git-cliff.
#
#   just release-paths op-challenger
[script('bash')]
release-paths component:
    set -euo pipefail
    # Shared Go infrastructure linked into every Go binary.
    go_shared="shared=go.*,op-core/,op-service/"
    # kona-host's local-path dependency closure per `cargo metadata`, excluding
    # rust/kona/crates/node/ because kona-node is a separate binary. Blind spot: a
    # dependency bump touching only rust/Cargo.lock appears in no path here.
    kona_host="kona-host=rust/kona/bin/host/,rust/kona/crates/proof/,rust/kona/crates/protocol/,rust/kona/crates/providers/,rust/kona/crates/utilities/,rust/alloy-op-evm/,rust/alloy-op-hardforks/,rust/op-alloy/,rust/op-revm/"
    specs=()
    case "{{ component }}" in
        op-node|op-batcher|op-proposer|op-supernode|op-dispute-mon)
            specs=("{{ component }}/" "$go_shared")
            ;;
        op-challenger)
            # The op-challenger image ships three binaries, so all three are
            # release-relevant: op-challenger itself, the cannon VM, and kona-host
            # built from the in-repo rust/ workspace. See the op-challenger-target
            # stage in ops/docker/op-stack-go/Dockerfile. A kona-host-only fix has
            # no op-challenger/ diff yet still changes what operators run.
            # kona-client is not in the image — it ships as a prestate hash.
            specs=("op-challenger/" "cannon/" "$kona_host" "$go_shared")
            ;;
        op-reth)
            specs=("rust/{{ component }}/" "rust/Cargo.toml" "rust/op-alloy/" "rust/alloy-op*/")
            ;;
        kona-*)
            specs=("rust/kona/" "rust/Cargo.toml" "rust/op-alloy/" "rust/alloy-op*/" "rust/op-revm/")
            ;;
        op-deployer)
            specs=("op-deployer/")
            ;;
        op-contracts)
            specs=("packages/contracts-bedrock/")
            ;;
        *)
            echo "error: component must be one of: op-node, op-batcher, op-proposer, op-challenger, op-dispute-mon, op-reth, op-deployer, op-contracts, op-supernode, kona-*; is {{ component }}" >&2
            exit 1
            ;;
    esac
    # `label=a,b` is private to this recipe: it is expanded to labelled pairs
    # here, so no consumer ever has to know the separators.
    for spec in "${specs[@]}"; do
        if [[ "$spec" == *=* ]]; then
            label="${spec%%=*}"
            group="${spec#*=}"
        else
            # A lone path is its own section; "cannon/" reads as cannon.
            label="${spec%/}"
            group="$spec"
        fi
        IFS=',' read -r -a group_paths <<< "$group"
        for p in "${group_paths[@]}"; do
            printf '%s\t%s\n' "$label" "$p"
        done
    done

# Generates release notes between two tags using git-cliff.
# <from> and <to> can be explicit tags (e.g. v1.16.5), or:
#   'latest'    - resolves to the latest stable tag (vX.Y.Z)
#   'latest-rc' - resolves to the latest RC tag (vX.Y.Z-rc.N)
#   'develop'   - (only for <to>) uses the develop branch tip with --unreleased
#
# Set <mode> to 'offline' to skip GitHub API calls (faster, but no PR metadata).
#
# Examples:
#   just release-notes op-node                          # latest stable -> latest RC (default)
#   just release-notes op-node latest develop           # all unreleased changes since the latest stable release
#   just release-notes op-node latest develop offline   # same, but without GitHub API calls
#   just release-notes op-node v1.16.5 v1.16.6          # explicit tags
#
# Requires GITHUB_TOKEN for git-cliff's GitHub integration (unless mode=offline):
#   GITHUB_TOKEN=$(gh auth token) just release-notes op-node
[script('zsh')]
release-notes component from='latest' to='latest-rc' mode='':
    set -euo pipefail
    if [ "{{ mode }}" != "offline" ] && [ -z "${GITHUB_TOKEN:-}" ]; then
        echo "warning: GITHUB_TOKEN is not set. Set it like: GITHUB_TOKEN=\$(gh auth token) just release-notes ..."
        exit 1
    fi
    resolve_tag() {
        case "$1" in
            latest)    git tag -l "{{ component }}/v*" --sort=-v:refname | grep -E '^[^/]+/v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 ;;
            latest-rc) git tag -l "{{ component }}/v*" --sort=-v:refname | grep -E '^[^/]+/v[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$' | head -1 ;;
            v[0-9]*) echo "{{ component }}/$1" ;;
            *)       echo "error: invalid tag '$1'; expected 'latest', 'latest-rc', or 'vX.Y.Z...'" >&2; return 1 ;;
        esac
    }
    from_tag=$(resolve_tag "{{ from }}")
    if [ -z "$from_tag" ]; then echo "error: could not resolve from tag '{{ from }}' for {{ component }}"; exit 1; fi
    # release-paths is the single source of truth for what ships in a component.
    # Assigning first (rather than piping) so `set -e` aborts on an unknown one.
    component_paths="$({{ just_executable() }} release-paths "{{ component }}")"
    include_path_args=()
    # Second column only — the label just names a section, which notes don't use.
    # Not `path`: this recipe is zsh, where $path is tied to $PATH and reading
    # into it empties PATH, so every later command silently vanishes.
    while IFS=$'\t' read -r _label include_path; do
        [ -z "$include_path" ] && continue
        # release-paths yields directories as prefixes; git-cliff wants globs.
        case "$include_path" in */) include_path="${include_path}**/*" ;; esac
        include_path_args+=(--include-path "$include_path")
    done <<< "$component_paths"
    if [ ${#include_path_args[@]} -eq 0 ]; then
        echo "error: no paths for component '{{ component }}'"
        exit 1
    fi
    tag_args=()
    if [ "{{ to }}" = "develop" ]; then
        tag_args=(--unreleased)
        range_end="develop"
    else
        to_tag=$(resolve_tag "{{ to }}")
        if [ -z "$to_tag" ]; then echo "error: could not resolve to tag '{{ to }}' for {{ component }}"; exit 1; fi
        tag_args=(--tag "$to_tag")
        range_end="$to_tag"
    fi
    echo "Generating release notes for ${from_tag}..${range_end}"
    offline_args=()
    if [ "{{ mode }}" = "offline" ]; then
        offline_args=(--offline)
    fi
    git cliff \
        --config .github/cliff.toml \
        "${include_path_args[@]}" \
        --tag-pattern "${from_tag}" \
        "${tag_args[@]}" \
        "${offline_args[@]}" \
        -- "${from_tag}..${range_end}"

# Run the rust-code-reviewer agent over the current branch (delegates to rust/justfile).
rust-review base='':
  cd rust && just rust-review "{{base}}"
