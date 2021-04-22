---
"@eth-optimism/l2geth": patch
---

Refactor the SyncService to more closely implement the specification. This includes using query params to select the backend from the DTL, trailing syncing of batches for the sequencer, syncing by batches as the verifier as well as unified code paths for transaction ingestion to prevent double ingestion or missed ingestion
