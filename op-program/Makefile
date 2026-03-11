DEPRECATED_TARGETS := op-program op-program-host op-program-client op-program-client-mips op-program-client-mips64 op-program-client-mips64-interop op-program-client-riscv check-custom-chains build-reproducible-prestate output-prestate-hash verify-reproducibility clean test capture-mainnet-genesis capture-sepolia-delta capture-sepolia-ecotone capture-sepolia-fjord capture-sepolia-granite capture-sepolia-holocene capture-sepolia-isthmus capture-chain-test-data verify-mainnet-genesis verify-sepolia-delta verify-sepolia-ecotone verify-sepolia-fjord verify-sepolia-granite verify-sepolia-holocene verify-sepolia-isthmus verify-compat analyze-op-program-client-current analyze-op-program-client-next run-vm-compat

include ../justfiles/deprecated.mk

# Used as an entrypoint by external scripts (e.g. Docker builds, CI pipelines)
# that will not be migrated to just. Keep as an explicit Make target without
# deprecation warning for backwards compatibility.
.PHONY: reproducible-prestate
reproducible-prestate:
	just reproducible-prestate
