//! Sync core of post-Holocene payload-attribute building.
//!
//! Lifted from `attributes/stateful.rs::prepare_payload_attributes`. The
//! IO-free portion lives here; the async wrapper does the L1 header / L1
//! receipts / L2 sysconfig lookups, decodes deposits, applies sysconfig
//! updates, and then composes the payload via [`prepare_payload_attributes`].
//!
//! No IO and no `tracing::*`. Errors that the async wrapper logs (block
//! mismatch resets, time-invariant violations, malformed sysconfig updates,
//! malformed deposit logs) are surfaced as structured values via
//! [`PreparedAttributes`].

use crate::BuilderError;
use alloc::{string::ToString, sync::Arc, vec, vec::Vec};
use alloy_consensus::{Eip658Value, Header, Receipt};
use alloy_eips::{BlockNumHash, eip2718::Encodable2718};
use alloy_primitives::{Address, B256, Bytes};
use alloy_rlp::Encodable;
use alloy_rpc_types_engine::PayloadAttributes;
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_hardforks::{Hardfork, Hardforks, Interop};
use kona_interop::DependencySet;
use kona_protocol::{
    DEPOSIT_EVENT_ABI_HASH, L1BlockInfoTx, L2BlockInfo, Predeploys, decode_deposit,
};
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// Returns `true` if this epoch hits the epoch-boundary branch (the L1 origin
/// changed in this block, so the caller must fetch L1 receipts).
pub(crate) const fn needs_receipts(l2_parent: &L2BlockInfo, epoch: &BlockNumHash) -> bool {
    l2_parent.l1_origin.number != epoch.number
}

/// Validates the sync-only "L2 parent vs. epoch" relationship before any IO
/// happens. Returns the appropriate [`BuilderError`] on a mismatch, or `None`
/// when the pair is consistent.
///
/// The signature mirrors op-node's split: the same-epoch branch checks the
/// L2 parent's L1-origin hash against the epoch hash; the new-epoch branch
/// checks the parent's L1-origin hash against the *epoch header's parent
/// hash*. The caller passes the fetched header in for the second check; on
/// the same-epoch path it can pass `None` and we skip the header check.
pub(crate) fn check_block_mismatch(
    l2_parent: &L2BlockInfo,
    epoch: &BlockNumHash,
    fetched_header_parent_hash: Option<B256>,
) -> Option<BuilderError> {
    if l2_parent.l1_origin.number == epoch.number {
        (l2_parent.l1_origin.hash != epoch.hash)
            .then_some(BuilderError::BlockMismatch(*epoch, l2_parent.l1_origin))
    } else {
        let parent_hash = fetched_header_parent_hash
            .expect("epoch-boundary path requires fetched header for parent hash check");
        (l2_parent.l1_origin.hash != parent_hash).then_some(BuilderError::BlockMismatchEpochReset(
            *epoch,
            l2_parent.l1_origin,
            parent_hash,
        ))
    }
}

/// Derive deposits as `Vec<Bytes>` from transaction receipts.
///
/// Sync mirror of the previously-async helper at the bottom of
/// `attributes/stateful.rs`. The original was async only because it lived
/// inside an `async fn` — it never awaited anything.
pub(crate) fn derive_deposits(
    block_hash: B256,
    receipts: &[Receipt],
    deposit_contract: Address,
) -> Result<Vec<Bytes>, crate::PipelineEncodingError> {
    let mut global_index = 0;
    let mut res = Vec::new();
    for r in receipts {
        if Eip658Value::Eip658(false) == r.status {
            continue;
        }
        for l in &r.logs {
            let curr_index = global_index;
            global_index += 1;
            if l.data.topics().first().is_none_or(|i| *i != DEPOSIT_EVENT_ABI_HASH) {
                continue;
            }
            if l.address != deposit_contract {
                continue;
            }
            let decoded = decode_deposit(block_hash, curr_index, l)?;
            res.push(decoded);
        }
    }
    Ok(res)
}

/// Inputs for [`prepare_payload_attributes`]. The async wrapper assembles
/// these from its IO sources.
#[derive(Debug)]
pub(crate) struct PrepareInputs<'a> {
    /// Rollup config.
    pub rollup_cfg: &'a RollupConfig,
    /// L1 chain config.
    pub l1_cfg: &'a L1ChainConfig,
    /// L2 parent block info.
    pub l2_parent: L2BlockInfo,
    /// Sys config at the L2 parent's block number, with any in-block updates
    /// already applied by the wrapper.
    pub sys_config: SystemConfig,
    /// The L1 header at `epoch.hash` (fetched by the wrapper).
    pub l1_header: Header,
    /// Pre-derived deposit transactions. Empty on the same-epoch path.
    pub deposit_transactions: Vec<Bytes>,
    /// Sequence number for the next L2 block.
    pub sequence_number: u64,
    /// Optional interop dependency set.
    pub dependency_set: Option<&'a Arc<DependencySet>>,
}

