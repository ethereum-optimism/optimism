## `kona-sources`

Data source types and utilities for the kona-node.

## Engine API client

[`EngineClient`] talks to the L2 execution layer's Engine API, authenticated with the engine's JWT secret. Each request has a deadline and records its duration. The client selects the Engine API method version:

- **Engine Forkchoice Updated**: V2 and V3, selected by the payload attributes' timestamp (V3 from Ecotone; V3 without attributes)
- **Engine Get Payload**: V2 to V5, selected by the payload's timestamp (V3 from Ecotone, V4 from Isthmus, V5 from Karst)
- **Engine New Payload**: V1 to V4, selected by the payload envelope's version
