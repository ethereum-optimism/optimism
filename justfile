import 'justfiles/git.just'

BEDROCK_TAGS_REMOTE := env('BEDROCK_TAGS_REMOTE', 'origin')
OP_STACK_GO_BUILDER := env('OP_STACK_GO_BUILDER', 'us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest')
PYTHON := env('PYTHON', 'python3')

TEST_TIMEOUT := env('TEST_TIMEOUT', '10m')

TEST_PKGS := "./op-alt-da/... ./op-batcher/... ./op-chain-ops/... ./op-node/... ./op-proposer/... ./op-challenger/... ./op-faucet/... ./op-dispute-mon/... ./op-conductor/... ./op-program/... ./op-service/... ./op-supervisor/... ./op-test-sequencer/... ./op-fetcher/... ./op-e2e/system/... ./op-e2e/e2eutils/... ./op-e2e/opgeth/... ./op-e2e/interop/... ./op-e2e/actions/altda ./op-e2e/actions/batcher ./op-e2e/actions/derivation ./op-e2e/actions/helpers ./op-e2e/actions/interop ./op-e2e/actions/proofs ./op-e2e/actions/proposer ./op-e2e/actions/safedb ./op-e2e/actions/sequencer ./op-e2e/actions/sync ./op-e2e/actions/upgrades ./packages/contracts-bedrock/scripts/checks/... ./op-dripper/... ./op-devstack/... ./op-deployer/pkg/deployer/artifacts/... ./op-deployer/pkg/deployer/broadcaster/... ./op-deployer/pkg/deployer/clean/... ./op-deployer/pkg/deployer/integration_test/ ./op-deployer/pkg/deployer/integration_test/cli/... ./op-deployer/pkg/deployer/standard/... ./op-deployer/pkg/deployer/state/... ./op-deployer/pkg/deployer/verify/... ./op-sync-tester/... ./op-supernode/..."

FRAUD_PROOF_TEST_PKGS := "./op-e2e/faultproofs/..."

RPC_TEST_PKGS := "./op-validator/pkg/validations/... ./op-deployer/pkg/deployer/bootstrap/... ./op-deployer/pkg/deployer/manage/... ./op-deployer/pkg/deployer/opcm/... ./op-deployer/pkg/deployer/pipeline/... ./op-deployer/pkg/deployer/upgrade/..."

ALL_TEST_PACKAGES := TEST_PKGS + " " + RPC_TEST_PKGS + " " + FRAUD_PROOF_TEST_PKGS

# Lists all available targets.
help:
  @just --list

# Builds Go components and contracts-bedrock.
build: build-go build-contracts

# Builds main Go components.
build-go: submodules op-node op-proposer op-batcher op-challenger op-dispute-mon op-program cannon

# Builds contracts-bedrock.
build-contracts:
  cd packages/contracts-bedrock && just build

# Builds the custom linter.
build-customlint:
  cd linter && just build

# Lints Go code with specific linters.
lint-go: build-customlint
  ./linter/bin/op-golangci-lint run ./...
  go mod tidy -diff

# Lints Go code with specific linters and fixes reported issues.
lint-go-fix: build-customlint
  ./linter/bin/op-golangci-lint run ./... --fix

# Checks that op-geth version in go.mod is valid.
check-op-geth-version:
  go run ./ops/scripts/check-op-geth-version

# Builds Docker images for Go components using buildx.
[script('bash')]
golang-docker:
  set -euo pipefail
  GIT_COMMIT=$(git rev-parse HEAD) \
  GIT_DATE=$(git show -s --format='%ct') \
  IMAGE_TAGS=$(git rev-parse HEAD),latest \
  docker buildx bake \
      --progress plain \
      --load \
      -f docker-bake.hcl \
      op-node op-batcher op-proposer op-challenger op-dispute-mon op-supervisor

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

# Builds op-program binary.
op-program:
  cd op-program && just op-program

# Builds cannon binary.
cannon:
  cd cannon && just cannon

# Builds reproducible prestate for op-program.
reproducible-prestate-op-program:
  cd op-program && just build-reproducible-prestate

# Builds reproducible prestate for kona.
reproducible-prestate-kona:
  cd rust && just build-kona-reproducible-prestate

