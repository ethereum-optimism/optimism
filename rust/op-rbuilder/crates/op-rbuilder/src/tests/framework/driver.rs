use core::time::Duration;

use alloy_eips::{BlockNumberOrTag, Encodable2718, eip7685::Requests};
use alloy_primitives::{B64, B256, Bytes, TxKind, U256, address, hex};
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadAttributes, PayloadStatusEnum};
use alloy_rpc_types_eth::Block;
use op_alloy_consensus::{OpTypedTransaction, TxDeposit};
use op_alloy_network::Optimism;
use op_alloy_rpc_types::Transaction;
use reth_optimism_node::OpPayloadAttributes;
use rollup_boost::OpExecutionPayloadEnvelope;

use super::{EngineApi, Ipc, LocalInstance, TransactionBuilder};
use crate::{
    args::OpRbuilderArgs,
    tests::{
        DEFAULT_DENOMINATOR, DEFAULT_ELASTICITY, ExternalNode, Protocol,
        framework::DEFAULT_GAS_LIMIT,
    },
    tx_signer::Signer,
};

/// The ChainDriver is a type that allows driving the op builder node to build new blocks manually
/// by calling the `build_new_block` method. It uses the Engine API to interact with the node
/// and the provider to fetch blocks and transactions.
pub struct ChainDriver<RpcProtocol: Protocol = Ipc> {
    engine_api: EngineApi<RpcProtocol>,
    provider: RootProvider<Optimism>,
    signer: Option<Signer>,
    gas_limit: Option<u64>,
    args: OpRbuilderArgs,
    validation_nodes: Vec<ExternalNode>,
}

// instantiation and configuration
impl<RpcProtocol: Protocol> ChainDriver<RpcProtocol> {
    const MIN_BLOCK_TIME: Duration = Duration::from_secs(1);

    /// Creates a new ChainDriver instance for a local instance of RBuilder running in-process
    /// communicating over IPC.
    pub async fn local(instance: &LocalInstance) -> eyre::Result<ChainDriver<Ipc>> {
        Ok(ChainDriver::<Ipc> {
            engine_api: instance.engine_api(),
            provider: instance.provider().await?,
            signer: Default::default(),
            gas_limit: None,
            args: instance.args().clone(),
            validation_nodes: vec![],
        })
    }

    /// Creates a new ChainDriver for some EL node instance.
    pub fn remote(
        provider: RootProvider<Optimism>,
        engine_api: EngineApi<RpcProtocol>,
    ) -> ChainDriver<RpcProtocol> {
        ChainDriver {
            engine_api,
            provider,
            signer: Default::default(),
            gas_limit: None,
            args: OpRbuilderArgs::default(),
            validation_nodes: vec![],
        }
    }

    /// Specifies the block builder signing key used to sign builder transactions.
    /// If not specified, a random signer will be used.
    pub fn with_signer(mut self, signer: Signer) -> Self {
        self.signer = Some(signer);
        self
    }

    /// Specifies a custom gas limit for blocks being built, otherwise the limit is
    /// set to a default value of 10_000_000.
    pub fn with_gas_limit(mut self, gas_limit: u64) -> Self {
        self.gas_limit = Some(gas_limit);
        self
    }

    /// Adds an external Optimism execution client node that will receive all newly built
    /// blocks by this driver and ensure that they are valid. This validation process is
    /// transparent and happens in the background when building new blocks.
    ///
    /// If there are validation nodes specified any newly built block will be submitted to
    /// the validation EL and the driver will fail if the block is rejected by the
    /// validation node.
    pub async fn with_validation_node(mut self, node: ExternalNode) -> eyre::Result<Self> {
        node.catch_up_with(self.provider()).await?;
        self.validation_nodes.push(node);
        Ok(self)
    }
}

// public test api
impl<RpcProtocol: Protocol> ChainDriver<RpcProtocol> {
    pub async fn build_new_block_with_no_tx_pool(&self) -> eyre::Result<Block<Transaction>> {
        self.build_new_block_with_txs_timestamp(vec![], Some(true), None, None, Some(0))
            .await
    }

    /// Builds a new block using the current state of the chain and the transactions in the pool.
    pub async fn build_new_block(&self) -> eyre::Result<Block<Transaction>> {
        self.build_new_block_with_txs(vec![]).await
    }

    /// Builds a new block with block_timestamp calculated as block time right before sending FCU
    pub async fn build_new_block_with_current_timestamp(
        &self,
        timestamp_jitter: Option<Duration>,
    ) -> eyre::Result<Block<Transaction>> {
        self.build_new_block_with_txs_timestamp(vec![], None, None, timestamp_jitter, Some(0))
            .await
    }

