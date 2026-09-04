//! Crate-private proposer dependency capabilities.

use std::sync::Arc;

use alloy_eips::BlockId;
use alloy_primitives::{Address, B256, U256};
use anyhow::Result;
use async_trait::async_trait;
use kona_sp1_super_range_executor::SuperRootAtTimestampResponse;

use crate::{
    contract::{BondDistributionMode, GameStatus, ProposalStatus, ZKGameArgs},
    prover::ProofKeys,
    superroot::SuperRootAt,
};

/// A compact L1 block observation used for proposer synchronization.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct L1BlockRef {
    pub(crate) hash: B256,
    pub(crate) number: u64,
    pub(crate) timestamp: u64,
}

/// The anchor root and its L2 sequence number.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct AnchorRoot {
    pub(crate) root: B256,
    pub(crate) sequence_number: U256,
}

/// A factory game entry before game-type-specific reads.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct FactoryGame {
    pub(crate) address: Address,
    pub(crate) game_type: u32,
}

/// The policy-relevant fields returned by `claimData`.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct GameClaim {
    pub(crate) status: u8,
    pub(crate) deadline: u64,
    pub(crate) parent_index: u32,
}

/// Immutable identity data read before validating a game's sequence number.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct GameIdentity {
    pub(crate) anchor_state_registry: Address,
    pub(crate) weth: Address,
    pub(crate) creator: Address,
    pub(crate) sequence_number: U256,
}

/// Validity data read after a game's sequence number is accepted.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct GameValidity {
    pub(crate) root_claim: B256,
    pub(crate) was_respected: bool,
    pub(crate) status: GameStatus,
    pub(crate) absolute_prestate: B256,
}

/// Base fields refreshed for every cached game before status-specific reads.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct GameLifecycle {
    pub(crate) proposal_status: ProposalStatus,
    pub(crate) deadline: u64,
    pub(crate) parent_index: u32,
    pub(crate) status: GameStatus,
    pub(crate) is_finalized: bool,
}

/// Bond-distribution and withdrawal fields for terminal lifecycle recovery.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct BondState {
    pub(crate) bond_distribution_mode: BondDistributionMode,
    pub(crate) credit: U256,
    pub(crate) refund_mode_credit: U256,
    pub(crate) withdrawal_amount: U256,
    pub(crate) withdrawal_timestamp: U256,
    pub(crate) delay: U256,
}

/// A withdrawal observation used by the latest-state claim preflight.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct WithdrawalState {
    pub(crate) amount: U256,
    pub(crate) timestamp: U256,
}

/// Independently failing fields from the latest-state claim preflight.
#[derive(Debug)]
pub(crate) struct ClaimPreflight {
    pub(crate) bond_distribution_mode: Result<BondDistributionMode>,
    pub(crate) credit: Result<U256>,
    pub(crate) refund_mode_credit: Result<U256>,
    pub(crate) withdrawal: Result<WithdrawalState>,
}

/// Latest and pending transaction counts for creation reconciliation.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct NonceState {
    pub(crate) pending: u64,
    pub(crate) latest: u64,
}

/// Current registry standing for a game.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct GameStanding {
    pub(crate) blacklisted: bool,
    pub(crate) retired: bool,
}

impl GameStanding {
    /// Returns whether the registry prevents using the game.
    pub(crate) const fn disallowed(self) -> bool {
        self.blacklisted || self.retired
    }
}

/// L1-backed inputs required before fetching a game's super-root span.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct ProofInputs {
    pub(crate) l1_head: B256,
    pub(crate) l1_head_number: u64,
    pub(crate) starting_root: B256,
    pub(crate) starting_sequence_number: u64,
    pub(crate) root_claim: B256,
    pub(crate) sequence_number: u64,
}

/// Super-root safety horizons used by proposal policy.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct ProposalHorizon {
    pub(crate) safe_timestamp: u64,
    pub(crate) finalized_timestamp: u64,
}

/// A `superroot_atTimestamp` response paired with its validated root material.
#[derive(Clone, Debug)]
pub(crate) struct SuperRootAtTimestamp {
    pub(crate) response: SuperRootAtTimestampResponse,
    pub(crate) root: Option<SuperRootAt>,
}

/// The on-chain facts and proposer identity bound into a game proof.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GameProofInputs {
    /// The game's pinned L1 head.
    pub l1_head: B256,
    /// Block number of the pinned L1 head.
    pub l1_head_number: u64,
    /// The agreed starting super root.
    pub starting_root: B256,
    /// Timestamp of the starting super root.
    pub starting_ts: u64,
    /// The claimed super root.
    pub root_claim: B256,
    /// Timestamp of the claim.
    pub claim_ts: u64,
    /// The game's absolute prestate.
    pub prestate: B256,
    /// The address bound into the proof's public values.
    pub prover: Address,
}

