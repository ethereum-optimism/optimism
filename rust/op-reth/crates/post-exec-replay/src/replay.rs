use crate::types::{
    PostExecReplayBlock, PostExecReplayConfig, PostExecReplayMismatch, PostExecReplayMismatchKind,
    PostExecReplayPayload, PostExecReplayPayloadEntry, PostExecReplayRefundEvent,
    PostExecReplayRefundKind, PostExecReplaySummary, PostExecReplayTx,
};
use alloy_consensus::{
    Block as AlloyBlock, BlockBody, BlockHeader as AlloyBlockHeader, TxReceipt, Typed2718,
};
use op_alloy_consensus::{
    OpTransaction, POST_EXEC_PAYLOAD_VERSION, POST_EXEC_TX_TYPE_ID, PostExecPayload,
    PostExecPayloadValidationError, build_post_exec_tx, parse_post_exec_payload_from_transactions,
};
use reth_evm::{Database, execute::BlockExecutor};
use reth_execution_errors::BlockExecutionError;
use reth_node_api::NodePrimitives;
use reth_optimism_evm::{
    ConfigurePostExecEvm, PostExecExecutorExt, WarmingRefundEvent, WarmingRefundKind,
};
use reth_primitives_traits::{Block, BlockHeader, RecoveredBlock, SignedTransaction};
use revm::database::State;
use std::collections::{BTreeMap, BTreeSet};

