use crate::{
    metrics::SDMReplayMetrics,
    types::{
        PostExecReplayBlock, PostExecReplayConfig, PostExecReplayMismatch,
        PostExecReplayMismatchKind, PostExecReplayMode, PostExecReplayPayload,
        PostExecReplayPayloadEntry, PostExecReplayRefundEvent, PostExecReplayRefundKind,
        PostExecReplaySummary, PostExecReplayTx,
    },
};
use alloy_consensus::{Block as AlloyBlock, BlockBody, BlockHeader, TxReceipt, Typed2718};
use op_alloy_consensus::{POST_EXEC_TX_TYPE_ID, PostExecPayload, SDMGasEntry, build_post_exec_tx};
use reth_evm::{Database, execute::BlockExecutor};
use reth_execution_errors::BlockExecutionError;
use reth_optimism_evm::{
    ConfigurePostExecEvm, PostExecExecutorExt, WarmingRefundEvent, WarmingRefundKind,
};
use reth_optimism_primitives::{OpBlock, OpPrimitives, OpTransactionSigned};
use reth_primitives_traits::{Block, RecoveredBlock};
use revm::database::{State, states::bundle_state::BundleRetention};
use std::collections::{BTreeMap, BTreeSet};

/// Replay error.
#[derive(Debug, thiserror::Error)]
pub enum PostExecReplayError {
    /// Unsupported replay configuration.
    #[error("unsupported replay mode: {0:?}")]
    UnsupportedMode(PostExecReplayMode),
    /// Execution failed.
    #[error(transparent)]
    Execution(#[from] BlockExecutionError),
}

#[derive(Debug, Clone)]
struct NormalizedBlock {
    replay_block: RecoveredBlock<OpBlock>,
    original_indexes: Vec<u64>,
    embedded_payload: Option<PostExecPayload>,
    post_exec_tx_index: Option<u64>,
}

/// Strip the synthetic post-exec tx from a block before replay while preserving original indexes.
pub fn strip_post_exec_tx_for_replay(
    block: &RecoveredBlock<OpBlock>,
) -> (RecoveredBlock<OpBlock>, Vec<u64>) {
    let normalized = normalize_block(block);
    (normalized.replay_block, normalized.original_indexes)
}

fn normalize_block(block: &RecoveredBlock<OpBlock>) -> NormalizedBlock {
    let (raw_block, senders) = block.clone().split();
    let (header, body) = raw_block.split();
    let BlockBody { transactions, ommers, withdrawals } = body;

    let mut replay_transactions = Vec::with_capacity(transactions.len());
    let mut replay_senders = Vec::with_capacity(senders.len());
    let mut original_indexes = Vec::with_capacity(transactions.len());
    let mut embedded_payload = None;
    let mut post_exec_tx_index = None;

    for (idx, (tx, sender)) in transactions.into_iter().zip(senders.into_iter()).enumerate() {
        if tx.ty() == POST_EXEC_TX_TYPE_ID {
            post_exec_tx_index = Some(idx as u64);
            if let OpTransactionSigned::PostExec(post_exec) = &tx {
                embedded_payload = Some(post_exec.inner().payload.clone());
            }
            continue;
        }

        original_indexes.push(idx as u64);
        replay_transactions.push(tx);
        replay_senders.push(sender);
    }

    let replay_block = RecoveredBlock::new_unhashed(
        AlloyBlock::new(
            header,
            BlockBody { transactions: replay_transactions, ommers, withdrawals },
        ),
        replay_senders,
    );

    NormalizedBlock { replay_block, original_indexes, embedded_payload, post_exec_tx_index }
}

const fn into_refund_kind(kind: WarmingRefundKind) -> PostExecReplayRefundKind {
    match kind {
        WarmingRefundKind::WarmAccount => PostExecReplayRefundKind::WarmAccount,
        WarmingRefundKind::WarmSload => PostExecReplayRefundKind::WarmSload,
        WarmingRefundKind::WarmSstore => PostExecReplayRefundKind::WarmSstore,
    }
}

fn into_refund_event(
    event: WarmingRefundEvent,
    claiming_replay_tx_index: u64,
    original_indexes: &[u64],
) -> PostExecReplayRefundEvent {
    let first_warmed_by_replay_tx_index = event.first_warmed_by_tx_index;
    let claiming_tx_index = original_indexes
        .get(claiming_replay_tx_index as usize)
        .copied()
        .unwrap_or(claiming_replay_tx_index);
    let first_warmed_by_tx_index = original_indexes
        .get(first_warmed_by_replay_tx_index as usize)
        .copied()
        .unwrap_or(first_warmed_by_replay_tx_index);

    PostExecReplayRefundEvent {
        claiming_replay_tx_index,
        claiming_tx_index,
        kind: into_refund_kind(event.kind),
        amount: event.amount,
        address: event.address,
        slot: event.slot,
        first_warmed_by_replay_tx_index,
        first_warmed_by_tx_index,
    }
}

fn build_payload_map(
    block_number: u64,
    block: &RecoveredBlock<OpBlock>,
    payload: &PostExecPayload,
    mismatches: &mut Vec<PostExecReplayMismatch>,
) -> BTreeMap<u64, u64> {
    let mut refunds = BTreeMap::new();
    let mut seen = BTreeSet::new();
    let tx_count = block.body().transactions.len() as u64;

    for entry in &payload.gas_refund_entries {
        if !seen.insert(entry.index) {
            mismatches.push(PostExecReplayMismatch {
                category: PostExecReplayMismatchKind::DuplicatePayloadIndex,
                block_num: block_number,
                tx_index: Some(entry.index),
                expected: None,
                actual: Some(entry.gas_refund),
                message: format!("duplicate payload entry for tx index {}", entry.index),
            });
            continue;
        }

        if entry.index >= tx_count {
            mismatches.push(PostExecReplayMismatch {
                category: PostExecReplayMismatchKind::PayloadIndexOutOfRange,
                block_num: block_number,
                tx_index: Some(entry.index),
                expected: Some(tx_count.saturating_sub(1)),
                actual: Some(entry.index),
                message: format!("payload entry targets out-of-range tx index {}", entry.index),
            });
            continue;
        }

        let tx = &block.body().transactions[entry.index as usize];
        if tx.is_deposit() {
            mismatches.push(PostExecReplayMismatch {
                category: PostExecReplayMismatchKind::PayloadTargetsDeposit,
                block_num: block_number,
                tx_index: Some(entry.index),
                expected: Some(0),
                actual: Some(entry.gas_refund),
                message: format!("payload entry targets deposit tx index {}", entry.index),
            });
            continue;
        }

        if tx.ty() == POST_EXEC_TX_TYPE_ID {
            mismatches.push(PostExecReplayMismatch {
                category: PostExecReplayMismatchKind::PayloadTargetsPostExec,
                block_num: block_number,
                tx_index: Some(entry.index),
                expected: Some(0),
                actual: Some(entry.gas_refund),
                message: format!("payload entry targets post-exec tx index {}", entry.index),
            });
            continue;
        }

        refunds.insert(entry.index, entry.gas_refund);
    }

    refunds
}

fn into_replay_payload(payload: PostExecPayload) -> PostExecReplayPayload {
    PostExecReplayPayload {
        version: payload.version,
        block_number: payload.block_number,
        gas_refund_entries: payload
            .gas_refund_entries
            .into_iter()
            .map(|entry| PostExecReplayPayloadEntry {
                index: entry.index,
                gas_refund: entry.gas_refund,
            })
            .collect(),
    }
}

struct CompareRefundsInput<'a> {
    block_number: u64,
    tx_index: u64,
    raw_gas_used: u64,
    replay_refund: u64,
    payload_refund: Option<u64>,
    receipt_refund: Option<u64>,
    config: &'a PostExecReplayConfig,
}