    /// Builds a new block with provided txs and timestamp
    pub async fn build_new_block_with_txs_timestamp(
        &self,
        txs: Vec<Bytes>,
        no_tx_pool: Option<bool>,
        block_timestamp: Option<Duration>,
        // Amount of time to lag before sending FCU. This tests late FCU scenarios
        timestamp_jitter: Option<Duration>,
        min_base_fee: Option<u64>,
    ) -> eyre::Result<Block<Transaction>> {
        let latest = self.latest().await?;

        // Add L1 block info as the first transaction in every L2 block
        // This deposit transaction contains L1 block metadata required by the L2 chain
        // Currently using hardcoded data from L1 block 124665056
        // If this info is not provided, Reth cannot decode the receipt for any transaction
        // in the block since it also includes this info as part of the result.
        // It does not matter if the to address (4200000000000000000000000000000000000015) is
        // not deployed on the L2 chain since Reth queries the block to get the info and not the contract.
        let block_info_tx: Bytes = {
            let deposit_tx = TxDeposit {
                source_hash: B256::default(),
                from: address!("DeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001"),
                to: TxKind::Call(address!("4200000000000000000000000000000000000015")),
                mint: 0,
                value: U256::default(),
                gas_limit: 210000,
                is_system_transaction: false,
                input: JOVIAN_DATA.into(),
            };

            // Create a temporary signer for the deposit
            let signer = self.signer.unwrap_or_else(Signer::random);
            let signed_tx = signer.sign_tx(OpTypedTransaction::Deposit(deposit_tx))?;
            signed_tx.encoded_2718().into()
        };

        let mut wait_until = None;
        // If block_timestamp we need to produce new timestamp according to current clocks
        let block_timestamp = if let Some(block_timestamp) = block_timestamp {
            block_timestamp.as_secs()
        } else {
            // We take the following second, until which we will need to wait before issuing FCU
            let latest_timestamp = (chrono::Utc::now().timestamp() + 1) as u64;
            wait_until = Some(latest_timestamp);
            latest_timestamp
                + Duration::from_millis(self.args.chain_block_time)
                    .as_secs()
                    .max(Self::MIN_BLOCK_TIME.as_secs())
        };

        // This step will alight time at which we send FCU. ideally we must send FCU and the beginning of the second.
        if let Some(wait_until) = wait_until {
            let sleep_time = Duration::from_secs(wait_until).saturating_sub(Duration::from_millis(
                chrono::Utc::now().timestamp_millis() as u64,
            ));
            if let Some(timestamp_jitter) = timestamp_jitter {
                tokio::time::sleep(sleep_time + timestamp_jitter).await;
            } else {
                tokio::time::sleep(sleep_time).await;
            }
        }

        let eip_1559_params: u64 =
            ((DEFAULT_DENOMINATOR as u64) << 32) | (DEFAULT_ELASTICITY as u64);

        let fcu_result = self
            .fcu(OpPayloadAttributes {
                payload_attributes: PayloadAttributes {
                    timestamp: block_timestamp,
                    parent_beacon_block_root: Some(B256::ZERO),
                    withdrawals: Some(vec![]),
                    ..Default::default()
                },
                transactions: Some(vec![block_info_tx].into_iter().chain(txs).collect()),
                gas_limit: Some(self.gas_limit.unwrap_or(DEFAULT_GAS_LIMIT)),
                no_tx_pool,
                min_base_fee,
                eip_1559_params: Some(B64::from(eip_1559_params)),
            })
            .await?;

        if fcu_result.payload_status.is_invalid() {
            return Err(eyre::eyre!("Forkchoice update failed: {fcu_result:?}"));
        }

        let payload_id = fcu_result
            .payload_id
            .ok_or_else(|| eyre::eyre!("Forkchoice update did not return a payload ID"))?;

        // wait for the block to be built for the specified chain block time
        if let Some(timestamp_jitter) = timestamp_jitter {
            tokio::time::sleep(
                Duration::from_millis(self.args.chain_block_time)
                    .max(Self::MIN_BLOCK_TIME)
                    .saturating_sub(timestamp_jitter),
            )
            .await;
        } else {
            tokio::time::sleep(
                Duration::from_millis(self.args.chain_block_time).max(Self::MIN_BLOCK_TIME),
            )
            .await;
        }

        let payload =
            OpExecutionPayloadEnvelope::V4(self.engine_api.get_payload(payload_id).await?);

        let OpExecutionPayloadEnvelope::V4(payload) = payload else {
            return Err(eyre::eyre!("Expected V4 payload, got something else"));
        };

        let payload = payload.execution_payload;

        if self
            .engine_api
            .new_payload(payload.clone(), vec![], B256::ZERO, Requests::default())
            .await?
            .status
            != PayloadStatusEnum::Valid
        {
            return Err(eyre::eyre!("Invalid validation status from builder"));
        }

        let new_block_hash = payload.payload_inner.payload_inner.payload_inner.block_hash;

        self.engine_api
            .update_forkchoice(latest.header.hash, new_block_hash, None)
            .await?;

        let block = self
            .provider
            .get_block_by_number(BlockNumberOrTag::Latest)
            .full()
            .await?
            .ok_or_else(|| eyre::eyre!("Failed to get latest block after building new block"))?;

        assert_eq!(
            block.header.hash, new_block_hash,
            "New block hash does not match expected hash"
        );

        for validation_node in &self.validation_nodes {
            // optionally for each external validation node, ensure
            // that they consider the block valid before returning it
            // to the test author.
            validation_node.post_block(&payload).await?;
        }

        Ok(block)
    }

