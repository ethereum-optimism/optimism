//! Utility methods used by protocol types.

use alloc::vec::Vec;
use alloy_consensus::{Transaction, TxType, Typed2718};
use alloy_primitives::B256;
use alloy_rlp::{Buf, Header};
use kona_genesis::{RollupConfig, SystemConfig};
use op_alloy_consensus::{OpBlock, decode_holocene_extra_data, decode_jovian_extra_data};

use crate::{
    L1BlockInfoBedrock, L1BlockInfoEcotone, L1BlockInfoIsthmus, L1BlockInfoTx,
    OpBlockConversionError, SpanBatchError, SpanDecodingError, info::L1BlockInfoJovian,
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
        L1BlockInfoTx::Bedrock(L1BlockInfoBedrock { l1_fee_scalar, .. }) => l1_fee_scalar,
        L1BlockInfoTx::Ecotone(L1BlockInfoEcotone {
            base_fee_scalar,
            blob_base_fee_scalar,
            ..
        }) |
        L1BlockInfoTx::Isthmus(L1BlockInfoIsthmus {
            base_fee_scalar,
            blob_base_fee_scalar,
            ..
        }) |
        L1BlockInfoTx::Jovian(L1BlockInfoJovian {
            base_fee_scalar, blob_base_fee_scalar, ..
        }) => {
            // Translate Ecotone values back into encoded scalar if needed.
            // We do not know if it was derived from a v0 or v1 scalar,
            // but v1 is fine, a 0 blob base fee has the same effect.
            let mut buf = B256::ZERO;
            buf[0] = 0x01;
            buf[24..28].copy_from_slice(blob_base_fee_scalar.to_be_bytes().as_ref());
            buf[28..32].copy_from_slice(base_fee_scalar.to_be_bytes().as_ref());
            buf.into()
        }
    };

    let mut cfg = SystemConfig {
        batcher_address: l1_info.batcher_address(),
        overhead: l1_info.l1_fee_overhead(),
        scalar: l1_fee_scalar,
        gas_limit: block.header.gas_limit,
        ..Default::default()
    };

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

/// Reads transaction data from a reader.
pub fn read_tx_data(r: &mut &[u8]) -> Result<(Vec<u8>, TxType), SpanBatchError> {
    let mut tx_data = Vec::new();
    let first_byte =
        *r.first().ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
    let mut tx_type = 0;
    if first_byte <= 0x7F {
        // EIP-2718: Non-legacy tx, so write tx type
        tx_type = first_byte;
        tx_data.push(tx_type);
        r.advance(1);
    }

    // Read the RLP header with a different reader pointer. This prevents the initial pointer from
    // being advanced in the case that what we read is invalid.
    let rlp_header = Header::decode(&mut (**r).as_ref())
        .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;

    let tx_payload = if rlp_header.list {
        // Grab the raw RLP for the transaction data from `r`. It was unaffected since we copied it.
        let payload_length_with_header = rlp_header.payload_length + rlp_header.length();
        let payload = r[0..payload_length_with_header].to_vec();
        r.advance(payload_length_with_header);
        Ok(payload)
    } else {
        Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))
    }?;
    tx_data.extend_from_slice(&tx_payload);

    Ok((
        tx_data,
        tx_type
            .try_into()
            .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionType))?,
    ))
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
}
