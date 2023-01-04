---
'@eth-optimism/common-ts': minor
'@eth-optimism/drippie-mon': patch
'@eth-optimism/fault-detector': patch
'@eth-optimism/replica-healthcheck': patch
---

Updates BaseServiceV2 so that options are secret by default. Services will have to explicitly mark options as "public" for those options to be logged and included in the metadata metric.
