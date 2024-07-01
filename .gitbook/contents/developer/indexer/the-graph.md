# The Graph 

Getting historical data on a smart contract can be frustrating when you’re building a dapp. [The Graph](https://thegraph.com/) provides an easy way to query smart contract data through APIs known as subgraphs. The Graph’s infrastructure relies on a decentralized network of indexers, enabling your dapp to become truly decentralized.

## Quick Start

These subgraphs only take a few minutes to set up. To get started, follow these three steps:

1. Initialize your subgraph project
2. Deploy & Publish
3. Query from your dapp

Pricing: **All developers receive 100K free queries per month on the decentralized network**. After these free queries, you only pay based on usage at $4 for every 100K queries.

Here’s a step by step walk through:

## 1. Initialize your subgraph project

### Create a subgraph on Subgraph Studio⁠

Go to the [Subgraph Studio](https://thegraph.com/studio/) and connect your wallet. Once your wallet is connected, you can begin by clicking “Create a Subgraph”. Please choose a good name for the subgraph because this name can’t be edited later. It is recommended to use Title Case: “Subgraph Name Chain Name.”

![Create a Subgraph](https://lh7-us.googleusercontent.com/docsz/AD_4nXf8OTdwMxlKQGKzIF_kYR7NPKeh9TmWnZBYxb7ft_YbdOdx_VVtbp6PslN7N1KGUzNpIDCmaXppdrllM1cw_J4L8Na03BXOWzJTK1POCve0nkRjQYgWJ60QHAdtQ4Niy83SMM8m0F0f-N-AJj4PDqDPlA5M?key=fnI6SyFgXU9SZRNX5C5vPQ)


You will then land on your subgraph’s page. All the CLI commands you need will be visible on the right side of the page:

![CLI commands](https://lh7-us.googleusercontent.com/docsz/AD_4nXe3YvCxiOH_LupSWe8zh9AmP-VrV4PlOq3f7Ix6hNlBUYcANUFuLuVIWR74OGiBs0nrugTyT0v3o6RPmTsgHONdv_ZJNWtcDWEkRntXPHlQGFcqmEBa-D6j4aoIPzUKYdOJMVUPu8O3fwjdZ4IaXXZoTzY?key=fnI6SyFgXU9SZRNX5C5vPQ)


### Install the Graph CLI⁠

On your local machine run the following:
```
npm install -g @graphprotocol/graph-cli
```
### Initialize your Subgraph⁠

You can copy this directly from your subgraph page to include your specific subgraph slug:
```
graph init --studio <SUBGRAPH_SLUG>
```
You’ll be prompted to provide some info on your subgraph like this:

![cli sample](https://lh7-us.googleusercontent.com/docsz/AD_4nXdTAUsUb5vbs3GtCrhKhuXM1xYoqqooYTxw6lfJfYtLJNP8GKVOhTPmjxlM1b6Qpx-pXNVOzRuc8BL12wZXqy4MIj8ja0tp15znfuJD_Mg84SSNj3JpQ4d31lNTxPYnpba4UOzZx8pmgOIsbI7vCz70v9gC?key=fnI6SyFgXU9SZRNX5C5vPQ)


Simply have your contract verified on the block explorer and the CLI will automatically obtain the ABI and set up your subgraph. The default settings will generate an entity for each event.

## 2. Deploy & Publish

### Deploy to Subgraph Studio⁠

First run these commands:

```bash
$ graph codegen
$ graph build
```

Then run these to authenticate and deploy your subgraph. You can copy these commands directly from your subgraph’s page in Studio to include your specific deploy key and subgraph slug:

```bash
$ graph auth --studio <DEPLOY_KEY>
$ graph deploy --studio <SUBGRAPH_SLUG>
```

You will be asked for a version label. You can enter something like v0.0.1, but you’re free to choose the format.

### Test your subgraph⁠

You can test your subgraph by making a sample query in the playground section. The Details tab will show you an API endpoint. You can use that endpoint to test from your dapp.

![Playground](https://lh7-us.googleusercontent.com/docsz/AD_4nXf3afwSins8_eO7BceGPN79VvwolDxmFNUnkPk0zAJCaUA-3-UAAjVvrMzwr7q9vNYWdrEUNgm2De2VfQpWauiT87RkFc-cVfoPSsQbYSgsmwhyY1-tpPdv2J1H4JAMq70nfWBhb8PszZBFjsbDAaJ5eto?key=fnI6SyFgXU9SZRNX5C5vPQ)


### Publish Your Subgraph to The Graph’s Decentralized Network

Once your subgraph is ready to be put into production, you can publish it to the decentralized network. On your subgraph’s page in Subgraph Studio, click on the Publish button:

![publish button](https://edgeandnode.notion.site/image/https%3A%2F%2Fprod-files-secure.s3.us-west-2.amazonaws.com%2Fa7d6afae-8784-4b15-a90e-ee8f6ee007ba%2F2f9c4526-123d-4164-8ea8-39959c8babbf%2FUntitled.png?table=block&id=37005371-76b4-4780-b044-040a570e3af6&spaceId=a7d6afae-8784-4b15-a90e-ee8f6ee007ba&width=1420&userId=&cache=v2)


Before you can query your subgraph, Indexers need to begin serving queries on it. In order to streamline this process, you can curate your own subgraph using GRT.

When publishing, you’ll see the option to curate your subgraph. As of May 2024, it is recommended that you curate your own subgraph with at least 3,000 GRT to ensure that it is indexed and available for querying as soon as possible.

![Publish screen](https://lh7-us.googleusercontent.com/docsz/AD_4nXerUr-IgWjwBZvp9Idvz5hTq8AFB0n_VlXCzyDtUxKaCTANT4gkk-2O77oW-a0ZWOh3hnqQsY7zcSaLeCQin9XU1NTX1RVYOLFX9MuVxBEqcMryqgnGQKx-MbDnOWKuMoLBhgyVWQereg3cdWtCPcTQKFU?key=fnI6SyFgXU9SZRNX5C5vPQ)

> **Note:** The Graph's smart contracts are all on Arbitrum One, even though your subgraph is indexing data from Ethereum, BSC or any other [supported chain](https://thegraph.com/docs/en/developing/supported-networks/). 

## 3. Query your Subgraph

Congratulations! You can now query your subgraph on the decentralized network!

For any subgraph on the decentralized network, you can start querying it by passing a GraphQL query into the subgraph’s query URL which can be found at the top of its Explorer page.

Here’s an example from the [CryptoPunks Ethereum subgraph](https://thegraph.com/explorer/subgraphs/HdVdERFUe8h61vm2fDyycHgxjsde5PbB832NHgJfZNqK) by Messari:

![Query URL](https://lh7-us.googleusercontent.com/docsz/AD_4nXebivsPOUjPHAa3UVtvxoYTFXaGBao9pQOAJvFK0S7Uv0scfL6TcTVjmNCzT4DgsIloAQyrPTCqHjFPtmjyrzoKkfSeV28FjS32F9-aJJm0ILAHey2gqMr7Seu4IqPz2d__QotsWG3OKv2dEghiD74eypzs?key=fnI6SyFgXU9SZRNX5C5vPQ)


The query URL for this subgraph is:

https://gateway-arbitrum.network.thegraph.com/api/**[api-key]**/subgraphs/id/HdVdERFUe8h61vm2fDyycHgxjsde5PbB832NHgJfZNqK

Now, you simply need to  fill in your own API Key to start sending GraphQL queries to this endpoint.

### Getting your own API Key

![API keys](https://lh7-us.googleusercontent.com/docsz/AD_4nXdz7H8hSRf2XqrU0jN3p3KbmuptHvQJbhRHOJh67nBfwh8RVnhTsCFDGA_JQUFizyMn7psQO0Vgk6Vy7cKYH47OyTq5PqycB0xxLyF4kSPsT7hYdMv2MEzAo433sJT6VlQbUAzgPnSxKI9a5Tn3ShSzaxI?key=fnI6SyFgXU9SZRNX5C5vPQ)


In Subgraph Studio, you’ll see the “API Keys” menu at the top of the page. Here you can create API Keys.

## Appendix

### Sample Query

This query shows the most expensive CryptoPunks sold.

```graphql
{
  trades(orderBy: priceETH, orderDirection: desc) {
    priceETH
    tokenId
  }
}

```

Passing this into the query URL returns this result:

```
{
  "data": {
    "trades": [
      {
        "priceETH": "124457.067524886018255505",
        "tokenId": "9998"
      },
      {
        "priceETH": "8000",
        "tokenId": "5822"
      },
//      ...
```

<aside>
💡 Trivia: Looking at the top sales on [CryptoPunks website](https://cryptopunks.app/cryptopunks/topsales) it looks like the top sale is Punk #5822, not #9998. Why? Because they censor the flash-loan sale that happened.

</aside>

### Sample code

```jsx
const axios = require('axios');

const graphqlQuery = `{
  trades(orderBy: priceETH, orderDirection: desc) {
    priceETH
    tokenId
  }
}`;
const queryUrl = 'https://gateway-arbitrum.network.thegraph.com/api/[api-key]/subgraphs/id/HdVdERFUe8h61vm2fDyycHgxjsde5PbB832NHgJfZNqK'

const graphQLRequest = {
  method: 'post',
  url: queryUrl,
  data: {
    query: graphqlQuery,
  },
};

// Send the GraphQL query
axios(graphQLRequest)
  .then((response) => {
    // Handle the response here
    const data = response.data.data
    console.log(data)

  })
  .catch((error) => {
    // Handle any errors
    console.error(error);
  });
```

### Additional resources:

- To explore all the ways you can optimize & customize your subgraph for a better performance, read more about [creating a subgraph here](https://thegraph.com/docs/en/developing/creating-a-subgraph/).
- For more information about querying data from your subgraph, read more [here](https://thegraph.com/docs/en/querying/querying-the-graph/).