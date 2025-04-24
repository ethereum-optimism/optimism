//! Loads and formats OP receipt RPC response.

use alloy_consensus::transaction::TransactionMeta;
use alloy_eips::eip2718::Encodable2718;
use alloy_rpc_types_eth::{Log, TransactionReceipt};
use op_alloy_consensus::{OpDepositReceipt, OpDepositReceiptWithBloom, OpReceiptEnvelope};
use op_alloy_rpc_types::{L1BlockInfo, OpTransactionReceipt, OpTransactionReceiptFields};
use reth_chainspec::ChainSpecProvider;
use reth_node_api::{FullNodeComponents, NodeTypes};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::RethL1BlockInfo;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::{OpReceipt, OpTransactionSigned};
use reth_rpc_eth_api::{helpers::LoadReceipt, FromEthApiError, RpcReceipt};
use reth_rpc_eth_types::{receipt::build_receipt, EthApiError};
use reth_storage_api::{ReceiptProvider, TransactionsProvider};

use crate::{OpEthApi, OpEthApiError};

impl<N> LoadReceipt for OpEthApi<N>
where
    Self: Send + Sync,
    N: FullNodeComponents<Types: NodeTypes<ChainSpec = OpChainSpec>>,
    Self::Provider: TransactionsProvider<Transaction = OpTransactionSigned>
        + ReceiptProvider<Receipt = OpReceipt>,
{
    async fn build_transaction_receipt(
        &self,
        tx: OpTransactionSigned,
        meta: TransactionMeta,
        receipt: OpReceipt,
    ) -> Result<RpcReceipt<Self::NetworkTypes>, Self::Error> {
        let (block, receipts) = self
            .inner
            .eth_api
            .cache()
            .get_block_and_receipts(meta.block_hash)
            .await
            .map_err(Self::Error::from_eth_err)?
            .ok_or(Self::Error::from_eth_err(EthApiError::HeaderNotFound(
                meta.block_hash.into(),
            )))?;

        let mut l1_block_info =
            reth_optimism_evm::extract_l1_info(block.body()).map_err(OpEthApiError::from)?;

        Ok(OpReceiptBuilder::new(
            &self.inner.eth_api.provider().chain_spec(),
            &tx,
            meta,
            &receipt,
            &receipts,
            &mut l1_block_info,
        )?
        .build())
    }
}

/// L1 fee and data gas for a non-deposit transaction, or deposit nonce and receipt version for a
/// deposit transaction.
#[derive(Debug, Clone)]
pub struct OpReceiptFieldsBuilder {
    /// Block number.
    pub block_number: u64,
    /// Block timestamp.
    pub block_timestamp: u64,
    /// The L1 fee for transaction.
    pub l1_fee: Option<u128>,
    /// L1 gas used by transaction.
    pub l1_data_gas: Option<u128>,
    /// L1 fee scalar.
    pub l1_fee_scalar: Option<f64>,
    /* ---------------------------------------- Bedrock ---------------------------------------- */
    /// The base fee of the L1 origin block.
    pub l1_base_fee: Option<u128>,
    /* --------------------------------------- Regolith ---------------------------------------- */
    /// Deposit nonce, if this is a deposit transaction.
    pub deposit_nonce: Option<u64>,
    /* ---------------------------------------- Canyon ----------------------------------------- */
    /// Deposit receipt version, if this is a deposit transaction.
    pub deposit_receipt_version: Option<u64>,
    /* ---------------------------------------- Ecotone ---------------------------------------- */
    /// The current L1 fee scalar.
    pub l1_base_fee_scalar: Option<u128>,
    /// The current L1 blob base fee.
    pub l1_blob_base_fee: Option<u128>,
    /// The current L1 blob base fee scalar.
    pub l1_blob_base_fee_scalar: Option<u128>,
    /// The current operator fee scalar.
    pub operator_fee_scalar: Option<u128>,
    /// The current L1 blob base fee scalar.
    pub operator_fee_constant: Option<u128>,
}

impl OpReceiptFieldsBuilder {
    /// Returns a new builder.
    pub const fn new(block_timestamp: u64, block_number: u64) -> Self {
        Self {
            block_number,
            block_timestamp,
            l1_fee: None,
            l1_data_gas: None,
            l1_fee_scalar: None,
            l1_base_fee: None,
            deposit_nonce: None,
            deposit_receipt_version: None,
            l1_base_fee_scalar: None,
            l1_blob_base_fee: None,
            l1_blob_base_fee_scalar: None,
            operator_fee_scalar: None,
            operator_fee_constant: None,
        }
    }

