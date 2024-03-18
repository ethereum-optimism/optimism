# Using The Graph

These subgraphs index the **StandardBridge**, the **LiquidityPool**, the **Boba DAO**, and the **TuringMonster** contracts.

<figure><img src="../../.gitbook/assets/requirements.png" alt=""><figcaption></figcaption></figure>

The global `graph` is required to deploy to **The Graph**. Make sure that you have various packages installed.

```bash
yarn global add @graphprotocol/graph-cli
yarn global add --dev @graphprotocol/graph-ts
```

<figure><img src="../../.gitbook/assets/building and running (1).png" alt=""><figcaption></figcaption></figure>

First, `cd` to either the **L1** or the **L2** folders, depending on where you will be deploying your subgraphs to. There are three subgraphs: Ethereum, Boba, and Goerli. A deploy key or access token is required to deploy subgraphs. Depending on which chain you are indexing, provide either `mainnet` or `goerli` as a setting to `yarn prepare:`.

### L1 Subgraphs

(below command untested)

```bash
graph auth --product hosted-service <ACCESS_TOKEN>
# or, graph auth --studio $DEPLOY_KEY
cd L1
yarn install
yarn prepare:mainnet
# or, yarn prepare:goerli
yarn codegen
yarn build
graph deploy --product hosted-service BOBANETWORK/boba-l2-subgraph
# or, graph deploy --studio boba-network-goerli
```

### L2 Subgraphs

(Below commands tested for deploy to Boba mainnet)

```bash
graph auth --product hosted-service <ACCESS_TOKEN>
# or, graph auth --studio $DEPLOY_KEY
cd L2
yarn install
yarn prepare:mainnet
yarn codegen
yarn build
graph deploy --product hosted-service BOBANETWORK/boba-l2-subgraph
# No graph on Boba Goerli available
```

_NOTE: When you log into https://thegraph.com/hosted-service/dashboard, you may have more than one account. Make sure that you are using the ACCESS\_TOKEN associated with the correct account, otherwise your depoyment will fail. You can cycle through your multiple accounts by clicking on your GitHub user ID or whatever other account is displayed next to your user Avatar._

<figure><img src="../../.gitbook/assets/example.png" alt=""><figcaption></figcaption></figure>

Here is some example queries to get you started:

```bash
# L2 Boba Mainnet Query

  curl -g -X POST \
    -H "Content-Type: application/json" \
    -d '{"query":"{ governorProposalCreateds {proposalId values description proposer}}"}' \
    https://api.thegraph.com/subgraphs/name/bobanetwork/boba-l2-subgraph
```

<figure><img src="../../.gitbook/assets/querying.png" alt=""><figcaption></figcaption></figure>

* The Mainnet Graph Node is hosted by **The Graph**. Visit https://thegraph.com/hosted-service/ to deploy your subgraphs. You can experiment here: [bobanetwork/boba-l2-subgraph](https://thegraph.com/hosted-service/subgraph/bobanetwork/boba-l2-subgraph?query=Example%20query).
* No testnet graph node available.
