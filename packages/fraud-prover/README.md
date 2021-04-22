# Fraud Prover

Contains an executable fraud prover.

## Configuration

All configuration is done via environment variables. See all variables at [.env.example](.env.example); copy into a `.env` file before running.

## Building & Running

1. Make sure dependencies are installed just run `yarn` in the base directory
2. Build `yarn build`
3. Run `yarn start`

## Testing & linting

### Local

- See lint errors with `yarn lint`; auto-fix with `yarn lint --fix`

## Fraud Prover

| Environment Variable   | Required? | Default Value         | Description            |
| -----------            | --------- | -------------         | -----------           |
| `L1_WALLET_KEY`        | Yes       | N/A                   | Private key for an account on Layer 1 (Ethereum) to be used to submit fraud proof transactions. |
| `L2_NODE_WEB3_URL`     | No        | http://localhost:9545 | HTTP endpoint for a Layer 2 (Optimism) Verifier node.  |
| `L1_NODE_WEB3_URL`     | No        | http://localhost:8545 | HTTP endpoint for a Layer 1 (Ethereum) node.      |
| `RELAY_GAS_LIMIT`      | No        | 9,000,000             | Maximum amount of gas to provide to fraud proof transactions (except for the "transaction execution" step). |
| `RUN_GAS_LIMIT`        | No        | 9,000,000             | Maximum amount of gas to provide to the "transaction execution" step. |
| `POLLING_INTERVAL`     | No        | 5,000                 | Time (in milliseconds) to wait while polling for new transactions. |
| `L2_BLOCK_OFFSET`      | No        | 1                     | Offset between the `CanonicalTransactionChain` contract on Layer 1 and the blocks on Layer 2. Currently defaults to 1, but will likely be removed as soon as possible. |
| `L1_BLOCK_FINALITY`    | No        | 0                     | Number of Layer 1 blocks to wait before considering a given event. |
| `L1_START_OFFSET`      | No        | 0                     | Layer 1 block number to start scanning for transactions from. |
| `FROM_L2_TRANSACTION_INDEX` | No        | 0                     | Layer 2 block number to start scanning for transactions from. |

