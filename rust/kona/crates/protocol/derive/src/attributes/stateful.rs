//! The [`AttributesBuilder`] and it's default implementation.

use crate::{
    AttributesBuilder, ChainProvider, L2ChainProvider, PipelineError, PipelineErrorKind,
    PipelineResult,
    core::attributes::{
        PrepareInputs, PreparedAttributes, check_block_mismatch, derive_deposits, needs_receipts,
    },
};
use alloc::{boxed::Box, fmt::Debug, sync::Arc, vec, vec::Vec};
use alloy_eips::BlockNumHash;
use alloy_primitives::Bytes;
use async_trait::async_trait;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// A stateful implementation of the [`AttributesBuilder`].
#[derive(Debug, Default)]
pub struct StatefulAttributesBuilder<L1P, L2P>
where
    L1P: ChainProvider + Debug,
    L2P: L2ChainProvider + Debug,
{
    /// The rollup config.
    rollup_cfg: Arc<RollupConfig>,
    /// The L1 config.
    l1_cfg: Arc<L1ChainConfig>,
    /// The system config fetcher.
    config_fetcher: L2P,
    /// The L1 receipts fetcher.
    receipts_fetcher: L1P,
    /// Optional interop dependency set. Required when interop is scheduled for the
    /// chain (`rollup_cfg.hardforks.interop_time.is_some()`); ignored otherwise.
    dependency_set: Option<Arc<DependencySet>>,
}

impl<L1P, L2P> StatefulAttributesBuilder<L1P, L2P>
where
    L1P: ChainProvider + Debug,
    L2P: L2ChainProvider + Debug,
{
    /// Create a new [`StatefulAttributesBuilder`].
    ///
    /// # Panics
    ///
    /// Panics if `rcfg.hardforks.interop_time.is_some() && dependency_set.is_none()`.
    /// A chain that has interop scheduled must have a dependency set provided,
    /// otherwise the builder would silently diverge from op-node on interop
    /// activation (emitting different number of upgrade transactions, or the wrong
    /// ordering). See issue #19311.
    pub fn new(
        rcfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        sys_cfg_fetcher: L2P,
        receipts: L1P,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        assert!(
            !(rcfg.hardforks.interop_time.is_some() && dependency_set.is_none()),
            "StatefulAttributesBuilder: interop is scheduled for this chain \
             (interop_time = {:?}) but no DependencySet was provided. \
             This would silently diverge from op-node on interop activation.",
            rcfg.hardforks.interop_time,
        );
        Self {
            rollup_cfg: rcfg,
            l1_cfg,
            config_fetcher: sys_cfg_fetcher,
            receipts_fetcher: receipts,
            dependency_set,
        }
    }
}

