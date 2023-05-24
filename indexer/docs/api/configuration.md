## Configuration

Configuration of the indexer is done with a .toml file

- Configure a known chain such as Optimism, Base, or BaseGoerli simply pass in a `preset = CHAIN_ID`.
- For custom chains additional configuration is needed to configure such as l1 and l2 contract addresses

Here is an example indexer.toml

```toml indexer.toml
[chain]
preset = 84531

[rpcs]
l1-rpc = "https://base-goerli.g.alchemy.com/v2/YOUR_API_KEY"
l2-rpc = "https://base-goerli.g.alchemy.com/v2/YOUR_API_KEY"

[db]
host = "127.0.0.1"
port = 4321
user = "postgres"
password = "postgres"

[api]
host: "127.0.0.1"
port: 8080

[metrics]
host: "127.0.0.1"
port: 7300
```

