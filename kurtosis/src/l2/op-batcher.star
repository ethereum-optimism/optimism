def run(
        plan,
        name,
        l1_eth_rpc,
        l2_eth_rpc,
        l2_rollup_rpc,
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/op-batcher',
        tag='develop',
        additional_env={},
):
    return plan.add_service(
        name=name,
        config=ServiceConfig(
            image='{}:{}'.format(image, tag),
            ports={
                'http': PortSpec(8545),
            },
            env_vars={
                'OP_BATCHER_L1_ETH_RPC': l1_eth_rpc,
                'OP_BATCHER_L2_ETH_RPC': l2_eth_rpc,
                'OP_BATCHER_ROLLUP_RPC': l2_rollup_rpc,
                'OP_BATCHER_MAX_CHANNEL_DURATION': '1',
                'OP_BATCHER_SUB_SAFETY_MARGIN': '4',
                'OP_BATCHER_POLL_INTERVAL': '1s',
                'OP_BATCHER_NUM_CONFIRMATIONS': '1',
                'OP_BATCHER_MNEMONIC': 'test test test test test test test test test test test junk',
                'OP_BATCHER_SEQUENCER_HD_PATH': "m/44'/60'/0'/0/0",
                'OP_BATCHER_BATCH_TYPE': '0',
                'OP_BATCHER_RPC_ADDR': '0.0.0.0',
            } | additional_env,
        )
    )
