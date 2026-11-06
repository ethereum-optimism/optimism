//! A sequenced Lagoon block with SDM refunds and the RPC accessors the family tests share.

use crate::lagoon::{self, launch_sdm_node, prestate_tracer};
use alloy_consensus::{BlockHeader, TxReceipt};
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use alloy_rpc_types_eth::{TransactionInput, TransactionRequest};
use alloy_rpc_types_trace::{
    common::TraceResult,
    geth::{DiffMode, GethDebugTracingOptions, GethTrace, PreStateConfig, PreStateFrame},
};
use jsonrpsee::{http_client::HttpClient, ws_client::WsClient};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{NodeHelperType, transaction::TransactionTestContext, wallet::Wallet};
use reth_optimism_node::OpNode;
use reth_optimism_payload_builder::OpBuiltPayload;
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::{ReceiptProvider, StateProviderFactory};
use reth_rpc_api::clients::DebugApiClient;

/// Index of the post-exec transaction in a [`SettledBlock`]; the regular transactions precede it.
pub(super) const POST_EXEC_INDEX: usize = 4;

/// Revert data returned by the reverting deployment in a [`SettledBlock`].
pub(super) const REVERT_OUTPUT: U256 = U256::from_limbs([42, 0, 0, 0]);

/// A Lagoon block sequenced by a node running the fixed test policy, which refunds every regular
/// transaction one gas: a real, imported block with a 0x7d transaction and a settled state.
///
/// The sender submits four transactions: a deployment of the init code `STOP` (refunded), a plain
/// transfer (not refunded, since it already sits on the EIP-7623 floor), a second deployment
/// (refunded) and a deployment whose init code reverts with [`REVERT_OUTPUT`].
pub(super) struct SettledBlock {
    pub(super) client: HttpClient,
    pub(super) node: NodeHelperType<OpNode>,
    pub(super) payload: OpBuiltPayload,
    pub(super) sender: Address,
}

impl SettledBlock {
    pub(super) async fn sequence() -> eyre::Result<Self> {
        let chain_spec = lagoon::chain_spec()?;
        let mut node = launch_sdm_node(&chain_spec).await?;
        let client = node.inner.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();
        let wallet = Wallet::default().with_chain_id(chain_spec.chain_id());
        let sender = wallet.inner.address();
        let deployments = [
            (TxKind::Create, Bytes::from_static(&[0x00])),
            (TxKind::Call(Address::repeat_byte(0x99)), Bytes::new()),
            (TxKind::Create, Bytes::from_static(&[0x00])),
            // PUSH1 42 PUSH1 0 MSTORE PUSH1 32 PUSH1 0 REVERT
            (
                TxKind::Create,
                Bytes::from_static(&[0x60, 0x2a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xfd]),
            ),
        ];
        assert_eq!(deployments.len(), POST_EXEC_INDEX, "one regular transaction per row");
        for (nonce, (to, init_code)) in deployments.into_iter().enumerate() {
            let transaction = TransactionRequest {
                chain_id: Some(wallet.chain_id),
                gas: Some(if matches!(to, TxKind::Call(_)) { 21_000 } else { 100_000 }),
                input: TransactionInput::new(init_code),
                max_fee_per_gas: Some(20_000_000_000),
                max_priority_fee_per_gas: Some(1_000_000_000),
                nonce: Some(nonce as u64),
                to: Some(to),
                ..Default::default()
            };
            let signed = TransactionTestContext::sign_tx(wallet.inner.clone(), transaction).await;
            node.rpc.inject_tx(signed.encoded_2718().into()).await?;
        }
        let payload = node.advance_block().await?;

        let fixture = Self { client, node, payload, sender };
        let transactions = fixture.transactions();
        assert_eq!(
            transactions.len(),
            POST_EXEC_INDEX + 1,
            "the regular transactions and the post-exec tx",
        );
        let OpTransactionSigned::PostExec(post_exec) = &transactions[POST_EXEC_INDEX] else {
            panic!("block must end in a post-exec tx, got {:?}", transactions[POST_EXEC_INDEX]);
        };
        let refund_entries = &post_exec.inner().payload.gas_refund_entries;
        assert!(
            refund_entries.iter().any(|entry| entry.index == 2 && entry.gas_refund > 0),
            "the second deployment must be refunded: {refund_entries:?}",
        );
        assert!(
            !refund_entries.iter().any(|entry| entry.index == 1),
            "the transfer must not be refunded: {refund_entries:?}",
        );
        Ok(fixture)
    }

