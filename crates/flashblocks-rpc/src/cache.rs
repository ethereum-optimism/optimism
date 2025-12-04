use alloy_consensus::BlockHeader;
use alloy_consensus::Transaction as _;
use alloy_consensus::TxReceipt;
use alloy_consensus::transaction::SignerRecoverable;
use alloy_consensus::transaction::TransactionMeta;
use alloy_primitives::{Address, Sealable, TxHash, U256};
use alloy_rpc_types::Withdrawals;
use alloy_rpc_types::{BlockTransactions, Header, TransactionInfo};
use arc_swap::ArcSwap;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_network::Optimism;
use op_alloy_rpc_types::OpTransactionReceipt;
use op_alloy_rpc_types::Transaction;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::extract_l1_info;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_primitives::{OpBlock, OpReceipt, OpTransactionSigned};
use reth_optimism_rpc::OpReceiptBuilder;
use reth_primitives::Recovered;
use reth_primitives_traits::block::body::BlockBody;

use reth_rpc_eth_api::transaction::ConvertReceiptInput;
use reth_rpc_eth_api::{RpcBlock, RpcReceipt};
use rollup_boost::FlashblockBuilder;
use rollup_boost_types::flashblocks::FlashblocksPayloadV1;
use rollup_boost_types::payload::{OpExecutionPayloadEnvelope, PayloadVersion};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, str::FromStr, sync::Arc};

#[derive(Debug, Deserialize, Serialize, Clone, Default)]
pub struct Metadata {
    pub receipts: HashMap<String, OpReceipt>,
    pub new_account_balances: HashMap<String, String>, // Address -> Balance (hex)
    pub block_number: u64,
}

#[derive(Clone)]
pub struct FlashblocksCache {
    inner: Arc<ArcSwap<FlashblocksCacheInner>>,
    // TODO: add arc_swap::Cache to speed it up even more
}

impl FlashblocksCache {
    pub fn new(chain_spec: Arc<OpChainSpec>) -> Self {
        Self {
            inner: Arc::new(ArcSwap::from_pointee(FlashblocksCacheInner::new(
                chain_spec,
            ))),
        }
    }

    pub fn get_block(&self, full: bool) -> Option<RpcBlock<Optimism>> {
        ArcSwap::load(&self.inner).get_block(full)
    }

    pub fn get_transaction_count(&self, address: Address) -> Option<u64> {
        ArcSwap::load(&self.inner).get_nonce(address)
    }

    pub fn get_balance(&self, address: Address) -> Option<U256> {
        ArcSwap::load(&self.inner).get_balance(address)
    }

    pub fn get_receipt(&self, tx_hash: &TxHash) -> Option<RpcReceipt<Optimism>> {
        ArcSwap::load(&self.inner).get_receipt(tx_hash)
    }

    pub fn process_payload(&self, payload: FlashblocksPayloadV1) -> eyre::Result<()> {
        let mut new_state = FlashblocksCacheInner::clone(&self.inner.load_full());
        new_state.process_payload(payload)?;
        self.inner.store(Arc::new(new_state));
        Ok(())
    }
}

#[derive(Clone)]
struct FlashblocksCacheInner {
    chain_spec: Arc<OpChainSpec>,
    builder: FlashblockBuilder,
    block: Option<OpBlock>,
    balance_cache: HashMap<Address, U256>,
    nonce_cache: HashMap<Address, u64>,
    receipts_cache: HashMap<TxHash, OpTransactionReceipt>,
}

impl FlashblocksCacheInner {
    pub fn new(chain_spec: Arc<OpChainSpec>) -> Self {
        Self {
            chain_spec,
            builder: FlashblockBuilder::new(),
            block: None,
            balance_cache: HashMap::new(),
            nonce_cache: HashMap::new(),
            receipts_cache: HashMap::new(),
        }
    }

    pub fn get_block(&self, full: bool) -> Option<RpcBlock<Optimism>> {
        let block = match &self.block {
            Some(block) => block,
            None => return None,
        };

        let header: alloy_consensus::Header = block.header.clone();
        let transactions = block.body.transactions.to_vec();

        if full {
            let transactions_with_senders = transactions
                .into_iter()
                .zip(block.body.recover_signers().unwrap());
            let converted_txs = transactions_with_senders
                .enumerate()
                .map(|(idx, (tx, sender))| {
                    let signed_tx_ec_recovered = Recovered::new_unchecked(tx.clone(), sender);
                    let tx_info = TransactionInfo {
                        hash: Some(tx.tx_hash()),
                        block_hash: Some(block.header.hash_slow()),
                        block_number: Some(block.number),
                        index: Some(idx as u64),
                        base_fee: block.base_fee_per_gas,
                    };
                    transform_tx(signed_tx_ec_recovered, tx_info)
                })
                .collect();
            Some(RpcBlock::<Optimism> {
                header: Header::from_consensus(header.seal_slow(), None, None),
                transactions: BlockTransactions::Full(converted_txs),
                uncles: Vec::new(),
                withdrawals: Some(Withdrawals::new(Vec::new())),
            })
        } else {
            let tx_hashes = transactions.into_iter().map(|tx| tx.tx_hash()).collect();
            Some(RpcBlock::<Optimism> {
                header: Header::from_consensus(header.seal_slow(), None, None),
                transactions: BlockTransactions::Hashes(tx_hashes),
                uncles: Vec::new(),
                withdrawals: Some(Withdrawals::new(Vec::new())),
            })
        }
    }

