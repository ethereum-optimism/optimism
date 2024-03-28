---
'@eth-optimism/sdk': major
---

Added new message status enum READY_TO_REPLAY. Previously the sdk returned READY_FOR_RELAY when messages were replayable.

- This new message status represents a very minor breaking change in what gets returned for errored transactions
- allows users of the sdk to discriminate between replayable messages and messages that are ready to be finalized for the first time.
- All other enum MessageStatus values are unchanged