# Builds reproducible prestates for op-program and kona in parallel.
[script('bash')]
reproducible-prestate:
  set -euo pipefail
  (cd op-program && just build-reproducible-prestate) &
  pid1=$!
  (cd rust && just build-kona-reproducible-prestate) &
  pid2=$!
  wait "$pid1" "$pid2"
  (cd op-program && just output-prestate-hash)
  (cd rust && just output-kona-prestate-hash)

# Builds cannon prestates.
cannon-prestates: cannon op-program
  go run ./op-program/builder/main.go build-all-prestates

# Cleans up unused dependencies in Go modules.
# Bypasses the Go module proxy for freshly released versions.
# See https://proxy.golang.org/ for more info.
mod-tidy:
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

# Builds op-program-client binary.
op-program-client:
  cd op-program && just op-program-client

# Builds op-program-host binary.
op-program-host:
  cd op-program && just op-program-host

# Makes pre-test setup.
make-pre-test:
  cd op-e2e && just pre-test

# Runs comprehensive Go tests across all packages.
[script('bash')]
go-tests: op-program-client op-program-host cannon build-contracts cannon-prestates make-pre-test
  set -euo pipefail
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="false"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  go test -parallel="$PARALLEL" -timeout={{TEST_TIMEOUT}} {{TEST_PKGS}}

# Runs comprehensive Go tests with -short flag.
[script('bash')]
go-tests-short: op-program-client op-program-host cannon build-contracts cannon-prestates make-pre-test
  set -euo pipefail
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="false"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  go test -short -parallel="$PARALLEL" -timeout={{TEST_TIMEOUT}} {{TEST_PKGS}}

# Internal: runs Go tests with gotestsum for CI.
[script('bash')]
_go-tests-ci-internal go_test_flags="":
  set -euo pipefail
  (cd cannon && just cannon elf)
  echo "Setting up test directories..."
  mkdir -p ./tmp/test-results ./tmp/testlogs
  echo "Running Go tests with gotestsum..."
  export ENABLE_KURTOSIS=true
  export OP_E2E_CANNON_ENABLED="false"
  export OP_E2E_USE_HTTP=true
  export ENABLE_ANVIL=true
  export PARALLEL=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
  export OP_TESTLOG_FILE_LOGGER_OUTDIR=$(realpath ./tmp/testlogs)
  export SEPOLIA_RPC_URL="https://ci-sepolia-l1-archive.optimism.io"
  export MAINNET_RPC_URL="https://ci-mainnet-l1-archive.optimism.io"
  export NAT_INTEROP_LOADTEST_TARGET=10
  export NAT_INTEROP_LOADTEST_TIMEOUT=30s
  ALL_PACKAGES="{{ALL_TEST_PACKAGES}}"
  if [ -n "${CIRCLE_NODE_TOTAL:-}" ] && [ "$CIRCLE_NODE_TOTAL" -gt 1 ]; then
      NODE_INDEX=${CIRCLE_NODE_INDEX:-0}
      NODE_TOTAL=${CIRCLE_NODE_TOTAL:-1}
      PARALLEL_PACKAGES=$(echo "$ALL_PACKAGES" | tr ' ' '\n' | awk -v idx="$NODE_INDEX" -v total="$NODE_TOTAL" 'NR % total == idx' | tr '\n' ' ')
      if [ -n "$PARALLEL_PACKAGES" ]; then
          echo "Node $NODE_INDEX/$NODE_TOTAL running packages: $PARALLEL_PACKAGES"
          ./ops/scripts/gotestsum-split.sh --format=testname \
              --junitfile=./tmp/test-results/results-"$NODE_INDEX".xml \
              --jsonfile=./tmp/testlogs/log-"$NODE_INDEX".json \
              --rerun-fails=3 \
              --rerun-fails-max-failures=50 \
              --packages="$PARALLEL_PACKAGES" \
              -- -parallel="$PARALLEL" -coverprofile=coverage-"$NODE_INDEX".out {{go_test_flags}} -timeout={{TEST_TIMEOUT}} -tags="ci"
      else
          echo "ERROR: Node $NODE_INDEX/$NODE_TOTAL has no packages to run! Perhaps parallelism is set too high? (ALL_TEST_PACKAGES has $(echo "$ALL_PACKAGES" | wc -w) packages)"
          exit 1
      fi
  else
      ./ops/scripts/gotestsum-split.sh --format=testname \
          --junitfile=./tmp/test-results/results.xml \
          --jsonfile=./tmp/testlogs/log.json \
          --rerun-fails=3 \
          --rerun-fails-max-failures=50 \
          --packages="$ALL_PACKAGES" \
          -- -parallel="$PARALLEL" -coverprofile=coverage.out {{go_test_flags}} -timeout={{TEST_TIMEOUT}} -tags="ci"
  fi

