#[cfg(test)]
mod tests {
    use crate::{EthApiOverrideServer, FlashblocksApiExt, FlashblocksOverlay, cache::Metadata};
    use alloy_consensus::Receipt;
    use alloy_genesis::Genesis;
    use alloy_primitives::{Address, B256, Bytes, TxHash, U256, address, b256};
    use alloy_provider::{Provider, RootProvider};
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_engine::PayloadId;
    use reth_node_builder::{Node, NodeBuilder, NodeConfig, NodeHandle};
    use reth_node_core::{
        args::{DiscoveryArgs, NetworkArgs, RpcServerArgs},
        exit::NodeExitFuture,
    };
    use reth_optimism_chainspec::OpChainSpecBuilder;
    use reth_optimism_node::{OpNode, args::RollupArgs};
    use reth_optimism_primitives::OpReceipt;
    use reth_provider::providers::BlockchainProvider;
    use reth_tasks::TaskManager;
    use rollup_boost::{
        ExecutionPayloadBaseV1, ExecutionPayloadFlashblockDeltaV1, FlashblocksPayloadV1,
    };
    use std::{any::Any, collections::HashMap, net::SocketAddr, str::FromStr, sync::Arc};
    use tokio::sync::{mpsc, oneshot};
    use url::Url;

    pub struct NodeContext {
        sender: mpsc::Sender<(FlashblocksPayloadV1, oneshot::Sender<()>)>,
        http_api_addr: SocketAddr,
        _node_exit_future: NodeExitFuture,
        _node: Box<dyn Any + Sync + Send>,
        _task_manager: TaskManager,
    }

    impl NodeContext {
        pub async fn send_payload(&self, payload: FlashblocksPayloadV1) -> eyre::Result<()> {
            let (tx, rx) = oneshot::channel();
            self.sender.send((payload, tx)).await?;
            rx.await?;
            Ok(())
        }

        pub async fn provider(&self) -> eyre::Result<RootProvider> {
            let url = format!("http://{}", self.http_api_addr);
            let client = RpcClient::builder().http(url.parse()?);

            Ok(RootProvider::new(client))
        }

        pub async fn send_test_payloads(&self) -> eyre::Result<()> {
            let base_payload = create_first_payload();
            self.send_payload(base_payload).await?;

            let second_payload = create_second_payload();
            self.send_payload(second_payload).await?;

            Ok(())
        }
    }

    async fn setup_node() -> eyre::Result<NodeContext> {
        let tasks = TaskManager::current();
        let exec = tasks.executor();

        let genesis: Genesis = serde_json::from_str(include_str!("assets/genesis.json")).unwrap();
        let chain_spec = Arc::new(
            OpChainSpecBuilder::base_mainnet()
                .genesis(genesis)
                .ecotone_activated()
                .build(),
        );

        let network_config = NetworkArgs {
            discovery: DiscoveryArgs {
                disable_discovery: true,
                ..DiscoveryArgs::default()
            },
            ..NetworkArgs::default()
        };

        // Use with_unused_ports() to let Reth allocate random ports and avoid port collisions
        let node_config = NodeConfig::new(chain_spec.clone())
            .with_network(network_config.clone())
            .with_rpc(RpcServerArgs::default().with_unused_ports().with_http())
            .with_unused_ports();

        let node = OpNode::new(RollupArgs::default());

        // Start websocket server to simulate the builder and send payloads back to the node
        let (sender, mut receiver) =
            mpsc::channel::<(FlashblocksPayloadV1, oneshot::Sender<()>)>(100);

        let NodeHandle {
            node,
            node_exit_future,
        } = NodeBuilder::new(node_config.clone())
            .testing_node(exec.clone())
            .with_types_and_provider::<OpNode, BlockchainProvider<_>>()
            .with_components(node.components_builder())
            .with_add_ons(node.add_ons())
            .extend_rpc_modules(move |ctx| {
                // We are not going to use the websocket connection to send payloads so we use
                // a dummy url.
                let flashblocks_overlay =
                    FlashblocksOverlay::new(Url::parse("ws://localhost:8546")?, chain_spec);

                let eth_api = ctx.registry.eth_api().clone();
                let api_ext = FlashblocksApiExt::new(eth_api.clone(), flashblocks_overlay.clone());

                ctx.modules.replace_configured(api_ext.into_rpc())?;

                tokio::spawn(async move {
                    while let Some((payload, tx)) = receiver.recv().await {
                        flashblocks_overlay.process_payload(payload).unwrap();
                        tx.send(()).unwrap();
                    }
                });

                Ok(())
            })
            .launch()
            .await?;

        let http_api_addr = node
            .rpc_server_handle()
            .http_local_addr()
            .ok_or_else(|| eyre::eyre!("Failed to get http api address"))?;

        Ok(NodeContext {
            sender,
            http_api_addr,
            _node_exit_future: node_exit_future,
            _node: Box::new(node),
            _task_manager: tasks,
        })
    }

