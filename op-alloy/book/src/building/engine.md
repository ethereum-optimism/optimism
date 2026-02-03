# RPC Engine Types

<a href="https://crates.io/crates/op-alloy-rpc-types-engine"><img src="https://img.shields.io/crates/v/op-alloy-rpc-types-engine.svg" alt="op-alloy-rpc-types-engine crate"></a>

The [`op-alloy-rpc-types-engine`][engine] crate provides Optimism types for interfacing
with the Engine API in the OP Stack.

Optimism defines a custom payload attributes type called [`OpPayloadAttributes`][attributes].
`OpPayloadAttributes` extends alloy's [`PayloadAttributes`][pa] with a few fields: transactions,
a flag for enabling the tx pool, the gas limit, and EIP 1559 parameters.

Optimism also returns a custom type for the `engine_getPayload` request for both V3 and
V4 payload envelopes. These are the [`OpExecutionPayloadEnvelopeV3`][v3] and
[`OpExecutionPayloadEnvelopeV4`][v4] types, which both wrap payload envelope types
from [`alloy-rpc-types-engine`][alloy-engine].


<!-- Links -->

[alloy-engine]: https://crates.io/crates/alloy-rpc-types-engine
[v3]: https://docs.rs/op-alloy-rpc-types-engine/latest/op_alloy_rpc_types_engine/payload/v3/struct.OpExecutionPayloadEnvelopeV3.html
[v4]: https://docs.rs/op-alloy-rpc-types-engine/latest/op_alloy_rpc_types_engine/payload/v4/struct.OpExecutionPayloadEnvelopeV4.html
[pa]: https://docs.rs/alloy-rpc-types-engine/latest/alloy_rpc_types_engine/payload/struct.PayloadAttributes.html
[attributes]: https://docs.rs/op-alloy-rpc-types-engine/latest/op_alloy_rpc_types_engine/struct.OpPayloadAttributes.html
[engine]: https://docs.rs/op-alloy-rpc-types-engine/latest/op_alloy_rpc_types_engine/
