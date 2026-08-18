//! Crate-private proposer dependency capabilities.

use alloy_eips::BlockId;
use alloy_primitives::{Address, B256, U256};
use anyhow::Result;
use async_trait::async_trait;

use crate::contract::{GameStatus, ProposalStatus, ZKGameArgs};

/// A compact L1 block observation used for proposer synchronization.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct L1BlockRef {
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

/// Bond fields read only for a defender-wins game.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct BondState {
    pub(crate) credit: U256,
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
    pub(crate) credit: Result<U256>,
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
    async fn game_status(&self, game: Address) -> Result<u8>;
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
    async fn game_standing(&self, game: Address, registry: Address) -> Result<GameStanding>;
    async fn proof_status(&self, game: Address) -> Result<u8>;
    async fn proof_inputs(&self, game: Address) -> Result<ProofInputs>;
    async fn anchor_state_registry(&self, game: Address) -> Result<Address>;
    async fn latest_l1_timestamp(&self) -> Result<u64>;
}

/// Host query time used for current super-root observations.
pub(crate) trait QueryTime: Send + Sync {
    fn unix_timestamp(&self) -> Result<u64>;
}
