---
'@eth-optimism/contracts-bedrock': patch
---

Makes finalizeWithdrawalTransaction not payable because it doesn't need to be and it was causing confusion throughout the codebase.