/// Result of a confirmed game-creation action.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct GameCreationReceipt {
    pub(crate) game_address: Address,
    pub(crate) transaction_hash: B256,
}

/// Read-only L1 observations consumed by proposer policy.
#[async_trait]
pub(crate) trait L1View: Send + Sync {
    async fn latest_head(&self) -> Result<Option<L1BlockRef>>;
    async fn block_ref(&self, number: u64) -> Result<Option<L1BlockRef>>;
    async fn registered_game_args(&self, block: BlockId) -> Result<ZKGameArgs>;
    async fn anchor_root(&self, registry: Address, block: BlockId) -> Result<AnchorRoot>;
    async fn latest_game_index(&self, block: BlockId) -> Result<Option<U256>>;
    async fn registered_anchor_game(&self, block: BlockId) -> Result<Address>;
    async fn factory_game(&self, index: U256, block: BlockId) -> Result<FactoryGame>;
    async fn game_claim(&self, game: Address, block: BlockId) -> Result<GameClaim>;
    async fn game_identity(&self, game: Address, block: BlockId) -> Result<GameIdentity>;
    async fn game_validity(&self, game: Address, block: BlockId) -> Result<GameValidity>;
    async fn game_lifecycle(
        &self,
        game: Address,
        registry: Address,
        block: BlockId,
    ) -> Result<GameLifecycle>;
    async fn parent_game_status(&self, parent_index: u32, block: BlockId) -> Result<u8>;
    async fn bond_state(
        &self,
        game: Address,
        weth: Address,
        proposer: Address,
        block: BlockId,
    ) -> Result<BondState>;
    async fn init_bond(&self) -> Result<U256>;
    async fn game_status(&self, game: Address, block: BlockId) -> Result<u8>;
    async fn claim_preflight(
        &self,
        game: Address,
        weth: Address,
        proposer: Address,
    ) -> ClaimPreflight;
    async fn weth_delay(&self, weth: Address) -> Result<U256>;
    async fn game_by_uuid(&self, root_claim: B256, extra_data: Vec<u8>) -> Result<Address>;
    async fn game_creator(&self, game: Address) -> Result<Address>;
    async fn nonce_state(&self, proposer: Address) -> Result<NonceState>;
    async fn respected_game_type(&self, block: BlockId) -> Result<u32>;
    async fn parent_standing(&self, game: Address, registry: Address) -> Result<GameStanding>;
    async fn game_standing(
        &self,
        game: Address,
        registry: Address,
        block: BlockId,
    ) -> Result<GameStanding>;
    async fn proof_status(&self, game: Address) -> Result<u8>;
    async fn proof_inputs(&self, game: Address) -> Result<ProofInputs>;
    async fn latest_l1_timestamp(&self) -> Result<u64>;
}

/// Super-root observations consumed by proposal and proof policy.
#[async_trait]
pub(crate) trait SuperRootSource: Send + Sync {
    async fn proposal_horizon(&self, timestamp: u64) -> Result<ProposalHorizon>;
    async fn super_root_at_timestamp(&self, timestamp: u64) -> Result<SuperRootAtTimestamp>;
}

/// Expensive witness collection and SP1 proof execution.
#[async_trait]
pub(crate) trait ProofEngine: Send + Sync {
    async fn prove(
        &self,
        game_address: Address,
        keys: Option<Arc<ProofKeys>>,
        game: GameProofInputs,
        responses: Vec<SuperRootAtTimestampResponse>,
    ) -> Result<Vec<u8>>;
    fn clear(&self, game_address: Address);
}

/// Confirmed proposer transaction effects.
#[async_trait]
pub(crate) trait ActionExecutor: Send + Sync {
    async fn create_game(
        &self,
        root_claim: B256,
        extra_data: Vec<u8>,
        init_bond: U256,
    ) -> Result<GameCreationReceipt>;
    async fn prove_game(&self, game: Address, proof: Vec<u8>) -> Result<B256>;
    async fn resolve_game(&self, game: Address) -> Result<B256>;
    async fn claim_credit(&self, game: Address, recipient: Address) -> Result<B256>;
}

/// Host query time used for current super-root observations.
pub(crate) trait QueryTime: Send + Sync {
    fn unix_timestamp(&self) -> Result<u64>;
}
