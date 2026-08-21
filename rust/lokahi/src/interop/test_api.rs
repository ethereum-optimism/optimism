//! The interop verifier's test-control RPC.
//!
//! Four methods in the `lokahi` namespace, on the same process-wide socket as `lokahi_chains`:
//! pause, resume, status, and one chain's sealed range. They are what an acceptance test needs in
//! order to observe a running verifier deterministically — hold it still, watch it cold-start,
//! check what backfill covered — and they are process-wide questions, which is why they live here
//! rather than on a chain's own socket.
//!
//! The Go counterpart is `apis.SupernodeInteropTestAPI` (`op-service/apis/supernode.go`), which
//! the in-process Go op-supernode implements by calling itself and lokahi implements by serving
//! these four methods. The wire types below are therefore not free: they are
//! `eth.SupernodeInteropStatus` and `eth.SupernodeSealedBlocks`
//! (`op-service/eth/supernode_interop_test_status.go`) field for field, `snake_case` tags included,
//! because the client on the other side decodes exactly those.
//!
//! # Test-only
//!
//! Nothing in production asks a supernode to stop verifying. These methods exist for the devstack
//! and are registered unconditionally anyway, because the alternative — a build- or config-gated
//! surface — would mean the binary an acceptance test drives is not the binary that ships.
//!
//! # Late binding
//!
//! The RPC server binds before the chains are composed, so that a harness which launched this
//! process has an address to wait for. The verifier does not exist yet at that point, so
//! [`InteropTestHandle`] is empty when it is registered and filled once composition finishes — the
//! same shape [`crate::query::QueryHandle`] uses, and for the same reason. A call arriving in that
//! window is told the supernode is starting rather than being given a zeroed status, which would
//! be a well-formed claim that cold start had finished.

use crate::interop::{InteropQueryError, InteropReader, InteropStatus, SealedBlocks};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId};
use jsonrpsee::{
    core::{RpcResult, async_trait},
    proc_macros::rpc,
    types::{ErrorObject, ErrorObjectOwned},
};
use serde::{Deserialize, Serialize};
use std::sync::{Arc, OnceLock};

/// The JSON-RPC error code every failure on this surface reports.
///
/// One code for all of them, and the generic server-error code at that: these are test-control
/// methods whose caller distinguishes failures by reading the message, and inventing an error
/// taxonomy a client does not branch on would be a contract nobody checks.
const TEST_CONTROL_ERROR: i32 = -32000;

/// `eth.SupernodeInteropStatus`.
///
/// The tags are `snake_case` because the Go struct's are: it is one of the few wire types in the
/// tree that is not `camelCase`, and matching it is the whole point of this type existing.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub(crate) struct WireInteropStatus {
    /// Cold-start attempts since this verifier started.
    pub(crate) backfill_attempts: u32,
    /// Whether cold start has finished.
    pub(crate) backfill_completed: bool,
    /// The configured interop activation timestamp.
    pub(crate) activation_timestamp: u64,
    /// The L2 timestamp the round loop began at; zero before cold start finishes.
    pub(crate) verification_start_timestamp: u64,
    /// The lowest L2 timestamp the verifier covers; zero before cold start finishes.
    pub(crate) first_verifiable_timestamp: u64,
}

impl From<InteropStatus> for WireInteropStatus {
    fn from(status: InteropStatus) -> Self {
        Self {
            backfill_attempts: status.backfill_attempts,
            backfill_completed: status.backfill_completed,
            activation_timestamp: status.activation_timestamp,
            verification_start_timestamp: status.verification_start_timestamp,
            first_verifiable_timestamp: status.first_verifiable_timestamp,
        }
    }
}

/// `eth.BlockID`: a plain JSON number alongside the hash, with no `hexutil` wrapper.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize, Deserialize)]
pub(crate) struct WireBlockId {
    /// The block hash.
    pub(crate) hash: B256,
    /// The block number.
    pub(crate) number: u64,
}

impl From<BlockNumHash> for WireBlockId {
    fn from(id: BlockNumHash) -> Self {
        Self { hash: id.hash, number: id.number }
    }
}

/// `eth.SupernodeSealedBlock`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize, Deserialize)]
pub(crate) struct WireSealedBlock {
    /// The block's number and hash.
    pub(crate) id: WireBlockId,
    /// The block's L2 timestamp.
    pub(crate) timestamp: u64,
}

