prebuild:
  ./scripts/checks/check-foundry-install.sh

dep-status:
  git submodule status

install:
  forge install

build: prebuild
  forge build

build-go-ffi:
  cd scripts/go-ffi && go build

autogen-invariant-docs:
  go run ./scripts/autogen/generate-invariant-docs .

test: build-go-ffi
  forge test

# Run Kontrol tests and build all dependencies.
test-kontrol: build-go-ffi build kontrol-summary-full test-kontrol-no-build

# Run Kontrol tests without dependencies.
test-kontrol-no-build:
  ./test/kontrol/scripts/run-kontrol.sh script

test-rerun: build-go-ffi
  forge test --rerun -vvv

# Run extra fuzz iterations for modified fuzz tests.
test-heavy-fuzz-modified-tests: build-go-ffi
  ./scripts/testing/test-heavy-fuzz-modified-tests.sh

genesis:
  forge script scripts/L2Genesis.s.sol:L2Genesis --sig 'runWithStateDump()'

coverage: build-go-ffi
  forge coverage || (bash -c "forge coverage 2>&1 | grep -q 'Stack too deep' && echo -e '\\033[1;33mWARNING\\033[0m: Coverage failed with stack too deep, so overriding and exiting successfully' && exit 0 || exit 1")

coverage-lcov: build-go-ffi
  forge coverage --report lcov || (bash -c "forge coverage --report lcov 2>&1 | grep -q 'Stack too deep' && echo -e '\\033[1;33mWARNING\\033[0m: Coverage failed with stack too deep, so overriding and exiting successfully' && exit 0 || exit 1")

deploy:
  ./scripts/deploy/deploy.sh

gas-snapshot-no-build:
  forge snapshot --match-contract GasBenchMark

statediff:
  ./scripts/utils/statediff.sh && git diff --exit-code

gas-snapshot: build-go-ffi gas-snapshot-no-build

gas-snapshot-check: build-go-ffi
  forge snapshot --match-contract GasBenchMark --check

# Check that the Kontrol deployment script has not changed.
kontrol-deployment-check:
  ./scripts/checks/check-kontrol-deployment.sh

# Generates default Kontrol summary.
kontrol-summary:
  ./test/kontrol/scripts/make-summary-deployment.sh

# Generates fault proofs Kontrol summary.
kontrol-summary-fp:
  KONTROL_FP_DEPLOYMENT=true ./test/kontrol/scripts/make-summary-deployment.sh

# Generates all Kontrol summaries (default and FP).
kontrol-summary-full: kontrol-summary kontrol-summary-fp

# Generates ABI snapshots for contracts.
snapshots-abi-storage:
  go run ./scripts/autogen/generate-snapshots .

# Generates core snapshots without building contracts. Currently just an alias for
# snapshots-abi-storage because we no longer run Kontrol snapshots here. Run
# kontrol-summary-full to build the Kontrol summaries if necessary.
snapshots-no-build: snapshots-abi-storage

# Builds contracts and then generates core snapshots.
snapshots: build snapshots-no-build

# Checks if the snapshots are up to date without building.
snapshots-check-no-build:
  ./scripts/checks/check-snapshots.sh --no-build

# Checks if the snapshots are up to date.
snapshots-check:
  ./scripts/checks/check-snapshots.sh

# Checks interface correctness without building.
interfaces-check-no-build:
  ./scripts/checks/check-interfaces.sh

# Checks that all interfaces are appropriately named and accurately reflect the corresponding
# contract that they're meant to represent. We run "clean" before building because leftover
# artifacts can cause the script to detect issues incorrectly.2
interfaces-check: clean build interfaces-check-no-build

semver-lock:
  forge script scripts/autogen/SemverLock.s.sol

validate-deploy-configs:
  ./scripts/checks/check-deploy-configs.sh

validate-spacers-no-build:
  go run ./scripts/checks/spacers

validate-spacers: build validate-spacers-no-build

# Cleans build artifacts and deployments.
# Removes everything inside of .testdata (except the .gitkeep file).
clean:
  rm -rf ./artifacts ./forge-artifacts ./cache ./scripts/go-ffi/go-ffi ./deployments/hardhat/*
  find ./.testdata -mindepth 1 -not -name '.gitkeep' -delete

pre-pr-no-build: gas-snapshot-no-build snapshots-no-build semver-lock autogen-invariant-docs lint

pre-pr: clean build-go-ffi build pre-pr-no-build

pre-pr-full: test validate-deploy-configs validate-spacers pre-pr

lint-forge-tests-check:
  go run ./scripts/checks/names

lint-contracts-check:
  forge fmt --check

lint-check: lint-contracts-check

lint-contracts-fix:
  forge fmt

lint-fix: lint-contracts-fix

lint: lint-fix lint-check
