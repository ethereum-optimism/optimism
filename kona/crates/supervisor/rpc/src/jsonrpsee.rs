//! The Optimism Supervisor RPC API using `jsonrpsee`

pub use jsonrpsee::{
    core::{RpcResult, SubscriptionResult},
    types::{ErrorCode, ErrorObjectOwned},
};

use crate::{SuperRootOutputRpc, SupervisorSyncStatus};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, BlockHash, ChainId, map::HashMap};
use jsonrpsee::proc_macros::rpc;
use kona_interop::{
    DependencySet, DerivedIdPair, DerivedRefPair, ExecutingDescriptor, ManagedEvent, SafetyLevel,
};
use kona_protocol::BlockInfo;
use kona_supervisor_types::{BlockSeal, HexStringU64, OutputV0, Receipts, SubscriptionEvent};
use serde::{Deserialize, Serialize};

/// Supervisor API for interop.
///
/// See spec <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#methods>.
// TODO:: add all the methods
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "supervisor"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "supervisor"))]
pub trait SupervisorApi {
    /// Gets the source block for a given derived block
    #[method(name = "crossDerivedToSource")]
    async fn cross_derived_to_source(
        &self,
        chain_id: HexStringU64,
        block_id: BlockNumHash,
    ) -> RpcResult<BlockInfo>;

    /// Returns the [`LocalUnsafe`] block for given chain.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_localunsafe>
    ///
    /// [`LocalUnsafe`]: SafetyLevel::LocalUnsafe
    #[method(name = "localUnsafe")]
    async fn local_unsafe(&self, chain_id: HexStringU64) -> RpcResult<BlockNumHash>;

    /// Returns the [`LocalSafe`] block for given chain.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_localsafe>
    ///
    /// [`LocalSafe`]: SafetyLevel::LocalSafe
    #[method(name = "localSafe")]
    async fn local_safe(&self, chain_id: HexStringU64) -> RpcResult<DerivedIdPair>;

    /// Returns the [`CrossSafe`] block for given chain.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_crosssafe>
    ///
    /// [`CrossSafe`]: SafetyLevel::CrossSafe
    #[method(name = "crossSafe")]
    async fn cross_safe(&self, chain_id: HexStringU64) -> RpcResult<DerivedIdPair>;

    /// Returns the [`Finalized`] block for the given chain.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_finalized>
    ///
    /// [`Finalized`]: SafetyLevel::Finalized
    #[method(name = "finalized")]
    async fn finalized(&self, chain_id: HexStringU64) -> RpcResult<BlockNumHash>;

    /// Returns the finalized L1 block that the supervisor is synced to.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_finalizedl1>
    #[method(name = "finalizedL1")]
    async fn finalized_l1(&self) -> RpcResult<BlockInfo>;

    /// Returns the [`SuperRootOutput`] at a specified timestamp, which represents the global
    /// state across all monitored chains. Contains the
    /// - Highest L1 [`BlockNumHash`] that is cross-safe among all chains
    /// - Timestamp of the super root
    /// - The [`SuperRoot`] hash
    /// - All chains [`ChainRootInfo`]s
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_superrootattimestamp>
    ///
    /// [`SuperRootOutput`]: kona_interop::SuperRootOutput
    /// [`SuperRoot`]: kona_interop::SuperRoot
    /// [`ChainRootInfo`]: kona_interop::ChainRootInfo
    #[method(name = "superRootAtTimestamp")]
    async fn super_root_at_timestamp(
        &self,
        timestamp: HexStringU64,
    ) -> RpcResult<SuperRootOutputRpc>;

    /// Verifies if an access-list references only valid messages w.r.t. locally configured minimum
    /// [`SafetyLevel`].
    #[method(name = "checkAccessList")]
    async fn check_access_list(
        &self,
        inbox_entries: Vec<B256>,
        min_safety: SafetyLevel,
        executing_descriptor: ExecutingDescriptor,
    ) -> RpcResult<()>;

    /// Describes superchain sync status.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_syncstatus>
    #[method(name = "syncStatus")]
    async fn sync_status(&self) -> RpcResult<SupervisorSyncStatus>;

    /// Returns the last derived block, for each chain, from the given L1 block. This block is at
    /// least [`LocalSafe`].
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisor_allsafederivedat>
    ///
    /// [`LocalSafe`]: SafetyLevel::LocalSafe
    #[method(name = "allSafeDerivedAt")]
    async fn all_safe_derived_at(
        &self,
        derived_from: BlockNumHash,
    ) -> RpcResult<HashMap<ChainId, BlockNumHash>>;

    /// Returns the [`DependencySet`] for the supervisor.
    ///
    /// Spec: <https://github.com/ethereum-optimism/specs/pull/684>
    /// TODO: Replace the link above after the PR is merged.
    #[method(name = "dependencySetV1")]
    async fn dependency_set_v1(&self) -> RpcResult<DependencySet>;
}