    fn create_first_payload() -> FlashblocksPayloadV1 {
        FlashblocksPayloadV1 {
            payload_id: PayloadId::new([0; 8]),
            index: 0,
            base: Some(ExecutionPayloadBaseV1 {
                parent_beacon_block_root: B256::default(),
                parent_hash: B256::default(),
                fee_recipient: Address::ZERO,
                prev_randao: B256::default(),
                block_number: 1,
                gas_limit: 0,
                timestamp: 0,
                extra_data: Bytes::new(),
                base_fee_per_gas: U256::ZERO,
            }),
            diff: ExecutionPayloadFlashblockDeltaV1::default(),
            metadata: serde_json::to_value(Metadata {
                block_number: 1,
                receipts: HashMap::default(),
                new_account_balances: HashMap::default(),
            })
            .unwrap(),
        }
    }

    const TEST_ADDRESS: Address = address!("0x1234567890123456789012345678901234567890");
    const PENDING_BALANCE: u64 = 4600;

    const TX1_HASH: TxHash =
        b256!("0x2be2e6f8b01b03b87ae9f0ebca8bbd420f174bef0fbcc18c7802c5378b78f548");
    const TX2_HASH: TxHash =
        b256!("0xa6155b295085d3b87a3c86e342fe11c3b22f9952d0d85d9d34d223b7d6a17cd8");

    fn create_second_payload() -> FlashblocksPayloadV1 {
        // Create second payload (index 1) with transactions
        // tx1 hash: 0x2be2e6f8b01b03b87ae9f0ebca8bbd420f174bef0fbcc18c7802c5378b78f548 (deposit transaction)
        // tx2 hash: 0xa6155b295085d3b87a3c86e342fe11c3b22f9952d0d85d9d34d223b7d6a17cd8
        let tx1 = Bytes::from_str("0x7ef8f8a042a8ae5ec231af3d0f90f68543ec8bca1da4f7edd712d5b51b490688355a6db794deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e200000044d000a118b00000000000000040000000067cb7cb0000000000077dbd4000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000014edd27304108914dd6503b19b9eeb9956982ef197febbeeed8a9eac3dbaaabdf000000000000000000000000fc56e7272eebbba5bc6c544e159483c4a38f8ba3").unwrap();
        let tx2 = Bytes::from_str("0xf8cd82016d8316e5708302c01c94f39635f2adf40608255779ff742afe13de31f57780b8646e530e9700000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000001bc16d674ec8000000000000000000000000000000000000000000000000000156ddc81eed2a36d68302948ba0a608703e79b22164f74523d188a11f81c25a65dd59535bab1cd1d8b30d115f3ea07f4cfbbad77a139c9209d3bded89091867ff6b548dd714109c61d1f8e7a84d14").unwrap();

        // Send another test flashblock payload
        let payload = FlashblocksPayloadV1 {
            payload_id: PayloadId::new([0; 8]),
            index: 1,
            base: None,
            diff: ExecutionPayloadFlashblockDeltaV1 {
                state_root: B256::default(),
                receipts_root: B256::default(),
                gas_used: 0,
                block_hash: B256::default(),
                transactions: vec![tx1, tx2],
                withdrawals: Vec::new(),
                logs_bloom: Default::default(),
                withdrawals_root: Default::default(),
                blob_gas_used: Default::default(),
            },
            metadata: serde_json::to_value(Metadata {
                block_number: 1,
                receipts: {
                    let mut receipts = HashMap::default();
                    receipts.insert(
                        TX1_HASH.to_string(), // transaction hash as string
                        OpReceipt::Legacy(Receipt {
                            status: true.into(),
                            cumulative_gas_used: 21000,
                            logs: vec![],
                        }),
                    );
                    receipts.insert(
                        TX2_HASH.to_string(), // transaction hash as string
                        OpReceipt::Legacy(Receipt {
                            status: true.into(),
                            cumulative_gas_used: 45000,
                            logs: vec![],
                        }),
                    );
                    receipts
                },
                new_account_balances: {
                    let mut map = HashMap::default();
                    map.insert(
                        TEST_ADDRESS.to_string(),
                        format!("0x{:x}", U256::from(PENDING_BALANCE)),
                    );
                    map
                },
            })
            .unwrap(),
        };

        payload
    }