fn compare_refunds(
    input: CompareRefundsInput<'_>,
    mismatches: &mut Vec<PostExecReplayMismatch>,
) -> bool {
    let CompareRefundsInput {
        block_number,
        tx_index,
        raw_gas_used,
        replay_refund,
        payload_refund,
        receipt_refund,
        config,
    } = input;
    let mut mismatch = false;

    if let Some(payload_refund) = payload_refund &&
        payload_refund > raw_gas_used
    {
        mismatch = true;
        mismatches.push(PostExecReplayMismatch {
            category: PostExecReplayMismatchKind::PayloadRefundExceedsRawGas,
            block_num: block_number,
            tx_index: Some(tx_index),
            expected: Some(raw_gas_used),
            actual: Some(payload_refund),
            message: format!("payload refund exceeds raw gas for tx index {}", tx_index),
        });
    }

    if config.compare_payload && payload_refund.unwrap_or_default() != replay_refund {
        mismatch = true;
        mismatches.push(PostExecReplayMismatch {
            category: PostExecReplayMismatchKind::PayloadRefundMismatch,
            block_num: block_number,
            tx_index: Some(tx_index),
            expected: payload_refund,
            actual: Some(replay_refund),
            message: format!("payload refund mismatch for tx index {}", tx_index),
        });
    }

    if config.compare_receipts && receipt_refund.unwrap_or_default() != replay_refund {
        mismatch = true;
        mismatches.push(PostExecReplayMismatch {
            category: PostExecReplayMismatchKind::ReceiptRefundMismatch,
            block_num: block_number,
            tx_index: Some(tx_index),
            expected: receipt_refund,
            actual: Some(replay_refund),
            message: format!("receipt refund mismatch for tx index {}", tx_index),
        });
    }

    mismatch
}

