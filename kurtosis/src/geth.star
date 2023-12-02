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
        datadir=None,
        jwt_secret=None,
        image='ethereum/client-go',
        tag='stable',
        additional_env={},
        additional_files={}
):
    env = {
        'GETH_HTTP_ADDR': '0.0.0.0',
        'GETH_WS_ADDR': '0.0.0.0',
        'GETH_AUTHRPC_ADDR': '0.0.0.0',
        'GETH_HTTP': 'true',
        'GETH_WS': 'true',
        'GETH_HTTP_API': 'eth,engine',
        'GETH_NODISCOVER': 'true',
        'GETH_SYNCMODE': 'full',
    }
    files = {}
    if datadir:
        env['GETH_DATADIR'] = '/data'
        files['/data'] = datadir
    if jwt_secret:
        files['/jwt'] = jwt_secret
        env['GETH_AUTHRPC_JWTSECRET'] = '/jwt/jwt-secret.txt'
    ports = {
        'http': PortSpec(8545),
        'ws': PortSpec(8546),
    }
    if additional_env.get('GETH_DEV', 'false') == 'false':
        ports['engine'] = PortSpec(8551)

    return plan.add_service(
        name=name,
        config=ServiceConfig(
            image='{}:{}'.format(image, tag),
            ports=ports,
            env_vars=env | additional_env,
            files=files | additional_files,
            ready_conditions=ReadyCondition(
                recipe=PostHttpRequestRecipe(
                    port_id='http',
                    endpoint='/',
                    content_type='application/json',
                    body='{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
                ),
                field='code',
                assertion='==',
                target_value=200,
                timeout='5s'
            )
        )
    )