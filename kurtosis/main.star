op_node = import_module('./src/l2/op-node.star')
op_geth = import_module('./src/l2/op-geth.star')
op_batcher = import_module('./src/l2/op-batcher.star')
op_proposer = import_module('./src/l2/op-proposer.star')
geth = import_module('./src/geth.star')
contracts = import_module('./src/l1/contracts.star')

default_devnet_config = {
    'genesis_contracts_path': None
}

def run(
        plan,
        playbook,
        config=None
):
    if playbook == 'devnet':
        devnet(plan, config)
    elif playbook == 'generate_allocs':
        generate_allocs(plan)

def devnet(plan, config):
    # Make an archive containing the Geth genesis file and the keystore password.
    geth_config = plan.upload_files(
        name='geth_bootstrap',
        src='/static/geth'
    )

    # Make an archive containing the Geth keystore.
    geth_keystore = plan.upload_files(
        name='geth_keystore',
        src='/static/keystore'
    )

    # Make an archive containing the Prysm L1 validator config.
    prysm_config = plan.upload_files(
        name='prysm_bootstrap',
        src='/static/prysm'
    )

    # Make an archive containing the JWT secret.
    jwt_secret = plan.upload_files(
        name='jwt_secret',
        src='/static/jwt-secret.txt'
    )

    scripts = plan.upload_files(
        name='scripts',
        src='/static/scripts'
    )

    genesis_l1_files = {
        '/scripts': scripts,
        '/config': prysm_config,
        '/execution-in': geth_config,
    }

    if config['genesis_contracts_path']:
        genesis_l1_allocs = plan.upload_files(
            name='genesis-l1-allocs',
            src=config['genesis_contracts_path'] + '/allocs'
        )
        genesis_l1_files['/allocs-in'] = genesis_l1_allocs

    # Generate the PoS genesis state and the Geth genesis file.
    plan.run_sh(
        run='bash /scripts/generate-l1-genesis.sh',
        image='mslipper/prysm-shell:latest',
        files=genesis_l1_files,
        store=[
            StoreSpec(src='/consensus/genesis.ssz', name='prysm_genesis'),
            StoreSpec(src='/execution-out/genesis.json', name='geth_genesis')
        ]
    )

    # Initialize Geth.
    geth_bootstrap = geth.init(
        plan=plan,
        genesis='geth_genesis',
        name='l1-geth'
    )

    # Run L1 Geth.
    l1_geth = geth.run(
        plan=plan,
        name='l1-geth',
        datadir=geth_bootstrap.files_artifacts[0],
        jwt_secret=jwt_secret,
        additional_env={
            'GETH_ALLOW_INSECURE_UNLOCK': 'true',
            'GETH_UNLOCK': '0x123463a4b065722e99115d6c222f267d9cabb524',
            'GETH_KEYSTORE': '/keystore',
            'GETH_PASSWORD': '/config/password.txt'
        },
        additional_files={
            '/config': geth_config,
            '/keystore': geth_keystore,
        }
    )

    # Run the L1 beacon chain.
    beacon_chain = plan.add_service(
        name='l1-beacon-chain',
        config=ServiceConfig(
            image='mslipper/prysm-shell:latest',
            cmd=[
                'beacon-chain',
                '--datadir=/data',
                '--min-sync-peers=0',
                '--genesis-state=/genesis/genesis.ssz',
                '--bootstrap-node=',
                '--chain-config-file=/config/config.yml',
                '--contract-deployment-block=0',
                '--chain-id=32382',
                '--rpc-host=0.0.0.0',
                '--grpc-gateway-host=0.0.0.0',
                '--execution-endpoint=http://{}:8551'.format(l1_geth.ip_address),
                '--accept-terms-of-use',
                '--jwt-secret=/jwt/jwt-secret.txt',
                '--suggested-fee-recipient=0x123463a4b065722e99115d6c222f267d9cabb524',
                '--minimum-peers-per-subnet=0',
                '--enable-debug-rpc-endpoints'
            ],
            files={
                '/genesis': 'prysm_genesis',
                '/config': prysm_config,
                '/jwt': jwt_secret,
            },
            ports={
                'beacon_rpc': PortSpec(4000),
            },
        ),
    )

    # Run the L1 validator.
    plan.add_service(
        name='l1-validator',
        config=ServiceConfig(
            image='mslipper/prysm-shell:latest',
            cmd=[
                'validator',
                '--datadir=/data',
                '--accept-terms-of-use',
                '--interop-num-validators=64',
                '--interop-start-index=0',
                '--force-clear-db',
                '--chain-config-file=/config/config.yml',
                '--config-file=/config/config.yml',
                '--beacon-rpc-provider={}:4000'.format(beacon_chain.ip_address),
            ],
            files={
                '/config': prysm_config,
            }
        ),
    )

    # Grab the genesis block so we can use that as the rollup anchor point.
    head_block = plan.request(
        service_name='l1-geth',
        recipe=PostHttpRequestRecipe(
            port_id='http',
            endpoint='/',
            content_type='application/json',
            body='{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}',
            extract={
                'hash': '.result.hash',
                'timestamp': '.result.timestamp',
            }
        ),
        acceptable_codes=[200],
    )

    # HACK: The timestamp is hexlified, so we need to unhexlify it.
    unhexlified_timestamp = plan.run_python(
        run="import sys; sys.stdout.write(str(int(sys.argv[1].replace('0x', ''), 16)))",
        args=[head_block['extract.timestamp']],
    )

    # Render the deploy config template.
    deploy_config = plan.render_templates(
        name='deploy-config',
        config={
            '32382.json': struct(
                template=read_file('/static/templates/deploy-config.json'),
                data={
                    'l1_chain_id': '32382',
                    'l1_starting_block_tag': head_block['extract.hash'],
                    'l2oo_starting_timestamp': unhexlified_timestamp.output,
                }
            )
        }
    )

    if config['genesis_contracts_path']:
        deployment_artifacts = plan.upload_files(
            name='genesis-l1-artifacts',
            src=config['genesis_contracts_path'] + '/artifacts'
        )
    else:
        deployer = contracts.deploy(
            plan=plan,
            name='deployer',
            l1_eth_rpc='http://{}:8545'.format(l1_geth.ip_address),
            l1_chain_id='32382',
            deploy_config=deploy_config,

        )
        deployment_artifacts = deployer.files_artifacts[0]

    # Generate the L2 genesis and rollup configs.
    plan.run_sh(
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:develop',
        run=' '.join([
            'op-node',
            'genesis',
            'l2',
            '--l1-rpc=http://{}:8545'.format(l1_geth.ip_address),
            '--deploy-config=/deploy-config/32382.json',
            '--deployment-dir=/deployment',
            '--outfile.l2=/genesis.json',
            '--outfile.rollup=/rollup.json',
        ]),
        files={
            '/deployment': deployment_artifacts,
            '/deploy-config': deploy_config,
        },
        store=[
            StoreSpec(src='/rollup.json', name='rollup_config'),
            StoreSpec(src='/genesis.json', name='op_geth_genesis'),
        ]
    )

    # Initialize the L2 Geth node.
    sequencer_op_geth_init = op_geth.init(
        plan=plan,
        genesis='op_geth_genesis',
        name='op-geth-sequencer-init',
    )

    # Run the L2 Geth node.
    sequencer_op_geth = op_geth.run(
        name='op-geth',
        plan=plan,
        datadir=sequencer_op_geth_init.files_artifacts[0],
        keystore=geth_keystore,
        jwt_secret=jwt_secret,
    )

    # Create an archive containing the sequencer's P2P key.
    p2p_key = plan.upload_files(
        name='p2p_key',
        src='/static/sequencer-p2p-key.txt'
    )

    # Run the L2 op-node.
    sequencer_op_node = op_node.run(
        plan=plan,
        name='op-node-sequencer',
        l1_eth_rpc='http://{}:8545'.format(l1_geth.ip_address),
        l2_engine_rpc='http://{}:8551'.format(sequencer_op_geth.ip_address),
        jwt_secret=jwt_secret,
        p2p_key=p2p_key
    )

    # Run the batcher.
    op_batcher.run(
        plan=plan,
        name='op-batcher-sequencer',
        l1_eth_rpc='http://{}:8545'.format(l1_geth.ip_address),
        l2_eth_rpc='http://{}:8545'.format(sequencer_op_geth.ip_address),
        l2_rollup_rpc='http://{}:9545'.format(sequencer_op_node.ip_address),
    )

    # Find the L2 Output Oracle address.
    l2oo_addr_finder = plan.run_sh(
        image='badouralix/curl-jq:latest',
        run='jq -r .address /deployment/L2OutputOracleProxy.json | tr -d "\\n"',
        files={
            '/deployment': deployment_artifacts,
        }
    )

    # Run the proposer.
    op_proposer.run(
        plan=plan,
        name='op-proposer-sequencer',
        l1_eth_rpc='http://{}:8545'.format(l1_geth.ip_address),
        l2_rollup_rpc='http://{}:9545'.format(sequencer_op_node.ip_address),
        l2oo_address=l2oo_addr_finder.output,
    )


def generate_allocs(plan):
    contracts.create_allocs(
        plan=plan,
        name='l1-allocs',
        l2oo_starting_timestamp=1234,
    )