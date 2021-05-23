---
'@eth-optimism/hardhat-ovm': patch
---

Instantiate the harhat ethers provider using the Hardhat network config if no provider URL is set, and set the provider at the end, so that the overridden `getSigner` method is used. 
