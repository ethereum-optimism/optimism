## Configuration

Configuration of the indexer is done with a .toml file

- To configure a known chain such as Optimism, Base, or BaseGoerli simply pass in a `preset = CHAIN_ID`.
- For custom chains additional configuration is needed to configure such as l1 and l2 contract addresses

Here is an example indexer.toml

```toml indexer.toml
[chain]
preset = 10

[rpcs]
l1-rpc = "https://eth-goerli.g.alchemy.com/v2/YOUR_API_KEY"
l2-rpc = "https://eth-goerli.g.alchemy.com/v2/YOUR_API_KEY"

[db]
host = "http://localhost"
port = 4321
user = 'postgres'
password = "postgres"

[api]
hostname: "127.0.0.1"
port: 8080

[metrics]
hostname: "127.0.0.1"
port: 7300
```

- **chain configuration**

Configure chain constants such as chainId, contract addresses ect.

For known chains pass in the chainId as `preset`

- **rpcs**

Configures the rpcs used to populate the indexer database

- **db**

Configures the Postgresql db

- **api**

Configures the api server

- **metrics**

Configures the metrics server

