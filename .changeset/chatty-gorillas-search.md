---
'@eth-optimism/contracts-bedrock': patch
---

Uses assert rather than a require statements to check for conditions we believe are unreachable.This is more semantically explicit, and should enable us to more effectively use some advanced analysis methods in our testing.
