# Build all Rust binaries (release) for sysgo tests.
build-rust-release:
  cd kona && cargo build --release --bin kona-node --bin kona-supervisor
  cd op-rbuilder && cargo build --release -p op-rbuilder --bin op-rbuilder
  cd rollup-boost && cargo build --release -p rollup-boost --bin rollup-boost

# Checks that TODO comments have corresponding issues.
todo-checker:
  ./ops/scripts/todo-checker.sh

# Runs semgrep on the entire monorepo.
semgrep:
  semgrep scan --config .semgrep/rules/ --error .

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
#   just update-op-geth 2f0528b
#   just update-op-geth v1.101602.4
#   just update-op-geth optimism
update-op-geth ref:
	@ref="{{ref}}"; \
	if [ -z "$ref" ]; then echo "error: provide a hash/tag/branch"; exit 1; fi; \
	tmpl=$(printf "\173\173.Version\175\175"); \
	ver=$(go list -m -f "$tmpl" github.com/ethereum-optimism/op-geth@"$ref"); \
	if [ -z "$ver" ]; then echo "error: couldn't resolve $ref"; exit 1; fi; \
	go mod edit -replace=github.com/ethereum/go-ethereum=github.com/ethereum-optimism/op-geth@"$ver"; \
	go mod tidy; \
	echo "Updated op-geth to $ver"

# e.g. GITHUB_TOKEN=$(gh auth token) just generate-release-notes op-batcher v1.16.3 v1.16.4-rc.1
generate-release-notes component from_tag to_tag:
    @component="{{ component }}"; \
    case "$component" in \
        op-batcher|op-node) \
            ;; \
        *) \
            echo "error: component must be one of: op-batcher, op-node"; \
            exit 1; \
            ;; \
    esac; \
    git cliff \
        --include-path {{ component }}/**/* \
        --include-path go.* \
        --include-path op-core/**/* \
        --include-path op-service/**/* \
        --config .github/cliff.toml \
        --tag-pattern {{ component }}/{{ from_tag }} \
        --tag {{ component }}/{{ to_tag }} \
        -- {{ component }}/{{ from_tag }}..{{ component }}/{{ to_tag }}
