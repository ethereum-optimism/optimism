### Deployment Instructions for `The Graph` Subgraphs

This setup here handles a single repository for all networks respectively.

The different networks can be obtained from this config file:
[networks.json](light-bridge/networks.json)

The subgraph.yaml file always contains the latest network chosen:
[subgraph.json](light-bridge/subgraph.yaml)

### Build

- Install deps \
  ```yarn```

- Generate `subgraph.yaml` \
  ```yarn prepare:boba```


- Build files \
  ```npx graph codegen && npx graph build```


- Build for a certain network \
  ``` yarn build --network NETWORK```

### Authenticate

- Authenticate with the Studio \
  ```npx graph auth --studio DEPLOYMENT_KEY```

### Deploy

- Deploy a single subgraph with the `network` option and the dedicated the-graph `PROJECT_NAME` \
  ```npx graph deploy --studio PROJECT_NAME --network NETWORK```

### Miscellaneous

#### LightBridge

| NETWORK          | PROJECT_NAME                  | Deployed |
|------------------|-------------------------------|----------|
| optimism-sepolia | light-bridge-optimism-sepolia | ✅        |
| arbitrum-sepolia | light-bridge-arbitrum-sepolia | ✅        |
| bsc              | light-bridge-bsc              | ✅        |
| x                | light-bridge-boba-bnb         | ❌        |
| boba-sepolia     | light-bridge-boba-sepolia     | ✅        |
| arbitrum-one     | light-bridge-arbitrum-one     | ✅        |
| mainnet          | light-bridge-mainnet          | ✅        |
| optimism         | light-bridge-optimism         | ✅        |
| boba             | light-bridge-boba-eth         | ✅        |
| sepolia          | light-bridge-sepolia          | ✅        |

#### DAO

| NETWORK | PROJECT_NAME | Deployed? |
|---------|--------------|-----------|
| x       | dao-boba-eth | ✅         |

#### Anchorage

| NETWORK | PROJECT_NAME             | Deployed |
|---------|--------------------------|----------|
| mainnet | anchorage-bridge-mainnet | ✅        |
| sepolia | anchorage-bridge-sepolia | ✅        |

#### Anchorage Boba

| NETWORK      | PROJECT_NAME                           | Deployed |
|--------------|----------------------------------------|----------|
| boba         | anchorage-bridge-boba-eth              | ✅        |
| boba-sepolia | anchorage-bridge-bridging-boba-sepolia | ✅        |
