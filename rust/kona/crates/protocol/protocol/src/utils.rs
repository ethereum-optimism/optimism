//! Utility methods used by protocol types.

use alloc::vec::Vec;
use alloy_consensus::{Transaction, Typed2718};
use alloy_op_hardforks::OpHardfork;
use alloy_primitives::{B256, U256};
use alloy_rlp::{Buf, Header};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_hardforks::{Hardfork, Hardforks};
use op_alloy_consensus::{OpBlock, OpTxType, decode_holocene_extra_data, decode_jovian_extra_data};

use crate::{
    L1BlockInfoBedrockOnlyFields as _, L1BlockInfoEcotoneBaseFields as _, L1BlockInfoTx,
    MAX_SPAN_BATCH_ELEMENTS, OpBlockConversionError, SpanBatchError, SpanDecodingError,
};

/// Converts the [`OpBlock`] to a partial [`SystemConfig`].
pub fn to_system_config(
    block: &OpBlock,
    rollup_config: &RollupConfig,
) -> Result<SystemConfig, OpBlockConversionError> {
    if block.header.number == rollup_config.genesis.l2.number {
        if block.header.hash_slow() != rollup_config.genesis.l2.hash {
            return Err(OpBlockConversionError::InvalidGenesisHash(
                rollup_config.genesis.l2.hash,
                block.header.hash_slow(),
            ));
        }
        return rollup_config
            .genesis
            .system_config
            .ok_or(OpBlockConversionError::MissingSystemConfigGenesis);
    }

    if block.body.transactions.is_empty() {
        return Err(OpBlockConversionError::EmptyTransactions(block.header.hash_slow()));
    }
    let Some(tx) = block.body.transactions[0].as_deposit() else {
        return Err(OpBlockConversionError::InvalidTxType(block.body.transactions[0].ty()));
    };

    let l1_info = L1BlockInfoTx::decode_calldata(tx.input().as_ref())?;
    let l1_fee_scalar = match l1_info {
        L1BlockInfoTx::Bedrock(block_info) => block_info.l1_fee_scalar(),
        L1BlockInfoTx::Ecotone(block_info) => {
            encode_scalar(block_info.blob_base_fee_scalar(), block_info.base_fee_scalar())
        }
        L1BlockInfoTx::Isthmus(block_info) => {
            encode_scalar(block_info.blob_base_fee_scalar(), block_info.base_fee_scalar())
        }
        L1BlockInfoTx::Jovian(block_info) => {
            encode_scalar(block_info.blob_base_fee_scalar(), block_info.base_fee_scalar())
        }
    };

    let mut cfg = SystemConfig {
        batcher_address: l1_info.batcher_address(),
        overhead: l1_info.l1_fee_overhead(),
        scalar: l1_fee_scalar,
        gas_limit: block.header.gas_limit,
        ..Default::default()
    };

    // Starting with Karst, each NUT-bundle fork adds one-time upgrade gas to its activation block's
    // gas limit so the upgrade transactions exceed the normal budget. Subtract it back here so the
    // reconstructed config holds the steady-state limit. This runs during reconstruction, before
    // update_with_receipts in the caller, so a setGasLimit in the same block's L1 origin takes
    // precedence. Mirrors op-node's PayloadToSystemConfig.
    let gas_to_strip = upgrade_gas_to_strip(rollup_config, block.header.timestamp);
    if gas_to_strip > 0 {
        cfg.gas_limit = cfg.gas_limit.checked_sub(gas_to_strip).ok_or(
            OpBlockConversionError::GasLimitBelowUpgradeGas {
                gas_limit: cfg.gas_limit,
                upgrade_gas: gas_to_strip,
            },
        )?;
    }

    // After holocene's activation, the EIP-1559 parameters are stored in the block header's nonce.
    if rollup_config.is_jovian_active(block.header.timestamp) {
        let (elasticity, denominator, min_base_fee) =
            decode_jovian_extra_data(&block.header.extra_data)?;
        cfg.eip1559_denominator = Some(denominator);
        cfg.eip1559_elasticity = Some(elasticity);
        cfg.min_base_fee = Some(min_base_fee);
    } else if rollup_config.is_holocene_active(block.header.timestamp) {
        let (elasticity, denominator) = decode_holocene_extra_data(&block.header.extra_data)?;
        cfg.eip1559_denominator = Some(denominator);
        cfg.eip1559_elasticity = Some(elasticity);
    }

    if rollup_config.is_isthmus_active(block.header.timestamp) {
        cfg.operator_fee_scalar = Some(l1_info.operator_fee_scalar());
        cfg.operator_fee_constant = Some(l1_info.operator_fee_constant());
    }

    if let Some(da_footprint) = l1_info.da_footprint() {
        cfg.da_footprint_gas_scalar = Some(da_footprint);
    }

    Ok(cfg)
}

