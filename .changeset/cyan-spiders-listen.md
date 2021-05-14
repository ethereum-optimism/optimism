---
'@eth-optimism/l2geth': patch
'@eth-optimism/data-transport-layer': patch
---

Fix bug with replica syncing where contract creations would fail in replicas but pass in the sequencer. This was due to the change from a custom batched tx serialization to the batch serialzation for txs being regular RLP encoding
