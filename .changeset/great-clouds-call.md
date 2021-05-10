---
"@eth-optimism/l2geth": patch
---

Return bytes from both ExecutionManager.run and ExecutionManager.simulateMessage and be sure to properly ABI decode the return values and the nested (bool, returndata)
