---
'@eth-optimism/contracts-bedrock': patch
---

OZ Audit fixes with a Low or informational severity:

- Hardcode constant values
- Require that msg.value == \_amount on ETH withdrawals
- use \_from in place of msg.sender when applicable in internal functions