/// Returns the one-time NUT-bundle upgrade gas to subtract from the system config reconstructed
/// from a block with the given timestamp. Starting with Karst, upgrade gas is added to a fork's
/// activation block to accommodate its NUT bundle; subtracting it again at the next block reverts
/// the gas limit to the steady state. Karst can opt out via `keep_karst_upgrade_gas` (for chains
/// that activated Karst with the leak baked into their history); later forks have no opt-out.
fn upgrade_gas_to_strip(rollup_config: &RollupConfig, block_timestamp: u64) -> u64 {
    for fork in OpHardfork::Karst.forks_from() {
        if !rollup_config.is_fork_activation_block(fork, block_timestamp) {
            continue;
        }
        // At most one fork activates per block, so the first activation fork found is the only one.
        if fork == OpHardfork::Karst && rollup_config.hardforks.keep_karst_upgrade_gas {
            return 0;
        }
        return upgrade_gas(fork);
    }
    0
}

/// Returns the one-time upgrade gas a NUT-bundle fork adds to its activation block. Forks without a
/// NUT bundle (everything before Karst) add none.
///
/// [`OpHardfork`] is `#[non_exhaustive]`, so an exhaustive match that fails to compile when a new
/// fork is added is not possible here; the `upgrade_gas_covers_known_forks` test guards instead,
/// failing when a new variant appears so that its upgrade gas is reviewed.
pub fn upgrade_gas(fork: OpHardfork) -> u64 {
    match fork {
        OpHardfork::Karst => Hardforks::KARST.upgrade_gas(),
        OpHardfork::Lagoon => Hardforks::LAGOON.upgrade_gas(),
        _ => 0,
    }
}

fn encode_scalar(blob_base_fee_scalar: u32, base_fee_scalar: u32) -> U256 {
    // Translate Ecotone values back into encoded scalar if needed.
    // We do not know if it was derived from a v0 or v1 scalar,
    // but v1 is fine, a 0 blob base fee has the same effect.
    let mut buf = B256::ZERO;
    buf[0] = 0x01;
    buf[24..28].copy_from_slice(blob_base_fee_scalar.to_be_bytes().as_ref());
    buf[28..32].copy_from_slice(base_fee_scalar.to_be_bytes().as_ref());
    buf.into()
}

/// Maximum EIP-2718 transaction-type byte. A leading byte above this is the RLP list header of a
/// legacy transaction rather than a type identifier.
const EIP2718_MAX_TX_TYPE: u8 = 0x7F;

