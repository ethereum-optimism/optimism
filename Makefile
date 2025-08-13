# provide JUSTFLAGS for just-backed targets
include ./justfiles/flags.mk

BEDROCK_TAGS_REMOTE?=origin
OP_STACK_GO_BUILDER?=us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest

# Requires at least Python v3.9; specify a minor version below if needed
PYTHON?=python3

help: ## Prints this help message
	@grep -h -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: build-go build-contracts ## Builds Go components and contracts-bedrock
.PHONY: build

build-go: submodules op-node op-proposer op-batcher op-challenger op-dispute-mon op-program cannon ## Builds main Go components
.PHONY: build-go

build-contracts:
	(cd packages/contracts-bedrock && just build)
.PHONY: build-contracts

lint-go: ## Lints Go code with specific linters
	golangci-lint run ./...
	go mod tidy -diff
.PHONY: lint-go

lint-go-fix: ## Lints Go code with specific linters and fixes reported issues
	golangci-lint run ./... --fix
.PHONY: lint-go-fix

golang-docker: ## Builds Docker images for Go components using buildx
	# We don't use a buildx builder here, and just load directly into regular docker, for convenience.
	GIT_COMMIT=$$(git rev-parse HEAD) \
	GIT_DATE=$$(git show -s --format='%ct') \
	IMAGE_TAGS=$$(git rev-parse HEAD),latest \
	docker buildx bake \
			--progress plain \
			--load \
			-f docker-bake.hcl \
			op-node op-batcher op-proposer op-challenger op-dispute-mon op-supervisor
.PHONY: golang-docker

docker-builder-clean: ## Removes the Docker buildx builder
	docker buildx rm buildx-build
.PHONY: docker-builder-clean

docker-builder: ## Creates a Docker buildx builder
	docker buildx create \
		--driver=docker-container --name=buildx-build --bootstrap --use
.PHONY: docker-builder

# add --print to dry-run
cross-op-node: ## Builds cross-platform Docker image for op-node
	# We don't use a buildx builder here, and just load directly into regular docker, for convenience.
	GIT_COMMIT=$$(git rev-parse HEAD) \
	GIT_DATE=$$(git show -s --format='%ct') \
	IMAGE_TAGS=$$(git rev-parse HEAD),latest \
	PLATFORMS="linux/arm64" \
	GIT_VERSION=$(shell tags=$$(git tag --points-at $(GITCOMMIT) | grep '^op-node/' | sed 's/op-node\///' | sort -V); \
             preferred_tag=$$(echo "$$tags" | grep -v -- '-rc' | tail -n 1); \
             if [ -z "$$preferred_tag" ]; then \
                 if [ -z "$$tags" ]; then \
                     echo "untagged"; \
                 else \
                     echo "$$tags" | tail -n 1; \
                 fi \
             else \
                 echo $$preferred_tag; \
             fi) \
	docker buildx bake \
			--progress plain \
			--builder=buildx-build \
			--load \
			--no-cache \
			-f docker-bake.hcl \
			op-node
.PHONY: cross-op-node

contracts-bedrock-docker: ## Builds Docker image for Bedrock contracts
	IMAGE_TAGS=$$(git rev-parse HEAD),latest \
	docker buildx bake \
			--progress plain \
			--load \
			-f docker-bake.hcl \
		  contracts-bedrock
.PHONY: contracts-bedrock-docker

submodules: ## Updates git submodules
	git submodule update --init --recursive
.PHONY: submodules


op-node: ## Builds op-node binary
	just $(JUSTFLAGS) ./op-node/op-node
.PHONY: op-node

generate-mocks-op-node: ## Generates mocks for op-node
	make -C ./op-node generate-mocks
.PHONY: generate-mocks-op-node

generate-mocks-op-service: ## Generates mocks for op-service
	make -C ./op-service generate-mocks
.PHONY: generate-mocks-op-service

op-batcher: ## Builds op-batcher binary
	just $(JUSTFLAGS) ./op-batcher/op-batcher
.PHONY: op-batcher

op-proposer: ## Builds op-proposer binary
	just $(JUSTFLAGS) ./op-proposer/op-proposer
.PHONY: op-proposer

op-challenger: ## Builds op-challenger binary
	make -C ./op-challenger op-challenger
.PHONY: op-challenger

op-dispute-mon: ## Builds op-dispute-mon binary
	make -C ./op-dispute-mon op-dispute-mon
.PHONY: op-dispute-mon

