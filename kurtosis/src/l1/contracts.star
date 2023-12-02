anvil = import_module('./anvil.star')
geth = import_module('../geth.star')


def create_allocs(
        plan,
        name,
        l2oo_starting_timestamp,
        l1_chain_id='32382'
):
    # Render the deploy config template.
    deploy_config = plan.render_templates(
        name=name + '-deploy-config',
        config={
            '{}.json'.format(l1_chain_id): struct(
                template=read_file('/static/templates/deploy-config.json'),
                data={
                    'l1_chain_id': l1_chain_id,
                    'l1_starting_block_tag': 'latest',
                    'l2oo_starting_timestamp': l2oo_starting_timestamp,
                }
            )
        }
    )

    anvil_l1 = anvil.run(
        plan=plan,
        name=name + '-anvil',
        additional_args=[
            '-a', '100', '--chain-id', l1_chain_id
        ]
    )

    scripts = plan.upload_files(
        name=name + '-scripts',
        src='/static/scripts',
    )

    create2 = plan.run_sh(
        image='ghcr.io/foundry-rs/foundry:nightly',
        run='sh /scripts/deploy-create2.sh http://{}:8545'.format(anvil_l1.ip_address),
        files={
            '/scripts': scripts,
        },
        store=[
            StoreSpec(src='/result/*', name=name + '-create2'),
        ]
    )

    deployment = deploy(
        plan=plan,
        name=name + '-deploy',
        l1_eth_rpc='http://{}:8545'.format(anvil_l1.ip_address),
        l1_chain_id=l1_chain_id,
        deploy_config=deploy_config,
        sender='0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
        additional_files={
            '/create2': create2.files_artifacts[0],
        }
    )

    return plan.run_sh(
        image='badouralix/curl-jq:latest',
        run='sh /scripts/generate-allocs.sh "http://{}:8545"'.format(anvil_l1.ip_address),
        files={
            '/dummy': deployment.files_artifacts[0],
            '/scripts': scripts,
        },
        store=[
            StoreSpec(src='/allocs/*', name=name + '-allocs'),
        ]
    )


def deploy(
        plan,
        name,
        l1_eth_rpc,
        l1_chain_id,
        deploy_config,
        sender='0x123463a4b065722e99115d6c222f267d9cabb524',
        image='us-docker.pkg.dev/oplabs-tools-artifacts/images/contracts-bedrock',
        tag='develop',
        additional_files={},
):
    return plan.run_sh(
        run=' '.join([
            'forge',
            'script',
            'scripts/Deploy.s.sol',
            '--rpc-url={}'.format(l1_eth_rpc),
            '--broadcast',
            '--sender={}'.format(sender),
            '--unlocked',
            '&&',
            'forge',
            'script',
            'scripts/Deploy.s.sol',
            "--sig='sync()'",
            '--rpc-url={}'.format(l1_eth_rpc),
        ]),
        image='{}:{}'.format(image, tag),
        files={
            '/opt/optimism/packages/contracts-bedrock/deploy-config': deploy_config,
        } | additional_files,
        store=[
            StoreSpec(src='/opt/optimism/packages/contracts-bedrock/deployments/{}/*'.format(l1_chain_id),
                      name=name + '-artifacts')
        ]
    )