    /// Applies [`L1BlockInfo`](op_revm::L1BlockInfo).
    pub fn l1_block_info(
        mut self,
        chain_spec: &OpChainSpec,
        tx: &OpTransactionSigned,
        l1_block_info: &mut op_revm::L1BlockInfo,
    ) -> Result<Self, OpEthApiError> {
        let raw_tx = tx.encoded_2718();
        let timestamp = self.block_timestamp;

        self.l1_fee = Some(
            l1_block_info
                .l1_tx_data_fee(chain_spec, timestamp, &raw_tx, tx.is_deposit())
                .map_err(|_| OpEthApiError::L1BlockFeeError)?
                .saturating_to(),
        );

        self.l1_data_gas = Some(
            l1_block_info
                .l1_data_gas(chain_spec, timestamp, &raw_tx)
                .map_err(|_| OpEthApiError::L1BlockGasError)?
                .saturating_add(l1_block_info.l1_fee_overhead.unwrap_or_default())
                .saturating_to(),
        );

        self.l1_fee_scalar = (!chain_spec.is_ecotone_active_at_timestamp(timestamp))
            .then_some(f64::from(l1_block_info.l1_base_fee_scalar) / 1_000_000.0);

        self.l1_base_fee = Some(l1_block_info.l1_base_fee.saturating_to());
        self.l1_base_fee_scalar = Some(l1_block_info.l1_base_fee_scalar.saturating_to());
        self.l1_blob_base_fee = l1_block_info.l1_blob_base_fee.map(|fee| fee.saturating_to());
        self.l1_blob_base_fee_scalar =
            l1_block_info.l1_blob_base_fee_scalar.map(|scalar| scalar.saturating_to());

        // If the operator fee params are both set to 0, we don't add them to the receipt.
        let operator_fee_scalar_has_non_zero_value: bool =
            l1_block_info.operator_fee_scalar.is_some_and(|scalar| !scalar.is_zero());

        let operator_fee_constant_has_non_zero_value =
            l1_block_info.operator_fee_constant.is_some_and(|constant| !constant.is_zero());

        if operator_fee_scalar_has_non_zero_value || operator_fee_constant_has_non_zero_value {
            self.operator_fee_scalar =
                l1_block_info.operator_fee_scalar.map(|scalar| scalar.saturating_to());
            self.operator_fee_constant =
                l1_block_info.operator_fee_constant.map(|constant| constant.saturating_to());
        }

        Ok(self)
    }

    /// Applies deposit transaction metadata: deposit nonce.
    pub const fn deposit_nonce(mut self, nonce: Option<u64>) -> Self {
        self.deposit_nonce = nonce;
        self
    }

    /// Applies deposit transaction metadata: deposit receipt version.
    pub const fn deposit_version(mut self, version: Option<u64>) -> Self {
        self.deposit_receipt_version = version;
        self
    }

    /// Builds the [`OpTransactionReceiptFields`] object.
    pub const fn build(self) -> OpTransactionReceiptFields {
        let Self {
            block_number: _,    // used to compute other fields
            block_timestamp: _, // used to compute other fields
            l1_fee,
            l1_data_gas: l1_gas_used,
            l1_fee_scalar,
            l1_base_fee: l1_gas_price,
            deposit_nonce,
            deposit_receipt_version,
            l1_base_fee_scalar,
            l1_blob_base_fee,
            l1_blob_base_fee_scalar,
            operator_fee_scalar,
            operator_fee_constant,
        } = self;

        OpTransactionReceiptFields {
            l1_block_info: L1BlockInfo {
                l1_gas_price,
                l1_gas_used,
                l1_fee,
                l1_fee_scalar,
                l1_base_fee_scalar,
                l1_blob_base_fee,
                l1_blob_base_fee_scalar,
                operator_fee_scalar,
                operator_fee_constant,
            },
            deposit_nonce,
            deposit_receipt_version,
        }
    }
}

/// Builds an [`OpTransactionReceipt`].
#[derive(Debug)]
pub struct OpReceiptBuilder {
    /// Core receipt, has all the fields of an L1 receipt and is the basis for the OP receipt.
    pub core_receipt: TransactionReceipt<OpReceiptEnvelope<Log>>,
    /// Additional OP receipt fields.
    pub op_receipt_fields: OpTransactionReceiptFields,
}

