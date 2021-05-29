---
'@eth-optimism/l2geth': patch
---

Fixes incorrect type parsing in the RollupClient. The gasLimit became greater than the largest safe JS number so it needs to be represented as a string
