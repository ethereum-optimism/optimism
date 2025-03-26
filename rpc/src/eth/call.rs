use super::OpNodeCore;
use crate::{OpEthApi, OpEthApiError};
use alloy_consensus::TxType;
use alloy_primitives::{Bytes, TxKind, U256};
use alloy_rpc_types_eth::transaction::TransactionRequest;
use op_revm::OpTransaction;
use reth_evm::{execute::BlockExecutorFactory, ConfigureEvm, EvmEnv, EvmFactory, SpecFor};
use reth_node_api::NodePrimitives;
use reth_rpc_eth_api::{
    helpers::{estimate::EstimateCall, Call, EthCall, LoadBlock, LoadState, SpawnBlocking},
    FromEthApiError, FromEvmError, FullEthApiTypes, IntoEthApiError,
};
use reth_rpc_eth_types::{revm_utils::CallFees, EthApiError, RpcInvalidTransactionError};
use reth_storage_api::{ProviderHeader, ProviderTx};
use revm::{context::TxEnv, context_interface::Block, Database};

impl<N> EthCall for OpEthApi<N>
where
    Self: EstimateCall + LoadBlock + FullEthApiTypes,
    N: OpNodeCore,
{
}

impl<N> EstimateCall for OpEthApi<N>
where
    Self: Call,
    Self::Error: From<OpEthApiError>,
    N: OpNodeCore,
{
}

impl<N> Call for OpEthApi<N>
where
    Self: LoadState<
            Evm: ConfigureEvm<
                Primitives: NodePrimitives<
                    BlockHeader = ProviderHeader<Self::Provider>,
                    SignedTx = ProviderTx<Self::Provider>,
                >,
                BlockExecutorFactory: BlockExecutorFactory<
                    EvmFactory: EvmFactory<Tx = OpTransaction<TxEnv>>,
                >,
            >,
            Error: FromEvmError<Self::Evm>,
        > + SpawnBlocking,
    Self::Error: From<OpEthApiError>,
    N: OpNodeCore,
{
    #[inline]
    fn call_gas_limit(&self) -> u64 {
        self.inner.eth_api.gas_cap()
    }

    #[inline]
    fn max_simulate_blocks(&self) -> u64 {
        self.inner.eth_api.max_simulate_blocks()
    }

    fn create_txn_env(
        &self,
        evm_env: &EvmEnv<SpecFor<Self::Evm>>,
        request: TransactionRequest,
        mut db: impl Database<Error: Into<EthApiError>>,
    ) -> Result<OpTransaction<TxEnv>, Self::Error> {
        // Ensure that if versioned hashes are set, they're not empty
        if request.blob_versioned_hashes.as_ref().is_some_and(|hashes| hashes.is_empty()) {
            return Err(RpcInvalidTransactionError::BlobTransactionMissingBlobHashes.into_eth_err())
        }

        let tx_type = if request.authorization_list.is_some() {
            TxType::Eip7702
        } else if request.sidecar.is_some() || request.max_fee_per_blob_gas.is_some() {
            TxType::Eip4844
        } else if request.max_fee_per_gas.is_some() || request.max_priority_fee_per_gas.is_some() {
            TxType::Eip1559
        } else if request.access_list.is_some() {
            TxType::Eip2930
        } else {
            TxType::Legacy
        } as u8;

        let TransactionRequest {
            from,
            to,
            gas_price,
            max_fee_per_gas,
            max_priority_fee_per_gas,
            gas,
            value,
            input,
            nonce,
            access_list,
            chain_id,
            blob_versioned_hashes,
            max_fee_per_blob_gas,
            authorization_list,
            transaction_type: _,
            sidecar: _,
        } = request;

        let CallFees { max_priority_fee_per_gas, gas_price, max_fee_per_blob_gas } =
            CallFees::ensure_fees(
                gas_price.map(U256::from),
                max_fee_per_gas.map(U256::from),
                max_priority_fee_per_gas.map(U256::from),
                U256::from(evm_env.block_env.basefee),
                blob_versioned_hashes.as_deref(),
                max_fee_per_blob_gas.map(U256::from),
                evm_env.block_env.blob_gasprice().map(U256::from),
            )?;

        let gas_limit = gas.unwrap_or(
            // Use maximum allowed gas limit. The reason for this
            // is that both Erigon and Geth use pre-configured gas cap even if
            // it's possible to derive the gas limit from the block:
            // <https://github.com/ledgerwatch/erigon/blob/eae2d9a79cb70dbe30b3a6b79c436872e4605458/cmd/rpcdaemon/commands/trace_adhoc.go#L956
            // https://github.com/ledgerwatch/erigon/blob/eae2d9a79cb70dbe30b3a6b79c436872e4605458/eth/ethconfig/config.go#L94>
            evm_env.block_env.gas_limit,
        );

        let chain_id = chain_id.unwrap_or(evm_env.cfg_env.chain_id);

        let caller = from.unwrap_or_default();

        let nonce = if let Some(nonce) = nonce {
            nonce
        } else {
            db.basic(caller).map_err(Into::into)?.map(|acc| acc.nonce).unwrap_or_default()
        };

        let base = TxEnv {
            tx_type,
            gas_limit,
            nonce,
            caller,
            gas_price: gas_price.saturating_to(),
            gas_priority_fee: max_priority_fee_per_gas.map(|v| v.saturating_to()),
            kind: to.unwrap_or(TxKind::Create),
            value: value.unwrap_or_default(),
            data: input
                .try_into_unique_input()
                .map_err(Self::Error::from_eth_err)?
                .unwrap_or_default(),
            chain_id: Some(chain_id),
            access_list: access_list.unwrap_or_default(),
            // EIP-4844 fields
            blob_hashes: blob_versioned_hashes.unwrap_or_default(),
            max_fee_per_blob_gas: max_fee_per_blob_gas
                .map(|v| v.saturating_to())
                .unwrap_or_default(),
            // EIP-7702 fields
            authorization_list: authorization_list.unwrap_or_default(),
        };

        Ok(OpTransaction { base, enveloped_tx: Some(Bytes::new()), deposit: Default::default() })
    }
}
