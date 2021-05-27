# @eth-optimism/core-utils

## What is this?

`@eth-optimism/core-utils` contains the Optimistic Virtual Machine core utilities.

## Getting started

### Building and usage

After cloning and switching to the repository, install dependencies:

```bash
$ yarn
```

Use the following commands to build, use, test, and lint:

```bash
$ yarn build
$ yarn start
$ yarn test
$ yarn lint
```

### L2 Fees

The Layer 2 fee is encoded in `tx.gasLimit`. The Layer 2 `gasLimit` is encoded
in the lower order bits of the `tx.gasLimit`. For this scheme to work, both the
L1 gas price and the L2 gas price must satisfy specific equations. There are
functions that help ensure that the correct gas prices are used.

- `roundL1GasPrice`
- `roundL2GasPrice`

The Layer 2 fee is based on both the cost of submitting the data to L1 as well
as the cost of execution on L2. To make libraries like `ethers` just work, the
return value of `eth_estimateGas` has been modified to return the fee. A new RPC
endpoint `eth_estimateExecutionGas` has been added that returns the L2 gas used.

To locally encode the `tx.gasLimit`, the `L2GasLimit` methods `encode` and
`decode` should be used.

```typescript
import { L2GasLimit, roundL1GasPrice, roundL2GasPrice } from '@eth-optimism/core-utils'
import { JsonRpcProvider } from 'ethers'

const provider = new JsonRpcProvider('https://mainnet.optimism.io')
const gasLimit = await provider.send('eth_estimateExecutionGas', [tx])

const encoded = L2GasLimit.encode({
  data: '0x',
  l1GasPrice: roundL1GasPrice(1),
  l2GasLimit: gasLimit,
  l2GasPrice: roundL2GasPrice(1),
})

const decoded = L2GasLimit.decode(encoded)
assert(decoded.eq(gasLimit))
```