/// Replay error.
#[derive(Debug, thiserror::Error)]
pub enum PostExecReplayError {
    /// Execution failed.
    #[error(transparent)]
    Execution(#[from] BlockExecutionError),
    /// Source block contains an invalid post-exec transaction structure.
    #[error(transparent)]
    InvalidPostExecPayload(#[from] PostExecPayloadValidationError),
}

/// Replay block with the synthetic post-exec transaction removed and original indexes retained.
pub(crate) type StrippedPostExecBlock<Tx, Header> =
    (RecoveredBlock<AlloyBlock<Tx, Header>>, Vec<u64>);

struct NormalizedBlock<Tx, Header>
where
    Tx: SignedTransaction + OpTransaction + Clone,
    Header: BlockHeader + AlloyBlockHeader + Clone,
    AlloyBlock<Tx, Header>: Block<Header = Header, Body = BlockBody<Tx, Header>>,
{
    replay_block: RecoveredBlock<AlloyBlock<Tx, Header>>,
    original_indexes: Vec<u64>,
    embedded_payload: Option<PostExecPayload>,
    post_exec_tx_index: Option<u64>,
}

/// Strip the synthetic post-exec tx from a block before replay while preserving original indexes.
pub fn strip_post_exec_tx_for_replay<Tx, Header>(
    block: &RecoveredBlock<AlloyBlock<Tx, Header>>,
) -> Result<StrippedPostExecBlock<Tx, Header>, PostExecReplayError>
where
    Tx: SignedTransaction + OpTransaction + Clone,
    Header: BlockHeader + AlloyBlockHeader + Clone,
    AlloyBlock<Tx, Header>: Block<Header = Header, Body = BlockBody<Tx, Header>>,
{
    let normalized = normalize_block(block)?;
    Ok((normalized.replay_block, normalized.original_indexes))
}

fn normalize_block<Tx, Header>(
    block: &RecoveredBlock<AlloyBlock<Tx, Header>>,
) -> Result<NormalizedBlock<Tx, Header>, PostExecReplayError>
where
    Tx: SignedTransaction + OpTransaction + Clone,
    Header: BlockHeader + AlloyBlockHeader + Clone,
    AlloyBlock<Tx, Header>: Block<Header = Header, Body = BlockBody<Tx, Header>>,
{
    let parsed_post_exec = parse_post_exec_payload_from_transactions(
        block.body().transactions.iter(),
        block.header().number(),
        // Replay is a debug/counterfactual tool, so the caller decides to run post-exec even for
        // blocks without an embedded payload. When a payload is embedded, still enforce the shared
        // structural rules (single trailing tx, anchored block number) used by consensus parsing.
        true,
    )?;
    let post_exec_tx_index = parsed_post_exec.as_ref().map(|parsed| parsed.tx_index);
    let embedded_payload = parsed_post_exec.map(|parsed| parsed.payload);

    let (raw_block, senders) = block.clone().split();
    let (header, body) = raw_block.split();
    let BlockBody { transactions, ommers, withdrawals } = body;

    let mut replay_transactions = Vec::with_capacity(transactions.len());
    let mut replay_senders = Vec::with_capacity(senders.len());
    let mut original_indexes = Vec::with_capacity(transactions.len());

    for (original_index, (tx, sender)) in
        transactions.into_iter().zip(senders.into_iter()).enumerate()
    {
        let original_index = original_index as u64;
        if Some(original_index) == post_exec_tx_index {
            continue;
        }

        original_indexes.push(original_index);
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

    Ok(NormalizedBlock { replay_block, original_indexes, embedded_payload, post_exec_tx_index })
}

const fn into_refund_kind(kind: WarmingRefundKind) -> PostExecReplayRefundKind {
    match kind {
        WarmingRefundKind::WarmAccount => PostExecReplayRefundKind::WarmAccount,
        WarmingRefundKind::WarmSload => PostExecReplayRefundKind::WarmSload,
        WarmingRefundKind::WarmSstore => PostExecReplayRefundKind::WarmSstore,
    }
}

fn original_tx_index(original_indexes: &[u64], replay_tx_index: u64) -> u64 {
    original_indexes.get(replay_tx_index as usize).copied().unwrap_or(replay_tx_index)
}

fn into_refund_event(
    event: WarmingRefundEvent,
    claiming_replay_tx_index: u64,
    original_indexes: &[u64],
) -> PostExecReplayRefundEvent {
    let first_warmed_by_replay_tx_index = event.first_warmed_by_tx_index;

    PostExecReplayRefundEvent {
        claiming_replay_tx_index,
        claiming_tx_index: original_tx_index(original_indexes, claiming_replay_tx_index),
        kind: into_refund_kind(event.kind),
        amount: event.amount,
        address: event.address,
        slot: event.slot,
        first_warmed_by_replay_tx_index,
        first_warmed_by_tx_index: original_tx_index(
            original_indexes,
            first_warmed_by_replay_tx_index,
        ),
    }
}

const fn replay_mismatch(
    category: PostExecReplayMismatchKind,
    block_num: u64,
    tx_index: Option<u64>,
    expected: Option<u64>,
    actual: Option<u64>,
    message: String,
) -> PostExecReplayMismatch {
    PostExecReplayMismatch { category, block_num, tx_index, expected, actual, message }
}

fn build_payload_map<Tx, Header>(
    block_number: u64,
    block: &RecoveredBlock<AlloyBlock<Tx, Header>>,
    payload: &PostExecPayload,
    mismatches: &mut Vec<PostExecReplayMismatch>,
) -> BTreeMap<u64, u64>
where
    Tx: SignedTransaction + OpTransaction + Typed2718,
    Header: BlockHeader,
    AlloyBlock<Tx, Header>: Block<Header = Header, Body = BlockBody<Tx, Header>>,
{
    let transactions = &block.body().transactions;
    let tx_count = transactions.len() as u64;
    let mut refunds = BTreeMap::new();
    let mut seen = BTreeSet::new();

    for entry in &payload.gas_refund_entries {
        let tx_index = entry.index;
        if !seen.insert(tx_index) {
            mismatches.push(replay_mismatch(
                PostExecReplayMismatchKind::DuplicatePayloadIndex,
                block_number,
                Some(tx_index),
                None,
                Some(entry.gas_refund),
                format!("duplicate payload entry for tx index {tx_index}"),
            ));
            continue;
        }

        if tx_index >= tx_count {
            mismatches.push(replay_mismatch(
                PostExecReplayMismatchKind::PayloadIndexOutOfRange,
                block_number,
                Some(tx_index),
                Some(tx_count.saturating_sub(1)),
                Some(tx_index),
                format!("payload entry targets out-of-range tx index {tx_index}"),
            ));
            continue;
        }

        let tx = &transactions[tx_index as usize];
        if tx.is_deposit() {
            mismatches.push(replay_mismatch(
                PostExecReplayMismatchKind::PayloadTargetsDeposit,
                block_number,
                Some(tx_index),
                Some(0),
                Some(entry.gas_refund),
                format!("payload entry targets deposit tx index {tx_index}"),
            ));
            continue;
        }

        if Typed2718::ty(tx) == POST_EXEC_TX_TYPE_ID {
            mismatches.push(replay_mismatch(
                PostExecReplayMismatchKind::PayloadTargetsPostExec,
                block_number,
                Some(tx_index),
                Some(0),
                Some(entry.gas_refund),
                format!("payload entry targets post-exec tx index {tx_index}"),
            ));
            continue;
        }

        refunds.insert(tx_index, entry.gas_refund);
    }

    refunds
}

fn into_replay_payload(payload: PostExecPayload) -> PostExecReplayPayload {
    PostExecReplayPayload {
        version: payload.version.into(),
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
        config,
    } = input;
    let mismatch_count = mismatches.len();

    if let Some(payload_refund) = payload_refund &&
        payload_refund > raw_gas_used
    {
        mismatches.push(replay_mismatch(
            PostExecReplayMismatchKind::PayloadRefundExceedsRawGas,
            block_number,
            Some(tx_index),
            Some(raw_gas_used),
            Some(payload_refund),
            format!("payload refund exceeds raw gas for tx index {tx_index}"),
        ));
    }

    if config.compare_payload && payload_refund.unwrap_or_default() != replay_refund {
        mismatches.push(replay_mismatch(
            PostExecReplayMismatchKind::PayloadRefundMismatch,
            block_number,
            Some(tx_index),
            payload_refund,
            Some(replay_refund),
            format!("payload refund mismatch for tx index {tx_index}"),
        ));
    }

    mismatches.len() != mismatch_count
}

/// Replay a historical block with post-exec enabled counterfactually.
pub fn replay_block<DB, EvmConfig, Tx, Header>(
    evm_config: &EvmConfig,
    db: DB,
    block: &RecoveredBlock<AlloyBlock<Tx, Header>>,
    config: PostExecReplayConfig,
) -> Result<PostExecReplayBlock, PostExecReplayError>
where
    DB: Database,
    EvmConfig: ConfigurePostExecEvm,
    EvmConfig::Primitives: NodePrimitives<
            Block = AlloyBlock<Tx, Header>,
            BlockBody = BlockBody<Tx, Header>,
            BlockHeader = Header,
            SignedTx = Tx,
        >,
    Tx: SignedTransaction + OpTransaction + Clone,
    Header: BlockHeader + AlloyBlockHeader + Clone,
    AlloyBlock<Tx, Header>: Block<Header = Header, Body = BlockBody<Tx, Header>>,
{
    let normalized = normalize_block(block)?;
    let block_number = block.header().number();
    let block_hash = block.hash();
    let post_exec_tx_present = normalized.post_exec_tx_index.is_some();

    let mut state = State::builder().with_database(db).with_bundle_update().build();
    let mut executor = evm_config
        .post_exec_executor_for_block(
            &mut state,
            normalized.replay_block.sealed_block(),
            reth_optimism_evm::PostExecMode::Produce,
        )
        .map_err(BlockExecutionError::other)?;

    executor.apply_pre_execution_changes()?;
    for tx in normalized.replay_block.clone_transactions_recovered() {
        executor.execute_transaction(tx)?;
    }
    let replay_entries = executor.take_post_exec_entries();
    let warming_events_by_tx = executor.take_warming_events_by_tx();
    let execution = executor.apply_post_execution_changes()?;

    let replay_payload = PostExecPayload {
        version: POST_EXEC_PAYLOAD_VERSION,
        block_number,
        gas_refund_entries: replay_entries.clone(),
    };
    let replay_refunds: BTreeMap<u64, u64> =
        replay_entries.iter().map(|entry| (entry.index, entry.gas_refund)).collect();

    let mut mismatches = Vec::new();
    let payload_refunds = match (config.compare_payload, normalized.embedded_payload.as_ref()) {
        (true, Some(payload)) => build_payload_map(block_number, block, payload, &mut mismatches),
        _ => BTreeMap::new(),
    };

    let mut txs = Vec::with_capacity(normalized.replay_block.body().transactions.len());
    let mut previous_cumulative_gas = 0_u64;

    for (replay_idx, (tx, &tx_index)) in normalized
        .replay_block
        .body()
        .transactions
        .iter()
        .zip(&normalized.original_indexes)
        .enumerate()
    {
        let replay_tx_index = replay_idx as u64;
        let cumulative_gas_used = execution.receipts[replay_idx].cumulative_gas_used();
        let canonical_gas_used = cumulative_gas_used.saturating_sub(previous_cumulative_gas);
        previous_cumulative_gas = cumulative_gas_used;

        let replay_refund = replay_refunds.get(&tx_index).copied().unwrap_or_default();
        let raw_gas_used = canonical_gas_used.saturating_add(replay_refund);
        let payload_refund = payload_refunds.get(&tx_index).copied();
        let refund_breakdown = warming_events_by_tx
            .get(replay_idx)
            .into_iter()
            .flatten()
            .copied()
            .map(|event| into_refund_event(event, replay_tx_index, &normalized.original_indexes))
            .collect();
        let mismatch = compare_refunds(
            CompareRefundsInput {
                block_number,
                tx_index,
                raw_gas_used,
                replay_refund,
                payload_refund,
                config: &config,
            },
            &mut mismatches,
        );

        txs.push(PostExecReplayTx {
            tx_index,
            replay_tx_index,
            tx_hash: *tx.tx_hash(),
            tx_type: tx.ty(),
            is_deposit_tx: tx.is_deposit(),
            raw_gas_used,
            canonical_gas_used,
            op_gas_refund_replay: replay_refund,
            op_gas_refund_payload: payload_refund,
            refund_breakdown,
            mismatch,
        });
    }

    let tx_count_user = txs.iter().filter(|tx| !tx.is_deposit_tx).count();
    let replay_refund_total = txs.iter().map(|tx| tx.op_gas_refund_replay).sum::<u64>();
    let payload_refund_total =
        txs.iter().map(|tx| tx.op_gas_refund_payload.unwrap_or_default()).sum::<u64>();
    let block_gas_used = txs.iter().map(|tx| tx.canonical_gas_used).sum::<u64>();
    let block_raw_gas_used = txs.iter().map(|tx| tx.raw_gas_used).sum::<u64>();

    let summary = PostExecReplaySummary {
        block_num: block_number,
        block_hash,
        tx_count_total: txs.len(),
        tx_count_user,
        post_exec_tx_present,
        post_exec_payload_entry_count: replay_entries.len(),
        block_gas_used,
        block_raw_gas_used,
        replay_refund_total,
        payload_refund_total,
        mismatch_count: mismatches.len(),
    };

    Ok(PostExecReplayBlock {
        config,
        block_num: block_number,
        block_hash,
        parent_hash: block.header().parent_hash(),
        post_exec_tx_present,
        post_exec_tx_index: normalized.post_exec_tx_index,
        embedded_payload: normalized.embedded_payload.map(into_replay_payload),
        synthesized_payload_bytes: build_post_exec_tx(block_number, replay_entries)
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
        CompareRefundsInput, PostExecReplayError, build_payload_map, compare_refunds,
        normalize_block, strip_post_exec_tx_for_replay,
    };
    use crate::{PostExecReplayConfig, PostExecReplayMismatchKind};
    use alloy_consensus::{BlockBody, Header, Sealable, SignableTransaction, TxLegacy};
    use alloy_primitives::{Address, Signature, U256};
    use op_alloy_consensus::{
        OpTxEnvelope, POST_EXEC_PAYLOAD_VERSION, TxDeposit, build_post_exec_tx,
    };
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

        let (replay_block, original_indexes) = strip_post_exec_tx_for_replay(&block).unwrap();
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

        let normalized = normalize_block(&block).unwrap();
        assert_eq!(normalized.post_exec_tx_index, Some(2));
        assert_eq!(normalized.original_indexes, vec![0, 1]);
        assert_eq!(normalized.embedded_payload.unwrap().gas_refund_entries, payload_entries);
        assert_eq!(normalized.replay_block.body().transactions.len(), 2);
    }

    #[test]
    fn normalize_block_reuses_shared_post_exec_structure_validation() {
        let post_exec: OpTransactionSigned =
            OpTransactionSigned::PostExec(build_post_exec_tx(0, vec![]).seal_slow());
        let user = user_tx();

        let block = RecoveredBlock::new_unhashed(
            alloy_consensus::Block::new(
                Header::default(),
                BlockBody {
                    transactions: vec![post_exec, user],
                    ommers: vec![],
                    withdrawals: None,
                },
            ),
            vec![Address::ZERO, Address::ZERO],
        );

        let err =
            normalize_block(&block).err().expect("non-trailing post-exec tx must fail validation");
        assert!(matches!(err, PostExecReplayError::InvalidPostExecPayload(_)));
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
            version: POST_EXEC_PAYLOAD_VERSION,
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
            mismatches.iter().map(|m| m.category).collect::<Vec<_>>(),
            vec![
                PostExecReplayMismatchKind::PayloadTargetsDeposit,
                PostExecReplayMismatchKind::PayloadTargetsPostExec,
                PostExecReplayMismatchKind::PayloadIndexOutOfRange,
                PostExecReplayMismatchKind::DuplicatePayloadIndex,
            ]
        );
    }

    #[test]
    fn compare_refunds_detects_tampered_payload_mismatches() {
        let config = PostExecReplayConfig { compare_payload: true };
        let mut mismatches = Vec::new();

        let mismatch = compare_refunds(
            CompareRefundsInput {
                block_number: 100,
                tx_index: 3,
                raw_gas_used: 40,
                replay_refund: 5,
                payload_refund: Some(7),
                config: &config,
            },
            &mut mismatches,
        );

        assert!(mismatch);
        assert_eq!(mismatches.len(), 1);
        assert_eq!(mismatches[0].category, PostExecReplayMismatchKind::PayloadRefundMismatch);
    }
}