/// Supervisor API for admin operations.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "admin"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "admin"))]
pub trait SupervisorAdminApi {
    /// Adds L2RPC to the supervisor.
    #[method(name = "addL2RPC")]
    async fn add_l2_rpc(&self, url: String, jwt_secret: String) -> RpcResult<()>;
}

/// Represents the topics for subscriptions in the Managed Mode API.
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SubscriptionTopic {
    /// The topic for events from the managed node.
    Events,
}

/// ManagedModeApi to send control signals to a managed node from supervisor
/// And get info for syncing the state with the given L2.
///
/// See spec <https://specs.optimism.io/interop/managed-mode.html>
/// Using the proc_macro to generate the client and server code.
/// Default namespace separator is `_`.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "interop"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "interop"))]
pub trait ManagedModeApi {
    /// Subscribe to the events from the managed node.
    /// Op-node provides the `interop-subscribe` method for subscribing to the events topic.
    /// Subscription notifications are then sent via the `interop-subscription` method as
    /// [`SubscriptionEvent`]s.
    // Currently, the `events` topic must be explicitly passed as a parameter to the subscription
    // request, even though this function is specifically intended to subscribe to the `events`
    // topic. todo: Find a way to eliminate the need to pass the topic explicitly.
    #[subscription(name = "subscribe" => "subscription", item = SubscriptionEvent, unsubscribe = "unsubscribe")]
    async fn subscribe_events(&self, topic: SubscriptionTopic) -> SubscriptionResult;

    /// Pull an event from the managed node.
    #[method(name = "pullEvent")]
    async fn pull_event(&self) -> RpcResult<ManagedEvent>;

    /// Control signals sent to the managed node from supervisor
    /// Update the cross unsafe block head
    #[method(name = "updateCrossUnsafe")]
    async fn update_cross_unsafe(&self, id: BlockNumHash) -> RpcResult<()>;

    /// Update the cross safe block head
    #[method(name = "updateCrossSafe")]
    async fn update_cross_safe(&self, derived: BlockNumHash, source: BlockNumHash)
    -> RpcResult<()>;

    /// Update the finalized block head
    #[method(name = "updateFinalized")]
    async fn update_finalized(&self, id: BlockNumHash) -> RpcResult<()>;

    /// Invalidate a block
    #[method(name = "invalidateBlock")]
    async fn invalidate_block(&self, seal: BlockSeal) -> RpcResult<()>;

    /// Send the next L1 block
    #[method(name = "provideL1")]
    async fn provide_l1(&self, next_l1: BlockInfo) -> RpcResult<()>;

    /// Get the genesis block ref for l1 and l2; Soon to be deprecated!
    #[method(name = "anchorPoint")]
    async fn anchor_point(&self) -> RpcResult<DerivedRefPair>;

    /// Reset the managed node to the pre-interop state
    #[method(name = "resetPreInterop")]
    async fn reset_pre_interop(&self) -> RpcResult<()>;

    /// Reset the managed node to the specified block heads
    #[method(name = "reset")]
    async fn reset(
        &self,
        local_unsafe: BlockNumHash,
        cross_unsafe: BlockNumHash,
        local_safe: BlockNumHash,
        cross_safe: BlockNumHash,
        finalized: BlockNumHash,
    ) -> RpcResult<()>;

    /// Sync methods that supervisor uses to sync with the managed node
    /// Fetch all receipts for a give block
    #[method(name = "fetchReceipts")]
    async fn fetch_receipts(&self, block_hash: BlockHash) -> RpcResult<Receipts>;

    /// Get block info for a given block number
    #[method(name = "l2BlockRefByNumber")]
    async fn l2_block_ref_by_number(&self, number: u64) -> RpcResult<BlockInfo>;

    /// Get the chain id
    #[method(name = "chainID")]
    async fn chain_id(&self) -> RpcResult<String>;

    /// Get the state_root, message_parser_storage_root, and block_hash at a given timestamp
    #[method(name = "outputV0AtTimestamp")]
    async fn output_v0_at_timestamp(&self, timestamp: u64) -> RpcResult<OutputV0>;

    /// Get the pending state_root, message_parser_storage_root, and block_hash at a given timestamp
    #[method(name = "pendingOutputV0AtTimestamp")]
    async fn pending_output_v0_at_timestamp(&self, timestamp: u64) -> RpcResult<OutputV0>;

    /// Get the l2 block ref for a given timestamp
    #[method(name = "l2BlockRefByTimestamp")]
    async fn l2_block_ref_by_timestamp(&self, timestamp: u64) -> RpcResult<BlockInfo>;
}