/// `eth.SupernodeSealedBlocks`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize, Deserialize)]
pub(crate) struct WireSealedBlocks {
    /// The earliest sealed block; meaningful only when `has_blocks`.
    pub(crate) first: WireSealedBlock,
    /// The most recent sealed block; meaningful only when `has_blocks`.
    pub(crate) latest: WireSealedBlock,
    /// Whether the store holds any sealed block at all.
    pub(crate) has_blocks: bool,
}

impl From<SealedBlocks> for WireSealedBlocks {
    fn from(sealed: SealedBlocks) -> Self {
        Self {
            first: WireSealedBlock {
                id: sealed.first.id.into(),
                timestamp: sealed.first.timestamp,
            },
            latest: WireSealedBlock {
                id: sealed.latest.id.into(),
                timestamp: sealed.latest.timestamp,
            },
            has_blocks: sealed.has_blocks,
        }
    }
}

/// The interop test-control methods of the `lokahi` namespace.
///
/// A chain id crosses this surface as a plain `u64`, matching `lokahi_chains`, rather than as the
/// decimal string `eth.ChainID` marshals itself into. The client is the devstack, which already
/// holds every chain id as a `uint64` when it addresses lokahi, and a `u256`-shaped identifier
/// would buy nothing on a surface whose chains all came out of one configuration file.
#[rpc(server, namespace = "lokahi")]
pub(crate) trait LokahiInteropTestApi {
    /// Stops the verifier when it reaches `timestamp`.
    ///
    /// Inclusive and forward-looking: a verifier already past `timestamp` still stops. Zero clears
    /// the pause, which is `apis.SupernodeInteropTestAPI.PauseInterop`'s documented behaviour and
    /// the reason this takes a bare `u64` rather than an option.
    #[method(name = "pauseInterop")]
    async fn pause_interop(&self, timestamp: u64) -> RpcResult<()>;

    /// Clears any pause, letting verification continue.
    #[method(name = "resumeInterop")]
    async fn resume_interop(&self) -> RpcResult<()>;

    /// Reads the verifier's test-visible progress.
    #[method(name = "interopStatus")]
    async fn interop_status(&self) -> RpcResult<WireInteropStatus>;

    /// Reports how far one chain's interop log store extends.
    ///
    /// Fails only when the chain is not one this supernode verifies; an empty store is reported
    /// through the result.
    #[method(name = "interopSealedBlocks")]
    async fn interop_sealed_blocks(&self, chain_id: u64) -> RpcResult<WireSealedBlocks>;
}

/// The handle the RPC server holds on a verifier that does not exist when it binds.
///
/// A [`OnceLock`], like [`crate::query::QueryHandle`]: the only transition is "not composed yet"
/// to "composed", and a type that cannot express a second one cannot accidentally serve two
/// different verifiers.
///
/// Empty for the life of the process when the chain set schedules no interop. That is not a
/// degraded mode — there is no verifier to control — and it is reported as such rather than as a
/// verifier reporting zeroes.
#[derive(Debug, Clone, Default)]
pub(crate) struct InteropTestHandle(Arc<OnceLock<InteropReader>>);

impl InteropTestHandle {
    /// Publishes the verifier's read handle to the RPC server.
    pub(crate) fn attach(&self, reader: InteropReader) {
        if self.0.set(reader).is_err() {
            // Unreachable: `run` composes once. Reported rather than panicked on, because a
            // supernode that is already answering correctly should not be stopped by it.
            tracing::warn!(
                target: "lokahi",
                "The interop test-control API was attached twice; keeping the first verifier"
            );
        }
    }

    /// Builds the RPC module serving these four methods.
    pub(crate) fn into_rpc_module(
        self,
    ) -> Result<jsonrpsee::RpcModule<()>, jsonrpsee::core::RegisterMethodError> {
        let mut module = jsonrpsee::RpcModule::new(());
        module.merge(LokahiInteropTestApiServer::into_rpc(self))?;
        Ok(module)
    }

    /// Returns the verifier, or says why there is none to control.
    fn reader(&self) -> RpcResult<&InteropReader> {
        self.0.get().ok_or_else(|| {
            Self::error(
                "this supernode has no interop verifier to control: either it is still starting, \
                 or its chain set schedules no interop",
            )
        })
    }

    /// Renders a query failure as this surface's JSON-RPC error.
    fn query_error(err: InteropQueryError) -> ErrorObjectOwned {
        Self::error(err.to_string())
    }