    pub(super) fn block_hash(&self) -> B256 {
        self.payload.block().hash()
    }

    pub(super) fn block_number(&self) -> u64 {
        self.payload.block().number()
    }

    pub(super) fn state_root(&self) -> B256 {
        self.payload.block().state_root()
    }

    pub(super) fn transactions(&self) -> &[OpTransactionSigned] {
        &self.payload.block().body().transactions
    }

    pub(super) fn transaction_hash(&self, index: usize) -> B256 {
        self.transactions()[index].tx_hash()
    }

    pub(super) async fn ws_client(&self) -> WsClient {
        self.node.inner.add_ons_handle.rpc_server_handles().rpc.ws_client().await.unwrap()
    }

    /// Gas each transaction used according to its receipt.
    pub(super) fn receipt_gas(&self) -> eyre::Result<Vec<u64>> {
        let mut cumulative_gas = 0;
        self.transactions()
            .iter()
            .map(|transaction| {
                let receipt =
                    self.node.inner.provider.receipt_by_hash(transaction.tx_hash())?.unwrap();
                let gas_used = receipt.cumulative_gas_used() - cumulative_gas;
                cumulative_gas = receipt.cumulative_gas_used();
                Ok(gas_used)
            })
            .collect()
    }

    /// Balance of `address` in the state the node committed for the block.
    pub(super) fn committed_balance(&self, address: Address) -> eyre::Result<U256> {
        let state = self.node.inner.provider.state_by_block_hash(self.block_hash())?;
        Ok(state.account_balance(&address)?.unwrap_or_default())
    }

    /// Balance of `address` right after the transaction at `index`, as `debug_accountAt` replays
    /// it.
    pub(super) async fn replayed_balance(
        &self,
        index: usize,
        address: Address,
    ) -> eyre::Result<U256> {
        let account = DebugApiClient::<TransactionRequest>::debug_account_info_at(
            &self.client,
            self.block_hash().into(),
            index.into(),
            address,
        )
        .await?
        .unwrap_or_default();
        Ok(account.balance)
    }

    pub(super) async fn block_traces(&self) -> eyre::Result<Vec<TraceResult<GethTrace, String>>> {
        Ok(DebugApiClient::<TransactionRequest>::debug_trace_block_by_hash(
            &self.client,
            self.block_hash(),
            None,
        )
        .await?)
    }

    pub(super) async fn trace_transaction(
        &self,
        index: usize,
        options: Option<GethDebugTracingOptions>,
    ) -> eyre::Result<GethTrace> {
        Ok(DebugApiClient::<TransactionRequest>::debug_trace_transaction(
            &self.client,
            self.transaction_hash(index),
            options,
        )
        .await?)
    }

    pub(super) async fn default_trace_gas_of(&self, index: usize) -> eyre::Result<u64> {
        match self.trace_transaction(index, None).await? {
            GethTrace::Default(frame) => Ok(frame.gas),
            other => panic!("unexpected default trace of transaction {index}: {other:?}"),
        }
    }

    pub(super) async fn prestate_diff(&self, index: usize) -> eyre::Result<DiffMode> {
        let options =
            prestate_tracer(PreStateConfig { diff_mode: Some(true), ..Default::default() });
        match self.trace_transaction(index, Some(options)).await? {
            GethTrace::PreStateTracer(PreStateFrame::Diff(diff)) => Ok(diff),
            other => panic!("unexpected prestate trace of transaction {index}: {other:?}"),
        }
    }
}
