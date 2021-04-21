---
"@eth-optimism/data-transport-layer": patch
---

Allow the DTL to provide data from either L1 or L2, configurable via a query param sent by the client.
The config option `default-backend` can be used to specify the backend to be
used if the query param is not specified. This allows it to be backwards
compatible with how the DTL was previously used.