impl OpReceiptBuilder {
    /// Returns a new builder.
    pub fn new(
        chain_spec: &OpChainSpec,
        transaction: &OpTransactionSigned,
        meta: TransactionMeta,
        receipt: &OpReceipt,
        all_receipts: &[OpReceipt],
        l1_block_info: &mut op_revm::L1BlockInfo,
    ) -> Result<Self, OpEthApiError> {
        let timestamp = meta.timestamp;
        let block_number = meta.block_number;
        let core_receipt =
            build_receipt(transaction, meta, receipt, all_receipts, None, |receipt_with_bloom| {
                match receipt {
                    OpReceipt::Legacy(_) => OpReceiptEnvelope::<Log>::Legacy(receipt_with_bloom),
                    OpReceipt::Eip2930(_) => OpReceiptEnvelope::<Log>::Eip2930(receipt_with_bloom),
                    OpReceipt::Eip1559(_) => OpReceiptEnvelope::<Log>::Eip1559(receipt_with_bloom),
                    OpReceipt::Eip7702(_) => OpReceiptEnvelope::<Log>::Eip7702(receipt_with_bloom),
                    OpReceipt::Deposit(receipt) => {
                        OpReceiptEnvelope::<Log>::Deposit(OpDepositReceiptWithBloom::<Log> {
                            receipt: OpDepositReceipt::<Log> {
                                inner: receipt_with_bloom.receipt,
                                deposit_nonce: receipt.deposit_nonce,
                                deposit_receipt_version: receipt.deposit_receipt_version,
                            },
                            logs_bloom: receipt_with_bloom.logs_bloom,
                        })
                    }
                }
            })?;

        let op_receipt_fields = OpReceiptFieldsBuilder::new(timestamp, block_number)
            .l1_block_info(chain_spec, transaction, l1_block_info)?
            .build();

        Ok(Self { core_receipt, op_receipt_fields })
    }

