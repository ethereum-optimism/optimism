---
'@eth-optimism/l2geth': patch
---

Removes config options that are no longer required. `ROLLUP_DATAPRICE`, `ROLLUP_EXECUTION_PRICE`, `ROLLUP_GAS_PRICE_ORACLE_ADDRESS` and `ROLLUP_ENABLE_L2_GAS_POLLING`. The oracle was moved to a predeploy 0x42.. address and polling is always enabled as it no longer needs to be backwards compatible
