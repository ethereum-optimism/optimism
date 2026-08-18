//! Production implementations of proposer dependency capabilities.

use std::{
    num::NonZeroUsize,
    sync::Arc,
    time::{SystemTime, UNIX_EPOCH},
};

use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::{Address, B256, U256};
use alloy_provider::Provider;
use alloy_rpc_types_eth::{TransactionReceipt, TransactionRequest};
use alloy_sol_types::SolEvent;
use alloy_transport_http::reqwest::Url;
use anyhow::{Context, Result, bail};
use async_trait::async_trait;
use kona_sp1_super_range_executor::{HostInputs, SuperRootAtTimestampResponse};

use crate::{
    FactoryTrait, TX_REVERTED_PREFIX, ZK_GAME_TYPE,
    config::RangeSplitCount,
    contract::{
        AnchorStateRegistry, DelayedWETH,
        DisputeGameFactory::{DisputeGameCreated, DisputeGameFactoryInstance},
        GameStatus, ProposalStatus, ZKDisputeGame, ZKGameArgs,
    },
    ports::{
        ActionExecutor, AnchorRoot, BondState, ClaimPreflight, FactoryGame, GameClaim,
        GameCreationReceipt, GameIdentity, GameLifecycle, GameStanding, GameValidity, L1BlockRef,
        L1View, NonceState, ProofEngine, ProofInputs, ProposalHorizon, QueryTime,
        SuperRootAtTimestamp, SuperRootSource, WithdrawalState,
    },
    prover::{ProofKeys, ProofProvider},
    proving::{GameProofInputs, prove_game_inner},
    signer::{FeeCaps, SignerLock},
    superroot::{ResponseSelection, SuperrootClient},
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

/// Production super-root observations over the configured supernode clients.
pub(crate) struct ProductionSuperRootSource {
    client: SuperrootClient,
}

impl ProductionSuperRootSource {
    pub(crate) const fn new(client: SuperrootClient) -> Self {
        Self { client }
    }
}

#[async_trait]
impl SuperRootSource for ProductionSuperRootSource {
    async fn proposal_horizon(&self, timestamp: u64) -> Result<ProposalHorizon> {
        let response =
            self.client.superroot_at_timestamp(timestamp, ResponseSelection::HighestL1).await?;
        Ok(ProposalHorizon {
            safe_timestamp: response.current_safe_timestamp,
            finalized_timestamp: response.current_finalized_timestamp,
        })
    }

    async fn super_root_at_timestamp(&self, timestamp: u64) -> Result<SuperRootAtTimestamp> {
        let response =
            self.client.superroot_at_timestamp(timestamp, ResponseSelection::PreferData).await?;
        let root = SuperrootClient::super_root_at(&response, timestamp)?;
        Ok(SuperRootAtTimestamp { response, root })
    }
}

/// Production witness collection and SP1 proof execution.
pub(crate) struct ProductionProofEngine {
    provider: ProofProvider,
    host_inputs: Arc<HostInputs>,
    split: RangeSplitCount,
    max_concurrent: NonZeroUsize,
}

impl ProductionProofEngine {
    pub(crate) const fn new(
        provider: ProofProvider,
        host_inputs: Arc<HostInputs>,
        split: RangeSplitCount,
        max_concurrent: NonZeroUsize,
    ) -> Self {
        Self { provider, host_inputs, split, max_concurrent }
    }
}

#[async_trait]
impl ProofEngine for ProductionProofEngine {
    async fn prove(
        &self,
        keys: Option<Arc<ProofKeys>>,
        game: GameProofInputs,
        responses: Vec<SuperRootAtTimestampResponse>,
    ) -> Result<Vec<u8>> {
        prove_game_inner(
            &self.provider,
            keys.as_deref(),
            &self.host_inputs,
            &game,
            &responses,
            self.split,
            self.max_concurrent,
        )
        .await
    }
}

/// Production transaction construction, serialized submission, and receipt decoding.
pub(crate) struct ProductionActionExecutor<P> {
    signer: SignerLock,
    provider: P,
    factory: DisputeGameFactoryInstance<P>,
    l1_rpc: Url,
    confirmation_timeout: u64,
    fee_caps: FeeCaps,
}

impl<P> ProductionActionExecutor<P>
where
    P: Provider + Clone,
{
    pub(crate) const fn new(
        signer: SignerLock,
        provider: P,
        factory: DisputeGameFactoryInstance<P>,
        l1_rpc: Url,
        confirmation_timeout: u64,
        fee_caps: FeeCaps,
    ) -> Self {
        Self { signer, provider, factory, l1_rpc, confirmation_timeout, fee_caps }
    }

    async fn submit(&self, request: TransactionRequest) -> Result<TransactionReceipt> {
        self.signer
            .send_transaction_request_with_timeout(
                self.l1_rpc.clone(),
                request,
                self.confirmation_timeout,
                self.fee_caps,
            )
            .await
    }

    fn confirmed_hash(receipt: TransactionReceipt) -> Result<B256> {
        if !receipt.status() {
            bail!("{TX_REVERTED_PREFIX} {receipt:?}");
        }
        Ok(receipt.transaction_hash)
    }

    fn confirmed_creation(receipt: TransactionReceipt) -> Result<GameCreationReceipt> {
        if !receipt.status() {
            bail!("{TX_REVERTED_PREFIX} {receipt:?}");
        }
        let game_address = receipt
            .inner
            .logs()
            .iter()
            .find_map(|log| {
                DisputeGameCreated::decode_log(&log.inner).ok().map(|event| event.disputeProxy)
            })
            .context("Could not find DisputeGameCreated event in transaction receipt logs")?;
        Ok(GameCreationReceipt { game_address, transaction_hash: receipt.transaction_hash })
    }
}

#[async_trait]
impl<P> ActionExecutor for ProductionActionExecutor<P>
where
    P: Provider + Clone + Send + Sync + 'static,
{
    async fn create_game(
        &self,
        root_claim: B256,
        extra_data: Vec<u8>,
        init_bond: U256,
    ) -> Result<GameCreationReceipt> {
        let request = self
            .factory
            .create(ZK_GAME_TYPE, root_claim, extra_data.into())
            .value(init_bond)
            .into_transaction_request();
        Self::confirmed_creation(self.submit(request).await?)
    }

    async fn prove_game(&self, game: Address, proof: Vec<u8>) -> Result<B256> {
        let request = ZKDisputeGame::new(game, self.provider.clone())
            .prove(proof.into())
            .into_transaction_request();
        Self::confirmed_hash(self.submit(request).await?)
    }

    async fn resolve_game(&self, game: Address) -> Result<B256> {
        let request =
            ZKDisputeGame::new(game, self.provider.clone()).resolve().into_transaction_request();
        Self::confirmed_hash(self.submit(request).await?)
    }

    async fn claim_credit(&self, game: Address, recipient: Address) -> Result<B256> {
        // claimCredit's implicit closeGame can exceed the old 200k limit.
        let request = ZKDisputeGame::new(game, self.provider.clone())
            .claimCredit(recipient)
            .into_transaction_request();
        Self::confirmed_hash(self.submit(request).await?)
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
    use std::{num::NonZeroUsize, sync::Arc};

    use alloy_consensus::{Eip658Value, Receipt, ReceiptEnvelope, ReceiptWithBloom};
    use alloy_primitives::{Address, B256, Bloom, Bytes, LogData, U256};
    use alloy_provider::{ProviderBuilder, mock::Asserter};
    use alloy_rpc_types_eth::TransactionReceipt;
    use alloy_sol_types::SolValue;
    use kona_sp1_client_utils::super_root::hash_super_root_proof;
    use kona_sp1_super_range_executor::{
        BlockId as SuperRootBlockId, ChainId, ChainIdAndOutput, SuperRootAtTimestampResponse,
        SuperRootResponseData, SuperV1, proof_from_super_v1,
    };

    use super::*;
    use crate::{
        TxErrorExt, contract::DisputeGameFactory, ports::L1View, prover::MockProofProvider,
        proving::is_unprovable,
    };

    fn view(asserter: Asserter) -> ProductionL1View<crate::L1Provider> {
        let provider = ProviderBuilder::default().connect_mocked_client(asserter);
        let factory = DisputeGameFactory::new(Address::ZERO, provider.clone());
        ProductionL1View::new(provider, factory, "http://127.0.0.1:1".parse().unwrap())
    }

    fn push_abi(asserter: &Asserter, value: impl SolValue) {
        let encoded: Bytes = value.abi_encode().into();
        asserter.push_success(&encoded);
    }

    fn superroot_response(timestamp: u64) -> SuperRootAtTimestampResponse {
        let super_v1 = SuperV1 {
            timestamp,
            chains: vec![ChainIdAndOutput {
                chain_id: ChainId(U256::from(10)),
                output: B256::repeat_byte(0xab),
            }],
        };
        let proof = proof_from_super_v1(&super_v1).unwrap();
        let super_root = B256::from(*hash_super_root_proof(&proof).unwrap());
        SuperRootAtTimestampResponse {
            current_l1: SuperRootBlockId { number: 12, hash: B256::ZERO },
            current_safe_timestamp: timestamp + 2,
            current_local_safe_timestamp: timestamp + 1,
            current_finalized_timestamp: timestamp,
            optimistic_at_timestamp: Default::default(),
            chain_ids: vec![ChainId(U256::from(10))],
            data: Some(SuperRootResponseData {
                verified_required_l1: SuperRootBlockId { number: 11, hash: B256::ZERO },
                super_v1,
                super_root,
            }),
        }
    }

    fn receipt_with_logs(
        status: bool,
        transaction_hash: B256,
        logs: Vec<alloy_rpc_types_eth::Log>,
    ) -> TransactionReceipt {
        TransactionReceipt {
            inner: ReceiptEnvelope::Legacy(ReceiptWithBloom::new(
                Receipt { status: Eip658Value::Eip658(status), cumulative_gas_used: 0, logs },
                Bloom::ZERO,
            )),
            transaction_hash,
            transaction_index: None,
            block_hash: None,
            block_number: None,
            gas_used: 0,
            effective_gas_price: 0,
            blob_gas_used: None,
            blob_gas_price: None,
            from: Address::ZERO,
            to: None,
            contract_address: None,
        }
    }

    fn receipt(status: bool, transaction_hash: B256) -> TransactionReceipt {
        receipt_with_logs(status, transaction_hash, Vec::new())
    }

    fn game_created_log(game_address: Address, root_claim: B256) -> alloy_rpc_types_eth::Log {
        let address_topic = B256::left_padding_from(game_address.as_slice());
        let game_type_topic = B256::from(U256::from(ZK_GAME_TYPE).to_be_bytes::<32>());
        alloy_rpc_types_eth::Log {
            inner: alloy_primitives::Log {
                address: Address::ZERO,
                data: LogData::new_unchecked(
                    vec![
                        DisputeGameCreated::SIGNATURE_HASH,
                        address_topic,
                        game_type_topic,
                        root_claim,
                    ],
                    Bytes::new(),
                ),
            },
            block_hash: None,
            block_number: None,
            block_timestamp: None,
            transaction_hash: None,
            transaction_index: None,
            log_index: None,
            removed: false,
        }
    }

    #[test]
    fn action_receipt_preserves_success_and_revert_classification() {
        let transaction_hash = B256::repeat_byte(0x11);
        assert_eq!(
            ProductionActionExecutor::<crate::L1Provider>::confirmed_hash(receipt(
                true,
                transaction_hash
            ))
            .unwrap(),
            transaction_hash
        );
        let error = ProductionActionExecutor::<crate::L1Provider>::confirmed_hash(receipt(
            false,
            transaction_hash,
        ))
        .unwrap_err();
        assert!(error.is_revert());
    }

    #[test]
    fn creation_receipt_decodes_event_after_confirmed_success() {
        let transaction_hash = B256::repeat_byte(0x11);
        let game_address = Address::repeat_byte(0x22);
        let root_claim = B256::repeat_byte(0x33);
        let result =
            ProductionActionExecutor::<crate::L1Provider>::confirmed_creation(receipt_with_logs(
                true,
                transaction_hash,
                vec![game_created_log(game_address, root_claim)],
            ))
            .unwrap();
        assert_eq!(result.game_address, game_address);
        assert_eq!(result.transaction_hash, transaction_hash);

        let missing = ProductionActionExecutor::<crate::L1Provider>::confirmed_creation(receipt(
            true,
            transaction_hash,
        ))
        .unwrap_err();
        assert!(!missing.is_revert());

        let reverted = ProductionActionExecutor::<crate::L1Provider>::confirmed_creation(receipt(
            false,
            transaction_hash,
        ))
        .unwrap_err();
        assert!(reverted.is_revert());
    }

    #[tokio::test]
    async fn production_proof_engine_preserves_execution_error_classification() {
        let previous = superroot_response(100);
        let current = superroot_response(101);
        let game = GameProofInputs {
            l1_head: B256::repeat_byte(0x99),
            l1_head_number: 8,
            starting_root: previous.data.as_ref().unwrap().super_root,
            starting_ts: 100,
            root_claim: current.data.as_ref().unwrap().super_root,
            claim_ts: 101,
            prestate: B256::repeat_byte(0x77),
            prover: Address::repeat_byte(0x66),
        };
        let engine = ProductionProofEngine::new(
            ProofProvider::Mock(MockProofProvider),
            Arc::new(HostInputs {
                l1_node_address: "http://127.0.0.1:1".into(),
                l1_beacon_address: "http://127.0.0.1:1".into(),
                l2_node_addresses: vec!["http://127.0.0.1:1".into()],
                rollup_config_paths: None,
                l1_config_path: None,
                dependency_set_path: None,
            }),
            RangeSplitCount::one(),
            NonZeroUsize::MIN,
        );

        let error = engine.prove(None, game, vec![previous, current]).await.unwrap_err();
        assert!(!is_unprovable(&error));
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
