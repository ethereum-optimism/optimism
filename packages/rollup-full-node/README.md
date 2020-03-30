# Dependencies
The `/exec/` scripts depend on [parity](https://github.com/paritytech/parity-ethereum/releases/tag/v2.5.13) being installed.

If you'd like to use a containerized version, you'll need to [install docker](https://docs.docker.com/install/).

For other dependencies, please refer to the root README of this repo.

# Setup
Run `yarn install` to install necessary dependencies.

# Building
Run `yarn build` to build the code. Note: `yarn all` may be used to build and run tests.

## Building Docker Image
_Make sure you're in the base directory_ (`cd ../..`)

Run `docker build -t optimism/rollup-full-node .` to build and tag the fullnode.

### Pushing Image to AWS Registry:
Note: Image registration and deployment to our internal dev environment is done automatically upon merge to the `master` branch.

Make sure the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) is installed and [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html#cli-quick-configuration)

1. Make sure you're authenticated: 
    ```
    aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node
    ```
2. Build and tag latest: 
    ```
    docker build -t optimism/rollup-full-node .
    ```
3. Tag the build: 
    ```
    optimism/rollup-full-node:latest <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node:latest
    ```
4. Push tag to ECR:
    ```
    docker push <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/rollup-full-node:latest
    ``` 

## Running in Docker
_Make sure you're in the base directory_ (`cd ../..`)

Run `docker-compose up --build` to build and run. If you don't need to build the full node or geth, omit the `--build`

When the containers are up, you should see the following output:
```
rollup-full-node_1  | <timestamp> info:rollup-fullnode Listening at http://0.0.0.0:8545
```

You can run a simple connectivity test against the rollup node by running:
```
curl -H "Content-Type: application/json" -d '{"jsonrpc": "2.0", "id": 9999999, "method": "net_version"}' http://0.0.0.0:8545
```
which should yield the response:
```
{"id":9999999,"jsonrpc":"2.0","result":"108"}
```

# Testing
Run `yarn test` to run the unit tests.

# Configuration
`/config/default.json` specifies the default configuration. Overrides will be read from environment variables with the same key.

`/config/parity/local-chain-config.json` configures the local parity chain. This should not normally need modification.

# Running the Aggregator Server
Run `yarn server:aggregator` to run the aggregator server.

# Running the Fullnode Server
Run `yarn server:fullnode` to run the fullnode server.

# Running a Persistent Chain
Run `./exec/startChain.sh` to start a local persistent blockchain.
Note: This chain will be initiated with a LOT of ETH in the following account:
* address: `0x77e3E8EF810e2eD36c396A80EC21379e345b862e`
* mnemonic: `response fresh afford leader twice silent table exist aisle pelican focus bird`

# Deleting Persistent Chain DB
Run `./exec/purgeChainDb.sh`

