# Gas Price Oracle

This service is responsible for updating the `gasPrice` in the `OVM_GasPriceOracle.sol` contract, so the Sequencer can fetch the latest `gasPrice` and update the L2 gas price over time.

## Configuration
All configuration is done via environment variables. See all variables at [.env.example](.env.example); copy into a `.env` file before running.

| Environment Variables               | Description                                                  | Default        |
| ----------------------------------- | ------------------------------------------------------------ | -------------- |
| L1_NODE_WEB3_URL                    | The endpoint of Layer 1                                      |                |
| L2_NODE_WEB3_URL                    | The endpoint of Layer 2                                      |                |
| DEPLOYER_PRIVATE_KEY                | The owner of `OVM_GasPriceOracle`                            |                |
| SEQUENCER_PRIVATE_KEY               | The private key of sequencer account                         |                |
| PROPOSER_PRIVATE_KEY                | The private key of proposer account                          |                |
| RELAYER_PRIVATE_KEY                 | The private key of relayer account                           |                |
| FAST_RELAYER_PRIVATE_KEY            | The private key of fast relayer account                      |                |
| GAS_PRICE_ORACLE_ADDRESS            | The address of `OVM_GasPriceOracle`                          |                |
| GAS_PRICE_ORACLE_FLOOR_PRICE        | The minimum L2 gas price                                     | 150000         |
| GAS_PRICE_ORACLE_ROOF_PRICE         | The maximum L2 gas price                                     | 20000000       |
| GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE | The gas price will be updated if it exceeds the minimum price percent change. | 0.1            |
| POLLING_INTERVAL                    | The polling interval                                         | 10 * 60 * 1000 |
| ETHERSCAN_API                       | The API Key of Etherscan                                     |                |

## Building & Running
1. Make sure dependencies are installed just run `yarn` in the base directory
2. Build `yarn build`
3. Run `yarn start`

## L2 Gas Fee

The gas fee of L2 is 

```
gasFee = gasPrice * gasLimit
```

The gas price is **0.015 GWei**. **DON'T UPDATE IT!**

Users can use the maximum gas limit, but it costs more than you actually have to pay. The estimated gas limit is based on `rollup_gasPrices`, `rollup_gasPrices` has `l1GasPrice` and `l2GasPrce`. 

```
estimatedGasLimit = calculateL1GasLimit(data) * L1GasFee + L2GasPrice * L2estimateExecutionGas
```

We update `l2GasPrice` based on our service cost.

## Algorithm

The service fetches the L1 ETH balances of `sequencer`, `proposer`, `relayer` and `fast relayer` in each polling interval. Based on the ETH balances, we can calculate the costs of maintaining the Layer 2.

* `L1ETHBalance`: The ETH balances of all accounts
* `L1ETHCostFee`: The ETH fees that we pay to maintain the Layer 2 since the gas oracle service starts

The service also fetches the L2 gas fees collected by us based on the `gasUsage * gasPrice` and increased L2 block numbers in each polling interval. We also calculate the average gas usage per block, so we can estimate the gas price.

* `L2ETHCollectFee`: The ETH fees that we collect from the Layer 2 transactions.
* `avgL2GasLimitPerBlock` : The average gas limit per block in each polling interval
* `numberOfBlocksInterval`: The increased number of blocks in each pooling interval

The estimated gas usages in the next interval are

```
estimatedGasUsage = avgL2GasLimitPerBlock * numberOfBlocksInterval
```

The estimated L2 gas price that we should charge in the next interval is

```
estimatedL2GasPrice = (L1ETHCostFee - L2ETHCollectFee) / estimatedGasUsage
```

When the estimated L2 gas price is lower than the `GAS_PRICE_ORACLE_FLOOR_PRICE`, we set the gas price as the `GAS_PRICE_ORACLE_FLOOR_PRICE`.

When the estimated L2 gas price is larger than the `GAS_PRICE_ORACLE_ROOF_PRICE`, we set the gas price as the `GAS_PRICE_ORACLE_ROOF_PRICE`.

When the new estimated L2 gas price is not in the range of `(1 + GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE) * latestGasPriceInContract` and `(1 - GAS_PRICE_ORACLE_MIN_PERCENT_CHANGE) * latestGasPriceInContract`, we update the gas price.

## Problem

1. `numberOfBlocksInterval` can affect the gas price significantly. The possible solution is to increase the `POLLING_INTERVAL` to 30 mins.
2. When the service starts, the gas price will be restored to the `GAS_PRICE_ORACLE_FLOOR_PRICE`.
3. The `GAS_PRICE_ORACLE_FLOOR_PRICE` and `GAS_PRICE_ORACLE_ROOF_PRICE` are not easy to be determined in the test environment.
