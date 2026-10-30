//! The Optimism RPC API using `jsonrpsee`

use crate::{
    OutputResponse, SafeHeadResponse, health::HealthzResponse,
    superroot::SuperRootAtTimestampResponse,
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::{B256, U64};
use core::net::IpAddr;
use ipnet::IpNet;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use kona_genesis::RollupConfig;
use kona_gossip::{PeerCount, PeerDump, PeerInfo, PeerStats};
use kona_protocol::SyncStatus;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

#[cfg_attr(all(target_arch = "wasm32", target_os = "unknown"), allow(unused_imports))]
use getrandom as _; // required for compiling wasm32-unknown-unknown

// Re-export apis defined in upstream `op-alloy-rpc-jsonrpsee`
pub use op_alloy_rpc_jsonrpsee::traits::{MinerApiExtServer, OpAdminApiServer};

/// Optimism specified rpc interface.
///
/// https://docs.optimism.io/builders/node-operators/json-rpc
/// https://github.com/ethereum-optimism/optimism/blob/8dd17a7b114a7c25505cd2e15ce4e3d0f7e3f7c1/op-node/node/api.go#L114
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "optimism"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "optimism"))]
pub trait RollupNodeApi {
    /// Get the output root at a specific block.
    #[method(name = "outputAtBlock")]
    async fn op_output_at_block(&self, block_number: BlockNumberOrTag)
    -> RpcResult<OutputResponse>;

    /// Get the safe head at an L1 block.
    #[method(name = "safeHeadAtL1Block")]
    async fn op_safe_head_at_l1_block(&self, block_number: U64) -> RpcResult<SafeHeadResponse>;

    /// Get the synchronization status.
    #[method(name = "syncStatus")]
    async fn op_sync_status(&self) -> RpcResult<SyncStatus>;

    /// Get the rollup configuration parameters.
    #[method(name = "rollupConfig")]
    async fn op_rollup_config(&self) -> RpcResult<RollupConfig>;

    /// Get the software version.
    #[method(name = "version")]
    async fn op_version(&self) -> RpcResult<String>;
}

/// The super root API op-node serves for its single chain, the non-interop counterpart of
/// op-supernode's.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "superroot"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "superroot"))]
pub trait SuperRootApi {
    /// Returns the super root at a timestamp, the output it is built from, and the L1 block that
    /// first made that output safe.
    #[method(name = "atTimestamp")]
    async fn superroot_at_timestamp(
        &self,
        timestamp: U64,
    ) -> RpcResult<SuperRootAtTimestampResponse>;
}

/// The opp2p namespace handles peer interactions.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "opp2p"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "opp2p"))]
pub trait OpP2PApi {
    /// Returns information of node
    #[method(name = "self")]
    async fn opp2p_self(&self) -> RpcResult<PeerInfo>;

    /// Returns information of peers
    #[method(name = "peerCount")]
    async fn opp2p_peer_count(&self) -> RpcResult<PeerCount>;

    /// Returns information of peers. If `connected` is true, only returns connected peers.
    #[method(name = "peers")]
    async fn opp2p_peers(&self, connected: bool) -> RpcResult<PeerDump>;

    /// Returns statistics of peers
    #[method(name = "peerStats")]
    async fn opp2p_peer_stats(&self) -> RpcResult<PeerStats>;

    /// Returns the discovery table
    #[method(name = "discoveryTable")]
    async fn opp2p_discovery_table(&self) -> RpcResult<Vec<String>>;

    /// Blocks the given peer: closes its connections, refuses its new ones in both directions, and
    /// ignores its gossip until it is unblocked.
    ///
    /// This applies to protected and static peers too: a static peer isn't dialed again while it
    /// is blocked. Unlike op-node, which leaves a blocked peer's open connections in place.
    #[method(name = "blockPeer")]
    async fn opp2p_block_peer(&self, peer: String) -> RpcResult<()>;

    /// Unblocks the given peer
    #[method(name = "unblockPeer")]
    async fn opp2p_unblock_peer(&self, peer: String) -> RpcResult<()>;

    /// Lists blocked peers
    #[method(name = "listBlockedPeers")]
    async fn opp2p_list_blocked_peers(&self) -> RpcResult<Vec<String>>;

    /// Blocks the given address
    #[method(name = "blockAddr")]
    async fn opp2p_block_addr(&self, ip: IpAddr) -> RpcResult<()>;

