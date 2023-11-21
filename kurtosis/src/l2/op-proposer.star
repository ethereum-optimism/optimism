def run (
        plan,
        name,
        l1_eth_rpc,
        l2_rollup_rpc,
        l2oo_address,
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/op-proposer',
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
                'OP_PROPOSER_L1_ETH_RPC': l1_eth_rpc,
                'OP_PROPOSER_ROLLUP_RPC': l2_rollup_rpc,
                'OP_PROPOSER_POLL_INTERVAL': '1s',
                'OP_PROPOSER_NUM_CONFIRMATIONS': '1',
                'OP_PROPOSER_MNEMONIC': 'test test test test test test test test test test test junk',
                'OP_PROPOSER_L2_OUTPUT_HD_PATH': "m/44'/60'/0'/0/1",
                'OP_PROPOSER_L2OO_ADDRESS': l2oo_address,
                'OP_PROPOSER_RPC_ADDR': '0.0.0.0'
            } | additional_env,
        )
    )