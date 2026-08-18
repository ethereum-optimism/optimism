//! Production implementations of proposer dependency capabilities.

use std::time::{SystemTime, UNIX_EPOCH};

use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::{Address, B256, U256};
use alloy_provider::Provider;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result};
use async_trait::async_trait;

use crate::{
    FactoryTrait, ZK_GAME_TYPE,
    contract::{
        AnchorStateRegistry, DelayedWETH, DisputeGameFactory::DisputeGameFactoryInstance,
        GameStatus, ProposalStatus, ZKDisputeGame, ZKGameArgs,
    },
    ports::{
        AnchorRoot, BondState, ClaimPreflight, FactoryGame, GameClaim, GameIdentity, GameLifecycle,
        GameStanding, GameValidity, L1BlockRef, L1View, NonceState, ProofInputs, QueryTime,
        WithdrawalState,
    },
};

/// Production L1 reads over Alloy providers and generated contract bindings.
pub(crate) struct ProductionL1View<P> {
    provider: P,
    factory: DisputeGameFactoryInstance<P>,
    l1_rpc: Url,
}

impl<P> ProductionL1View<P> {
    pub(crate) const fn new(
        provider: P,
        factory: DisputeGameFactoryInstance<P>,
        l1_rpc: Url,
    ) -> Self {
        Self { provider, factory, l1_rpc }
    }
}

