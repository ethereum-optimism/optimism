---
'@eth-optimism/data-transport-layer': patch
---

Allow the L1 gas price to be fetched from either the sequencer or a L1 provider based on the config `--l1-gas-price-backend` as well as overriding the config by using a query param. Valid values are `l1` or `l2` and it defaults to `l1`