    pub fn reset(&mut self) {
        self.block = None;
        self.builder = FlashblockBuilder::new();
        self.balance_cache.clear();
        self.nonce_cache.clear();
        self.receipts_cache.clear();
    }

    pub fn process_payload(&mut self, payload: FlashblocksPayloadV1) -> eyre::Result<()> {
        // Convert metadata with error handling
        let metadata: Metadata = match serde_json::from_value(payload.metadata.clone()) {
            Ok(m) => m,
            Err(e) => {
                return Err(eyre::eyre!("Failed to deserialize metadata: {}", e));
            }
        };

        if payload.index == 0 {
            self.reset();
        }

        self.builder.extend(payload)?;

        let execution_payload = match self.builder.build_envelope(PayloadVersion::V4)? {
            OpExecutionPayloadEnvelope::V4(envelope) => envelope.execution_payload.payload_inner,
            _ => return Err(eyre::eyre!("Invalid payload version")),
        };

        let block: OpBlock = match execution_payload.try_into_block() {
            Ok(block) => block,
            Err(e) => {
                return Err(eyre::eyre!(
                    "Failed to convert execution payload to block: {}",
                    e
                ));
            }
        };

        // Update the nonce for each transaction
        let mut nonce_map = HashMap::new();
        let mut all_receipts = Vec::new();

        for tx in block.body.transactions.iter() {
            if let Ok(from) = tx.recover_signer() {
                let nonce = nonce_map.get(&from).copied().unwrap_or(0);
                nonce_map.insert(from, nonce + 1);
            }

            // update the receipts
            let receipt = metadata
                .receipts
                .get(&tx.tx_hash().to_string())
                .expect("Receipt should exist");

            all_receipts.push(receipt.clone());
        }
        for (address, nonce) in nonce_map.iter() {
            self.nonce_cache.insert(*address, *nonce);
        }

        if !block.body.transactions.is_empty() {
            // The first transaction in an Op block is the L1 info transaction.
            let mut l1_block_info =
                extract_l1_info(&block.body).expect("failed to extract l1 info");
            let block_number = block.number();
            let base_fee = block.base_fee_per_gas();
            let block_hash = block.hash_slow();
            let excess_blob_gas = block.excess_blob_gas();
            let timestamp = block.timestamp();
            let mut gas_used = 0;
            let mut next_log_index = 0;

            // build the receipts
            for (indx, tx) in block.body.transactions.iter().enumerate() {
                let receipt = all_receipts
                    .get(indx)
                    .expect("Receipt should exist for transaction");
                let meta = TransactionMeta {
                    tx_hash: tx.tx_hash(),
                    index: indx as u64,
                    block_hash,
                    block_number,
                    base_fee,
                    excess_blob_gas,
                    timestamp,
                };
                let input: ConvertReceiptInput<'_, OpPrimitives> = ConvertReceiptInput {
                    receipt: receipt.clone(),
                    tx: tx.try_to_recovered_ref()?,
                    gas_used: receipt.cumulative_gas_used() - gas_used,
                    next_log_index,
                    meta,
                };

                gas_used = receipt.cumulative_gas_used();
                next_log_index += receipt.logs().len();

                let rpc_receipt =
                    OpReceiptBuilder::new(&self.chain_spec.clone(), input, &mut l1_block_info)
                        .expect("failed to build receipt")
                        .build();

                self.receipts_cache
                    .insert(tx.tx_hash(), rpc_receipt.clone());
            }
        }

        self.block = Some(block);

        // Store account balances
        for (address, balance) in metadata.new_account_balances.iter() {
            let address = Address::from_str(address)
                .map_err(|e| eyre::eyre!("Failed to parse address: {}", e))?;
            let balance = U256::from_str(balance)
                .map_err(|e| eyre::eyre!("Failed to parse balance: {}", e))?;

            self.balance_cache.insert(address, balance);
        }

        Ok(())
    }

    pub fn get_balance(&self, address: Address) -> Option<U256> {
        self.balance_cache.get(&address).cloned()
    }

    pub fn get_nonce(&self, address: Address) -> Option<u64> {
        self.nonce_cache.get(&address).cloned()
    }

    pub fn get_receipt(&self, tx_hash: &TxHash) -> Option<RpcReceipt<Optimism>> {
        self.receipts_cache.get(tx_hash).cloned()
    }
}

fn transform_tx(tx: Recovered<OpTransactionSigned>, tx_info: TransactionInfo) -> Transaction {
    let tx = tx.convert::<OpTxEnvelope>();

    let TransactionInfo {
        block_hash,
        block_number,
        index: transaction_index,
        base_fee,
        ..
    } = tx_info;

    let effective_gas_price = if tx.is_deposit() {
        // For deposits, we must always set the `gasPrice` field to 0 in rpc
        // deposit tx don't have a gas price field, but serde of `Transaction` will take care of
        // it
        0
    } else {
        base_fee
            .map(|base_fee| {
                tx.effective_tip_per_gas(base_fee).unwrap_or_default() + base_fee as u128
            })
            .unwrap_or_else(|| tx.max_fee_per_gas())
    };

    Transaction {
        inner: alloy_rpc_types_eth::Transaction {
            inner: tx,
            block_hash,
            block_number,
            transaction_index,
            effective_gas_price: Some(effective_gas_price),
        },
        deposit_nonce: None, // TODO
        deposit_receipt_version: None,
    }
}