#[async_trait]
impl<P> L1View for ProductionL1View<P>
where
    P: Provider + Clone + Send + Sync + 'static,
{
    async fn latest_head(&self) -> Result<Option<L1BlockRef>> {
        Ok(self.provider.get_block_by_number(BlockNumberOrTag::Latest).await?.map(|block| {
            L1BlockRef { number: block.header.number, timestamp: block.header.timestamp }
        }))
    }

    async fn block_ref(&self, number: u64) -> Result<Option<L1BlockRef>> {
        Ok(self.provider.get_block_by_number(BlockNumberOrTag::Number(number)).await?.map(
            |block| L1BlockRef { number: block.header.number, timestamp: block.header.timestamp },
        ))
    }

    async fn registered_game_args(&self, block: BlockId) -> Result<ZKGameArgs> {
        let raw = self.factory.gameArgs(ZK_GAME_TYPE).block(block).call().await?;
        ZKGameArgs::decode(&raw)
            .with_context(|| format!("game type {ZK_GAME_TYPE} has no valid registered game args"))
    }

    async fn anchor_root(&self, registry: Address, block: BlockId) -> Result<AnchorRoot> {
        let anchor = AnchorStateRegistry::new(registry, self.provider.clone())
            .getAnchorRoot()
            .block(block)
            .call()
            .await?;
        Ok(AnchorRoot { root: anchor._0, sequence_number: anchor._1 })
    }

    async fn latest_game_index(&self, block: BlockId) -> Result<Option<U256>> {
        self.factory.fetch_latest_game_index(block).await
    }

    async fn registered_anchor_game(&self, block: BlockId) -> Result<Address> {
        let args = self.registered_game_args(block).await?;
        Ok(AnchorStateRegistry::new(args.anchor_state_registry, self.provider.clone())
            .anchorGame()
            .block(block)
            .call()
            .await?)
    }

    async fn factory_game(&self, index: U256, block: BlockId) -> Result<FactoryGame> {
        let game = self.factory.gameAtIndex(index).block(block).call().await?;
        Ok(FactoryGame { address: game.proxy_, game_type: game.gameType_ })
    }

    async fn game_claim(&self, game: Address, block: BlockId) -> Result<GameClaim> {
        let claim =
            ZKDisputeGame::new(game, self.provider.clone()).claimData().block(block).call().await?;
        Ok(GameClaim {
            status: claim.status,
            deadline: claim.deadline,
            parent_index: claim.parentIndex,
        })
    }

    async fn game_identity(&self, game: Address, block: BlockId) -> Result<GameIdentity> {
        let contract = ZKDisputeGame::new(game, self.provider.clone());
        let anchor_state_registry = contract.anchorStateRegistry().block(block).call().await?;
        let weth = contract.weth().block(block).call().await?;
        let creator = contract.gameCreator().block(block).call().await?;
        let sequence_number = contract.l2SequenceNumber().block(block).call().await?;
        Ok(GameIdentity { anchor_state_registry, weth, creator, sequence_number })
    }

    async fn game_validity(&self, game: Address, block: BlockId) -> Result<GameValidity> {
        let contract = ZKDisputeGame::new(game, self.provider.clone());
        let root_claim = contract.rootClaim().block(block).call().await?;
        let was_respected = contract.wasRespectedGameTypeWhenCreated().block(block).call().await?;
        let status = GameStatus::try_from(contract.status().block(block).call().await?)?;
        let absolute_prestate = contract.absolutePrestate().block(block).call().await?;
        Ok(GameValidity { root_claim, was_respected, status, absolute_prestate })
    }

    async fn game_lifecycle(
        &self,
        game: Address,
        registry: Address,
        block: BlockId,
    ) -> Result<GameLifecycle> {
        let claim = self.game_claim(game, block).await?;
        let proposal_status = ProposalStatus::try_from(claim.status)?;
        let status = GameStatus::try_from(
            ZKDisputeGame::new(game, self.provider.clone()).status().block(block).call().await?,
        )?;
        let is_finalized = AnchorStateRegistry::new(registry, self.provider.clone())
            .isGameFinalized(game)
            .block(block)
            .call()
            .await?;
        Ok(GameLifecycle {
            proposal_status,
            deadline: claim.deadline,
            parent_index: claim.parent_index,
            status,
            is_finalized,
        })
    }

    async fn parent_game_status(&self, parent_index: u32, block: BlockId) -> Result<u8> {
        let parent =
            self.factory.fetch_game_address_by_index(U256::from(parent_index), block).await?;
        Ok(ZKDisputeGame::new(parent, self.factory.provider().clone())
            .status()
            .block(block)
            .call()
            .await?)
    }

    async fn bond_state(
        &self,
        game: Address,
        weth: Address,
        proposer: Address,
        block: BlockId,
    ) -> Result<BondState> {
        let credit = ZKDisputeGame::new(game, self.provider.clone())
            .credit(proposer)
            .block(block)
            .call()
            .await?;
        let weth = DelayedWETH::new(weth, self.provider.clone());
        let withdrawal = weth.withdrawals(game, proposer).block(block).call().await?;
        let delay = weth.delay().block(block).call().await?;
        Ok(BondState {
            credit,
            withdrawal_amount: withdrawal.amount,
            withdrawal_timestamp: withdrawal.timestamp,
            delay,
        })
    }

    async fn init_bond(&self) -> Result<U256> {
        self.factory.fetch_init_bond(ZK_GAME_TYPE).await
    }

    async fn game_status(&self, game: Address) -> Result<u8> {
        Ok(ZKDisputeGame::new(game, self.provider.clone()).status().call().await?)
    }

    async fn claim_preflight(
        &self,
        game: Address,
        weth: Address,
        proposer: Address,
    ) -> ClaimPreflight {
        let credit = ZKDisputeGame::new(game, self.provider.clone()).credit(proposer).call().await;
        let withdrawal = DelayedWETH::new(weth, self.provider.clone())
            .withdrawals(game, proposer)
            .call()
            .await
            .map(|withdrawal| WithdrawalState {
                amount: withdrawal.amount,
                timestamp: withdrawal.timestamp,
            })
            .map_err(Into::into);
        ClaimPreflight { credit: credit.map_err(Into::into), withdrawal }
    }

    async fn weth_delay(&self, weth: Address) -> Result<U256> {
        Ok(DelayedWETH::new(weth, self.provider.clone()).delay().call().await?)
    }

    async fn game_by_uuid(&self, root_claim: B256, extra_data: Vec<u8>) -> Result<Address> {
        Ok(self.factory.games(ZK_GAME_TYPE, root_claim, extra_data.into()).call().await?.proxy_)
    }

    async fn game_creator(&self, game: Address) -> Result<Address> {
        Ok(ZKDisputeGame::new(game, self.provider.clone()).gameCreator().call().await?)
    }

    async fn nonce_state(&self, proposer: Address) -> Result<NonceState> {
        let pending = self.provider.get_transaction_count(proposer).pending().await?;
        let latest = self.provider.get_transaction_count(proposer).latest().await?;
        Ok(NonceState { pending, latest })
    }

    async fn respected_game_type(&self, block: BlockId) -> Result<u32> {
        let args = self.registered_game_args(block).await?;
        Ok(AnchorStateRegistry::new(args.anchor_state_registry, self.provider.clone())
            .respectedGameType()
            .call()
            .await?)
    }

    async fn parent_standing(&self, game: Address, registry: Address) -> Result<GameStanding> {
        let registry = AnchorStateRegistry::new(registry, self.provider.clone());
        let blacklisted = registry.isGameBlacklisted(game).call().await?;
        let retired = registry.isGameRetired(game).call().await?;
        Ok(GameStanding { blacklisted, retired })
    }

    async fn game_standing(&self, game: Address, registry: Address) -> Result<GameStanding> {
        let registry = AnchorStateRegistry::new(registry, self.provider.clone());
        let blacklisted = registry.isGameBlacklisted(game);
        let retired = registry.isGameRetired(game);
        let (blacklisted, retired) = tokio::try_join!(blacklisted.call(), retired.call())?;
        Ok(GameStanding { blacklisted, retired })
    }

    async fn proof_status(&self, game: Address) -> Result<u8> {
        Ok(ZKDisputeGame::new(game, self.provider.clone()).claimData().call().await?.status)
    }

    async fn proof_inputs(&self, game: Address) -> Result<ProofInputs> {
        let contract = ZKDisputeGame::new(game, self.provider.clone());
        let l1_head = contract.l1Head().call().await?;
        let starting = contract.startingProposal().call().await?;
        let root_claim = contract.rootClaim().call().await?;
        let sequence_number = contract
            .l2SequenceNumber()
            .call()
            .await?
            .try_into()
            .context("l2SequenceNumber exceeds u64")?;
        let starting_sequence_number = starting
            .l2SequenceNumber
            .try_into()
            .context("starting l2SequenceNumber exceeds u64")?;
        let l1_head_number =
            kona_sp1_super_range_executor::fetch_l1_head_number(self.l1_rpc.as_str(), l1_head)
                .await?;
        Ok(ProofInputs {
            l1_head,
            l1_head_number,
            starting_root: starting.root,
            starting_sequence_number,
            root_claim,
            sequence_number,
        })
    }

    async fn anchor_state_registry(&self, game: Address) -> Result<Address> {
        Ok(ZKDisputeGame::new(game, self.provider.clone()).anchorStateRegistry().call().await?)
    }

    async fn latest_l1_timestamp(&self) -> Result<u64> {
        Ok(self
            .provider
            .get_block_by_number(BlockNumberOrTag::Latest)
            .await?
            .context("failed to fetch latest L1 block")?
            .header
            .timestamp)
    }
}

