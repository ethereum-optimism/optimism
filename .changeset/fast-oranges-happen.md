---
'@eth-optimism/contracts': minor
'@eth-optimism/integration-tests': patch
'@eth-optimism/l2geth': patch
---

Add a new Standard Token Bridge, to handle deposits and withdrawals of any ERC20 token.
For projects developing a custom bridge, if you were previously importing `iAbs_BaseCrossDomainMessenger`, you should now
import `iOVM_CrossDomainMessenger`.
