//! The experimental `opstack` block-building namespace.
//!
//! op-node serves these methods when started with `--experimental.sequencer-api`
//! (`op-node/node/api.go`, `opstackAPI`); the op-test-sequencer's standard builder, committer and
//! publisher drive block building through them. The wire types below mirror the Go types
//! (`op-service/eth`, `op-service/signer`) field for field, and the error codes are the ones
//! `op-service/apis/opstack.go` defines, so a client built against op-node reads the same
//! responses from a kona-node.

use alloy_primitives::{B256, FixedBytes};
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use serde::{Deserialize, Serialize};

/// The build could not start or complete for a transient reason; the caller may retry.
///
/// `BuildErrCodeTemporary` in `op-service/apis/opstack.go`.
pub const BUILD_ERR_CODE_TEMPORARY: i32 = -40100;

/// The engine's pre-state disagrees with the request; a reset is needed to resolve it.
///
/// `BuildErrCodePrestate` in `op-service/apis/opstack.go`.
pub const BUILD_ERR_CODE_PRESTATE: i32 = -40101;

/// The input was invalid: the execution layer rejected the payload or attributes.
///
/// `BuildErrCodeInvalidInput` in `op-service/apis/opstack.go`.
pub const BUILD_ERR_CODE_INVALID_INPUT: i32 = -40110;

/// The named payload is not a build job the execution layer knows.
///
/// `BuildErrCodeUnknownPayload` in `op-service/apis/opstack.go`.
pub const BUILD_ERR_CODE_UNKNOWN_PAYLOAD: i32 = -40120;

/// Any other failure.
///
/// `BuildErrCodeOther` in `op-service/apis/opstack.go`.
pub const BUILD_ERR_CODE_OTHER: i32 = -40199;

/// A block named by hash and number: Go's `eth.BlockID` (`op-service/eth/id.go`), whose JSON
/// carries the number as a plain integer.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct OpStackBlockId {
    /// The block hash.
    pub hash: B256,
    /// The block number, as the plain JSON number Go's `uint64` marshals to.
    pub number: u64,
}

/// A started build job: Go's `eth.PayloadInfo` (`op-service/eth/types.go`).
///
/// The timestamp travels with the id because it is what selects the `engine_getPayload` version
/// when the job is sealed or cancelled, exactly as op-node's `GetPayload` selects it.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct PayloadInfo {
    /// The execution layer's identifier for the build job.
    pub id: alloy_rpc_types_engine::PayloadId,
    /// The timestamp of the payload being built, as the plain JSON number Go marshals.
    pub timestamp: u64,
}

/// A signed execution payload envelope: Go's `opsigner.SignedExecutionPayloadEnvelope`
/// (`op-service/signer/signed_envelope.go`).
///
/// The signature is the 65-byte `eth.Bytes65`, kept as raw bytes rather than decomposed: what the
/// gossip topic carries is these bytes, and `commitBlockV1` — like op-node's — never reads them.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SignedExecutionPayloadEnvelope {
    /// The execution payload and, post-Ecotone, its parent beacon block root.
    pub envelope: OpExecutionPayloadEnvelope,
    /// The 65-byte signature over the envelope.
    pub signature: FixedBytes<65>,
}

/// The experimental `opstack` block-building interface, method for method op-node's `opstackAPI`
/// (`op-node/node/api.go`).
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "opstack"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "opstack"))]
pub trait OpStackApi {
    /// Starts a block-building job on the given parent with the given attributes.
    #[method(name = "openBlockV1")]
    async fn open_block_v1(
        &self,
        parent: OpStackBlockId,
        attrs: OpPayloadAttributes,
    ) -> RpcResult<PayloadInfo>;

    /// Cancels a block-building job.
    #[method(name = "cancelBlockV1")]
    async fn cancel_block_v1(&self, id: PayloadInfo) -> RpcResult<()>;

    /// Completes a block-building job, returning the built payload without canonicalizing it:
    /// the block becomes canonical only when `commitBlockV1` is called with it.
    #[method(name = "sealBlockV1")]
    async fn seal_block_v1(&self, id: PayloadInfo) -> RpcResult<OpExecutionPayloadEnvelope>;

    /// Processes the block and makes it the canonical unsafe head of the chain.
    #[method(name = "commitBlockV1")]
    async fn commit_block_v1(&self, signed: SignedExecutionPayloadEnvelope) -> RpcResult<()>;

    /// Publishes the signed block on the chain's p2p gossip topic, signature as given.
    #[method(name = "publishBlockV1")]
    async fn publish_block_v1(&self, signed: SignedExecutionPayloadEnvelope) -> RpcResult<()>;
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b256;

    /// `eth.BlockID` marshals its number as a plain JSON integer, not a quantity string.
    #[test]
    fn block_id_reads_gos_json() {
        let id: OpStackBlockId = serde_json::from_str(
            r#"{"hash":"0x37b28ab063dad3b53f7c07b8b0da4b332bf3839503b28f6a689a4bfbef85b02c","number":42}"#,
        )
        .expect("a Go eth.BlockID deserializes");
        assert_eq!(
            id.hash,
            b256!("0x37b28ab063dad3b53f7c07b8b0da4b332bf3839503b28f6a689a4bfbef85b02c")
        );
        assert_eq!(id.number, 42);
    }

    /// `eth.PayloadInfo` carries an 8-byte hex id and a plain-integer timestamp, and comes back
    /// out byte for byte: this is the value a caller echoes into `sealBlockV1`.
    #[test]
    fn payload_info_round_trips_gos_json() {
        let go = r#"{"id":"0x0102030405060708","timestamp":1234}"#;
        let info: PayloadInfo = serde_json::from_str(go).expect("a Go eth.PayloadInfo parses");
        assert_eq!(info.id.0.as_slice(), &[1, 2, 3, 4, 5, 6, 7, 8]);
        assert_eq!(info.timestamp, 1234);
        assert_eq!(serde_json::to_string(&info).expect("serializes"), go);
    }

    /// The signed envelope keeps Go's field names and the 65-byte hex signature.
    #[test]
    fn signed_envelope_reads_gos_field_names() {
        let payload =
            alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(&alloy_consensus::Block::<
                op_alloy_consensus::OpTxEnvelope,
            >::default());
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        let signed =
            SignedExecutionPayloadEnvelope { envelope, signature: FixedBytes::repeat_byte(0x11) };

        let json = serde_json::to_value(&signed).expect("serializes");
        assert!(json["envelope"]["executionPayload"].is_object());
        assert_eq!(json["signature"], serde_json::json!(format!("0x{}", "11".repeat(65))));

        let back: SignedExecutionPayloadEnvelope =
            serde_json::from_value(json).expect("deserializes");
        assert_eq!(back, signed);
    }
}
