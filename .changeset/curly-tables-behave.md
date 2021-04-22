---
"@eth-optimism/contracts": patch
---

Fix bridge contracts upgradeability by changing `Abs_L1TokenGateway.DEFAULT_FINALIZE_DEPOSIT_L2_GAS` from a storage var to an internal constant.
Additionally, make some bridge functions virtual so they could be overriden in child contracts.