/// Replay a historical block with post-exec enabled counterfactually.
pub fn replay_block<DB, EvmConfig>(
    evm_config: &EvmConfig,
    db: DB,
    block: &RecoveredBlock<OpBlock>,
    config: PostExecReplayConfig,
) -> Result<PostExecReplayBlock, PostExecReplayError>
where
    DB: Database,
    EvmConfig: ConfigurePostExecEvm<Primitives = OpPrimitives>,
{
    let metrics = SDMReplayMetrics::default();

    if config.mode != PostExecReplayMode::CounterfactualEnabled {
        metrics.record_block(&[PostExecReplayMismatchKind::UnsupportedMode]);
        return Err(PostExecReplayError::UnsupportedMode(config.mode));
    }

    let normalized = normalize_block(block);

    let mut state = State::builder().with_database(db).with_bundle_update().build();
    let mut executor = evm_config
        .post_exec_executor_for_block(
            &mut state,
            normalized.replay_block.sealed_block(),
            reth_optimism_evm::PostExecMode::Produce,
        )
        .map_err(BlockExecutionError::other)?;

    executor.apply_pre_execution_changes()?;
    for tx in normalized.replay_block.transactions_recovered() {
        executor.execute_transaction(tx)?;
    }
    let replay_entries: Vec<SDMGasEntry> = executor.take_post_exec_entries();
    let warming_events_by_tx = executor.take_warming_events_by_tx();
    let execution = executor.apply_post_execution_changes()?;

    state.merge_transitions(BundleRetention::Reverts);

    let replay_payload = PostExecPayload {
        version: 1,
        block_number: block.header().number(),
        gas_refund_entries: replay_entries.clone(),
    };
    let replay_refunds: BTreeMap<u64, u64> =
        replay_entries.iter().map(|entry| (entry.index, entry.gas_refund)).collect();

    let mut mismatches = Vec::new();
    let payload_refunds = normalized
        .embedded_payload
        .as_ref()
        .map(|payload| build_payload_map(block.header().number(), block, payload, &mut mismatches))
        .unwrap_or_default();

    let receipt_refunds = payload_refunds.clone();
    let mut txs = Vec::with_capacity(normalized.replay_block.body().transactions.len());
    let mut previous_cumulative_gas = 0_u64;

    for (replay_idx, tx) in normalized.replay_block.body().transactions.iter().enumerate() {
        let tx_index = normalized.original_indexes[replay_idx];
        let cumulative_gas_used = execution.receipts[replay_idx].cumulative_gas_used();
        let canonical_gas_used = cumulative_gas_used.saturating_sub(previous_cumulative_gas);
        previous_cumulative_gas = cumulative_gas_used;

        let replay_refund = replay_refunds.get(&tx_index).copied().unwrap_or_default();
        let raw_gas_used = canonical_gas_used.saturating_add(replay_refund);
        let payload_refund = payload_refunds.get(&tx_index).copied();
        let receipt_refund = receipt_refunds.get(&tx_index).copied();
        let refund_breakdown = warming_events_by_tx
            .get(replay_idx)
            .cloned()
            .unwrap_or_default()
            .into_iter()
            .map(|event| into_refund_event(event, replay_idx as u64, &normalized.original_indexes))
            .collect::<Vec<_>>();
        let mismatch = compare_refunds(
            CompareRefundsInput {
                block_number: block.header().number(),
                tx_index,
                raw_gas_used,
                replay_refund,
                payload_refund,
                receipt_refund,
                config: &config,
            },
            &mut mismatches,
        );

        txs.push(PostExecReplayTx {
            tx_index,
            replay_tx_index: replay_idx as u64,
            tx_hash: tx.tx_hash(),
            tx_type: tx.ty(),
            is_deposit_tx: tx.is_deposit(),
            gas_used: canonical_gas_used,
            raw_gas_used,
            canonical_gas_used,
            op_gas_refund_replay: replay_refund,
            op_gas_refund_payload: payload_refund,
            op_gas_refund_receipt: receipt_refund,
            effective_gas: canonical_gas_used,
            refund_breakdown,
            mismatch,
        });
    }

    let tx_count_user = txs.iter().filter(|tx| !tx.is_deposit_tx).count();
    let replay_refund_total = txs.iter().map(|tx| tx.op_gas_refund_replay).sum::<u64>();
    let payload_refund_total =
        txs.iter().map(|tx| tx.op_gas_refund_payload.unwrap_or_default()).sum::<u64>();
    let receipt_refund_total =
        txs.iter().map(|tx| tx.op_gas_refund_receipt.unwrap_or_default()).sum::<u64>();
    let block_gas_used = txs.iter().map(|tx| tx.gas_used).sum::<u64>();
    let block_raw_gas_used = txs.iter().map(|tx| tx.raw_gas_used).sum::<u64>();

    let summary = PostExecReplaySummary {
        block_num: block.header().number(),
        block_hash: block.hash(),
        tx_count_total: txs.len(),
        tx_count_user,
        post_exec_tx_present: normalized.post_exec_tx_index.is_some(),
        post_exec_payload_entry_count: replay_entries.len(),
        block_gas_used,
        block_raw_gas_used,
        replay_refund_total,
        payload_refund_total,
        node_receipt_refund_total: receipt_refund_total,
        block_effective_gas: block_gas_used,
        mismatch_count: mismatches.len(),
        replay_mode: config.mode,
    };

    metrics.record_block(&mismatches.iter().map(|m| m.category.clone()).collect::<Vec<_>>());

    Ok(PostExecReplayBlock {
        config,
        block_num: block.header().number(),
        block_hash: block.hash(),
        parent_hash: block.header().parent_hash(),
        post_exec_tx_present: normalized.post_exec_tx_index.is_some(),
        post_exec_tx_index: normalized.post_exec_tx_index,
        embedded_payload: normalized.embedded_payload.map(into_replay_payload),
        synthesized_payload_bytes: build_post_exec_tx(block.header().number(), replay_entries)
            .payload
            .to_rlp_bytes(),
        synthesized_payload: into_replay_payload(replay_payload),
        txs,
        mismatches,
        summary,
    })
}

