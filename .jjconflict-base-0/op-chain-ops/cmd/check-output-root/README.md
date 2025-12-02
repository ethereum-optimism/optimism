# Overview

Generates an output root for a given block using only the execution client RPC endpoint.

## Prerequisites:

1. git clone or pull the latest develop branch of the optimism repo
2. Go Installed
  - You can follow the instructions in the [CONTRIBUTING.md](http://CONTRIBUTING.md) to install all software dependencies of the repo
3. RPC URL for the **L2** chain execution client you want to generate a output root for.
  - **Important**: The RPC endpoint must be trusted as it provide the chain state used to compute the output root.

## Usage:

```bash
go run op-chain-ops/cmd/check-output-root/ --l2-eth-rpc $RPC_URL --block-num $BLOCK_NUM
```

Output:

```text
0xfefc68b1c0aa7f6e744a8c74084142cf3daa8692179fd5b9ff46c6eacdffe9aa
```

## Environment Variables

Alternatively, you can use environment variables to configure the script:

- `CHECK_OUTPUT_ROOT_L2_ETH_RPC`: L2 execution client RPC endpoint.
- `CHECK_OUTPUT_ROOT_BLOCK_NUM`: Block number to calculate the output root for.