    /// Builds a new block using the current state of the chain and the transactions in the pool with a list
    /// of mandatory builder transactions. Those are usually deposit transactions.
    pub async fn build_new_block_with_txs(
        &self,
        txs: Vec<Bytes>,
    ) -> eyre::Result<Block<Transaction>> {
        let latest = self.latest().await?;
        let latest_timestamp = Duration::from_secs(latest.header.timestamp);
        let block_timestamp = latest_timestamp + Self::MIN_BLOCK_TIME;

        self.build_new_block_with_txs_timestamp(txs, None, Some(block_timestamp), None, Some(0))
            .await
    }

    /// Retreives the latest built block and returns only a list of transaction
    /// hashes from its body.
    pub async fn latest(&self) -> eyre::Result<Block<Transaction>> {
        self.provider
            .get_block_by_number(alloy_eips::BlockNumberOrTag::Latest)
            .await?
            .ok_or_else(|| eyre::eyre!("Failed to get latest block"))
    }

    /// Retreives the latest built block and returns a list of full transaction
    /// contents in its body.
    pub async fn latest_full(&self) -> eyre::Result<Block<Transaction>> {
        self.provider
            .get_block_by_number(alloy_eips::BlockNumberOrTag::Latest)
            .full()
            .await?
            .ok_or_else(|| eyre::eyre!("Failed to get latest full block"))
    }

    /// retreives a specific block by its number or tag and returns a list of transaction
    /// hashes from its body.
    pub async fn get_block(
        &self,
        number: BlockNumberOrTag,
    ) -> eyre::Result<Option<Block<Transaction>>> {
        Ok(self.provider.get_block_by_number(number).await?)
    }

    /// retreives a specific block by its number or tag and returns a list of full transaction
    /// contents in its body.
    pub async fn get_block_full(
        &self,
        number: BlockNumberOrTag,
    ) -> eyre::Result<Option<Block<Transaction>>> {
        Ok(self.provider.get_block_by_number(number).full().await?)
    }

    /// Returns a transaction builder that can be used to create and send transactions.
    pub fn create_transaction(&self) -> TransactionBuilder {
        TransactionBuilder::new(self.provider.clone())
    }

    /// Returns a reference to the underlying alloy provider that is used to
    /// interact with the chain.
    pub const fn provider(&self) -> &RootProvider<Optimism> {
        &self.provider
    }
}

// internal methods
impl<RpcProtocol: Protocol> ChainDriver<RpcProtocol> {
    async fn fcu(&self, attribs: OpPayloadAttributes) -> eyre::Result<ForkchoiceUpdated> {
        let latest = self.latest().await?.header.hash;
        let response = self
            .engine_api
            .update_forkchoice(latest, latest, Some(attribs))
            .await?;

        Ok(response)
    }
}

// L1 block info for OP mainnet block 124665056 (stored in input of tx at index 0)
// https://optimistic.etherscan.io/tx/0x312e290cf36df704a2217b015d6455396830b0ce678b860ebfcc30f41403d7b1
// It has the following modifications:
// 1. Function signature  support Jovian: cast sig "setL1BlockValuesJovian()" => 0x3db6be2b
// 2. Zero operator fee scalar
// 3. Zero operator fee constant
// 4. DA footprint of 400 applied
// See: // https://specs.optimism.io/protocol/jovian/l1-attributes.html for Jovian specs.
const JOVIAN_DATA: &[u8] = &hex!(
    "3db6be2b0000146b000f79c500000000000000040000000066d052e700000000013ad8a
    3000000000000000000000000000000000000000000000000000000003ef12787000000
    00000000000000000000000000000000000000000000000000000000012fdf87b89884a
    61e74b322bbcf60386f543bfae7827725efaaf0ab1de2294a5900000000000000000000
    00006887246668a3b87f54deb3b94ba47a6f63f32985
    00000000
    0000000000000000
    0190"
);