/// Production host-clock source.
pub(crate) struct SystemQueryTime;

impl QueryTime for SystemQueryTime {
    fn unix_timestamp(&self) -> Result<u64> {
        Ok(SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs())
    }
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{Address, B256, Bytes, U256};
    use alloy_provider::{ProviderBuilder, mock::Asserter};
    use alloy_sol_types::SolValue;

    use super::*;
    use crate::{contract::DisputeGameFactory, ports::L1View};

    fn view(asserter: Asserter) -> ProductionL1View<crate::L1Provider> {
        let provider = ProviderBuilder::default().connect_mocked_client(asserter);
        let factory = DisputeGameFactory::new(Address::ZERO, provider.clone());
        ProductionL1View::new(provider, factory, "http://127.0.0.1:1".parse().unwrap())
    }

    fn push_abi(asserter: &Asserter, value: impl SolValue) {
        let encoded: Bytes = value.abi_encode().into();
        asserter.push_success(&encoded);
    }

    #[tokio::test]
    async fn claim_preflight_preserves_independent_read_failures() {
        let asserter = Asserter::new();
        asserter.push_failure_msg("credit unavailable");
        push_abi(&asserter, (U256::from(2), U256::from(3)));
        let result =
            view(asserter).claim_preflight(Address::ZERO, Address::ZERO, Address::ZERO).await;
        assert!(result.credit.is_err());
        assert_eq!(result.withdrawal.unwrap().amount, U256::from(2));

        let asserter = Asserter::new();
        push_abi(&asserter, U256::from(1));
        asserter.push_failure_msg("withdrawal unavailable");
        let result =
            view(asserter).claim_preflight(Address::ZERO, Address::ZERO, Address::ZERO).await;
        assert_eq!(result.credit.unwrap(), U256::from(1));
        assert!(result.withdrawal.is_err());
    }

    #[tokio::test]
    async fn parent_standing_stops_after_the_first_failed_read() {
        let asserter = Asserter::new();
        asserter.push_failure_msg("blacklist unavailable");
        push_abi(&asserter, true);
        let view = view(asserter.clone());

        assert!(view.parent_standing(Address::ZERO, Address::ZERO).await.is_err());
        assert_eq!(asserter.read_q().len(), 1);
    }

    #[tokio::test]
    async fn parent_status_uses_the_factory_provider() {
        let game = Address::left_padding_from(&[0x11]);
        let factory_asserter = Asserter::new();
        push_abi(&factory_asserter, (ZK_GAME_TYPE, 1_u64, game));
        push_abi(&factory_asserter, U256::from(2));

        let factory_provider =
            ProviderBuilder::default().connect_mocked_client(factory_asserter.clone());
        let factory = DisputeGameFactory::new(Address::ZERO, factory_provider);
        let read_provider = ProviderBuilder::default().connect_mocked_client(Asserter::new());
        let view =
            ProductionL1View::new(read_provider, factory, "http://127.0.0.1:1".parse().unwrap());

        assert_eq!(view.parent_game_status(0, BlockId::latest()).await.unwrap(), 2);
        assert!(factory_asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn lifecycle_stops_after_an_invalid_claim_status() {
        let asserter = Asserter::new();
        push_abi(
            &asserter,
            (0_u32, U256::from(u8::MAX), Address::ZERO, Address::ZERO, 1_u64, B256::ZERO),
        );
        push_abi(&asserter, U256::ZERO);
        push_abi(&asserter, false);
        let view = view(asserter.clone());

        assert!(
            view.game_lifecycle(Address::ZERO, Address::ZERO, BlockId::latest()).await.is_err()
        );
        assert_eq!(asserter.read_q().len(), 2);
    }

    #[tokio::test]
    async fn validity_stops_after_an_invalid_game_status() {
        let asserter = Asserter::new();
        push_abi(&asserter, B256::ZERO);
        push_abi(&asserter, true);
        push_abi(&asserter, U256::from(u8::MAX));
        push_abi(&asserter, B256::ZERO);
        let view = view(asserter.clone());

        assert!(view.game_validity(Address::ZERO, BlockId::latest()).await.is_err());
        assert_eq!(asserter.read_q().len(), 1);
    }
}
