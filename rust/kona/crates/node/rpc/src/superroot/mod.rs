//! Serves `superroot_atTimestamp` for a single chain: the answer op-supernode gives for a
//! dependency set, reduced to the one chain this node follows, so proofs tooling can point at a
//! kona-node where it would otherwise need a supernode.

mod server;
pub use server::SuperRootRpc;

mod types;
pub use types::{
    ChainIdAndOutput, DecimalChainId, OutputV0Response, OutputWithRequiredL1,
    SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1Response,
};