op-program: ## Builds op-program binary
	make -C ./op-program op-program
.PHONY: op-program

cannon:  ## Builds cannon binary
	make -C ./cannon cannon
.PHONY: cannon

reproducible-prestate:   ## Builds reproducible-prestate binary
	make -C ./op-program reproducible-prestate
.PHONY: reproducible-prestate

cannon-prestates: cannon op-program
	go run ./op-program/builder/main.go build-all-prestates
.PHONY: cannon-prestates

mod-tidy: ## Cleans up unused dependencies in Go modules
	# Below GOPRIVATE line allows mod-tidy to be run immediately after
	# releasing new versions. This bypasses the Go modules proxy, which
	# can take a while to index new versions.
	#
	# See https://proxy.golang.org/ for more info.
	export GOPRIVATE="github.com/ethereum-optimism" && go mod tidy
.PHONY: mod-tidy

clean: ## Removes all generated files under bin/
	rm -rf ./bin
	cd packages/contracts-bedrock/ && forge clean
.PHONY: clean

nuke: clean ## Completely clean the project directory
	git clean -Xdf
.PHONY: nuke

test-unit: ## Runs unit tests for individual components
	make -C ./op-node test
	make -C ./op-proposer test
	make -C ./op-batcher test
	make -C ./op-e2e test
	(cd packages/contracts-bedrock && just test)
.PHONY: test-unit

# Remove the baseline-commit to generate a base reading & show all issues
semgrep: ## Runs Semgrep checks
	$(eval DEV_REF := $(shell git rev-parse develop))
	SEMGREP_REPO_NAME=ethereum-optimism/optimism semgrep ci --baseline-commit=$(DEV_REF)
.PHONY: semgrep

op-program-client: ## Builds op-program-client binary
	make -C ./op-program op-program-client
.PHONY: op-program-client

op-program-host: ## Builds op-program-host binary
	make -C ./op-program op-program-host
.PHONY: op-program-host

make-pre-test: ## Makes pre-test setup
	make -C ./op-e2e pre-test
.PHONY: make-pre-test

# Common prerequisites and package list for Go tests
TEST_DEPS := op-program-client op-program-host cannon build-contracts cannon-prestates make-pre-test

# Excludes: op-validator, op-deployer/pkg/{validation,deployer/{bootstrap,manage,opcm,pipeline,upgrade}} (need RPC)
TEST_PKGS := \
	./op-alt-da/... \
	./op-batcher/... \
	./op-chain-ops/... \
	./op-node/... \
	./op-proposer/... \
	./op-challenger/... \
	./op-faucet/... \
	./op-dispute-mon/... \
	./op-conductor/... \
	./op-program/... \
	./op-service/... \
	./op-supervisor/... \
	./op-test-sequencer/... \
	./op-fetcher/... \
	./op-e2e/system/... \
	./op-e2e/e2eutils/... \
	./op-e2e/opgeth/... \
	./op-e2e/interop/... \
	./op-e2e/actions/... \
	./packages/contracts-bedrock/scripts/checks/... \
	./op-dripper/... \
	./devnet-sdk/... \
	./op-acceptance-tests/... \
	./kurtosis-devnet/... \
	./op-devstack/... \
	./op-deployer/pkg/deployer/artifacts/... \
	./op-deployer/pkg/deployer/broadcaster/... \
	./op-deployer/pkg/deployer/clean/... \
	./op-deployer/pkg/deployer/integration_test/... \
	./op-deployer/pkg/deployer/standard/... \
	./op-deployer/pkg/deployer/state/... \
	./op-deployer/pkg/deployer/verify/... \
	./op-sync-tester/...

FRAUD_PROOF_TEST_PKGS := \
	./op-e2e/faultproofs/...

# Includes: op-validator, op-deployer/pkg/{bootstrap,manage,opcm,pipeline,upgrade} (need RPC)
RPC_TEST_PKGS := \
	./op-validator/pkg/validations/... \
	./op-deployer/pkg/deployer/bootstrap/... \
	./op-deployer/pkg/deployer/manage/... \
	./op-deployer/pkg/deployer/opcm/... \
	./op-deployer/pkg/deployer/pipeline/... \
	./op-deployer/pkg/deployer/upgrade/...

# Common test environment variables
# For setting PARALLEL, nproc is for linux, sysctl for Mac and then fallback to 4 if neither is available
define DEFAULT_TEST_ENV_VARS
export ENABLE_KURTOSIS=true && \
export OP_E2E_CANNON_ENABLED="false" && \
export OP_E2E_USE_HTTP=true && \
export ENABLE_ANVIL=true && \
export PARALLEL=$$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
endef