    /// Builds [`OpTransactionReceipt`] by combing core (l1) receipt fields and additional OP
    /// receipt fields.
    pub fn build(self) -> OpTransactionReceipt {
        let Self { core_receipt: inner, op_receipt_fields } = self;

        let OpTransactionReceiptFields { l1_block_info, .. } = op_receipt_fields;

        OpTransactionReceipt { inner, l1_block_info }
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloy_consensus::{Block, BlockBody};
    use alloy_primitives::{hex, U256};
    use op_alloy_network::eip2718::Decodable2718;
    use reth_optimism_chainspec::{BASE_MAINNET, OP_MAINNET};

    /// OP Mainnet transaction at index 0 in block 124665056.
    ///
    /// <https://optimistic.etherscan.io/tx/0x312e290cf36df704a2217b015d6455396830b0ce678b860ebfcc30f41403d7b1>
    const TX_SET_L1_BLOCK_OP_MAINNET_BLOCK_124665056: [u8; 251] = hex!(
        "7ef8f8a0683079df94aa5b9cf86687d739a60a9b4f0835e520ec4d664e2e415dca17a6df94deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e200000146b000f79c500000000000000040000000066d052e700000000013ad8a3000000000000000000000000000000000000000000000000000000003ef1278700000000000000000000000000000000000000000000000000000000000000012fdf87b89884a61e74b322bbcf60386f543bfae7827725efaaf0ab1de2294a590000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985"
    );

    /// OP Mainnet transaction at index 1 in block 124665056.
    ///
    /// <https://optimistic.etherscan.io/tx/0x1059e8004daff32caa1f1b1ef97fe3a07a8cf40508f5b835b66d9420d87c4a4a>
    const TX_1_OP_MAINNET_BLOCK_124665056: [u8; 1176] = hex!(
        "02f904940a8303fba78401d6d2798401db2b6d830493e0943e6f4f7866654c18f536170780344aa8772950b680b904246a761202000000000000000000000000087000a300de7200382b55d40045000000e5d60e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003a0000000000000000000000000000000000000000000000000000000000000022482ad56cb0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000120000000000000000000000000dc6ff44d5d932cbd77b52e5612ba0529dc6226f1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000044095ea7b300000000000000000000000021c4928109acb0659a88ae5329b5374a3024694c0000000000000000000000000000000000000000000000049b9ca9a6943400000000000000000000000000000000000000000000000000000000000000000000000000000000000021c4928109acb0659a88ae5329b5374a3024694c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000024b6b55f250000000000000000000000000000000000000000000000049b9ca9a694340000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000415ec214a3950bea839a7e6fbb0ba1540ac2076acd50820e2d5ef83d0902cdffb24a47aff7de5190290769c4f0a9c6fabf63012986a0d590b1b571547a8c7050ea1b00000000000000000000000000000000000000000000000000000000000000c080a06db770e6e25a617fe9652f0958bd9bd6e49281a53036906386ed39ec48eadf63a07f47cf51a4a40b4494cf26efc686709a9b03939e20ee27e59682f5faa536667e"
    );

    /// Timestamp of OP mainnet block 124665056.
    ///
    /// <https://optimistic.etherscan.io/block/124665056>
    const BLOCK_124665056_TIMESTAMP: u64 = 1724928889;

    /// L1 block info for transaction at index 1 in block 124665056.
    ///
    /// <https://optimistic.etherscan.io/tx/0x1059e8004daff32caa1f1b1ef97fe3a07a8cf40508f5b835b66d9420d87c4a4a>
    const TX_META_TX_1_OP_MAINNET_BLOCK_124665056: OpTransactionReceiptFields =
        OpTransactionReceiptFields {
            l1_block_info: L1BlockInfo {
                l1_gas_price: Some(1055991687), // since bedrock l1 base fee
                l1_gas_used: Some(4471),
                l1_fee: Some(24681034813),
                l1_fee_scalar: None,
                l1_base_fee_scalar: Some(5227),
                l1_blob_base_fee: Some(1),
                l1_blob_base_fee_scalar: Some(1014213),
                operator_fee_scalar: None,
                operator_fee_constant: None,
            },
            deposit_nonce: None,
            deposit_receipt_version: None,
        };

    #[test]
    fn op_receipt_fields_from_block_and_tx() {
        // rig
        let tx_0 = OpTransactionSigned::decode_2718(
            &mut TX_SET_L1_BLOCK_OP_MAINNET_BLOCK_124665056.as_slice(),
        )
        .unwrap();

        let tx_1 =
            OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice())
                .unwrap();

        let block: Block<OpTransactionSigned> = Block {
            body: BlockBody { transactions: [tx_0, tx_1.clone()].to_vec(), ..Default::default() },
            ..Default::default()
        };

        let mut l1_block_info =
            reth_optimism_evm::extract_l1_info(&block.body).expect("should extract l1 info");

        // test
        assert!(OP_MAINNET.is_fjord_active_at_timestamp(BLOCK_124665056_TIMESTAMP));

        let receipt_meta = OpReceiptFieldsBuilder::new(BLOCK_124665056_TIMESTAMP, 124665056)
            .l1_block_info(&OP_MAINNET, &tx_1, &mut l1_block_info)
            .expect("should parse revm l1 info")
            .build();

        let L1BlockInfo {
            l1_gas_price,
            l1_gas_used,
            l1_fee,
            l1_fee_scalar,
            l1_base_fee_scalar,
            l1_blob_base_fee,
            l1_blob_base_fee_scalar,
            operator_fee_scalar,
            operator_fee_constant,
        } = receipt_meta.l1_block_info;

        assert_eq!(
            l1_gas_price, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_gas_price,
            "incorrect l1 base fee (former gas price)"
        );
        assert_eq!(
            l1_gas_used, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_gas_used,
            "incorrect l1 gas used"
        );
        assert_eq!(
            l1_fee, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_fee,
            "incorrect l1 fee"
        );
        assert_eq!(
            l1_fee_scalar, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_fee_scalar,
            "incorrect l1 fee scalar"
        );
        assert_eq!(
            l1_base_fee_scalar,
            TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_base_fee_scalar,
            "incorrect l1 base fee scalar"
        );
        assert_eq!(
            l1_blob_base_fee,
            TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_blob_base_fee,
            "incorrect l1 blob base fee"
        );
        assert_eq!(
            l1_blob_base_fee_scalar,
            TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_blob_base_fee_scalar,
            "incorrect l1 blob base fee scalar"
        );
        assert_eq!(
            operator_fee_scalar,
            TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.operator_fee_scalar,
            "incorrect operator fee scalar"
        );
        assert_eq!(
            operator_fee_constant,
            TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.operator_fee_constant,
            "incorrect operator fee constant"
        );
    }

    #[test]
    fn op_non_zero_operator_fee_params_included_in_receipt() {
        let tx_1 =
            OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice())
                .unwrap();

        let mut l1_block_info = op_revm::L1BlockInfo::default();

        l1_block_info.operator_fee_scalar = Some(U256::ZERO);
        l1_block_info.operator_fee_constant = Some(U256::from(2));

        let receipt_meta = OpReceiptFieldsBuilder::new(BLOCK_124665056_TIMESTAMP, 124665056)
            .l1_block_info(&OP_MAINNET, &tx_1, &mut l1_block_info)
            .expect("should parse revm l1 info")
            .build();

