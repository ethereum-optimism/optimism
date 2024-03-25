---
'@eth-optimism/sdk': patch
---

Fixed bug where replayable transactions would fail `finalize` if they previously were marked as errors but replayable.