# Additional CI-specific environment variables
define CI_ENV_VARS
export OP_TESTLOG_FILE_LOGGER_OUTDIR=$$(realpath ./tmp/testlogs) && \
export SEPOLIA_RPC_URL="https://ci-sepolia-l1-archive.optimism.io" && \
export MAINNET_RPC_URL="https://ci-mainnet-l1-archive.optimism.io" && \
export NAT_INTEROP_LOADTEST_TARGET=10 && \
export NAT_INTEROP_LOADTEST_TIMEOUT=30s
endef

# Test timeout (can be overridden via environment)
TEST_TIMEOUT ?= 10m

go-tests: $(TEST_DEPS) ## Runs comprehensive Go tests across all packages (cached for fast repeated runs)
	$(DEFAULT_TEST_ENV_VARS) && \
	go test -parallel=$$PARALLEL -timeout=$(TEST_TIMEOUT) $(TEST_PKGS)
.PHONY: go-tests

go-tests-short: $(TEST_DEPS) ## Runs comprehensive Go tests with -short flag
	$(DEFAULT_TEST_ENV_VARS) && \
	go test -short -parallel=$$PARALLEL -timeout=$(TEST_TIMEOUT) $(TEST_PKGS)
.PHONY: go-tests-short

go-tests-short-ci: ## Runs short Go tests with gotestsum for CI (assumes deps built by CI)
	@echo "Setting up test directories..."
	mkdir -p ./tmp/test-results ./tmp/testlogs
	@echo "Running Go tests with gotestsum..."
	$(DEFAULT_TEST_ENV_VARS) && \
	$(CI_ENV_VARS) && \
	gotestsum --format=testname \
		--junitfile=./tmp/test-results/results.xml \
		--jsonfile=./tmp/testlogs/log.json \
		--rerun-fails=3 \
		--rerun-fails-max-failures=50 \
		--packages="$(TEST_PKGS) $(RPC_TEST_PKGS) $(FRAUD_PROOF_TEST_PKGS)" \
		-- -parallel=$$PARALLEL -coverprofile=coverage.out -short -timeout=$(TEST_TIMEOUT) -tags="ci"
.PHONY: go-tests-short-ci

go-tests-ci: ## Runs comprehensive Go tests with gotestsum for CI (assumes deps built by CI)
	@echo "Setting up test directories..."
	mkdir -p ./tmp/test-results ./tmp/testlogs
	@echo "Running Go tests with gotestsum..."
	$(DEFAULT_TEST_ENV_VARS) && \
	$(CI_ENV_VARS) && \
	gotestsum --format=testname \
		--junitfile=./tmp/test-results/results.xml \
		--jsonfile=./tmp/testlogs/log.json \
		--rerun-fails=3 \
		--rerun-fails-max-failures=50 \
		--packages="$(TEST_PKGS) $(RPC_TEST_PKGS) $(FRAUD_PROOF_TEST_PKGS)" \
		-- -parallel=$$PARALLEL -coverprofile=coverage.out -timeout=$(TEST_TIMEOUT) -tags="ci"
.PHONY: go-tests-ci

go-tests-fraud-proofs-ci: ## Runs fraud proofs Go tests with gotestsum for CI (assumes deps built by CI)
	@echo "Setting up test directories..."
	mkdir -p ./tmp/test-results ./tmp/testlogs
	@echo "Running Go tests with gotestsum..."
	$(DEFAULT_TEST_ENV_VARS) && \
	$(CI_ENV_VARS) && \
	export OP_E2E_CANNON_ENABLED="true" && \
	gotestsum --format=testname \
		--junitfile=./tmp/test-results/results.xml \
		--jsonfile=./tmp/testlogs/log.json \
		--rerun-fails=3 \
		--rerun-fails-max-failures=50 \
		--packages="$(FRAUD_PROOF_TEST_PKGS)" \
		-- -parallel=$$PARALLEL -coverprofile=coverage.out -timeout=$(TEST_TIMEOUT)
.PHONY: go-tests-fraud-proofs-ci

test: go-tests ## Runs comprehensive Go tests (alias for go-tests)
.PHONY: test

update-op-geth: ## Updates the Geth version used in the project
	./ops/scripts/update-op-geth.py
.PHONY: update-op-geth
