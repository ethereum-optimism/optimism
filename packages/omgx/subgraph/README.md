# Boba Network Bridges Subgraphs

These subgraphs index the **StandardBridge** contracts and **LiquidityPool** contracts.

## Requirements

The global graph is required to deploy to **The Graph Node**.

```bash
npm install -g @graphprotocol/graph-cli
```

## Building & Running

### L1 Subgraphs

The deploy key is required to deploy subgraphs to **The Graph Node**. 

```bash
graph auth  --studio $DEPLOY_KEY
yarn install
yarn prepare:mainet # yarn prepare:rinkeby
yarn codegen
yarn build
graph deploy --studio boba-network # graph deploy --studio boba-network-rinkeby
```

### L2 Subgraphs

The admin port is not public. 

```bash
yarn install
yarn prepare:mainet # yarn prepare:rinkeby
yarn codegen
yarn build
yarn create:subgraph:mainnet  # yarn create:subgraph:rinkeby
yarn deploy:subgraph:mainnet  # yarn deploy:subgraph:rinkeby
```

## Querying

### L2 Subgraphs

> Mainnet: https://graph.mainnet.boba.network/subgraphs/name/boba/Bridges/graphql

> Rinkeby: https://graph.rinkeby.boba.network/subgraphs/name/boba/Bridges/graphql