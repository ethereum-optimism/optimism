# Local Node

The following document outlines the steps to initiate a fresh local L1 and local L2 setup, along with other rollup components. These instructions can be followed to start a new L2 on testnets.

## Step 1

To begin, Step 1 involves launching a pristine local L1 network using a minimal genesis file. This genesis file exclusively includes developer accounts with a certain amount of ETH allocated to each.

```json
{
  "config": {
    "chainId": 910,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "muirGlacierBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "arrowGlacierBlock": 0,
    "grayGlacierBlock": 0,
    "clique": {
      "period": 3,
      "epoch": 30000
    }
  },
  "nonce": "0x0",
  "timestamp": "0x642c5346",
  "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000ca062b0fd91172d89bcd4bb084ac4e21972cc4670000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": "0xe4e1c0",
  "difficulty": "0x1",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": {
    "f39fd6e51aad88f6f4ce6ab8827279cfffb92266": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "70997970C51812dc3A010C7d01b50e0d17dc79C8": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "3C44CdDdB6a900fa2b585dd299e03d12FA4293BC": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "90F79bf6EB2c4f870365E785982E1f101E93b906": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "15d34AAf54267DB7D7c367839AAf71A00a2C6A65": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    },
    "9965507D1a55bcC2695C58ba16FB37d819B0A4dc": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    }
  },
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "baseFeePerGas": "0x3b9aca00"
}
```

Once you have successfully generated the appropriate `genesis-l1.json` file, you can initiate the L1 node within `op-bedrock` by adjusting the volume associated with the genesis file.

```bash
cd ops-bedrock
docker-compose up l1 -d -V
```

## Step 2

To proceed, it is necessary to include the appropriate configuration file in `packages/contracts-bedrock/deploy-config`, along with an export file named `network-name.ts`. The meaning of configuration settings can be located in `docs/op-stack/src/docs/build/conf.md`. Once this is done, the new network must be added to `hardhat.config.ts`. Additionally, an `.env` file should be added to the designated folder.

```yaml
# RPC for the L1 network to deploy to
L1_RPC=http://localhost:8545

# Private key for the deployer account
PRIVATE_KEY_DEPLOYER=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

Run

```
yarn hardhat deploy --network network-name
```

This will start the deployment of all L1 contracts to your local L1 node. The deployment files will be generated in the `packages/contracts-bedrock/deployments/network-name` folder.

Note:

To proceed, you need to obtain the `l1StartingBlockTag` and `l2OutputOracleStartingTimestamp` values from the local L1 node. You can select any random block as the starting point for the L2. Once you initiate the rollup node and L2, the rollup node will generate a series of empty blocks to synchronize with the L1 timestamp.

## Step 3 (Optional)

In this step, a custom local BOBA L1 token is deployed specifically for testing purposes. You have the flexibility to choose any standard ERC20 token contract and deploy it on the local node. It is essential to obtain the contract address as it will be needed for generating the L2 genesis file.

## Step 4

To create an L2 genesis file that exclusively consists of the pre-deployed contracts, you can utilize the `boba-chain-ops/cmd/boba-devnet` tool. This tool enables the generation of a specialized L2 genesis file tailored to your requirements.

```bash
go run ./cmd/boba-devnet --deploy-config=boba-configuration.json --hardhat-deployments=packages/contracts-bedrock/deployments --network=network-name --l1-rpc=http://localhost:8545 --outfile-l2="genesis-l2.json" --outfile-rollup="rollup.json"
```

The `genesis-l2` file is utilized to initiate the L2 node, while the `rollup.json` file is employed to initiate the rollup node.

Note:

In the `boba-configuration.json` file, it is necessary to include the L1 BOBA contract address. However, if you prefer not to use it, you have the option to select a random address other than the zero address.

```json
"l1BobaTokenAddress": "0x663F3ad617193148711d28f5334eE4Ed07016602"
```

## Step 5

To initiate the `l2` and `op-node` components, you need to adjust their volumes accordingly. Once these components are started, you will observe that the `op-node` generates a series of empty blocks to synchronize with the most recent L1 timestamp. Then you can launch `op-batcher` and `op-proposer` by providing the correct private key as a parameter.