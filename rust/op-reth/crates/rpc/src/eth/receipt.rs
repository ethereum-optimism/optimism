//! Loads and formats OP receipt RPC response.

use crate::{OpEthApi, OpEthApiError, eth::RpcNodeCore};
use alloy_consensus::{BlockHeader, Receipt, ReceiptWithBloom, TxReceipt};
use alloy_eips::eip2718::Encodable2718;
use alloy_rpc_types_eth::{Log, TransactionReceipt};
use op_alloy_consensus::{OpReceipt, OpTransaction, parse_post_exec_payload_from_transactions};
use op_alloy_rpc_types::{L1BlockInfo, OpTransactionReceipt, OpTransactionReceiptFields};
use op_revm::tx_da_footprint;
use reth_chainspec::{ChainSpecProvider, EthChainSpec};
use reth_node_api::NodePrimitives;
use reth_optimism_evm::RethL1BlockInfo;
use reth_optimism_forks::OpHardforks;
use reth_primitives_traits::{BlockBody, SealedBlock, SealedHeaderFor};
use reth_rpc_eth_api::{
    RpcConvert,
    helpers::LoadReceipt,
    transaction::{ConvertReceiptInput, ReceiptConverter},
};
use reth_rpc_eth_types::{EthApiError, receipt::build_receipt};
use reth_storage_api::BlockReader;
use std::fmt::Debug;

impl<N, Rpc> LoadReceipt for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError>,
{
}

/// Converter for OP receipts.
#[derive(Debug, Clone)]
pub struct OpReceiptConverter<Provider> {
    provider: Provider,
}

impl<Provider> OpReceiptConverter<Provider> {
    /// Creates a new [`OpReceiptConverter`].
    pub const fn new(provider: Provider) -> Self {
        Self { provider }
    }
}