/// Outcome of the sync core. Mirrors the post-IO errors of the async
/// `prepare_payload_attributes`.
#[derive(Debug)]
pub(crate) enum PreparedAttributes {
    /// Successfully built attributes.
    Ok(OpPayloadAttributes),
    /// Time invariant broken between L1 and L2.
    BrokenTimeInvariant(BuilderError),
    /// The L1 info transaction could not be built.
    L1InfoTxBuild(BuilderError),
}

/// Composes payload attributes from pre-fetched inputs. Mirrors
/// `op-node/rollup/derive/attributes.go` from the time-invariant check
/// onwards.
pub(crate) fn prepare_payload_attributes(inputs: PrepareInputs<'_>) -> PreparedAttributes {
    let PrepareInputs {
        rollup_cfg,
        l1_cfg,
        l2_parent,
        sys_config,
        l1_header,
        deposit_transactions,
        sequence_number,
        dependency_set,
    } = inputs;

    let next_l2_time = l2_parent.block_info.timestamp + rollup_cfg.block_time;
    if next_l2_time < l1_header.timestamp {
        return PreparedAttributes::BrokenTimeInvariant(BuilderError::BrokenTimeInvariant(
            l2_parent.l1_origin,
            next_l2_time,
            BlockNumHash { hash: l1_header.hash_slow(), number: l1_header.number },
            l1_header.timestamp,
        ));
    }

    let mut upgrade_transactions: Vec<Bytes> = if rollup_cfg.is_ecotone_active(next_l2_time) &&
        !rollup_cfg.is_ecotone_active(l2_parent.block_info.timestamp)
    {
        Hardforks::ECOTONE.txs().collect()
    } else {
        vec![]
    };
    if rollup_cfg.is_fjord_active(next_l2_time) &&
        !rollup_cfg.is_fjord_active(l2_parent.block_info.timestamp)
    {
        upgrade_transactions.append(&mut Hardforks::FJORD.txs().collect());
    }
    if rollup_cfg.is_isthmus_active(next_l2_time) &&
        !rollup_cfg.is_isthmus_active(l2_parent.block_info.timestamp)
    {
        upgrade_transactions.append(&mut Hardforks::ISTHMUS.txs().collect());
    }
    if rollup_cfg.is_jovian_active(next_l2_time) &&
        !rollup_cfg.is_jovian_active(l2_parent.block_info.timestamp)
    {
        upgrade_transactions.append(&mut Hardforks::JOVIAN.txs().collect());
    }
    // Starting with Karst, upgrade transactions carry their own gas budget
    // that is added to the block gas limit at the fork activation block.
    let mut upgrade_gas: u64 = 0;
    if rollup_cfg.is_karst_active(next_l2_time) &&
        !rollup_cfg.is_karst_active(l2_parent.block_info.timestamp)
    {
        upgrade_transactions.append(&mut Hardforks::KARST.txs().collect());
        upgrade_gas += Hardforks::KARST.upgrade_gas();
    }
    if rollup_cfg.is_interop_active(next_l2_time) &&
        !rollup_cfg.is_interop_active(l2_parent.block_info.timestamp)
    {
        upgrade_transactions.append(&mut Hardforks::INTEROP.txs().collect());

        let dep_set = dependency_set
            .expect("dependency_set must be Some when interop is active — constructor invariant");
        if dep_set.dependencies.len() > 1 {
            upgrade_transactions.extend(Interop::cross_l2_inbox_txs());
        }
    }

    let l1_info_tx_envelope = match L1BlockInfoTx::try_new_with_deposit_tx(
        rollup_cfg,
        l1_cfg,
        &sys_config,
        sequence_number,
        &l1_header,
        next_l2_time,
    ) {
        Ok((_, tx)) => tx,
        Err(e) => {
            return PreparedAttributes::L1InfoTxBuild(BuilderError::Custom(e.to_string()));
        }
    };
    let mut encoded_l1_info_tx = Vec::with_capacity(l1_info_tx_envelope.length());
    l1_info_tx_envelope.encode_2718(&mut encoded_l1_info_tx);

    let mut txs = Vec::with_capacity(1 + deposit_transactions.len() + upgrade_transactions.len());
    txs.push(encoded_l1_info_tx.into());
    txs.extend(deposit_transactions);
    txs.extend(upgrade_transactions);

    let withdrawals = rollup_cfg.is_canyon_active(next_l2_time).then(Vec::default);

    let parent_beacon_root = rollup_cfg
        .is_ecotone_active(next_l2_time)
        .then(|| l1_header.parent_beacon_block_root.unwrap_or_default());

    PreparedAttributes::Ok(OpPayloadAttributes {
        payload_attributes: PayloadAttributes {
            timestamp: next_l2_time,
            prev_randao: l1_header.mix_hash,
            suggested_fee_recipient: Predeploys::SEQUENCER_FEE_VAULT,
            parent_beacon_block_root: parent_beacon_root,
            withdrawals,
            slot_number: None,
        },
        transactions: Some(txs),
        no_tx_pool: Some(true),
        gas_limit: Some(
            u64::from_be_bytes(alloy_primitives::U64::from(sys_config.gas_limit).to_be_bytes()) +
                upgrade_gas,
        ),
        eip_1559_params: sys_config.eip_1559_params(
            rollup_cfg,
            l2_parent.block_info.timestamp,
            next_l2_time,
        ),
        min_base_fee: rollup_cfg
            .is_jovian_active(next_l2_time)
            .then(|| sys_config.min_base_fee.unwrap_or_default()),
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use kona_registry::L1Config;

    fn make_inputs<'a>(
        rcfg: &'a RollupConfig,
        l1cfg: &'a L1ChainConfig,
        l1_header: Header,
        _epoch: BlockNumHash,
        parent: L2BlockInfo,
    ) -> PrepareInputs<'a> {
        PrepareInputs {
            rollup_cfg: rcfg,
            l1_cfg: l1cfg,
            l2_parent: parent,
            sys_config: SystemConfig::default(),
            l1_header,
            deposit_transactions: vec![],
            sequence_number: 0,
            dependency_set: None,
        }
    }

    #[test]
    fn block_mismatch_same_epoch_diff_hash() {
        let parent = L2BlockInfo {
            l1_origin: BlockNumHash { hash: B256::left_padding_from(&[0xFF]), number: 1 },
            ..Default::default()
        };
        let epoch = BlockNumHash { hash: B256::ZERO, number: 1 };
        let err = check_block_mismatch(&parent, &epoch, None);
        assert!(matches!(err, Some(BuilderError::BlockMismatch(_, _))));
    }

    #[test]
    fn block_mismatch_epoch_boundary_bad_parent() {
        let parent = L2BlockInfo {
            l1_origin: BlockNumHash { hash: B256::left_padding_from(&[0xAA]), number: 1 },
            ..Default::default()
        };
        let epoch = BlockNumHash { hash: B256::ZERO, number: 2 };
        // Header.parent_hash != l2_parent.l1_origin.hash → triggers reset.
        let err = check_block_mismatch(&parent, &epoch, Some(B256::ZERO));
        assert!(matches!(err, Some(BuilderError::BlockMismatchEpochReset(..))));
    }

    #[test]
    fn block_mismatch_same_epoch_match() {
        let parent = L2BlockInfo {
            l1_origin: BlockNumHash { hash: B256::ZERO, number: 1 },
            ..Default::default()
        };
        let epoch = BlockNumHash { hash: B256::ZERO, number: 1 };
        assert!(check_block_mismatch(&parent, &epoch, None).is_none());
    }

    #[test]
    fn single_block_without_forks() {
        let rcfg = RollupConfig { block_time: 10, ..Default::default() };
        let l1cfg: L1ChainConfig = L1Config::sepolia().into();
        let header = Header { timestamp: 100, ..Default::default() };
        let hash = header.hash_slow();
        let epoch = BlockNumHash { hash, number: 1 };
        let parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: 1,
                timestamp: 100,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: 1 },
            seq_num: 0,
        };
        let inputs = make_inputs(&rcfg, &l1cfg, header, epoch, parent);
        match prepare_payload_attributes(inputs) {
            PreparedAttributes::Ok(attrs) => {
                assert_eq!(attrs.payload_attributes.timestamp, 110);
                assert_eq!(attrs.transactions.unwrap().len(), 1);
            }
            other => panic!("expected Ok, got {other:?}"),
        }
    }

    #[test]
    fn broken_time_invariant() {
        let rcfg = RollupConfig { block_time: 10, ..Default::default() };
        let l1cfg: L1ChainConfig = L1Config::sepolia().into();
        let header = Header { timestamp: 1_000, ..Default::default() };
        let hash = header.hash_slow();
        let epoch = BlockNumHash { hash, number: 1 };
        let parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: 1,
                timestamp: 100,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: 1 },
            seq_num: 0,
        };
        let inputs = make_inputs(&rcfg, &l1cfg, header, epoch, parent);
        let out = prepare_payload_attributes(inputs);
        assert!(matches!(out, PreparedAttributes::BrokenTimeInvariant(_)));
    }

    #[test]
    fn ecotone_activation_emits_upgrade_txs() {
        let rcfg = RollupConfig {
            block_time: 2,
            hardforks: HardForkConfig { ecotone_time: Some(102), ..Default::default() },
            ..Default::default()
        };
        let l1cfg: L1ChainConfig = L1Config::sepolia().into();
        let header = Header { timestamp: 100, ..Default::default() };
        let hash = header.hash_slow();
        let epoch = BlockNumHash { hash, number: 1 };
        let parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: 1,
                timestamp: 100,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: 1 },
            seq_num: 0,
        };
        let inputs = make_inputs(&rcfg, &l1cfg, header, epoch, parent);
        match prepare_payload_attributes(inputs) {
            PreparedAttributes::Ok(attrs) => {
                // 1 L1InfoTx + 6 ecotone upgrade txs = 7.
                assert_eq!(attrs.transactions.unwrap().len(), 7);
            }
            other => panic!("expected Ok, got {other:?}"),
        }
    }
}