/// Reads transaction data from a reader.
pub fn read_tx_data(r: &mut &[u8]) -> Result<(Vec<u8>, OpTxType), SpanBatchError> {
    let first_byte =
        *r.first().ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
    let tx_type_id = if first_byte <= EIP2718_MAX_TX_TYPE {
        r.advance(1);
        first_byte
    } else {
        u8::from(OpTxType::Legacy)
    };

    // Read the RLP header with a different reader pointer. This prevents the initial pointer from
    // being advanced in the case that what we read is invalid.
    let rlp_header = Header::decode(&mut (**r).as_ref())
        .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
    if !rlp_header.list {
        return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    let payload_length_with_header = rlp_header.payload_length + rlp_header.length();
    if payload_length_with_header > MAX_SPAN_BATCH_ELEMENTS as usize {
        return Err(SpanBatchError::TooBigSpanBatchSize);
    }
    if r.len() < payload_length_with_header {
        return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    let tx_type = match OpTxType::try_from(tx_type_id) {
        // Deposits are not valid span-batch transactions, and an unknown byte is invalid too.
        Ok(OpTxType::Deposit) | Err(_) => {
            return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionType));
        }
        Ok(ty) => ty,
    };

    let is_typed_tx = tx_type != OpTxType::Legacy;
    let tx_data_capacity = payload_length_with_header + usize::from(is_typed_tx);
    let mut tx_data = Vec::with_capacity(tx_data_capacity);
    if is_typed_tx {
        tx_data.push(u8::from(tx_type));
    }
    tx_data.extend_from_slice(&r[..payload_length_with_header]);
    r.advance(payload_length_with_header);

    Ok((tx_data, tx_type))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{RAW_BEDROCK_INFO_TX, RAW_ECOTONE_INFO_TX, RAW_ISTHMUS_INFO_TX};
    use alloc::vec;
    use alloy_eips::eip1898::BlockNumHash;
    use alloy_primitives::{U256, address, bytes, uint};
    use kona_genesis::{ChainGenesis, HardForkConfig};

    #[test]
    fn test_to_system_config_invalid_genesis_hash() {
        let block = OpBlock::default();
        let rollup_config = RollupConfig::default();
        let err = to_system_config(&block, &rollup_config).unwrap_err();
        assert_eq!(
            err,
            OpBlockConversionError::InvalidGenesisHash(
                rollup_config.genesis.l2.hash,
                block.header.hash_slow(),
            )
        );
    }

    #[test]
    fn test_to_system_config_missing_system_config_genesis() {
        let block = OpBlock::default();
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            ..Default::default()
        };
        let err = to_system_config(&block, &rollup_config).unwrap_err();
        assert_eq!(err, OpBlockConversionError::MissingSystemConfigGenesis);
    }

    #[test]
    fn test_to_system_config_from_genesis() {
        let block = OpBlock::default();
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                system_config: Some(SystemConfig::default()),
                ..Default::default()
            },
            ..Default::default()
        };
        let config = to_system_config(&block, &rollup_config).unwrap();
        assert_eq!(config, SystemConfig::default());
    }

    #[test]
    fn test_to_system_config_empty_txs() {
        let block = OpBlock {
            header: alloy_consensus::Header { number: 1, ..Default::default() },
            ..Default::default()
        };
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            ..Default::default()
        };
        let err = to_system_config(&block, &rollup_config).unwrap_err();
        assert_eq!(err, OpBlockConversionError::EmptyTransactions(block_hash));
    }

    #[test]
    fn test_to_system_config_non_deposit() {
        let block = OpBlock {
            header: alloy_consensus::Header { number: 1, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Legacy(
                    alloy_consensus::Signed::new_unchecked(
                        alloy_consensus::TxLegacy {
                            chain_id: Some(1),
                            nonce: 1,
                            gas_price: 1,
                            gas_limit: 1,
                            to: alloy_primitives::TxKind::Create,
                            value: U256::ZERO,
                            input: alloy_primitives::Bytes::new(),
                        },
                        alloy_primitives::Signature::new(U256::ZERO, U256::ZERO, false),
                        Default::default(),
                    ),
                )],
                ..Default::default()
            },
        };
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            ..Default::default()
        };
        let err = to_system_config(&block, &rollup_config).unwrap_err();
        assert_eq!(err, OpBlockConversionError::InvalidTxType(0));
    }

    #[test]
    fn test_constructs_bedrock_system_config() {
        let block = OpBlock {
            header: alloy_consensus::Header { number: 1, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Deposit(
                    alloy_primitives::Sealed::new(op_alloy_consensus::TxDeposit {
                        input: alloy_primitives::Bytes::from(&RAW_BEDROCK_INFO_TX),
                        ..Default::default()
                    }),
                )],
                ..Default::default()
            },
        };
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            ..Default::default()
        };
        let config = to_system_config(&block, &rollup_config).unwrap();
        let expected = SystemConfig {
            batcher_address: address!("6887246668a3b87f54deb3b94ba47a6f63f32985"),
            overhead: uint!(188_U256),
            scalar: uint!(684000_U256),
            gas_limit: 0,
            base_fee_scalar: None,
            blob_base_fee_scalar: None,
            eip1559_denominator: None,
            eip1559_elasticity: None,
            operator_fee_scalar: None,
            operator_fee_constant: None,
            min_base_fee: None,
            da_footprint_gas_scalar: None,
        };
        assert_eq!(config, expected);
    }

    #[test]
    fn test_to_system_config_strips_karst_upgrade_gas() {
        const BASE_GAS_LIMIT: u64 = 30_000_000;
        let karst_gas = Hardforks::KARST.upgrade_gas();
        let karst_time = 100u64;
        let block = OpBlock {
            header: alloy_consensus::Header {
                number: 1,
                timestamp: karst_time,
                // The activation block carries base + the one-time upgrade gas.
                gas_limit: BASE_GAS_LIMIT + karst_gas,
                // Jovian extra data: version 0x01 + denominator + elasticity + minBaseFee
                // (Karst is active, which implies Jovian).
                extra_data: bytes!("010000beef0000babe0000000000000000"),
                ..Default::default()
            },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Deposit(
                    alloy_primitives::Sealed::new(op_alloy_consensus::TxDeposit {
                        input: alloy_primitives::Bytes::from(&RAW_ISTHMUS_INFO_TX),
                        ..Default::default()
                    }),
                )],
                ..Default::default()
            },
        };
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            block_time: 2,
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            hardforks: HardForkConfig { karst_time: Some(karst_time), ..Default::default() },
            ..Default::default()
        };
        assert!(rollup_config.is_first_karst_block(block.header.timestamp));
        let config = to_system_config(&block, &rollup_config).unwrap();
        assert_eq!(config.gas_limit, BASE_GAS_LIMIT, "Karst upgrade gas must be stripped");
    }

    #[test]
    fn test_to_system_config_keeps_karst_upgrade_gas() {
        const BASE_GAS_LIMIT: u64 = 30_000_000;
        let karst_gas = Hardforks::KARST.upgrade_gas();
        let block = OpBlock {
            header: alloy_consensus::Header {
                number: 1,
                timestamp: 100,
                gas_limit: BASE_GAS_LIMIT + karst_gas,
                extra_data: bytes!("010000beef0000babe0000000000000000"),
                ..Default::default()
            },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Deposit(
                    alloy_primitives::Sealed::new(op_alloy_consensus::TxDeposit {
                        input: alloy_primitives::Bytes::from(&RAW_ISTHMUS_INFO_TX),
                        ..Default::default()
                    }),
                )],
                ..Default::default()
            },
        };
        // keep_karst_upgrade_gas = true: the activation block's inflated gas limit is kept,
        // so an already-affected chain's history still validates.
        let rollup_config = RollupConfig {
            block_time: 2,
            hardforks: HardForkConfig {
                karst_time: Some(100),
                keep_karst_upgrade_gas: true,
                ..Default::default()
            },
            ..Default::default()
        };
        assert!(rollup_config.is_first_karst_block(block.header.timestamp));
        let config = to_system_config(&block, &rollup_config).unwrap();
        assert_eq!(
            config.gas_limit,
            BASE_GAS_LIMIT + karst_gas,
            "upgrade gas must be kept when keep_karst_upgrade_gas is set"
        );
    }

    #[test]
    fn test_upgrade_gas_to_strip() {
        let karst_gas = Hardforks::KARST.upgrade_gas();
        let lagoon_gas = Hardforks::LAGOON.upgrade_gas();
        let (karst_time, lagoon_time) = (1000u64, 2000u64);
        let cfg = RollupConfig {
            block_time: 2,
            hardforks: HardForkConfig {
                karst_time: Some(karst_time),
                lagoon_time: Some(lagoon_time),
                ..Default::default()
            },
            ..Default::default()
        };

        // Each NUT-bundle fork strips its own upgrade gas at its activation block, nowhere else.
        assert_eq!(upgrade_gas_to_strip(&cfg, karst_time), karst_gas);
        assert_eq!(upgrade_gas_to_strip(&cfg, karst_time - 2), 0);
        assert_eq!(upgrade_gas_to_strip(&cfg, karst_time + 2), 0);
        assert_eq!(upgrade_gas_to_strip(&cfg, lagoon_time), lagoon_gas);

        // keep_karst_upgrade_gas opts out of Karst only; later forks have no opt-out.
        let kept = RollupConfig {
            block_time: 2,
            hardforks: HardForkConfig {
                karst_time: Some(karst_time),
                lagoon_time: Some(lagoon_time),
                keep_karst_upgrade_gas: true,
                ..Default::default()
            },
            ..Default::default()
        };
        assert_eq!(upgrade_gas_to_strip(&kept, karst_time), 0);
        assert_eq!(upgrade_gas_to_strip(&kept, lagoon_time), lagoon_gas);
    }

    #[test]
    fn upgrade_gas_covers_known_forks() {
        // OpHardfork is #[non_exhaustive], so upgrade_gas can't fail to compile on a new variant.
        // This guards instead: when a fork is added upstream, VARIANTS grows and this fails,
        // prompting a review of whether the new fork ships a NUT bundle.
        assert_eq!(OpHardfork::VARIANTS.len(), 11, "new OpHardfork variant: review upgrade_gas()");
        for &fork in OpHardfork::VARIANTS {
            let gas = upgrade_gas(fork);
            match fork {
                OpHardfork::Karst | OpHardfork::Lagoon => {
                    assert!(gas > 0, "{fork:?} must reserve upgrade gas")
                }
                _ => assert_eq!(gas, 0, "{fork:?} must not reserve upgrade gas"),
            }
        }
    }

    #[test]
    fn test_constructs_ecotone_system_config() {
        let block = OpBlock {
            header: alloy_consensus::Header {
                number: 1,
                // Holocene EIP1559 parameters stored in the extra data.
                extra_data: bytes!("000000beef0000babe"),
                ..Default::default()
            },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Deposit(
                    alloy_primitives::Sealed::new(op_alloy_consensus::TxDeposit {
                        input: alloy_primitives::Bytes::from(&RAW_ECOTONE_INFO_TX),
                        ..Default::default()
                    }),
                )],
                ..Default::default()
            },
        };
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        assert!(rollup_config.is_holocene_active(block.header.timestamp));
        let config = to_system_config(&block, &rollup_config).unwrap();
        let expected = SystemConfig {
            batcher_address: address!("6887246668a3b87f54deb3b94ba47a6f63f32985"),
            overhead: U256::ZERO,
            scalar: uint!(
                452312848583266388373324160190187140051835877600158453279134670530344387928_U256
            ),
            gas_limit: 0,
            base_fee_scalar: None,
            blob_base_fee_scalar: None,
            eip1559_denominator: Some(0xbeef),
            eip1559_elasticity: Some(0xbabe),
            operator_fee_scalar: None,
            operator_fee_constant: None,
            min_base_fee: None,
            da_footprint_gas_scalar: None,
        };
        assert_eq!(config, expected);
    }

    #[test]
    fn test_constructs_isthmus_system_config() {
        let block = OpBlock {
            header: alloy_consensus::Header {
                number: 1,
                // Holocene EIP1559 parameters stored in the extra data.
                extra_data: bytes!("000000beef0000babe"),
                ..Default::default()
            },
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Deposit(
                    alloy_primitives::Sealed::new(op_alloy_consensus::TxDeposit {
                        input: alloy_primitives::Bytes::from(&RAW_ISTHMUS_INFO_TX),
                        ..Default::default()
                    }),
                )],
                ..Default::default()
            },
        };
        let block_hash = block.header.hash_slow();
        let rollup_config = RollupConfig {
            genesis: ChainGenesis {
                l2: BlockNumHash { hash: block_hash, ..Default::default() },
                ..Default::default()
            },
            hardforks: HardForkConfig {
                holocene_time: Some(0),
                isthmus_time: Some(0),
                ..Default::default()
            },
            ..Default::default()
        };
        assert!(rollup_config.is_holocene_active(block.header.timestamp));
        let config = to_system_config(&block, &rollup_config).unwrap();
        let expected = SystemConfig {
            batcher_address: address!("6887246668a3b87f54deb3b94ba47a6f63f32985"),
            overhead: U256::ZERO,
            scalar: uint!(
                452312848583266388373324160190187140051835877600158453279134670530344387928_U256
            ),
            gas_limit: 0,
            base_fee_scalar: None,
            blob_base_fee_scalar: None,
            eip1559_denominator: Some(0xbeef),
            eip1559_elasticity: Some(0xbabe),
            operator_fee_scalar: Some(0xabcd),
            operator_fee_constant: Some(0xdcba),
            min_base_fee: None,
            da_footprint_gas_scalar: None,
        };
        assert_eq!(config, expected);
    }

    #[test]
    fn test_read_tx_data_truncated_payload() {
        // RLP list header claiming 100 bytes of payload, but only 3 bytes actually present.
        // 0xf8 0x64 = list header with 1-byte length prefix, payload length 100
        let mut data: &[u8] = &[0xf8, 0x64, 0x00, 0x00, 0x00];
        let err = read_tx_data(&mut data).unwrap_err();
        assert_eq!(err, SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    #[test]
    fn test_read_tx_data_exceeds_max_span_batch_elements() {
        // RLP list header claiming MAX_SPAN_BATCH_ELEMENTS + 1 bytes of payload.
        // 0xfa = list with 3-byte length prefix (0xf7 + 3), then 0x989681 = 10_000_001.
        // Header::decode validates the claimed length against the buffer, so we must provide
        // a buffer large enough for decoding to succeed before the max size check triggers.
        let mut data = vec![0u8; MAX_SPAN_BATCH_ELEMENTS as usize + 5];
        data[0] = 0xfa;
        data[1] = 0x98;
        data[2] = 0x96;
        data[3] = 0x81;
        let mut slice: &[u8] = &data;
        let err = read_tx_data(&mut slice).unwrap_err();
        assert_eq!(err, SpanBatchError::TooBigSpanBatchSize);
    }
}
