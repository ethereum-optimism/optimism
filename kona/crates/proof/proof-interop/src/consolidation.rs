//! Interop dependency resolution and consolidation logic.

use crate::{BootInfo, OptimisticBlock, OracleInteropProvider, PreState};
use alloc::vec::Vec;
use alloy_consensus::{Header, Sealed};
use alloy_eips::Encodable2718;
use alloy_evm::{EvmFactory, FromRecoveredTx, FromTxWithEncoded};
use alloy_op_evm::block::OpTxEnv;
use alloy_primitives::{Address, B256, Bytes, Sealable, TxKind, U256, address};
use alloy_rpc_types_engine::PayloadAttributes;
use core::fmt::Debug;
use kona_executor::{Eip1559ValidationError, ExecutorError, StatelessL2Builder};
use kona_interop::{MessageGraph, MessageGraphError};
use kona_mpt::OrderedListWalker;
use kona_preimage::CommsClient;
use kona_proof::{errors::OracleProviderError, l2::OracleL2ChainProvider};
use kona_protocol::OutputRoot;
use kona_registry::{HashMap, ROLLUP_CONFIGS};
use op_alloy_consensus::{InteropBlockReplacementDepositSource, OpTxEnvelope, OpTxType, TxDeposit};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use op_revm::OpSpecId;
use revm::context::BlockEnv;
use thiserror::Error;
use tracing::{error, info};

/// The [SuperchainConsolidator] holds a [MessageGraph] and is responsible for recursively
/// consolidating the blocks within the graph, per [message validity rules].
///
/// [message validity rules]: https://specs.optimism.io/interop/messaging.html#invalid-messages
#[derive(Debug)]
pub struct SuperchainConsolidator<'a, C, Evm>
where
    C: CommsClient,
{
    /// The [BootInfo] of the program.
    boot_info: &'a mut BootInfo,
    /// The [OracleInteropProvider] used for the message graph.
    interop_provider: OracleInteropProvider<C>,
    /// The [OracleL2ChainProvider]s used for re-execution of invalid blocks, keyed by chain ID.
    l2_providers: HashMap<u64, OracleL2ChainProvider<C>>,
    /// The inner [`EvmFactory`] to create EVM instances for re-execution of bad blocks.
    evm_factory: Evm,
}