    #[tokio::test]
    async fn test_get_block_by_number_pending() -> eyre::Result<()> {
        reth_tracing::init_test_tracing();
        let node = setup_node().await?;
        let provider = node.provider().await?;

        let latest_block = provider
            .get_block_by_number(alloy_eips::BlockNumberOrTag::Latest)
            .await?
            .expect("latest block expected");
        assert_eq!(latest_block.number(), 0);

        // Querying pending block when it does not exists yet
        let pending_block = provider
            .get_block_by_number(alloy_eips::BlockNumberOrTag::Pending)
            .await?;
        assert_eq!(pending_block.is_none(), true);

        let base_payload = create_first_payload();
        node.send_payload(base_payload).await?;

        // Query pending block after sending the base payload with an empty delta
        let pending_block = provider
            .get_block_by_number(alloy_eips::BlockNumberOrTag::Pending)
            .await?
            .expect("pending block expected");

        assert_eq!(pending_block.number(), 1);
        assert_eq!(pending_block.transactions.hashes().len(), 0);

        let second_payload = create_second_payload();
        node.send_payload(second_payload).await?;

        // Query pending block after sending the second payload with two transactions
        let block = provider
            .get_block_by_number(alloy_eips::BlockNumberOrTag::Pending)
            .await?
            .expect("pending block expected");

        assert_eq!(block.number(), 1);
        assert_eq!(block.transactions.hashes().len(), 2);

        Ok(())
    }

    #[tokio::test]
    async fn test_get_balance_pending() -> eyre::Result<()> {
        reth_tracing::init_test_tracing();
        let node = setup_node().await?;
        let provider = node.provider().await?;

        node.send_test_payloads().await?;

        let balance = provider.get_balance(TEST_ADDRESS).await?;
        assert_eq!(balance, U256::ZERO);

        let pending_balance = provider.get_balance(TEST_ADDRESS).pending().await?;
        assert_eq!(pending_balance, U256::from(PENDING_BALANCE));

        Ok(())
    }

    #[tokio::test]
    async fn test_get_transaction_receipt_pending() -> eyre::Result<()> {
        reth_tracing::init_test_tracing();
        let node = setup_node().await?;
        let provider = node.provider().await?;

        let receipt = provider.get_transaction_receipt(TX1_HASH).await?;
        assert_eq!(receipt.is_none(), true);

        node.send_test_payloads().await?;

        let receipt = provider
            .get_transaction_receipt(TX1_HASH)
            .await?
            .expect("receipt expected");
        assert_eq!(receipt.gas_used, 21000);

        // TODO: Add a new payload and validate that the receipts from the previous payload
        // are not returned.

        Ok(())
    }
}