impl<Provider, N> ReceiptConverter<N> for OpReceiptConverter<Provider>
where
    N: NodePrimitives<SignedTx: OpTransaction, Receipt = OpReceipt>,
    Provider:
        BlockReader<Block = N::Block> + ChainSpecProvider<ChainSpec: OpHardforks> + Debug + 'static,
{
    type RpcReceipt = OpTransactionReceipt;
    type RpcLog = Log;
    type Error = OpEthApiError;

    fn convert_log(
        &self,
        log: Log,
        _receipt: &N::Receipt,
        _header: &SealedHeaderFor<N>,
    ) -> Result<Self::RpcLog, Self::Error> {
        Ok(log)
    }

    fn convert_receipts(
        &self,
        inputs: Vec<ConvertReceiptInput<'_, N>>,
    ) -> Result<Vec<Self::RpcReceipt>, Self::Error> {
        let Some(block_number) = inputs.first().map(|r| r.meta.block_number) else {
            return Ok(Vec::new());
        };

        let block = self
            .provider
            .block_by_number(block_number)?
            .ok_or(EthApiError::HeaderNotFound(block_number.into()))?;

        self.convert_receipts_with_block(inputs, &SealedBlock::new_unhashed(block))
    }

    fn convert_receipts_with_block(
        &self,
        inputs: Vec<ConvertReceiptInput<'_, N>>,
        block: &SealedBlock<N::Block>,
    ) -> Result<Vec<Self::RpcReceipt>, Self::Error> {
        let chain_spec = self.provider.chain_spec();
        let mut l1_block_info = match reth_optimism_evm::extract_l1_info(block.body()) {
            Ok(l1_block_info) => l1_block_info,
            Err(err) => {
                let genesis_number = chain_spec.genesis().number.unwrap_or_default();
                // If it is the genesis block (i.e. block number is 0), there is no L1 info, so
                // we return an empty l1_block_info.
                if block.header().number() == genesis_number {
                    return Ok(vec![]);
                }
                return Err(err.into());
            }
        };

        let mut receipts = Vec::with_capacity(inputs.len());
        let sdm_active =
            reth_optimism_evm::is_sdm_active_at_timestamp(&chain_spec, block.header().timestamp());
        let post_exec_payload = parse_post_exec_payload_from_transactions(
            block.body().transactions(),
            block.header().number(),
            sdm_active,
        )?
        .map(|parsed| parsed.payload);

        for input in inputs {
            // We must clear this cache as different L2 transactions can have different
            // L1 costs. A potential improvement here is to only clear the cache if the
            // new transaction input has changed, since otherwise the L1 cost wouldn't.
            l1_block_info.clear_tx_l1_cost();

            let op_gas_refund = post_exec_payload
                .as_ref()
                .and_then(|payload| payload.gas_refund_for_idx(input.meta.index));

            receipts.push(
                OpReceiptBuilder::new(&chain_spec, input, &mut l1_block_info, op_gas_refund)?
                    .build(),
            );
        }

        Ok(receipts)
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
    /// Post-exec block-level warming refund for this transaction.
    pub op_gas_refund: Option<u64>,
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
    /* ---------------------------------------- Isthmus ---------------------------------------- */
    /// The current operator fee scalar.
    pub operator_fee_scalar: Option<u128>,
    /// The current L1 blob base fee scalar.
    pub operator_fee_constant: Option<u128>,
    /* ---------------------------------------- Jovian ----------------------------------------- */
    /// The current DA footprint gas scalar.
    pub da_footprint_gas_scalar: Option<u16>,
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
            op_gas_refund: None,
            deposit_nonce: None,
            deposit_receipt_version: None,
            l1_base_fee_scalar: None,
            l1_blob_base_fee: None,
            l1_blob_base_fee_scalar: None,
            operator_fee_scalar: None,
            operator_fee_constant: None,
            da_footprint_gas_scalar: None,
        }
    }

    /// Applies [`L1BlockInfo`](op_revm::L1BlockInfo).
    pub fn l1_block_info<T: Encodable2718 + OpTransaction>(
        mut self,
        chain_spec: &impl OpHardforks,
        tx: &T,
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
        let has_operator_fee = l1_block_info.operator_fee_scalar.is_some_and(|s| !s.is_zero()) ||
            l1_block_info.operator_fee_constant.is_some_and(|c| !c.is_zero());

        if has_operator_fee {
            self.operator_fee_scalar =
                l1_block_info.operator_fee_scalar.map(|scalar| scalar.saturating_to());
            self.operator_fee_constant =
                l1_block_info.operator_fee_constant.map(|constant| constant.saturating_to());
        }

        self.da_footprint_gas_scalar = l1_block_info.da_footprint_gas_scalar;

        Ok(self)
    }

    /// Applies post-exec block-level warming refund metadata.
    pub const fn op_gas_refund(mut self, op_gas_refund: Option<u64>) -> Self {
        self.op_gas_refund = op_gas_refund;
        self
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
            op_gas_refund,
            deposit_nonce,
            deposit_receipt_version,
            l1_base_fee_scalar,
            l1_blob_base_fee,
            l1_blob_base_fee_scalar,
            operator_fee_scalar,
            operator_fee_constant,
            da_footprint_gas_scalar,
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
                da_footprint_gas_scalar,
            },
            op_gas_refund,
            deposit_nonce,
            deposit_receipt_version,
        }
    }
}

/// Builds an [`OpTransactionReceipt`].
#[derive(Debug)]
pub struct OpReceiptBuilder {
    /// Core receipt, has all the fields of an L1 receipt and is the basis for the OP receipt.
    pub core_receipt: TransactionReceipt<ReceiptWithBloom<OpReceipt<Log>>>,
    /// Additional OP receipt fields.
    pub op_receipt_fields: OpTransactionReceiptFields,
}

