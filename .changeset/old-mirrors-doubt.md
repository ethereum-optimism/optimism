---
"@eth-optimism/integration-tests": patch
"@eth-optimism/l2geth": patch
---

Correctly set the OVM context based on the L1 values during `eth_call`. This will also set it during `eth_estimateGas`. Add tests for this in the integration tests
