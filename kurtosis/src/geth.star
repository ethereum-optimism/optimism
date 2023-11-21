def init(
        plan,
        genesis,
        name='op-geth',
        image='ethereum/client-go',
        tag='stable',
):
    return plan.run_sh(
        run='geth --datadir /data init /config/genesis.json',
        image='{}:{}'.format(image, tag),
        files={
            '/config': genesis,
        },
        store=[
            StoreSpec(src='/data/*', name='{}_datadir'.format(name))
        ]
    )

def run(
        plan,
        name,
        datadir,
        jwt_secret,
        image='ethereum/client-go',
        tag='stable',
        additional_env={},
        additional_files={}
):
    return plan.add_service(
        name=name,
        config=ServiceConfig(
            image='{}:{}'.format(image, tag),
            ports={
                'http': PortSpec(8545),
                'engine': PortSpec(8551),
                'ws': PortSpec(8546),
            },
            env_vars={
                'GETH_HTTP_ADDR': '0.0.0.0',
                'GETH_WS_ADDR': '0.0.0.0',
                'GETH_AUTHRPC_ADDR': '0.0.0.0',
                'GETH_HTTP': 'true',
                'GETH_WS': 'true',
                'GETH_HTTP_API': 'eth,engine',
                'GETH_DATADIR': '/data',
                'GETH_NODISCOVER': 'true',
                'GETH_SYNCMODE': 'full',
                'GETH_AUTHRPC_JWTSECRET': '/jwt/jwt-secret.txt',
            } | additional_env,
            files={
                '/data': datadir,
                '/jwt': jwt_secret,
            } | additional_files,
            ready_conditions=ReadyCondition(
                recipe=PostHttpRequestRecipe(
                    port_id='http',
                    endpoint='/',
                    content_type='application/json',
                    body='{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}'
                ),
                field='code',
                assertion='==',
                target_value=200,
                timeout='5s'
            )
        )
    )