impl<'a, C, Evm> SuperchainConsolidator<'a, C, Evm>
where
    C: CommsClient + Debug + Send + Sync,
    Evm: EvmFactory<Spec = OpSpecId, BlockEnv = BlockEnv> + Send + Sync + Debug + Clone + 'static,
    <Evm as EvmFactory>::Tx:
        FromTxWithEncoded<OpTxEnvelope> + FromRecoveredTx<OpTxEnvelope> + OpTxEnv,
{
    /// Creates a new [SuperchainConsolidator] with the given providers and [Header]s.
    ///
    /// [Header]: alloy_consensus::Header
    pub const fn new(
        boot_info: &'a mut BootInfo,
        interop_provider: OracleInteropProvider<C>,
        l2_providers: HashMap<u64, OracleL2ChainProvider<C>>,
        evm_factory: Evm,
    ) -> Self {
        Self { boot_info, interop_provider, l2_providers, evm_factory }
    }

    /// Recursively consolidates the dependencies of the blocks within the [MessageGraph].
    ///
    /// This method will recurse until all invalid cross-chain dependencies have been resolved,
    /// re-executing deposit-only blocks for chains with invalid dependencies as needed.
    pub async fn consolidate(&mut self) -> Result<(), ConsolidationError> {
        info!(target: "superchain_consolidator", "Consolidating superchain");

        loop {
            match self.consolidate_once().await {
                Ok(()) => {
                    info!(target: "superchain_consolidator", "Superchain consolidation complete");
                    return Ok(());
                }
                Err(ConsolidationError::MessageGraph(MessageGraphError::InvalidMessages(_))) => {
                    // If invalid messages are still present in the graph, continue the loop.
                    continue;
                }
                Err(e) => {
                    error!(target: "superchain_consolidator", "Error consolidating superchain: {:?}", e);
                    return Err(e);
                }
            }
        }
    }

    /// Performs a single iteration of the consolidation process.
    ///
    /// Step-wise:
    /// 1. Derive a new [MessageGraph] from the current set of local safe [Header]s.
    /// 2. Resolve the [MessageGraph].
    /// 3. If any invalid messages are found, re-execute the bad block(s) only deposit transactions,
    ///    and bubble up the error.
    ///
    /// [Header]: alloy_consensus::Header
    async fn consolidate_once(&mut self) -> Result<(), ConsolidationError> {
        // Derive the message graph from the current set of block headers.
        let graph = MessageGraph::derive(
            self.interop_provider.local_safe_heads(),
            &self.interop_provider,
            &self.boot_info.rollup_configs,
        )
        .await?;

        // Attempt to resolve the message graph. If there were any invalid messages found, we must
        // initiate a re-execution of the original block, with only deposit transactions.
        if let Err(MessageGraphError::InvalidMessages(invalid_chains)) = graph.resolve().await {
            self.re_execute_deposit_only(&invalid_chains.keys().copied().collect::<Vec<_>>())
                .await?;
            return Err(MessageGraphError::InvalidMessages(invalid_chains).into());
        }

        Ok(())
    }

    /// Re-executes the original blocks, keyed by their chain IDs, with only their deposit
    /// transactions.
    async fn re_execute_deposit_only(
        &mut self,
        chain_ids: &[u64],
    ) -> Result<(), ConsolidationError> {
        for chain_id in chain_ids {
            // Find the optimistic block header for the chain ID.
            let header = self
                .interop_provider
                .local_safe_heads()
                .get(chain_id)
                .ok_or(MessageGraphError::EmptyDependencySet)?;

            // Look up the parent header for the block.
            let parent_header =
                self.interop_provider.header_by_hash(*chain_id, header.parent_hash).await?;

            // Traverse the transactions trie of the block to re-execute.
            let trie_walker = OrderedListWalker::try_new_hydrated(
                header.transactions_root,
                &self.interop_provider,
            )
            .map_err(OracleProviderError::TrieWalker)?;
            let transactions = trie_walker.into_iter().map(|(_, rlp)| rlp).collect::<Vec<_>>();

            // Explicitly panic if a block sent off for re-execution already contains nothing but
            // deposits.
            assert!(
                !transactions.iter().all(|f| !f.is_empty() && f[0] == OpTxType::Deposit),
                "Impossible case; Block with only deposits found to be invalid. Something has gone horribly wrong!"
            );

            // Fetch the rollup config + provider for the current chain ID.
            let rollup_config = ROLLUP_CONFIGS
                .get(chain_id)
                .or_else(|| self.boot_info.rollup_configs.get(chain_id))
                .ok_or(ConsolidationError::MissingRollupConfig(*chain_id))?;
            let l2_provider = self
                .l2_providers
                .get(chain_id)
                .ok_or(ConsolidationError::MissingLocalProvider(*chain_id))?;

            let PreState::TransitionState(ref mut transition_state) =
                self.boot_info.agreed_pre_state
            else {
                return Err(ConsolidationError::InvalidPreStateVariant);
            };
            let original_optimistic_block = transition_state
                .pending_progress
                .iter_mut()
                .find(|block| block.block_hash == header.hash())
                .ok_or(MessageGraphError::EmptyDependencySet)?;

            // Filter out all transactions that are not deposits to start.
            let mut transactions = transactions
                .into_iter()
                .filter(|t| !t.is_empty() && t[0] == OpTxType::Deposit)
                .collect::<Vec<_>>();

            // Add the deposit replacement system transaction at the end of the list.
            transactions.push(Self::craft_replacement_transaction(
                header,
                original_optimistic_block.output_root,
            ));

            // Re-craft the execution payload, trimming off all non-deposit transactions.
            let deposit_only_payload = OpPayloadAttributes {
                payload_attributes: PayloadAttributes {
                    timestamp: header.timestamp,
                    prev_randao: header.mix_hash,
                    suggested_fee_recipient: header.beneficiary,
                    withdrawals: Default::default(),
                    parent_beacon_block_root: header.parent_beacon_block_root,
                },
                transactions: Some(transactions),
                no_tx_pool: Some(true),
                gas_limit: Some(header.gas_limit),
                eip_1559_params: rollup_config
                    .is_holocene_active(header.timestamp)
                    .then(|| {
                        // SAFETY: After the Holocene hardfork, blocks must have the EIP-1559
                        // parameters of the chain placed within the
                        // header's `extra_data` field. This slice index +
                        // conversion cannot fail unless the protocol rules
                        // have been violated.
                        header.extra_data.get(1..9).and_then(|s| s.try_into().ok()).ok_or(
                            ExecutorError::InvalidExtraData(Eip1559ValidationError::Decode(
                                op_alloy_consensus::EIP1559ParamError::NoEIP1559Params,
                            )),
                        )
                    })
                    .transpose()?,
                min_base_fee: rollup_config
                    .is_jovian_active(header.timestamp)
                    .then(|| {
                        header
                            .extra_data
                            .get(9..17)
                            .and_then(|s| <[u8; 8]>::try_from(s).ok())
                            .map(u64::from_be_bytes)
                            .ok_or(ExecutorError::InvalidExtraData(Eip1559ValidationError::Decode(
                                op_alloy_consensus::EIP1559ParamError::MinBaseFeeNotSet,
                            )))
                    })
                    .transpose()?,
            };

            // Create a new stateless L2 block executor for the current chain.
            let mut executor = StatelessL2Builder::new(
                rollup_config,
                self.evm_factory.clone(),
                l2_provider.clone(),
                l2_provider.clone(),
                parent_header.seal_slow(),
            );

            // Execute the block and take the new header. At this point, the block is guaranteed to
            // be canonical.
            let new_header = executor.build_block(deposit_only_payload)?.header;
            let new_output_root = executor.compute_output_root()?;

            // Replace the original optimistic block with the deposit only block.
            *original_optimistic_block = OptimisticBlock::new(new_header.hash(), new_output_root);

            // Replace the original header with the new header.
            self.interop_provider.replace_local_safe_head(*chain_id, new_header);
        }

        Ok(())
    }

    /// Forms the replacement transaction inserted into a deposit-only block in the event that a
    /// block is reduced due to invalid messages.
    ///
    /// <https://specs.optimism.io/interop/derivation.html#optimistic-block-deposited-transaction>
    fn craft_replacement_transaction(old_header: &Sealed<Header>, old_output_root: B256) -> Bytes {
        const REPLACEMENT_SENDER: Address = address!("deaddeaddeaddeaddeaddeaddeaddeaddead0002");
        const REPLACEMENT_GAS: u64 = 36000;

        let source = InteropBlockReplacementDepositSource::new(old_output_root);
        let output_root = OutputRoot::from_parts(
            old_header.state_root,
            old_header.withdrawals_root.unwrap_or_default(),
            old_header.hash(),
        );
        let replacement_tx = OpTxEnvelope::Deposit(
            TxDeposit {
                source_hash: source.source_hash(),
                from: REPLACEMENT_SENDER,
                to: TxKind::Call(Address::ZERO),
                mint: 0,
                value: U256::ZERO,
                gas_limit: REPLACEMENT_GAS,
                is_system_transaction: false,
                input: output_root.encode().into(),
            }
            .seal(),
        );

        replacement_tx.encoded_2718().into()
    }
}

/// An error type for the [SuperchainConsolidator] struct.
#[derive(Debug, Error)]
pub enum ConsolidationError {
    /// An invalid pre-state variant was passed to the consolidator.
    #[error("Invalid PreState variant")]
    InvalidPreStateVariant,
    /// Missing a rollup configuration.
    #[error("Missing rollup configuration for chain ID {0}")]
    MissingRollupConfig(u64),
    /// Missing a local L2 chain provider.
    #[error("Missing local L2 chain provider for chain ID {0}")]
    MissingLocalProvider(u64),
    /// An error occurred during consolidation.
    #[error(transparent)]
    MessageGraph(#[from] MessageGraphError<OracleProviderError>),
    /// An error occurred during execution.
    #[error(transparent)]
    Executor(#[from] ExecutorError),
    /// An error occurred during RLP decoding.
    #[error(transparent)]
    OracleProvider(#[from] OracleProviderError),
}
