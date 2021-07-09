---
'@eth-optimism/l2geth': patch
---

Allow zero gas price transactions from the `OVM_GasPriceOracle.owner` when enforce fees is set to true. This is to prevent the need to manage an additional hot wallet as well as prevent any situation where a bug causes the fees to go too high that it is not possible to lower the fee by sending a transaction
