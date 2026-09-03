//! An [`L2ChainProvider`] that prefers blocks the node has already imported over JSON-RPC.

use crate::{AlloyL2ChainProvider, AlloyL2ChainProviderError};
use alloy_primitives::B256;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_derive::L2ChainProvider;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo, to_system_config};
use op_alloy_consensus::OpBlock;
use std::sync::Arc;

/// Blocks the node has imported, looked up by hash.
///
/// A source may drop any block at any time; derivation only loses a fetch.
#[async_trait]
pub trait ImportedL2Blocks: Debug + Send + Sync {
    /// The imported block with `hash` and its derivation summary, if the source still holds it.
    async fn imported_l2_block(&self, hash: B256) -> Option<(L2BlockInfo, Arc<OpBlock>)>;
}

/// Answers derivation's L2 lookups from blocks the node has already imported, falling back to
/// JSON-RPC for anything the source does not hold.
///
/// Only the hash-keyed lookups are served locally. A height does not identify a block across a
/// reorg, so the by-number lookups stay on the authoritative RPC.
#[derive(Debug, Clone)]
pub struct BufferedAlloyL2ChainProvider {
    /// Blocks the node has imported.
    imported: Arc<dyn ImportedL2Blocks>,
    /// The authoritative fallback.
    rpc: AlloyL2ChainProvider,
}

impl BufferedAlloyL2ChainProvider {
    /// Creates a new [`BufferedAlloyL2ChainProvider`] over the given block source and RPC
    /// provider.
    pub fn new(imported: Arc<dyn ImportedL2Blocks>, rpc: AlloyL2ChainProvider) -> Self {
        Self { imported, rpc }
    }
}

#[async_trait]
impl BatchValidationProvider for BufferedAlloyL2ChainProvider {
    type Error = AlloyL2ChainProviderError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        self.rpc.l2_block_info_by_number(number).await
    }

    async fn l2_block_info_by_hash(&mut self, hash: B256) -> Result<L2BlockInfo, Self::Error> {
        match self.imported.imported_l2_block(hash).await {
            Some((info, _)) => Ok(info),
            None => self.rpc.l2_block_info_by_hash(hash).await,
        }
    }

    async fn block_by_number(&mut self, number: u64) -> Result<Arc<OpBlock>, Self::Error> {
        self.rpc.block_by_number(number).await
    }
}

