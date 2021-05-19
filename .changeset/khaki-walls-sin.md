---
'@eth-optimism/l2geth': patch
---

Adds the `debug_ingestTransactions` endpoint that takes a list of RPC transactions and applies each of them to the state sequentially. This is useful for testing purposes
