//! A real L2 chain over a scripted L1 chain, for tests of the sync start and reorg walks.

use crate::test_utils::RpcMock;
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256, U256};
use kona_genesis::RollupConfig;
use kona_protocol::{L1BlockInfoBedrock, L2BlockInfo};
use serde_json::json;

/// An L2 block as the engine's RPC returns it.
pub type L2RpcBlock = <op_alloy_network::Optimism as alloy_network::Network>::BlockResponse;

/// L1 block `number` on `fork`: fork 0 is the chain derivation read, fork 1 replaced it from
/// some height on.
pub fn test_l1_block(number: u64, fork: u8) -> BlockNumHash {
    BlockNumHash { number, hash: B256::left_padding_from(&[fork, number as u8]) }
}

/// Scripts the L1 node's lookup of block `number`, on the chain whose blocks from
/// `replaced_from` on are on fork 1.
pub fn expect_l1_block(l1: &RpcMock, number: u64, replaced_from: u64) {
    let mut block = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
    block.header.hash = test_l1_block(number, u8::from(number >= replaced_from)).hash;
    block.header.inner.number = number;
    l1.expect_params("eth_getBlockByNumber", json!([format!("{number:#x}"), false]), block);
}

/// A real L2 chain from block 18 to block 26, so the unsafe walks can fetch and decode its
/// blocks. Blocks 18 to 20 have L1 origins 99, 100 and 100, blocks 21 to 24 have L1 origins
/// 101 to 104, and blocks 25 and 26 share L1 origin 105.
#[derive(Debug)]
pub struct TestChain {
    blocks: Vec<(L2BlockInfo, L2RpcBlock)>,
}

impl Default for TestChain {
    fn default() -> Self {
        Self::new()
    }
}

impl TestChain {
    /// Builds the chain.
    pub fn new() -> Self {
        let mut blocks: Vec<(L2BlockInfo, L2RpcBlock)> = Vec::new();
        for number in 18..=26 {
            let origin = match number {
                18 => 99,
                19 | 20 => 100,
                25 | 26 => 105,
                n => 80 + n,
            };
            let info = L1BlockInfoBedrock::new(
                origin,
                0,
                0,
                test_l1_block(origin, 0).hash,
                0,
                Address::ZERO,
                U256::ZERO,
                U256::ZERO,
            );
            let deposit = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
                op_alloy_consensus::TxDeposit {
                    input: info.encode_calldata(),
                    ..Default::default()
                },
            ));
            let header = alloy_consensus::Header {
                number,
                parent_hash: blocks.last().map_or(B256::ZERO, |(parent, _)| parent.block_info.hash),
                timestamp: number,
                ..Default::default()
            };
            let consensus = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope> {
                header: header.clone(),
                body: alloy_consensus::BlockBody {
                    transactions: vec![deposit.clone()],
                    ..Default::default()
                },
            };
            let block_info =
                L2BlockInfo::from_block_and_genesis(&consensus, &RollupConfig::default().genesis)
                    .unwrap();
            let rpc = L2RpcBlock {
                header: alloy_rpc_types_eth::Header {
                    hash: block_info.block_info.hash,
                    inner: header,
                    ..Default::default()
                },
                transactions: alloy_rpc_types_eth::BlockTransactions::Full(vec![
                    op_alloy_rpc_types::Transaction {
                        inner: alloy_rpc_types_eth::Transaction {
                            inner: alloy_consensus::transaction::Recovered::new_unchecked(
                                deposit,
                                Address::ZERO,
                            ),
                            block_hash: Some(block_info.block_info.hash),
                            block_number: Some(number),
                            effective_gas_price: Some(0),
                            transaction_index: Some(0),
                            block_timestamp: None,
                        },
                        deposit_nonce: None,
                        deposit_receipt_version: None,
                    },
                ]),
                ..Default::default()
            };
            blocks.push((block_info, rpc));
        }
        Self { blocks }
    }

    /// Block `number`'s info.
    pub fn info(&self, number: u64) -> L2BlockInfo {
        self.block(number).0
    }

    /// Block `number` as the engine's RPC returns it.
    pub fn rpc(&self, number: u64) -> &L2RpcBlock {
        &self.block(number).1
    }

    /// Scripts the engine's lookup of block `number` by hash, with transactions.
    pub fn expect_by_hash(&self, l2: &RpcMock, number: u64) {
        let (info, rpc) = self.block(number);
        l2.expect_params("eth_getBlockByHash", json!([info.block_info.hash, true]), rpc);
    }

    /// Scripts the engine's lookups of its finalized, safe and latest blocks.
    pub fn expect_heads(&self, l2: &RpcMock, finalized: u64, safe: u64, latest: u64) {
        for (tag, number) in [("finalized", finalized), ("safe", safe), ("latest", latest)] {
            l2.expect_params("eth_getBlockByNumber", json!([tag, true]), self.rpc(number));
        }
    }

    fn block(&self, number: u64) -> &(L2BlockInfo, L2RpcBlock) {
        self.blocks
            .iter()
            .find(|(info, _)| info.block_info.number == number)
            .unwrap_or_else(|| panic!("no block {number} in the test chain"))
    }
}
