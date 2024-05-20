# Using GoldSky

GoldSky is an easy to use alternative to The Graph for indexing on-chain data.

## Official Subgraphs

### 1. Lightbridge
- [ETH Mainnet](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/light-bridge-mainnet/v1/gn)
- [Boba ETH](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/light-bridge-boba-eth/v1/gn)
- [BNB Mainnet](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/light-bridge-bsc/v1/gn)
- [Boba BNB](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/light-bridge-boba-bnb/v1/gn)
- [Arbitrum One](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/light-bridge-arbitrum-one/v1/gn)
- [Optimism Mainnet](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/light-bridge-optimism/v1/gn)

### 2. DAO
- [Boba ETH](https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/dao-boba-eth/v1/gn)


## Deploy your own Subgraphs
<figure><img src="../../.gitbook/assets/requirements.png" alt=""><figcaption></figcaption></figure>

The `goldsky`-CLI is required to deploy to **GoldSky**. Make sure that you have various packages installed. Please refer to the [official documentation](https://goldsky.com/).

```bash
curl https://goldsky.com | sh
```

### The Graph
If you are using **The Graph** right now, you can easily migrate your existing subgraphs over to Goldsky. The [official GoldSky documentation](https://docs.goldsky.com/introduction) is very helpful in that matter.


<figure><img src="../../.gitbook/assets/building and running (1).png" alt=""><figcaption></figcaption></figure>

First you need to login to your GoldSky account:

```bash
goldsky login
````

Then you can deploy your subgraph in one of the following ways (taken from the official GoldSky docs):

1. **Build and deploy from source**
```bash
cd <your-subgraph-directory>
graph build # Build your subgraph as normal.
goldsky subgraph deploy my-subgraph/1.0.0
# Can add --path <build target> to target a specific subgraph build directory
```

2. **Migrate from The Graph's hosted service**
```bash
goldsky subgraph deploy your-subgraph-name/your-version --from-url <your-subgraph-query-url>
```

3. **Migrate from any existing endpoint**
```bash
goldsky subgraph deploy your-subgraph-name/your-version --from-ipfs-hash <deployment-hash>
```

4. **Build and deploy from ABI + contract address**
```bash
goldsky subgraph deploy your-subgraph-name/your-version --from-abi <path-to-config-file>
```

_NOTE: When you log into https://thegraph.com/hosted-service/dashboard, you may have more than one account. Make sure that you are using the ACCESS\_TOKEN associated with the correct account, otherwise your depoyment will fail. You can cycle through your multiple accounts by clicking on your GitHub user ID or whatever other account is displayed next to your user Avatar._

<figure><img src="../../.gitbook/assets/example.png" alt=""><figcaption></figcaption></figure>

Here is some example queries to get you started:

```bash
# L2 Boba Mainnet Query

  curl -g -X POST \
    -H "Content-Type: application/json" \
    -d '{"query":"{ proposalQueueds {id eta, transactionHash_}}"}' \
   https://api.goldsky.com/api/public/project_clq6jph4q9t2p01uja7p1f0c3/subgraphs/dao-boba-eth/v1/gn
```

<figure><img src="../../.gitbook/assets/querying.png" alt=""><figcaption></figcaption></figure>

To get the endpoints you can either login on the official [GoldSky dashboard](https://app.goldsky.com/dashboard/subgraphs) or use the following command:

```bash
goldsky subgraph list
```
