use alloy_primitives::U256;
use reth::{core::primitives::SealedBlock, payload::PayloadId};
use reth_optimism_payload_builder::OpBuiltPayload as RethOpBuiltPayload;
use reth_optimism_primitives::OpBlock;
use serde::{Deserialize, Serialize};

pub(super) const AGENT_VERSION: &str = "op-rbuilder/1.0.0";
pub(super) const FLASHBLOCKS_STREAM_PROTOCOL: p2p::StreamProtocol =
    p2p::StreamProtocol::new("/flashblocks/1.0.0");

#[derive(Clone, Debug, PartialEq, Deserialize, Serialize)]
pub(super) enum Message {
    OpBuiltPayload(OpBuiltPayload),
}

impl p2p::Message for Message {
    fn protocol(&self) -> p2p::StreamProtocol {
        FLASHBLOCKS_STREAM_PROTOCOL
    }
}

/// Internal type analogous to [`reth_optimism_payload_builder::OpBuiltPayload`]
/// which additionally implements `Serialize` and `Deserialize` for p2p transmission.
#[derive(Clone, Debug, PartialEq, Deserialize, Serialize)]
pub(crate) struct OpBuiltPayload {
    /// Identifier of the payload
    pub(crate) id: PayloadId,
    /// Sealed block
    pub(crate) block: SealedBlock<OpBlock>,
    /// The fees of the block
    pub(crate) fees: U256,
}

impl From<RethOpBuiltPayload> for Message {
    fn from(value: RethOpBuiltPayload) -> Self {
        Message::OpBuiltPayload(value.into())
    }
}

impl From<OpBuiltPayload> for Message {
    fn from(value: OpBuiltPayload) -> Self {
        Message::OpBuiltPayload(value)
    }
}

impl From<OpBuiltPayload> for RethOpBuiltPayload {
    fn from(value: OpBuiltPayload) -> Self {
        RethOpBuiltPayload::new(value.id, value.block.into(), value.fees, None)
    }
}

impl From<RethOpBuiltPayload> for OpBuiltPayload {
    fn from(value: RethOpBuiltPayload) -> Self {
        OpBuiltPayload {
            id: value.id(),
            block: value.block().clone(),
            fees: value.fees(),
        }
    }
}