    /// Unblocks the given address
    #[method(name = "unblockAddr")]
    async fn opp2p_unblock_addr(&self, ip: IpAddr) -> RpcResult<()>;

    /// Lists blocked addresses
    #[method(name = "listBlockedAddrs")]
    async fn opp2p_list_blocked_addrs(&self) -> RpcResult<Vec<IpAddr>>;

    /// Blocks the given subnet
    #[method(name = "blockSubnet")]
    async fn opp2p_block_subnet(&self, subnet: IpNet) -> RpcResult<()>;

    /// Unblocks the given subnet
    #[method(name = "unblockSubnet")]
    async fn opp2p_unblock_subnet(&self, subnet: IpNet) -> RpcResult<()>;

    /// Lists blocked subnets
    #[method(name = "listBlockedSubnets")]
    async fn opp2p_list_blocked_subnets(&self) -> RpcResult<Vec<IpNet>>;

    /// Protects the given peer
    #[method(name = "protectPeer")]
    async fn opp2p_protect_peer(&self, peer: String) -> RpcResult<()>;

    /// Unprotects the given peer
    #[method(name = "unprotectPeer")]
    async fn opp2p_unprotect_peer(&self, peer: String) -> RpcResult<()>;

    /// Connects to the peer at the given multiaddr, which must end in its `/p2p/<peer id>`, and
    /// returns once the peer is connected.
    ///
    /// Respects the operator's blocks: if the peer, its IP address or its subnet is blocked, it
    /// fails at once with the reason, and dials nothing. Unblock first to connect such a peer.
    /// It also fails at once if the peer has reached its redial limit. A dial that fails, for
    /// example because nothing listens at the address, returns the dial's error, and one that
    /// doesn't finish within 30 seconds times out, as in op-node.
    #[method(name = "connectPeer")]
    async fn opp2p_connect_peer(&self, peer: String) -> RpcResult<()>;

    /// Disconnects from the given peer, and returns once its connections have closed. A static
    /// peer is dialed again about a second later.
    #[method(name = "disconnectPeer")]
    async fn opp2p_disconnect_peer(&self, peer: String) -> RpcResult<()>;
}

/// The admin namespace for the consensus node.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "admin"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "admin"))]
#[async_trait]
pub trait AdminApi {
    /// Validates the payload's block hash and queues it for engine processing.
    ///
    /// Returns after queue admission, before execution. Submitted payloads are not gossiped.
    #[method(name = "postUnsafePayload")]
    async fn admin_post_unsafe_payload(&self, payload: OpExecutionPayloadEnvelope)
    -> RpcResult<()>;

    /// Checks if the sequencer is active.
    #[method(name = "sequencerActive")]
    async fn admin_sequencer_active(&self) -> RpcResult<bool>;

    /// Starts the sequencer at the expected unsafe head.
    #[method(name = "startSequencer")]
    async fn admin_start_sequencer(&self, head: B256) -> RpcResult<()>;

    /// Stops the sequencer.
    #[method(name = "stopSequencer")]
    async fn admin_stop_sequencer(&self) -> RpcResult<B256>;

    /// Checks if the conductor is enabled.
    #[method(name = "conductorEnabled")]
    async fn admin_conductor_enabled(&self) -> RpcResult<bool>;

    /// Gets the recover mode.
    #[method(name = "adminRecoverMode")]
    async fn admin_recover_mode(&self) -> RpcResult<bool>;

    /// Sets the recover mode.
    #[method(name = "setRecoverMode")]
    async fn admin_set_recover_mode(&self, mode: bool) -> RpcResult<()>;

    /// Locally bypasses conductor leadership checks and commits until the node restarts.
    #[method(name = "overrideLeader")]
    async fn admin_override_leader(&self) -> RpcResult<()>;

    /// Resets the derivation pipeline.
    #[method(name = "resetDerivationPipeline")]
    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()>;
}

/// The admin namespace for the consensus node.
#[cfg_attr(not(feature = "client"), rpc(server))]
#[cfg_attr(feature = "client", rpc(server, client))]
pub trait HealthzApi {
    /// Gets the health of the kona-node.
    #[method(name = "healthz")]
    async fn healthz(&self) -> RpcResult<HealthzResponse>;
}