#[cfg(test)]
mod tests {
    use super::{
        CompareRefundsInput, build_payload_map, compare_refunds, normalize_block,
        strip_post_exec_tx_for_replay,
    };
    use crate::{PostExecReplayConfig, PostExecReplayMismatchKind, PostExecReplayMode};
    use alloy_consensus::{BlockBody, Header, Sealable, SignableTransaction, TxLegacy};
    use alloy_primitives::{Address, Signature, U256};
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit, build_post_exec_tx};
    use reth_optimism_primitives::OpTransactionSigned;
    use reth_primitives_traits::RecoveredBlock;

    fn user_tx() -> OpTransactionSigned {
        OpTxEnvelope::Legacy(TxLegacy::default().into_signed(Signature::new(
            U256::ZERO,
            U256::ZERO,
            false,
        )))
    }

    #[test]
    fn strips_post_exec_tx_and_preserves_original_indexes() {
        let deposit: OpTransactionSigned = OpTxEnvelope::Deposit(TxDeposit::default().seal_slow());
        let user = user_tx();
        let post_exec: OpTransactionSigned =
            OpTransactionSigned::PostExec(build_post_exec_tx(0, vec![]).seal_slow());

        let block = RecoveredBlock::new_unhashed(
            alloy_consensus::Block::new(
                Header::default(),
                BlockBody {
                    transactions: vec![deposit, user, post_exec],
                    ommers: vec![],
                    withdrawals: None,
                },
            ),
            vec![Address::ZERO, Address::ZERO, Address::ZERO],
        );

        let (replay_block, original_indexes) = strip_post_exec_tx_for_replay(&block);
        assert_eq!(replay_block.body().transactions.len(), 2);
        assert_eq!(original_indexes, vec![0, 1]);
    }

    #[test]
    fn normalize_block_extracts_embedded_payload_and_post_exec_index() {
        let deposit: OpTransactionSigned = OpTxEnvelope::Deposit(TxDeposit::default().seal_slow());
        let user = user_tx();
        let payload_entries = vec![op_alloy_consensus::SDMGasEntry { index: 1, gas_refund: 9 }];
        let post_exec: OpTransactionSigned = OpTransactionSigned::PostExec(
            build_post_exec_tx(0, payload_entries.clone()).seal_slow(),
        );

        let block = RecoveredBlock::new_unhashed(
            alloy_consensus::Block::new(
                Header::default(),
                BlockBody {
                    transactions: vec![deposit, user, post_exec],
                    ommers: vec![],
                    withdrawals: None,
                },
            ),
            vec![Address::ZERO, Address::ZERO, Address::ZERO],
        );

        let normalized = normalize_block(&block);
        assert_eq!(normalized.post_exec_tx_index, Some(2));
        assert_eq!(normalized.original_indexes, vec![0, 1]);
        assert_eq!(normalized.embedded_payload.unwrap().gas_refund_entries, payload_entries);
        assert_eq!(normalized.replay_block.body().transactions.len(), 2);
    }

    #[test]
    fn build_payload_map_reports_invalid_targets_and_duplicates() {
        let deposit: OpTransactionSigned = OpTxEnvelope::Deposit(TxDeposit::default().seal_slow());
        let user = user_tx();
        let post_exec: OpTransactionSigned =
            OpTransactionSigned::PostExec(build_post_exec_tx(0, vec![]).seal_slow());
        let block = RecoveredBlock::new_unhashed(
            alloy_consensus::Block::new(
                Header::default(),
                BlockBody {
                    transactions: vec![deposit, user, post_exec],
                    ommers: vec![],
                    withdrawals: None,
                },
            ),
            vec![Address::ZERO, Address::ZERO, Address::ZERO],
        );
        let payload = op_alloy_consensus::PostExecPayload {
            version: 1,
            block_number: 100,
            gas_refund_entries: vec![
                op_alloy_consensus::SDMGasEntry { index: 0, gas_refund: 1 },
                op_alloy_consensus::SDMGasEntry { index: 2, gas_refund: 2 },
                op_alloy_consensus::SDMGasEntry { index: 8, gas_refund: 3 },
                op_alloy_consensus::SDMGasEntry { index: 1, gas_refund: 4 },
                op_alloy_consensus::SDMGasEntry { index: 1, gas_refund: 5 },
            ],
        };

        let mut mismatches = Vec::new();
        let refunds = build_payload_map(100, &block, &payload, &mut mismatches);

        assert_eq!(refunds.get(&1), Some(&4));
        assert_eq!(refunds.len(), 1);
        assert_eq!(
            mismatches.iter().map(|m| m.category.clone()).collect::<Vec<_>>(),
            vec![
                PostExecReplayMismatchKind::PayloadTargetsDeposit,
                PostExecReplayMismatchKind::PayloadTargetsPostExec,
                PostExecReplayMismatchKind::PayloadIndexOutOfRange,
                PostExecReplayMismatchKind::DuplicatePayloadIndex,
            ]
        );
    }

    #[test]
    fn compare_refunds_detects_tampered_payload_and_receipt_mismatches() {
        let config = PostExecReplayConfig {
            mode: PostExecReplayMode::CounterfactualEnabled,
            compare_payload: true,
            compare_receipts: true,
        };
        let mut mismatches = Vec::new();

        let mismatch = compare_refunds(
            CompareRefundsInput {
                block_number: 100,
                tx_index: 3,
                raw_gas_used: 40,
                replay_refund: 5,
                payload_refund: Some(7),
                receipt_refund: Some(7),
                config: &config,
            },
            &mut mismatches,
        );

        assert!(mismatch);
        assert_eq!(mismatches.len(), 2);
        assert_eq!(mismatches[0].category, PostExecReplayMismatchKind::PayloadRefundMismatch);
        assert_eq!(mismatches[1].category, PostExecReplayMismatchKind::ReceiptRefundMismatch);
    }
}
