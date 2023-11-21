geth = import_module('../geth.star')

def init(
        plan,
        genesis,
        name='op-geth',
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/op-geth',
        tag='latest',
):
    return geth.init(
        plan=plan,
        genesis=genesis,
        name=name,
        image=image,
        tag=tag,
    )

def run(
        plan,
        datadir,
        keystore,
        jwt_secret,
        name='op-geth',
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/op-geth',
        tag='latest',
        additional_env={},
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
                'GETH_GCMODE': 'archive',
            } | additional_env,
            files={
                '/data': datadir,
                '/keystore': keystore,
                '/jwt': jwt_secret,
            },
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