# Runs short Go tests with gotestsum for CI.
go-tests-short-ci:
  just _go-tests-ci-internal "-short"

# Runs comprehensive Go tests with gotestsum for CI.
go-tests-ci:
  just _go-tests-ci-internal ""

# Runs action tests for kona with gotestsum for CI.
go-tests-ci-kona-action:
  just _go-tests-ci-internal "-count=1 -timeout 60m -run Test_ProgramAction"

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
  export SEPOLIA_RPC_URL="https://ci-sepolia-l1-archive.optimism.io"
  export MAINNET_RPC_URL="https://ci-mainnet-l1-archive.optimism.io"
  export NAT_INTEROP_LOADTEST_TARGET=10
  export NAT_INTEROP_LOADTEST_TIMEOUT=30s
  ./ops/scripts/gotestsum-split.sh --format=testname \
      --junitfile=./tmp/test-results/results.xml \
      --jsonfile=./tmp/testlogs/log.json \
      --rerun-fails=3 \
      --rerun-fails-max-failures=50 \
      --packages="{{FRAUD_PROOF_TEST_PKGS}}" \
      -- -parallel="$PARALLEL" -coverprofile=coverage.out -timeout={{TEST_TIMEOUT}}

# Runs comprehensive Go tests (alias for go-tests).
test: go-tests

# Updates the Geth version used in the project.
update-op-geth:
  ./ops/scripts/update-op-geth.py

# Build all Rust binaries (release) for sysgo tests.
build-rust-release:
  cd rust && cargo build --release --bin kona-node
  cd op-rbuilder && cargo build --release -p op-rbuilder --bin op-rbuilder
  cd rollup-boost && cargo build --release -p rollup-boost --bin rollup-boost

# Checks that locked NUT bundles have not been modified.
check-nut-locks:
  go run ./ops/scripts/check-nut-locks

# Checks that TODO comments have corresponding issues.
todo-checker:
  ./ops/scripts/todo-checker.sh

# Runs semgrep tests.
semgrep-test:
  semgrep scan --test --config .semgrep/rules/ .semgrep/tests/

# Runs shellcheck.
shellcheck:
  find . -type f -name '*.sh' -not -path '*/node_modules/*' -not -path './packages/contracts-bedrock/lib/*' -not -path './packages/contracts-bedrock/kout*/*' -exec sh -c 'echo "Checking $1"; shellcheck "$1"' _ {} \;
  find . -type f -name '*.sh' -not -path '*/node_modules/*' -not -path './packages/contracts-bedrock/lib/*' -not -path './packages/contracts-bedrock/kout*/*' -exec shfmt --diff {} \;

# Format shell scripts with shfmt.
shfmt-fix:
  find . -type f -name '*.sh' -not -path '*/node_modules/*' -not -path './packages/contracts-bedrock/lib/*' -not -path './packages/contracts-bedrock/kout*/*' -exec shfmt --write {} \;

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
    include_path_args=()
    case "{{ component }}" in
        op-node|op-batcher|op-proposer|op-challenger)
            include_path_args=(
                --include-path "{{ component }}/**/*"
                --include-path "go.*"
                --include-path "op-core/**/*"
                --include-path "op-service/**/*"
            )
            ;;
        op-reth)
            include_path_args=(
                --include-path "rust/{{ component }}/**/*"
                --include-path "rust/Cargo.toml"
                --include-path "rust/op-alloy/**/*"
                --include-path "rust/alloy-op*/**/*"
            )
            ;;
        kona-*)
            include_path_args=(
                --include-path "rust/kona/**/*"
                --include-path "rust/Cargo.toml"
                --include-path "rust/op-alloy/**/*"
                --include-path "rust/alloy-op*/**/*"
            )
            ;;
        *)
            echo "error: component must be one of: op-node, op-batcher, op-proposer, op-challenger, op-reth, kona-*; is {{ component }}"
            exit 1
            ;;
    esac
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
