---
"@eth-optimism/contracts": minor
---

- Remove ETH_SIGNED_MESSAGE and standardize on EIP-155 messages
- Reduce nonce size to uint64
- Ban ovmCALLER opcode when it would return ZERO
- Add value transfer in OVM_ECDSAContractAccount
