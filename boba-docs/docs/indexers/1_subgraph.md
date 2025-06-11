# Using The Graph

The Graph is a decentralized protocol for indexing and querying data from blockchains.

## Query Existing Subgraphs

Learn how to [query existing subgraphs](https://thegraph.com/docs/en/querying/querying-the-graph/) using the official documentation.

- [Boba Light Bridge ETH](https://thegraph.com/explorer/subgraphs/34UUDYdzZFX7afhysqX7JodzKngJmjsFKJDtWsTxi9UA?view=Query&chain=arbitrum-one)
- [Boba Blocks](https://thegraph.com/explorer/subgraphs/5d1ZCJQCEqsfCqLRRU5iQ9ewg79tuNqZLPMkgUcpmLsD?view=Query&chain=arbitrum-one)
- [Boba Sushiswap](https://thegraph.com/explorer/subgraphs/EC3ZtCpCaV5GyyhyPNHs584wdGA72nud7qcuxWNTfPr4?view=Query&chain=arbitrum-one)

## Create a Subgraph
This guide will quickly take you through how to initialize, create, and deploy your subgraph to Subgraph Studio.

This guide is written assuming that you have:

- A crypto wallet
- A smart contract address on Boba Network

### 1. Create a subgraph on Subgraph Studio

Go to the [Subgraph Studio](https://thegraph.com/studio/) and connect your wallet.

Once your wallet is connected, you can begin by clicking “Create a Subgraph." It is recommended to name the subgraph in Title Case: "Subgraph Name Chain Name."

### 2. Install the Graph CLI

The Graph CLI is written in TypeScript and you will need to have `node` and either `npm` or `yarn` installed to use it. Check that you have the most recent CLI version installed.

On your local machine, run one of the following commands:

Using [npm](https://www.npmjs.com/):

```sh
npm install -g @graphprotocol/graph-cli@latest
```

Using [yarn](https://yarnpkg.com/):

```sh
yarn global add @graphprotocol/graph-cli
```

### 3. Initialize your subgraph from an existing contract

Initialize your subgraph from an existing contract by running the initialize command:

```sh
graph init --studio <SUBGRAPH_SLUG>
```

> You can find commands for your specific subgraph on the subgraph page in [Subgraph Studio](https://thegraph.com/studio/).

When you initialize your subgraph, the CLI tool will ask you for the following information:

- Protocol: choose the protocol your subgraph will be indexing data from
- Subgraph slug: create a name for your subgraph. Your subgraph slug is an identifier for your subgraph.
- Directory to create the subgraph in: choose your local directory
- Ethereum network(optional): you may need to specify which EVM-compatible network your subgraph will be indexing data from
- Contract address: Locate the smart contract address you’d like to query data from
- ABI: If the ABI is not autopopulated, you will need to input it manually as a JSON file
- Start Block: it is suggested that you input the start block to save time while your subgraph indexes blockchain data. You can locate the start block by finding the block where your contract was deployed.
- Contract Name: input the name of your contract
- Index contract events as entities: it is suggested that you set this to true as it will automatically add mappings to your subgraph for every emitted event
- Add another contract(optional): you can add another contract

### 4. Write your subgraph

The previous commands create a scaffold subgraph that you can use as a starting point for building your subgraph. When making changes to the subgraph, you will mainly work with three files:

- Manifest (`subgraph.yaml`) - The manifest defines what datasources your subgraphs will index.
- Schema (`schema.graphql`) - The GraphQL schema defines what data you wish to retrieve from the subgraph.
- AssemblyScript Mappings (`mapping.ts`) - This is the code that translates data from your datasources to the entities defined in the schema.

For more information on how to write your subgraph, see [Creating a Subgraph](https://thegraph.com/docs/en/developing/creating-a-subgraph/).

### 5. Deploy to Subgraph Studio

Once your subgraph is written, run the following commands:

```sh
graph codegen
graph build
```

- Authenticate and deploy your subgraph. The deploy key can be found on the Subgraph page in Subgraph Studio.

```sh
graph auth --studio <DEPLOY_KEY>
graph deploy --studio <SUBGRAPH_SLUG>
```

You will be asked for a version label. It's strongly recommended to use [semver](https://semver.org/) for versioning like `0.0.1`. That said, you are free to choose any string as version such as:`v1`, `version1`, `asdf`.

### 6. Test your subgraph

In Subgraph Studio's playground environment, you can test your subgraph by making a sample query.

The logs will tell you if there are any errors with your subgraph. The logs of an operational subgraph will look like this:

If your subgraph is failing, you can query the subgraph health by using the GraphiQL Playground. Note that you can leverage the query below and input your deployment ID for your subgraph. In this case, `Qm...` is the deployment ID (which can be located on the Subgraph page under **Details**). The query below will tell you when a subgraph fails, so you can debug accordingly:

```graphql
{
  indexingStatuses(subgraphs: ["Qm..."]) {
    node
    synced
    health
    fatalError {
      message
      block {
        number
        hash
      }
      handler
    }
    nonFatalErrors {
      message
      block {
        number
        hash
      }
      handler
    }
    chains {
      network
      chainHeadBlock {
        number
      }
      earliestBlock {
        number
      }
      latestBlock {
        number
      }
      lastHealthyBlock {
        number
      }
    }
    entityCount
  }
}
```

### 7. Publish your subgraph to The Graph’s Decentralized Network

Once your subgraph has been deployed to Subgraph Studio, you have tested it out, and you are ready to put it into production, you can then publish it to the decentralized network.

In Subgraph Studio, you will be able to click the publish button on the top right of your subgraph's page.

The Graph's upgrade Indexer will begin serving queries on your subgraph regardless of subgraph curation, and it will provide you with **100,000 free queries per month.**

For a higher quality of service and stronger redundancy, you can [curate](https://thegraph.com/docs/en/publishing/publishing-a-subgraph/#adding-signal-to-your-subgraph) your subgraph to attract more Indexers. At the time of writing, it is recommended that you curate your own subgraph with at least 3,000 GRT to ensure 3-5 additional Indexers begin serving queries on your subgraph.

To save on gas costs, you can curate your subgraph in the same transaction that you published it by selecting this button when you publish your subgraph to The Graph’s decentralized network:

### 8. Query your subgraph

Now, you can query your subgraph by sending GraphQL queries to your subgraph’s Query URL, which you can find by clicking on the query button.

If you don't have your API key, you can query via the free, rate-limited development query URL, which can be used for development and staging.

For more information about querying data from your subgraph, read more [here](https://thegraph.com/docs/en/querying/querying-the-graph/).