impl OpReceiptBuilder {
    /// Returns a new builder.
    pub fn new<N>(
        chain_spec: &impl OpHardforks,
        input: ConvertReceiptInput<'_, N>,
        l1_block_info: &mut op_revm::L1BlockInfo,
        op_gas_refund: Option<u64>,
    ) -> Result<Self, OpEthApiError>
    where
        N: NodePrimitives<SignedTx: OpTransaction, Receipt = OpReceipt>,
    {
        let timestamp = input.meta.timestamp;
        let block_number = input.meta.block_number;
        let tx_signed = *input.tx.inner();
        let mut core_receipt = build_receipt(input, None, |receipt, next_log_index, meta| {
            let map_logs = move |receipt: alloy_consensus::Receipt| {
                let Receipt { status, cumulative_gas_used, logs } = receipt;
                let logs = Log::collect_for_receipt(next_log_index, meta, logs);
                Receipt { status, cumulative_gas_used, logs }
            };
            let mapped_receipt: OpReceipt<Log> = match receipt {
                OpReceipt::Legacy(receipt) => OpReceipt::Legacy(map_logs(receipt)),
                OpReceipt::Eip2930(receipt) => OpReceipt::Eip2930(map_logs(receipt)),
                OpReceipt::Eip1559(receipt) => OpReceipt::Eip1559(map_logs(receipt)),
                OpReceipt::Eip7702(receipt) => OpReceipt::Eip7702(map_logs(receipt)),
                OpReceipt::PostExec(receipt) => OpReceipt::PostExec(map_logs(receipt)),
                OpReceipt::Deposit(receipt) => OpReceipt::Deposit(receipt.map_inner(map_logs)),
            };
            mapped_receipt.into_with_bloom()
        });

        // In jovian, we're using the blob gas used field to store the current da
        // footprint's value.
        // We're computing the jovian blob gas used before building the receipt since the inputs get
        // consumed by the `build_receipt` function.
        if chain_spec.is_jovian_active_at_timestamp(timestamp) {
            // Estimate the transaction's DA footprint from its encoded size and footprint scalar.
            // Jovian specs: <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/exec-engine.md#da-footprint-block-limit>
            let tx_da_footprint = tx_da_footprint(
                tx_signed,
                l1_block_info.da_footprint_gas_scalar.unwrap_or_default().into(),
            );

            core_receipt.blob_gas_used = Some(tx_da_footprint);
        }

        // OP deposit-receipt spec: for a deposit contract-creation tx (`to == null`), the
        // `depositNonce` "helps derive the correct `contractAddress` meta-data, instead of
        // assuming the nonce was zero". The deposit nonce is the sender's real L2 nonce, persisted
        // on the receipt; a deposit tx's own `nonce()` is hard-coded to 0. Available from Regolith
        // onward (before Regolith the receipt has no deposit nonce, so the address stays
        // `CREATE(from, 0)`). See
        // <https://specs.optimism.io/protocol/deposits.html#deposit-receipt>.
        //
        // `build_receipt` (chain-agnostic, from `reth_rpc_eth_types`) derives the address as
        // `CREATE(from, tx.nonce())` = `CREATE(from, 0)`, while the contract is actually deployed
        // at `CREATE(from, depositNonce)`. Without this override
        // `eth_getCode(receipt.contractAddress)` returns `0x` whenever the deposit sender's L2
        // nonce > 0.
        if core_receipt.contract_address.is_some() &&
            let OpReceipt::Deposit(deposit_receipt) = &core_receipt.inner.receipt &&
            let Some(deposit_nonce) = deposit_receipt.deposit_nonce
        {
            core_receipt.contract_address = Some(core_receipt.from.create(deposit_nonce));
        }

        let op_receipt_fields = OpReceiptFieldsBuilder::new(timestamp, block_number)
            .l1_block_info(chain_spec, tx_signed, l1_block_info)?
            .op_gas_refund(op_gas_refund)
            .build();

        Ok(Self { core_receipt, op_receipt_fields })
    }

    /// Builds [`OpTransactionReceipt`] by combining core (l1) receipt fields and additional OP
    /// receipt fields.
    pub fn build(self) -> OpTransactionReceipt {
        let Self { core_receipt: inner, op_receipt_fields } = self;

        let OpTransactionReceiptFields {
            l1_block_info,
            op_gas_refund,
            deposit_nonce: _,
            deposit_receipt_version: _,
        } = op_receipt_fields;

        OpTransactionReceipt { inner, l1_block_info, op_gas_refund }
    }
}

#[cfg(test)]
mod tests;
