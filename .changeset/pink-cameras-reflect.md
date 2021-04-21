---
"@eth-optimism/contracts": patch
"@eth-optimism/hardhat-ovm": patch
---

Use optimistic-solc to compile the SequencerEntrypoint. Also introduces a cache invalidation mechanism for hardhat-ovm so that we can push new compiler versions.