#[async_trait]
impl L2ChainProvider for BufferedAlloyL2ChainProvider {
    type Error = AlloyL2ChainProviderError;

    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
        rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error> {
        if hash == rollup_config.genesis.l2.hash &&
            let Some(system_config) = rollup_config.genesis.system_config
        {
            return Ok(system_config);
        }
        // The source is only ever a cache, so any failure defers to the authoritative RPC.
        if let Some((_, block)) = self.imported.imported_l2_block(hash).await &&
            let Ok(system_config) = to_system_config(&block, &rollup_config)
        {
            return Ok(system_config);
        }
        self.rpc.system_config_by_l2_hash(hash, rollup_config).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use httpmock::prelude::*;
    use kona_genesis::ChainGenesis;
    use op_alloy_network::Optimism;
    use std::collections::HashMap;

    const GENESIS_HASH: B256 = B256::repeat_byte(0xaa);

    /// A source holding exactly the blocks it was built with.
    #[derive(Debug, Default)]
    struct Held(HashMap<B256, (L2BlockInfo, Arc<OpBlock>)>);

    #[async_trait]
    impl ImportedL2Blocks for Held {
        async fn imported_l2_block(&self, hash: B256) -> Option<(L2BlockInfo, Arc<OpBlock>)> {
            self.0.get(&hash).cloned()
        }
    }

    /// A config whose genesis block is `GENESIS_HASH`.
    fn rollup_config() -> Arc<RollupConfig> {
        Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: alloy_eips::BlockNumHash { hash: GENESIS_HASH, number: 0 },
                system_config: Some(SystemConfig { gas_limit: 30_000_000, ..Default::default() }),
                ..Default::default()
            },
            ..Default::default()
        })
    }

    fn rpc(server: &MockServer, config: Arc<RollupConfig>) -> AlloyL2ChainProvider {
        AlloyL2ChainProvider::new(
            RootProvider::<Optimism>::new(RpcClient::new_http(server.base_url().parse().unwrap())),
            config,
            8,
        )
    }

    /// A provider over an empty source, so every lookup falls through to `server`.
    fn provider(server: &MockServer, config: Arc<RollupConfig>) -> BufferedAlloyL2ChainProvider {
        BufferedAlloyL2ChainProvider::new(Arc::new(Held::default()), rpc(server, config))
    }

    /// Answers the RPC block lookup with `null`, i.e. "no such block".
    fn mount_block_lookup(server: &MockServer) -> httpmock::Mock<'_> {
        server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockByHash");
            then.status(200)
                .header("content-type", "application/json")
                .body(r#"{"jsonrpc":"2.0","id":0,"result":null}"#);
        })
    }

    #[tokio::test]
    async fn the_genesis_config_is_answered_locally() {
        let server = MockServer::start();
        let rpc_lookup = mount_block_lookup(&server);
        let config = rollup_config();

        let system_config = provider(&server, config.clone())
            .system_config_by_l2_hash(GENESIS_HASH, config.clone())
            .await
            .unwrap();

        rpc_lookup.assert_calls(0);
        assert_eq!(Some(system_config), config.genesis.system_config);
    }

    #[tokio::test]
    async fn imported_blocks_are_converted_without_touching_the_rpc() {
        // The point of the source: a block the node imported is turned into a SystemConfig
        // locally. Genesis sits at an unrelated hash so nothing short-circuits, and the block
        // carries a real L1-info deposit so the conversion has to actually decode it.
        let header =
            alloy_consensus::Header { number: 7, gas_limit: 30_000_000, ..Default::default() };
        let block = OpBlock {
            header,
            body: alloy_consensus::BlockBody {
                transactions: vec![op_alloy_consensus::OpTxEnvelope::Deposit(
                    alloy_primitives::Sealed::new(op_alloy_consensus::TxDeposit {
                        input: alloy_primitives::Bytes::from(
                            &kona_protocol::test_utils::RAW_ECOTONE_INFO_TX,
                        ),
                        ..Default::default()
                    }),
                )],
                ..Default::default()
            },
        };
        let block_hash = block.header.hash_slow();

        let config = Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: alloy_eips::BlockNumHash { hash: GENESIS_HASH, number: 0 },
                system_config: Some(SystemConfig::default()),
                ..Default::default()
            },
            ..Default::default()
        });
        let l2_block_info = L2BlockInfo::from_block_and_genesis(&block, &config.genesis)
            .expect("the deposit describes the block's L1 origin");

        let server = MockServer::start();
        let rpc_lookup = mount_block_lookup(&server);
        let held = Held(HashMap::from([(block_hash, (l2_block_info, Arc::new(block)))]));
        let mut provider =
            BufferedAlloyL2ChainProvider::new(Arc::new(held), rpc(&server, config.clone()));

        let system_config = provider.system_config_by_l2_hash(block_hash, config).await.unwrap();
        assert_eq!(provider.l2_block_info_by_hash(block_hash).await.unwrap(), l2_block_info);

        rpc_lookup.assert_calls(0);
        // Decoded from the deposit's calldata, not echoed from genesis.
        assert_eq!(
            system_config.batcher_address,
            alloy_primitives::address!("6887246668a3b87f54deb3b94ba47a6f63f32985")
        );
        assert_eq!(system_config.gas_limit, 30_000_000, "read from the imported block's header");
    }

    #[tokio::test]
    async fn unknown_blocks_fall_back_to_the_rpc() {
        let server = MockServer::start();
        let rpc_lookup = mount_block_lookup(&server);
        let config = rollup_config();

        let err = provider(&server, config.clone())
            .system_config_by_l2_hash(B256::repeat_byte(0xbb), config)
            .await
            .unwrap_err();

        rpc_lookup.assert_calls(1);
        assert!(matches!(err, AlloyL2ChainProviderError::BlockHashNotFound(_)));
    }
}