        let L1BlockInfo { operator_fee_scalar, operator_fee_constant, .. } =
            receipt_meta.l1_block_info;

        assert_eq!(operator_fee_scalar, Some(0), "incorrect operator fee scalar");
        assert_eq!(operator_fee_constant, Some(2), "incorrect operator fee constant");
    }

    #[test]
    fn op_zero_operator_fee_params_not_included_in_receipt() {
        let tx_1 =
            OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice())
                .unwrap();

        let mut l1_block_info = op_revm::L1BlockInfo::default();

        l1_block_info.operator_fee_scalar = Some(U256::ZERO);
        l1_block_info.operator_fee_constant = Some(U256::ZERO);

        let receipt_meta = OpReceiptFieldsBuilder::new(BLOCK_124665056_TIMESTAMP, 124665056)
            .l1_block_info(&OP_MAINNET, &tx_1, &mut l1_block_info)
            .expect("should parse revm l1 info")
            .build();

        let L1BlockInfo { operator_fee_scalar, operator_fee_constant, .. } =
            receipt_meta.l1_block_info;

        assert_eq!(operator_fee_scalar, None, "incorrect operator fee scalar");
        assert_eq!(operator_fee_constant, None, "incorrect operator fee constant");
    }

    // <https://github.com/paradigmxyz/reth/issues/12177>
    #[test]
    fn base_receipt_gas_fields() {
        // https://basescan.org/tx/0x510fd4c47d78ba9f97c91b0f2ace954d5384c169c9545a77a373cf3ef8254e6e
        let system = hex!(
            "7ef8f8a0389e292420bcbf9330741f72074e39562a09ff5a00fd22e4e9eee7e34b81bca494deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000008dd00101c120000000000000004000000006721035b00000000014189960000000000000000000000000000000000000000000000000000000349b4dcdc000000000000000000000000000000000000000000000000000000004ef9325cc5991ce750960f636ca2ffbb6e209bb3ba91412f21dd78c14ff154d1930f1f9a0000000000000000000000005050f69a9786f081509234f1a7f4684b5e5b76c9"
        );
        let tx_0 = OpTransactionSigned::decode_2718(&mut &system[..]).unwrap();

        let block: alloy_consensus::Block<OpTransactionSigned> = Block {
            body: BlockBody { transactions: vec![tx_0], ..Default::default() },
            ..Default::default()
        };
        let mut l1_block_info =
            reth_optimism_evm::extract_l1_info(&block.body).expect("should extract l1 info");

        // https://basescan.org/tx/0xf9420cbaf66a2dda75a015488d37262cbfd4abd0aad7bb2be8a63e14b1fa7a94
        let tx = hex!(
            "02f86c8221058034839a4ae283021528942f16386bb37709016023232523ff6d9daf444be380841249c58bc080a001b927eda2af9b00b52a57be0885e0303c39dd2831732e14051c2336470fd468a0681bf120baf562915841a48601c2b54a6742511e535cf8f71c95115af7ff63bd"
        );
        let tx_1 = OpTransactionSigned::decode_2718(&mut &tx[..]).unwrap();

        let receipt_meta = OpReceiptFieldsBuilder::new(1730216981, 21713817)
            .l1_block_info(&BASE_MAINNET, &tx_1, &mut l1_block_info)
            .expect("should parse revm l1 info")
            .build();

        let L1BlockInfo {
            l1_gas_price,
            l1_gas_used,
            l1_fee,
            l1_fee_scalar,
            l1_base_fee_scalar,
            l1_blob_base_fee,
            l1_blob_base_fee_scalar,
            operator_fee_scalar,
            operator_fee_constant,
        } = receipt_meta.l1_block_info;

        assert_eq!(l1_gas_price, Some(14121491676), "incorrect l1 base fee (former gas price)");
        assert_eq!(l1_gas_used, Some(1600), "incorrect l1 gas used");
        assert_eq!(l1_fee, Some(191150293412), "incorrect l1 fee");
        assert!(l1_fee_scalar.is_none(), "incorrect l1 fee scalar");
        assert_eq!(l1_base_fee_scalar, Some(2269), "incorrect l1 base fee scalar");
        assert_eq!(l1_blob_base_fee, Some(1324954204), "incorrect l1 blob base fee");
        assert_eq!(l1_blob_base_fee_scalar, Some(1055762), "incorrect l1 blob base fee scalar");
        assert_eq!(operator_fee_scalar, None, "incorrect operator fee scalar");
        assert_eq!(operator_fee_constant, None, "incorrect operator fee constant");
    }
}
