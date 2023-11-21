def run(
        plan,
        name,
        l1_eth_rpc,
        l2_engine_rpc,
        jwt_secret,
        p2p_key,
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node',
        tag='develop',
        additional_env={},
):
    return plan.add_service(
        name=name,
        config=ServiceConfig(
            image='{}:{}'.format(image, tag),
            ports={
                'http': PortSpec(9545),
                'p2p': PortSpec(9003),
            },
            env_vars={
                'OP_NODE_L1_ETH_RPC': l1_eth_rpc,
                'OP_NODE_L2_ENGINE_RPC': l2_engine_rpc,
                'OP_NODE_L2_ENGINE_AUTH': '/jwt/jwt-secret.txt',
                'OP_NODE_SEQUENCER_ENABLED': 'true',
                'OP_NODE_SEQUENCER_L1_CONFS': '0',
                'OP_NODE_VERIFIER_L1_CONFS': '0',
                'OP_NODE_P2P_LISTEN_TCP_PORT': '9003',
                'OP_NODE_P2P_SEQUENCER_KEY': '5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a',
                'OP_NODE_ROLLUP_CONFIG': '/config/rollup.json',
                'OP_NODE_RPC_ADDR': '0.0.0.0',
                'OP_NODE_P2P_LISTEN_IP': '0.0.0.0',
                'OP_NODE_P2P_PRIV_PATH': '/p2p/sequencer-p2p-key.txt'
            } | additional_env,
            files={
                '/config': 'rollup_config',
                '/jwt': jwt_secret,
                '/p2p': p2p_key
            },
            ready_conditions=ReadyCondition(
                recipe=PostHttpRequestRecipe(
                    port_id='http',
                    endpoint='/',
                    content_type='application/json',
                    body='{"jsonrpc":"2.0","method":"optimism_version","params":[],"id":1}'
                ),
                field='code',
                assertion='==',
                target_value=200,
                timeout='5s'
            )
        )
    )
