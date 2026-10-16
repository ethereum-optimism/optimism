## `kona-sources`

Data source types and utilities for the kona-node.

## Providers and RPC clients

[`l1_provider`] and [`l2_provider`] build the providers the node reads L1 and L2 through; the L2 provider authenticates with the engine's JWT secret. Each request they send has a deadline.

[`RpcClientLayer`] records metrics and logs for the requests a JSON-RPC client sends, by client and method: the requests and the bytes of their params, how long they took, and the responses by error, with the bytes of their results. The node's JSON-RPC clients use it: the L1 and L2 providers, the conductor client, the remote signer client and the derivation delegate client, as the clients `l1`, `l2`, `conductor`, `signer` and `derivation_delegate`.

## Engine API client

[`EngineApi`] talks to the L2 execution layer's Engine API through a provider that [`l2_provider`] builds. The client selects the Engine API method version:

- **Engine Forkchoice Updated**: V2 and V3, selected by the payload attributes' timestamp (V3 from Ecotone; V3 without attributes)
- **Engine Get Payload**: V2 to V5, selected by the payload's timestamp (V3 from Ecotone, V4 from Isthmus, V5 from Karst)
- **Engine New Payload**: V1 to V4, selected by the payload envelope's version
