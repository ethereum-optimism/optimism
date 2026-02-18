//! Test utilities for `kona-interop`.

#![allow(missing_docs, unreachable_pub, unused)]

use crate::{ExecutingMessage, MessageIdentifier, traits::InteropProvider};
use alloy_consensus::{Header, Receipt, ReceiptWithBloom, Sealed};
use alloy_primitives::{Address, B256, Bytes, Log, LogData, U256, map::HashMap};
use alloy_sol_types::{SolEvent, SolValue};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::Predeploys;
use op_alloy_consensus::OpReceiptEnvelope;

#[derive(Debug, Clone, Default)]
pub struct MockInteropProvider {
    pub headers: HashMap<u64, HashMap<u64, Sealed<Header>>>,
    pub receipts: HashMap<u64, HashMap<u64, Vec<OpReceiptEnvelope>>>,
}

impl MockInteropProvider {
    pub const fn new(
        headers: HashMap<u64, HashMap<u64, Sealed<Header>>>,
        receipts: HashMap<u64, HashMap<u64, Vec<OpReceiptEnvelope>>>,
    ) -> Self {
        Self { headers, receipts }
    }
}

#[derive(thiserror::Error, Debug, Eq, PartialEq)]
#[error("Mock interop provider error")]
pub struct InteropProviderError;

#[async_trait]
impl InteropProvider for MockInteropProvider {
    type Error = InteropProviderError;

    async fn header_by_number(&self, chain_id: u64, number: u64) -> Result<Header, Self::Error> {
        Ok(self
            .headers
            .get(&chain_id)
            .and_then(|headers| headers.get(&number))
            .unwrap()
            .inner()
            .clone())
    }

    async fn receipts_by_number(
        &self,
        chain_id: u64,
        number: u64,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error> {
        Ok(self.receipts.get(&chain_id).and_then(|receipts| receipts.get(&number)).unwrap().clone())
    }

    async fn receipts_by_hash(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error> {
        Ok(self
            .receipts
            .get(&chain_id)
            .and_then(|receipts| {
                let headers = self.headers.get(&chain_id).unwrap();
                let number =
                    headers.values().find(|header| header.hash() == block_hash).unwrap().number;
                receipts.get(&number)
            })
            .unwrap()
            .clone())
    }
}

#[derive(Default, Debug)]
pub struct SuperchainBuilder {
    chains: HashMap<u64, ChainBuilder>,
}

impl SuperchainBuilder {
    pub fn new() -> Self {
        Self { chains: HashMap::default() }
    }

    pub fn chain(&mut self, chain_id: u64) -> &mut ChainBuilder {
        self.chains.entry(chain_id).or_default()
    }

    /// Builds the scenario into the format needed for testing
    pub fn build(
        self,
    ) -> (HashMap<u64, Sealed<Header>>, HashMap<u64, RollupConfig>, MockInteropProvider) {
        let mut headers_map = HashMap::default();
        let mut receipts_map = HashMap::default();
        let mut sealed_headers = HashMap::default();
        let mut rollup_cfgs = HashMap::default();

        for (chain_id, chain) in self.chains {
            let header = chain.header;
            let header_hash = header.hash_slow();
            let sealed_header = header.seal(header_hash);

            let mut chain_headers = HashMap::default();
            chain_headers.insert(sealed_header.number, sealed_header.clone());
            headers_map.insert(chain_id, chain_headers);

            let mut chain_receipts = HashMap::default();
            chain_receipts.insert(sealed_header.number, chain.receipts);
            receipts_map.insert(chain_id, chain_receipts);

            sealed_headers.insert(chain_id, sealed_header);
            rollup_cfgs.insert(chain_id, chain.rollup_config);
        }

        (sealed_headers, rollup_cfgs, MockInteropProvider::new(headers_map, receipts_map))
    }
}

#[derive(Default, Debug)]
pub struct ChainBuilder {
    pub rollup_config: RollupConfig,
    pub header: Header,
    pub receipts: Vec<OpReceiptEnvelope>,
}

impl ChainBuilder {
    pub fn modify_rollup_cfg(&mut self, f: impl FnOnce(&mut RollupConfig)) -> &mut Self {
        f(&mut self.rollup_config);
        self
    }

    pub fn with_block_time(&mut self, block_time: u64) -> &mut Self {
        self.modify_rollup_cfg(|cfg| cfg.block_time = block_time)
    }

    pub fn with_interop_activation_time(&mut self, activation: u64) -> &mut Self {
        self.modify_rollup_cfg(|cfg| cfg.hardforks.interop_time = Some(activation))
    }

    pub fn modify_header(&mut self, f: impl FnOnce(&mut Header)) -> &mut Self {
        f(&mut self.header);
        self
    }

    pub fn with_timestamp(&mut self, timestamp: u64) -> &mut Self {
        self.modify_header(|h| h.timestamp = timestamp)
    }

    pub fn add_initiating_message(&mut self, message_data: Bytes) -> &mut Self {
        let receipt = OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt {
                logs: vec![Log {
                    address: Address::ZERO,
                    data: LogData::new(vec![], message_data).unwrap(),
                }],
                ..Default::default()
            },
            ..Default::default()
        });
        self.receipts.push(receipt);
        self
    }

    pub fn add_executing_message(&mut self, builder: ExecutingMessageBuilder) -> &mut Self {
        let receipt = OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt {
                logs: vec![Log {
                    address: Predeploys::CROSS_L2_INBOX,
                    data: LogData::new(
                        vec![ExecutingMessage::SIGNATURE_HASH, builder.message_hash],
                        MessageIdentifier {
                            origin: builder.origin_address,
                            blockNumber: U256::from(builder.origin_block_number),
                            logIndex: U256::from(builder.origin_log_index),
                            timestamp: U256::from(builder.origin_timestamp),
                            chainId: U256::from(builder.origin_chain_id),
                        }
                        .abi_encode()
                        .into(),
                    )
                    .unwrap(),
                }],
                ..Default::default()
            },
            ..Default::default()
        });
        self.receipts.push(receipt);
        self
    }
}

#[derive(Default, Debug)]
pub struct ExecutingMessageBuilder {
    pub message_hash: B256,
    pub origin_address: Address,
    pub origin_log_index: u64,
    pub origin_chain_id: u64,
    pub origin_block_number: u64,
    pub origin_timestamp: u64,
}

impl ExecutingMessageBuilder {
    pub const fn with_message_hash(mut self, message_hash: B256) -> Self {
        self.message_hash = message_hash;
        self
    }

    pub const fn with_origin_address(mut self, origin_address: Address) -> Self {
        self.origin_address = origin_address;
        self
    }

    pub const fn with_origin_log_index(mut self, origin_log_index: u64) -> Self {
        self.origin_log_index = origin_log_index;
        self
    }

    pub const fn with_origin_chain_id(mut self, origin_chain_id: u64) -> Self {
        self.origin_chain_id = origin_chain_id;
        self
    }

    pub const fn with_origin_block_number(mut self, origin_block_number: u64) -> Self {
        self.origin_block_number = origin_block_number;
        self
    }

    pub const fn with_origin_timestamp(mut self, origin_timestamp: u64) -> Self {
        self.origin_timestamp = origin_timestamp;
        self
    }
}
