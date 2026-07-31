# Local Development

To run a local development network, you can use either Kurtosis or docker compose to spin up the op-stack with rollup boost. This would include spinning up:

- sequencer `op-node` and `op-geth`
- builder `op-node` and `op-geth`
- rollup boost `rollup-boost`
- ethereum l1 devnet with l2 contracts deployed
- `op-proposer` and `op-batcher`

## Kurtosis

Kurtosis is a tool to manage containerized services. To run the devnet with Kurtosis, you can follow the instructions first to [install kurtosis](https://docs.kurtosis.com/quickstart).

See the [optimism package](https://github.com/ethpandaops/optimism-package/blob/main/README.md#configuration) for the full set of configuration options.

To clean up the devnet, you can run:

```bash
kurtosis clean -a
```

## Docker Compose Setup

To run the devnet with docker compose, you can checkout the devnet setup in the [flashbots optimism repo](https://github.com/flashbots/optimism).

```bash
git clone https://github.com/flashbots/optimism.git
cd optimism
git checkout sidecar-docker
```

To run the devnet, you can run:

```bash
make devnet-up
```

In this setup, there is a prefunded test account to send test transactions at:

- address: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
- private key: ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

To clean up the devnet, you can run:

```bash
make devnet-down; make devnet-clean
```

This will run a local devnet with an unmodified op-geth as the builder and rollup boost. To run the devnet with `op-rbuilder`, checkout the instructions in the [rbuilder repo](https://github.com/flashbots/op-rbuilder?tab=readme-ov-file#local-devnet).