#[async_trait]
impl<L1P, L2P> AttributesBuilder for StatefulAttributesBuilder<L1P, L2P>
where
    L1P: ChainProvider + Debug + Send,
    L2P: L2ChainProvider + Debug + Send,
{
    async fn prepare_payload_attributes(
        &mut self,
        l2_parent: L2BlockInfo,
        epoch: BlockNumHash,
    ) -> PipelineResult<OpPayloadAttributes> {
        let mut sys_config = self
            .config_fetcher
            .system_config_by_number(l2_parent.block_info.number, self.rollup_cfg.clone())
            .await
            .map_err(Into::into)?;

        let l1_header =
            self.receipts_fetcher.header_by_hash(epoch.hash).await.map_err(Into::into)?;

        // Sync block-mismatch check — short-circuits before deposit IO when
        // the L2 parent / epoch pair is inconsistent.
        let header_parent_for_check =
            needs_receipts(&l2_parent, &epoch).then_some(l1_header.parent_hash);
        if let Some(err) = check_block_mismatch(&l2_parent, &epoch, header_parent_for_check) {
            return Err(PipelineErrorKind::Reset(err.into()));
        }

        let (deposit_transactions, sequence_number): (Vec<Bytes>, u64) = if needs_receipts(
            &l2_parent, &epoch,
        ) {
            let receipts =
                self.receipts_fetcher.receipts_by_hash(epoch.hash).await.map_err(Into::into)?;
            let deposits =
                derive_deposits(epoch.hash, &receipts, self.rollup_cfg.deposit_contract_address)
                    .map_err(|e| PipelineError::BadEncoding(e).crit())?;
            let (updates, errors) = sys_config.update_with_receipts(
                &receipts,
                self.rollup_cfg.l1_system_config_address,
                self.rollup_cfg.is_ecotone_active(l1_header.timestamp),
            );
            for kind in &updates {
                info!(target: "attributes", epoch = epoch.number, %kind, "Applied system config update");
            }
            for err in &errors {
                warn!(target: "attributes", ?err, epoch = epoch.number, "Malformed system config update (skipped)");
            }
            (deposits, 0)
        } else {
            (vec![], l2_parent.seq_num + 1)
        };

        match crate::core::attributes::prepare_payload_attributes(PrepareInputs {
            rollup_cfg: &self.rollup_cfg,
            l1_cfg: &self.l1_cfg,
            l2_parent,
            sys_config,
            l1_header,
            deposit_transactions,
            sequence_number,
            dependency_set: self.dependency_set.as_ref(),
        }) {
            PreparedAttributes::Ok(attrs) => Ok(attrs),
            PreparedAttributes::BrokenTimeInvariant(err) => {
                Err(PipelineErrorKind::Reset(err.into()))
            }
            PreparedAttributes::L1InfoTxBuild(err) => {
                Err(PipelineError::AttributesBuilder(err).crit())
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        BuilderError,
        errors::ResetError,
        test_utils::{TestChainProvider, TestSystemConfigL2Fetcher},
    };
    use alloc::vec;
    use alloy_consensus::{Eip658Value, Header, Receipt};
    use alloy_primitives::{B256, Bytes, Log, LogData, U64, U256, address};
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_genesis::{CONFIG_UPDATE_TOPIC, HardForkConfig, SystemConfig};
    use kona_protocol::{BlockInfo, DEPOSIT_EVENT_ABI_HASH, DepositError, Predeploys};
    use kona_registry::L1Config;

    fn generate_valid_log() -> Log {
        let deposit_contract = address!("1111111111111111111111111111111111111111");
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(128).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        // Copy the u128 mint value
        let mint: [u8; 16] = 10_u128.to_be_bytes();
        data[80..96].copy_from_slice(&mint);
        // Copy the tx value
        let value: [u8; 32] = U256::from(100).to_be_bytes();
        data[96..128].copy_from_slice(&value);
        // Copy the gas limit
        let gas: [u8; 8] = 1000_u64.to_be_bytes();
        data[128..136].copy_from_slice(&gas);
        // Copy the isCreation flag
        data[136] = 1;
        let from = address!("2222222222222222222222222222222222222222");
        let mut from_bytes = vec![0u8; 32];
        from_bytes[12..32].copy_from_slice(from.as_slice());
        let to = address!("3333333333333333333333333333333333333333");
        let mut to_bytes = vec![0u8; 32];
        to_bytes[12..32].copy_from_slice(to.as_slice());
        Log {
            address: deposit_contract,
            data: LogData::new_unchecked(
                vec![
                    DEPOSIT_EVENT_ABI_HASH,
                    B256::from_slice(&from_bytes),
                    B256::from_slice(&to_bytes),
                    B256::default(),
                ],
                Bytes::from(data),
            ),
        }
    }

    fn generate_valid_receipt() -> Receipt {
        let mut bad_dest_log = generate_valid_log();
        bad_dest_log.data.topics_mut()[1] = B256::default();
        let mut invalid_topic_log = generate_valid_log();
        invalid_topic_log.data.topics_mut()[0] = B256::default();
        Receipt {
            status: Eip658Value::Eip658(true),
            logs: vec![generate_valid_log(), bad_dest_log, invalid_topic_log],
            ..Default::default()
        }
    }

    #[test]
    fn test_derive_deposits_empty() {
        let receipts = vec![];
        let deposit_contract = address!("0000000000000000000000000000000000000000");
        let result = derive_deposits(B256::default(), &receipts, deposit_contract);
        assert!(result.unwrap().is_empty());
    }

    #[test]
    fn test_derive_deposits_non_deposit_events_filtered_out() {
        let deposit_contract = address!("1111111111111111111111111111111111111111");
        let mut invalid = generate_valid_receipt();
        invalid.logs[0].data = LogData::new_unchecked(vec![], Bytes::default());
        let receipts = vec![generate_valid_receipt(), generate_valid_receipt(), invalid];
        let result = derive_deposits(B256::default(), &receipts, deposit_contract);
        assert_eq!(result.unwrap().len(), 5);
    }

    #[test]
    fn test_derive_deposits_non_deposit_contract_addr() {
        let deposit_contract = address!("1111111111111111111111111111111111111111");
        let mut invalid = generate_valid_receipt();
        invalid.logs[0].address = alloy_primitives::Address::default();
        let receipts = vec![generate_valid_receipt(), generate_valid_receipt(), invalid];
        let result = derive_deposits(B256::default(), &receipts, deposit_contract);
        assert_eq!(result.unwrap().len(), 5);
    }

    #[test]
    fn test_derive_deposits_decoding_errors() {
        let deposit_contract = address!("1111111111111111111111111111111111111111");
        let mut invalid = generate_valid_receipt();
        invalid.logs[0].data =
            LogData::new_unchecked(vec![DEPOSIT_EVENT_ABI_HASH], Bytes::default());
        let receipts = vec![generate_valid_receipt(), generate_valid_receipt(), invalid];
        let result = derive_deposits(B256::default(), &receipts, deposit_contract);
        let downcasted = result.unwrap_err();
        assert_eq!(downcasted, DepositError::UnexpectedTopicsLen(1).into());
    }

    #[test]
    fn test_derive_deposits_succeeds() {
        let deposit_contract = address!("1111111111111111111111111111111111111111");
        let receipts = vec![generate_valid_receipt(), generate_valid_receipt()];
        let result = derive_deposits(B256::default(), &receipts, deposit_contract);
        assert_eq!(result.unwrap().len(), 4);
    }

    #[tokio::test]
    async fn test_prepare_payload_block_mismatch_epoch_reset() {
        let cfg = Arc::new(RollupConfig::default());
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header::default();
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo { hash: B256::ZERO, number: l2_number, ..Default::default() },
            l1_origin: BlockNumHash { hash: B256::left_padding_from(&[0xFF]), number: 2 },
            seq_num: 0,
        };
        let expected =
            BuilderError::BlockMismatchEpochReset(epoch, l2_parent.l1_origin, B256::default());
        let err = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap_err();
        assert_eq!(err, PipelineErrorKind::Reset(expected.into()));
    }

    #[tokio::test]
    async fn test_prepare_payload_block_mismatch() {
        let cfg = Arc::new(RollupConfig::default());
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header::default();
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo { hash: B256::ZERO, number: l2_number, ..Default::default() },
            l1_origin: BlockNumHash { hash: B256::ZERO, number: l2_number },
            seq_num: 0,
        };
        let expected = BuilderError::BlockMismatch(epoch, l2_parent.l1_origin);
        let err = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap_err();
        assert_eq!(err, PipelineErrorKind::Reset(ResetError::AttributesBuilder(expected)));
    }

    #[tokio::test]
    async fn test_prepare_payload_broken_time_invariant() {
        let block_time = 10;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig { block_time, ..Default::default() });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo { hash: B256::ZERO, number: l2_number, ..Default::default() },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let next_l2_time = l2_parent.block_info.timestamp + block_time;
        let block_id = BlockNumHash { hash, number: 0 };
        let expected = BuilderError::BrokenTimeInvariant(
            l2_parent.l1_origin,
            next_l2_time,
            block_id,
            timestamp,
        );
        let err = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap_err();
        assert_eq!(err, PipelineErrorKind::Reset(ResetError::AttributesBuilder(expected)));
    }

    #[tokio::test]
    async fn test_prepare_payload_without_forks() {
        let block_time = 10;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig { block_time, ..Default::default() });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let prev_randao = header.mix_hash;
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let next_l2_time = l2_parent.block_info.timestamp + block_time;
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        let expected = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: next_l2_time,
                prev_randao,
                suggested_fee_recipient: Predeploys::SEQUENCER_FEE_VAULT,
                parent_beacon_block_root: None,
                withdrawals: None,
                slot_number: None,
            },
            transactions: payload.transactions.clone(),
            no_tx_pool: Some(true),
            gas_limit: Some(u64::from_be_bytes(
                alloy_primitives::U64::from(SystemConfig::default().gas_limit).to_be_bytes(),
            )),
            eip_1559_params: None,
            min_base_fee: None,
        };
        assert_eq!(payload, expected);
        assert_eq!(payload.transactions.unwrap().len(), 1);
    }

    #[tokio::test]
    async fn test_prepare_payload_with_canyon() {
        let block_time = 10;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig { canyon_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let prev_randao = header.mix_hash;
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let next_l2_time = l2_parent.block_info.timestamp + block_time;
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        let expected = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: next_l2_time,
                prev_randao,
                suggested_fee_recipient: Predeploys::SEQUENCER_FEE_VAULT,
                parent_beacon_block_root: None,
                withdrawals: Some(Vec::default()),
                slot_number: None,
            },
            transactions: payload.transactions.clone(),
            no_tx_pool: Some(true),
            gas_limit: Some(u64::from_be_bytes(
                alloy_primitives::U64::from(SystemConfig::default().gas_limit).to_be_bytes(),
            )),
            eip_1559_params: None,
            min_base_fee: None,
        };
        assert_eq!(payload, expected);
        assert_eq!(payload.transactions.unwrap().len(), 1);
    }

    #[tokio::test]
    async fn test_prepare_payload_with_ecotone() {
        let block_time = 2;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig { ecotone_time: Some(102), ..Default::default() },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let parent_beacon_block_root = Some(header.parent_beacon_block_root.unwrap_or_default());
        let prev_randao = header.mix_hash;
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let next_l2_time = l2_parent.block_info.timestamp + block_time;
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        let expected = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: next_l2_time,
                prev_randao,
                suggested_fee_recipient: Predeploys::SEQUENCER_FEE_VAULT,
                parent_beacon_block_root,
                withdrawals: Some(vec![]),
                slot_number: None,
            },
            transactions: payload.transactions.clone(),
            no_tx_pool: Some(true),
            gas_limit: Some(u64::from_be_bytes(
                alloy_primitives::U64::from(SystemConfig::default().gas_limit).to_be_bytes(),
            )),
            eip_1559_params: None,
            min_base_fee: None,
        };
        assert_eq!(payload, expected);
        assert_eq!(payload.transactions.unwrap().len(), 7);
    }

    #[tokio::test]
    async fn test_prepare_payload_with_fjord() {
        let block_time = 2;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig { fjord_time: Some(102), ..Default::default() },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let prev_randao = header.mix_hash;
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let mut builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let next_l2_time = l2_parent.block_info.timestamp + block_time;
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        let expected = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: next_l2_time,
                prev_randao,
                suggested_fee_recipient: Predeploys::SEQUENCER_FEE_VAULT,
                parent_beacon_block_root: Some(B256::ZERO),
                withdrawals: Some(vec![]),
                slot_number: None,
            },
            transactions: payload.transactions.clone(),
            no_tx_pool: Some(true),
            gas_limit: Some(u64::from_be_bytes(
                alloy_primitives::U64::from(SystemConfig::default().gas_limit).to_be_bytes(),
            )),
            eip_1559_params: None,
            min_base_fee: None,
        };
        assert_eq!(payload.transactions.as_ref().unwrap().len(), 10);
        assert_eq!(payload, expected);
    }

    #[tokio::test]
    async fn test_syscfg_update_error_is_nonfatal() {
        // When a receipt contains a malformed system config log, update_with_receipts
        // returns an error. This must NOT halt the pipeline — the error should be
        // logged as a warning and attributes building should continue successfully.
        // This matches op-node's attributes.go:97-99 which uses a blank identifier
        // assignment: `_ = UpdateSystemConfigWithL1Receipts(...)`.
        let block_time = 10;
        let timestamp = 100;
        let l1_sys_config_addr = address!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
        let deposit_contract = address!("1111111111111111111111111111111111111111");
        let cfg = Arc::new(RollupConfig {
            block_time,
            l1_system_config_address: l1_sys_config_addr,
            deposit_contract_address: deposit_contract,
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());

        // L2 parent is at number 1, its l1_origin is at genesis (number 0, hash = parent_hash
        // of the epoch header). The epoch is at number 1 — different from l1_origin.number,
        // so the code enters the first-block-of-epoch branch that fetches receipts.
        let epoch_header = Header { timestamp, ..Default::default() };
        let epoch_hash = epoch_header.hash_slow();
        let parent_origin_hash = epoch_header.parent_hash;

        let mut provider = TestChainProvider::default();
        provider.insert_header(epoch_hash, epoch_header);

        // Insert a malformed receipt: the log is from the system config address but has
        // only 1 topic (CONFIG_UPDATE_TOPIC) instead of the required >= 3 topics.
        // This causes update_with_receipts to return Err(InvalidTopicLen(1)).
        let bad_log = Log {
            address: l1_sys_config_addr,
            data: LogData::new_unchecked(vec![CONFIG_UPDATE_TOPIC], Bytes::default()),
        };
        let bad_receipt = Receipt {
            status: Eip658Value::Eip658(true),
            logs: vec![bad_log],
            ..Receipt::default()
        };
        provider.insert_receipts(epoch_hash, vec![bad_receipt]);

        let l2_number = 1u64;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());

        let mut builder =
            StatefulAttributesBuilder::new(cfg.clone(), l1_cfg, fetcher, provider, None);
        let epoch = BlockNumHash { hash: epoch_hash, number: 1 };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: epoch_hash,
            },
            l1_origin: BlockNumHash { hash: parent_origin_hash, number: 0 },
            seq_num: 0,
        };
        let result = builder.prepare_payload_attributes(l2_parent, epoch).await;
        assert!(
            result.is_ok(),
            "system config update failure should be non-fatal, got: {:?}",
            result.unwrap_err()
        );
    }

    // ---------------------------------------------------------------------------
    // Interop activation gating tests (issue #19311)
    //
    // These tests verify that kona's interop activation tx stream byte-matches
    // op-node's output: 7 base txs always, + 2 CrossL2Inbox txs only when the
    // dependency set has >1 chain. Go reference: op-node/rollup/derive/
    // attributes.go:171-185.
    // ---------------------------------------------------------------------------

    fn build_interop_dep_set(chain_count: usize) -> Arc<DependencySet> {
        use alloc::collections::BTreeMap;
        use kona_interop::ChainDependency;
        // `ChainDependency` is zero-sized today; the set's cardinality is what
        // gates the CrossL2Inbox predeploy pair, not the values.
        #[allow(clippy::zero_sized_map_values)]
        let dependencies =
            (1..=chain_count as u64).map(|id| (id, ChainDependency {})).collect::<BTreeMap<_, _>>();
        Arc::new(DependencySet { dependencies, override_message_expiry_window: None })
    }

    /// Interop-activated, single-chain `dep-set` → base 7 interop txs, no `CrossL2Inbox` pair.
    #[tokio::test]
    async fn test_prepare_payload_with_interop_single_chain() {
        let block_time = 2;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig {
                regolith_time: Some(50),
                canyon_time: Some(50),
                delta_time: Some(50),
                ecotone_time: Some(50),
                fjord_time: Some(50),
                granite_time: Some(50),
                holocene_time: Some(50),
                isthmus_time: Some(50),
                jovian_time: Some(50),
                karst_time: Some(50),
                interop_time: Some(102),
                ..Default::default()
            },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let dep_set = build_interop_dep_set(1);
        let mut builder =
            StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, Some(dep_set));
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        // 1 L1InfoTx + 7 interop base txs + 0 CrossL2Inbox txs = 8.
        assert_eq!(payload.transactions.unwrap().len(), 1 + 7);
    }

    /// Interop-activated, multi-chain `dep-set` → base 7 + `CrossL2Inbox` 2 = 9 upgrade txs.
    #[tokio::test]
    async fn test_prepare_payload_with_interop_multi_chain() {
        let block_time = 2;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig {
                regolith_time: Some(50),
                canyon_time: Some(50),
                delta_time: Some(50),
                ecotone_time: Some(50),
                fjord_time: Some(50),
                granite_time: Some(50),
                holocene_time: Some(50),
                isthmus_time: Some(50),
                jovian_time: Some(50),
                karst_time: Some(50),
                interop_time: Some(102),
                ..Default::default()
            },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let dep_set = build_interop_dep_set(2);
        let mut builder =
            StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, Some(dep_set));
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        // 1 L1InfoTx + 7 interop base txs + 2 CrossL2Inbox txs = 10.
        assert_eq!(payload.transactions.unwrap().len(), 1 + 7 + 2);
    }

    /// Single-chain interop: ordering matches Go — base txs [0..7] equal the
    /// golden `interop_base_tx_*.hex` files.
    #[tokio::test]
    async fn test_prepare_payload_with_interop_base_tx_ordering_matches_go() {
        use alloy_primitives::hex;
        let block_time = 2;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig {
                regolith_time: Some(50),
                canyon_time: Some(50),
                delta_time: Some(50),
                ecotone_time: Some(50),
                fjord_time: Some(50),
                granite_time: Some(50),
                holocene_time: Some(50),
                isthmus_time: Some(50),
                jovian_time: Some(50),
                karst_time: Some(50),
                interop_time: Some(102),
                ..Default::default()
            },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let dep_set = build_interop_dep_set(1);
        let mut builder =
            StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, Some(dep_set));
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        let txs = payload.transactions.unwrap();
        let expected: [&str; 7] = [
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_0.hex"),
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_1.hex"),
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_2.hex"),
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_3.hex"),
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_4.hex"),
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_5.hex"),
            include_str!("../../../hardforks/src/bytecode/interop_base_tx_6.hex"),
        ];
        for (i, expected_hex) in expected.iter().enumerate() {
            let expected_bytes: Bytes = hex::decode(expected_hex.replace('\n', "")).unwrap().into();
            // txs[0] is the L1 info tx; interop base txs start at txs[1].
            assert_eq!(txs[1 + i], expected_bytes, "interop base tx {i} diverges from Go");
        }
    }

    /// Multi-chain interop: `CrossL2Inbox` pair appended at tail (positions 8..10),
    /// matching op-node's ordering.
    #[tokio::test]
    async fn test_prepare_payload_with_interop_multi_chain_appends_cross_l2_inbox_at_tail() {
        use alloy_primitives::hex;
        let block_time = 2;
        let timestamp = 100;
        let cfg = Arc::new(RollupConfig {
            block_time,
            hardforks: HardForkConfig {
                regolith_time: Some(50),
                canyon_time: Some(50),
                delta_time: Some(50),
                ecotone_time: Some(50),
                fjord_time: Some(50),
                granite_time: Some(50),
                holocene_time: Some(50),
                isthmus_time: Some(50),
                jovian_time: Some(50),
                karst_time: Some(50),
                interop_time: Some(102),
                ..Default::default()
            },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let l2_number = 1;
        let mut fetcher = TestSystemConfigL2Fetcher::default();
        fetcher.insert(l2_number, SystemConfig::default());
        let mut provider = TestChainProvider::default();
        let header = Header { timestamp, ..Default::default() };
        let hash = header.hash_slow();
        provider.insert_header(hash, header);
        let dep_set = build_interop_dep_set(2);
        let mut builder =
            StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, Some(dep_set));
        let epoch = BlockNumHash { hash, number: l2_number };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: l2_number,
                timestamp,
                parent_hash: hash,
            },
            l1_origin: BlockNumHash { hash, number: l2_number },
            seq_num: 0,
        };
        let payload = builder.prepare_payload_attributes(l2_parent, epoch).await.unwrap();
        let txs = payload.transactions.unwrap();
        let expected_0: Bytes = hex::decode(
            include_str!("../../../hardforks/src/bytecode/interop_cross_l2_inbox_tx_0.hex")
                .replace('\n', ""),
        )
        .unwrap()
        .into();
        let expected_1: Bytes = hex::decode(
            include_str!("../../../hardforks/src/bytecode/interop_cross_l2_inbox_tx_1.hex")
                .replace('\n', ""),
        )
        .unwrap()
        .into();
        // txs[0]=L1Info, txs[1..=7]=base, txs[8..=9]=CrossL2Inbox.
        assert_eq!(txs[8], expected_0);
        assert_eq!(txs[9], expected_1);
    }

    /// Constructor panics fast when interop is scheduled but no dependency set was provided.
    #[test]
    #[should_panic(expected = "no DependencySet was provided")]
    fn test_stateful_builder_new_panics_when_interop_scheduled_without_dependency_set() {
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { interop_time: Some(100), ..Default::default() },
            ..Default::default()
        });
        let l1_cfg = Arc::new(L1Config::sepolia().into());
        let fetcher = TestSystemConfigL2Fetcher::default();
        let provider = TestChainProvider::default();
        let _builder = StatefulAttributesBuilder::new(cfg, l1_cfg, fetcher, provider, None);
    }
}
