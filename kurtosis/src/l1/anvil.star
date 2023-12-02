def run(
        plan,
        name,
        image='ghcr.io/foundry-rs/foundry',
        tag='nightly',
        additional_args=[],
):
    return plan.add_service(
        name=name,
        config=ServiceConfig(
            image='{}:{}'.format(image, tag),
            cmd=[' '.join(['anvil'] + additional_args)],
            ports={
                'http': PortSpec(8545),
            },
            env_vars={
                'ANVIL_IP_ADDR': '0.0.0.0'
            }
        )
    )