    /// Builds this surface's JSON-RPC error from a message.
    fn error(message: impl Into<String>) -> ErrorObjectOwned {
        ErrorObject::owned(TEST_CONTROL_ERROR, message.into(), None::<()>)
    }
}

#[async_trait]
impl LokahiInteropTestApiServer for InteropTestHandle {
    async fn pause_interop(&self, timestamp: u64) -> RpcResult<()> {
        // Zero is "clear", as it is on the Go surface. Expressed as `None` past this point so the
        // verifier cannot confuse a clear with a pause at timestamp zero.
        let pause = (timestamp != 0).then_some(timestamp);
        self.reader()?.set_pause(pause).await.map_err(Self::query_error)
    }

    async fn resume_interop(&self) -> RpcResult<()> {
        self.reader()?.set_pause(None).await.map_err(Self::query_error)
    }

    async fn interop_status(&self) -> RpcResult<WireInteropStatus> {
        Ok(self.reader()?.status().await.map_err(Self::query_error)?.into())
    }

    async fn interop_sealed_blocks(&self, chain_id: u64) -> RpcResult<WireSealedBlocks> {
        Ok(self
            .reader()?
            .sealed_blocks(ChainId::from(chain_id))
            .await
            .map_err(Self::query_error)?
            .into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// The Go client decodes `eth.SupernodeInteropStatus`, whose tags are `snake_case` rather than
    /// this tree's usual `camelCase`. A rename here is invisible in Rust and lands on the other
    /// side as every field reading zero, so the spelling is asserted rather than assumed.
    #[test]
    fn the_status_wire_spelling_is_the_go_struct_tags() {
        let json = serde_json::to_value(WireInteropStatus::from(InteropStatus {
            backfill_attempts: 3,
            backfill_completed: true,
            activation_timestamp: 1_700,
            verification_start_timestamp: 1_710,
            first_verifiable_timestamp: 1_705,
        }))
        .expect("status serializes");

        assert_eq!(
            json,
            serde_json::json!({
                "backfill_attempts": 3,
                "backfill_completed": true,
                "activation_timestamp": 1_700,
                "verification_start_timestamp": 1_710,
                "first_verifiable_timestamp": 1_705,
            })
        );
    }

    /// Same contract for the sealed range, including the nested `eth.BlockID`: `hash` and a plain
    /// numeric `number`, not a hexadecimal one.
    #[test]
    fn the_sealed_range_wire_spelling_is_the_go_struct_tags() {
        use crate::interop::query::SealedBlock;

        let json = serde_json::to_value(WireSealedBlocks::from(SealedBlocks {
            first: SealedBlock {
                id: BlockNumHash { number: 10, hash: B256::repeat_byte(0xaa) },
                timestamp: 1_000,
            },
            latest: SealedBlock {
                id: BlockNumHash { number: 20, hash: B256::repeat_byte(0xbb) },
                timestamp: 1_020,
            },
            has_blocks: true,
        }))
        .expect("sealed range serializes");

        assert_eq!(
            json,
            serde_json::json!({
                "first": {
                    "id": { "hash": B256::repeat_byte(0xaa), "number": 10 },
                    "timestamp": 1_000,
                },
                "latest": {
                    "id": { "hash": B256::repeat_byte(0xbb), "number": 20 },
                    "timestamp": 1_020,
                },
                "has_blocks": true,
            })
        );
    }

    /// An empty store is a truthful answer, not a failure: `has_blocks` false with both ends
    /// zeroed is what the Go client turns into "backfill has sealed nothing for this chain yet".
    #[test]
    fn an_empty_store_serializes_as_no_blocks() {
        let json = serde_json::to_value(WireSealedBlocks::from(SealedBlocks::default()))
            .expect("empty range serializes");
        assert_eq!(json["has_blocks"], serde_json::json!(false));
        assert_eq!(json["first"]["id"]["number"], serde_json::json!(0));
    }

    /// Without a verifier the surface says so. A zeroed status would be a well-formed claim that
    /// cold start had finished at activation timestamp zero, which is what a test polling
    /// `backfill_completed` would act on.
    #[tokio::test]
    async fn an_unattached_handle_refuses_rather_than_answering() {
        let handle = InteropTestHandle::default();
        let err = handle.interop_status().await.expect_err("must refuse");
        assert!(err.message().contains("no interop verifier"), "{err}");

        let err = handle.pause_interop(1_000).await.expect_err("must refuse");
        assert_eq!(err.code(), TEST_CONTROL_ERROR);
    }
}
