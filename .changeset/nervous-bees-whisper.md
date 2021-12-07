---
'@eth-optimism/l2geth': patch
---

Return a better error message for when the nonce is too high. Previously it would return `nonce too low` for even when the nonce was too high. Now it will return `nonce too high` to the user
