# @eth-optimism/web3.js-plugin

This web3.js plugin adds utility functions for estimating L1 and L2 gas for OP chains by wrapping the [GasPriceOracle](../contracts-bedrock/contracts/l2/GasPriceOracle.sol) contract

The GasPriceOracle is [deployed to Optimism](https://optimistic.etherscan.io/address/0x420000000000000000000000000000000000000F) and other OP chains at a predeployed address of `0x420000000000000000000000000000000000000F`

For more detailed information about gas fees on Optimism's Layer 2, you can visit the [official documentation](https://community.optimism.io/docs/developers/build/transaction-fees/#the-l2-execution-fee)

## Installation

This plugin is intended to be [registered](https://docs.web3js.org/guides/web3_plugin_guide/plugin_users#registering-the-plugin) onto an instance of `Web3`. It has a [peerDependency](https://nodejs.org/es/blog/npm/peer-dependencies) of `web3` version `4.x`, so make sure you have that latest version of `web3` installed for your project before installing the plugin

### Installing the Plugin

```bash
pnpm install @eth-optimism/web3.js-plugin
```

```bash
npm install @eth-optimism/web3.js-plugin
```

```bash
yarn add @eth-optimism/web3.js-plugin
```

### Registering the Plugin

```typescript
import Web3 from 'web3'
import { OptimismPlugin } from '@eth-optimism/web3.js-plugin'

const web3 = new Web3('http://yourProvider.com')
web3.registerPlugin(new OptimismPlugin())
```

You will now have access to the following functions under the `op` namespace, i.e. `web3.op.someMethod`

## API

| Function Name                            | Returns                                                                                                                                                                                                                |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [estimateFees](#estimatefees)            | The combined estimated L1 and L2 fees for a transaction                                                                                                                                                                |
| [getL1Fee](#getl1fee)                    | The L1 portion of the fee based on the size of the [RLP](https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/) encoded transaction, the current L1 base fee, and other various dynamic parameters |
| [getL2Fee](#getl2fee)                    | The L2 portion of the fee based on the simulated execution of the provided transaction and current `gasPrice`                                                                                                          |
| [getBaseFee](#getbasefee)                | The current L2 base fee                                                                                                                                                                                                |
| [getDecimals](#getdecimals)              | The decimals used in the scalar                                                                                                                                                                                        |
| [getGasPrice](#getgasprice)              | The current L2 gas price                                                                                                                                                                                               |
| [getL1GasUsed](#getl1gasused)            | The amount of L1 gas estimated to be used to execute a transaction                                                                                                                                                     |
| [getL1BaseFee](#getdegetl1basefeecimals) | The L1 base fee                                                                                                                                                                                                        |
| [getOverhead](#getoverhead)              | The current overhead                                                                                                                                                                                                   |
| [getScalar](#getscalar)                  | The current fee scalar                                                                                                                                                                                                 |
| [getVersion](#getversion)                | The current version of `GasPriceOracle`                                                                                                                                                                                |

---

### `estimateFees`

Computes the total (L1 + L2) fee estimate to execute a transaction

```typescript
async estimateFees(transaction: Transaction, returnFormat?: ReturnFormat)
```

#### Parameters

- `transaction: Transaction` - An unsigned web3.js [transaction](https://docs.web3js.org/api/web3-types/interface/Transaction) object
- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][1] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The estimated total fee as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
import Web3 from 'web3'
import { OptimismPlugin } from '@eth-optimism/web3.js-plugin'
import {
  l2StandardBridgeABI,
  l2StandardBridgeAddress,
} from '@eth-optimism/contracts-ts'

const web3 = new Web3('https://mainnet.optimism.io')
web3.registerPlugin(new OptimismPlugin())

const l2BridgeContract = new web3.eth.Contract(
  l2StandardBridgeABI,
  optimistAddress[420]
)
const encodedWithdrawMethod = l2BridgeContract.methods
  .withdraw(
    // l2 token address
    '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
    // amount
    Web3.utils.toWei('0.00000001', 'ether'),
    // l1 gas
    0,
    // extra data
    '0x00'
  )
  .encodeABI()

const totalFee = await web3.op.estimateFees({
  chainId: 10,
  data: encodedWithdrawMethod,
  value: Web3.utils.toWei('0.00000001', 'ether'),
  type: 2,
  to: '0x420000000000000000000000000000000000000F',
  from: '0x6387a88a199120aD52Dd9742C7430847d3cB2CD4',
  maxFeePerGas: Web3.utils.toWei('0.2', 'gwei'),
  maxPriorityFeePerGas: Web3.utils.toWei('0.1', 'gwei'),
})

console.log(totalFee) // 26608988767659n
```

##### Formatting Response as a Hex String

```typescript
import Web3 from 'web3'
import { OptimismPlugin } from '@eth-optimism/web3.js-plugin'
import {
  l2StandardBridgeABI,
  l2StandardBridgeAddress,
} from '@eth-optimism/contracts-ts'

const web3 = new Web3('https://mainnet.optimism.io')
web3.registerPlugin(new OptimismPlugin())

const l2BridgeContract = new web3.eth.Contract(
  l2StandardBridgeABI,
  optimistAddress[420]
)
const encodedWithdrawMethod = l2BridgeContract.methods
  .withdraw(
    // l2 token address
    '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
    // amount
    Web3.utils.toWei('0.00000001', 'ether'),
    // l1 gas
    0,
    // extra data
    '0x00'
  )
  .encodeABI()

const totalFee = await web3.op.estimateFees(
  {
    chainId: 10,
    data: encodedWithdrawMethod,
    value: Web3.utils.toWei('0.00000001', 'ether'),
    type: 2,
    to: '0x420000000000000000000000000000000000000F',
    from: '0x6387a88a199120aD52Dd9742C7430847d3cB2CD4',
    maxFeePerGas: Web3.utils.toWei('0.2', 'gwei'),
    maxPriorityFeePerGas: Web3.utils.toWei('0.1', 'gwei'),
  },
  { number: FMT_NUMBER.HEX, bytes: FMT_BYTES.HEX }
)

console.log(totalFee) // 0x18336352c5ab
```

### `getL1Fee`

Computes the L1 portion of the fee based on the size of the rlp encoded input transaction, the current L1 base fee, and the various dynamic parameters

```typescript
async getL1Fee(transaction: Transaction, returnFormat?: ReturnFormat)
```

#### Parameters

- `transaction: Transaction` - An unsigned web3.js [transaction](https://docs.web3js.org/api/web3-types/interface/Transaction) object
- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][1] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The estimated L1 fee as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
import { Contract } from 'web3'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'

const optimistContract = new Contract(optimistABI, optimistAddress[420])
const encodedBurnMethod = optimistContract.methods
  .burn('0x77194aa25a06f932c10c0f25090f3046af2c85a6')
  .encodeABI()

const l1Fee = await web3.op.getL1Fee({
  chainId: 10,
  data: encodedBurnMethod,
  type: 2,
})

console.log(l1Fee) // 18589035222172n
```

##### Formatting Response as a Hex String

```typescript
import { Contract } from 'web3'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'

const optimistContract = new Contract(optimistABI, optimistAddress[420])
const encodedBurnMethod = optimistContract.methods
  .burn('0x77194aa25a06f932c10c0f25090f3046af2c85a6')
  .encodeABI()

const l1Fee = await web3.op.getL1Fee(
  {
    chainId: 10,
    data: encodedBurnMethod,
    type: 2,
  },
  { number: FMT_NUMBER.HEX, bytes: FMT_BYTES.HEX }
)

console.log(l1Fee) // 0x10e818d7549c
```

### `getL2Fee`

Retrieves the amount of L2 gas estimated to execute `transaction`

```typescript
async getL2Fee(transaction: Transaction, returnFormat?: ReturnFormat)
```

#### Parameters

- `transaction: Transaction` - An unsigned web3.js [transaction](https://docs.web3js.org/api/web3-types/interface/Transaction) object
- `options?: { blockNumber?: BlockNumberOrTag, returnFormat?: ReturnFormat }` - An optional object with properties:
  - `blockNumber?: BlockNumberOrTag` - Specifies what block to use for gas estimation. Can be either:
    - **Note** Specifying a block to estimate L2 gas for is currently not working
    - A web3.js [Numbers](https://docs.web3js.org/api/web3-types#Numbers)
    - A web3.js [BlockTags](https://docs.web3js.org/api/web3-types/enum/BlockTags)
    - If not provided, `BlockTags.LATEST` is used
  - `returnFormat?: ReturnFormat` - A web3.js [DataFormat][1] object that specifies how to format number and bytes values
    - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The estimated total fee as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
import { Contract } from 'web3'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'

const optimistContract = new Contract(optimistABI, optimistAddress[420])
const encodedBurnMethod = optimistContract.methods
  .burn('0x77194aa25a06f932c10c0f25090f3046af2c85a6')
  .encodeABI()

const l2Fee = await web3.op.getL2Fee({
  chainId: '0xa',
  data: encodedBurnMethod,
  type: '0x2',
  to: optimistAddress[420],
  from: '0x77194aa25a06f932c10c0f25090f3046af2c85a6',
})

console.log(l2Fee) // 2659500n
```

##### Formatting Response as a Hex String

```typescript
import { Contract } from 'web3'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'

const optimistContract = new Contract(optimistABI, optimistAddress[420])
const encodedBurnMethod = optimistContract.methods
  .burn('0x77194aa25a06f932c10c0f25090f3046af2c85a6')
  .encodeABI()

const l2Fee = await web3.op.getL2Fee(
  {
    chainId: '0xa',
    data: encodedBurnMethod,
    type: '0x2',
    to: optimistAddress[420],
    from: '0x77194aa25a06f932c10c0f25090f3046af2c85a6',
  },
  {
    returnFormat: { number: FMT_NUMBER.HEX, bytes: FMT_BYTES.HEX },
  }
)

console.log(l2Fee) // 0x2894ac
```

### `getBaseFee`

Retrieves the current L2 base fee

```typescript
async getBaseFee(returnFormat?: ReturnFormat)
```

#### Parameters

- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][1] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The L2 base fee as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
const baseFee = await web3.op.getBaseFee()

console.log(baseFee) // 68n
```

##### Formatting Response as a Hex String

```typescript
const baseFee = await web3.op.getBaseFee({
  number: FMT_NUMBER.HEX,
  bytes: FMT_BYTES.HEX,
})

console.log(baseFee) // 0x44
```

### `getDecimals`

Retrieves the decimals used in the scalar

```typescript
async getDecimals(returnFormat?: ReturnFormat)
```

#### Parameters

- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][3] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The number of decimals as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
const decimals = await web3.op.getDecimals()

console.log(decimals) // 6n
```

##### Formatting Response as a Hex String

```typescript
const decimals = await web3.op.getDecimals({
  number: FMT_NUMBER.HEX,
  bytes: FMT_BYTES.HEX,
})

console.log(decimals) // 0x6
```

### `getGasPrice`

Retrieves the current L2 gas price (base fee)

```typescript
async getGasPrice(returnFormat?: ReturnFormat)
```

#### Parameters

- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][3] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The current L2 gas price as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
const gasPrice = await web3.op.getGasPrice()

console.log(gasPrice) // 77n
```

##### Formatting Response as a Hex String

```typescript
const gasPrice = await web3.op.getGasPrice({
  number: FMT_NUMBER.HEX,
  bytes: FMT_BYTES.HEX,
})

console.log(gasPrice) // 0x4d
```

### `getL1GasUsed`

Computes the amount of L1 gas used for {transaction}. Adds the overhead which represents the per-transaction gas overhead of posting the {transaction} and state roots to L1. Adds 68 bytes of padding to account for the fact that the input does not have a signature.

```typescript
async getL1GasUsed(transaction: Transaction, returnFormat?: ReturnFormat)
```

#### Parameters

- `transaction: Transaction` - An unsigned web3.js [transaction](https://docs.web3js.org/api/web3-types/interface/Transaction) object
- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][3] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The amount of gas as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
import { Contract } from 'web3'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'

const optimistContract = new Contract(optimistABI, optimistAddress[420])
const encodedBurnMethod = optimistContract.methods
  .burn('0x77194aa25a06f932c10c0f25090f3046af2c85a6')
  .encodeABI()

const l1GasUsed = await web3.op.getL1GasUsed({
  chainId: 10,
  data: encodedBurnMethod,
  type: 2,
})

console.log(l1GasUsed) // 1884n
```

##### Formatting Response as a Hex String

```typescript
import { Contract } from 'web3'
import { optimistABI, optimistAddress } from '@eth-optimism/contracts-ts'

const optimistContract = new Contract(optimistABI, optimistAddress[420])
const encodedBurnMethod = optimistContract.methods
  .burn('0x77194aa25a06f932c10c0f25090f3046af2c85a6')
  .encodeABI()

const l1GasUsed = await web3.op.getL1GasUsed(
  {
    chainId: 10,
    data: encodedBurnMethod,
    type: 2,
  },
  { number: FMT_NUMBER.HEX, bytes: FMT_BYTES.HEX }
)

console.log(l1GasUsed) // 0x75c
```

### `getL1BaseFee`

Retrieves the latest known L1 base fee

```typescript
async getL1BaseFee(returnFormat?: ReturnFormat)
```

#### Parameters

- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][3] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The L1 base fee as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
const baseFee = await web3.op.getL1BaseFee()

console.log(baseFee) // 13752544112n
```

##### Formatting Response as a Hex String

```typescript
const baseFee = await web3.op.getL1BaseFee({
  number: FMT_NUMBER.HEX,
  bytes: FMT_BYTES.HEX,
})

console.log(baseFee) // 0x333b72b70
```

### `getOverhead`

Retrieves the current fee overhead

```typescript
async getOverhead(returnFormat?: ReturnFormat)
```

#### Parameters

- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][3] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The current overhead as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
const overhead = await web3.op.getOverhead()

console.log(overhead) // 188n
```

##### Formatting Response as a Hex String

```typescript
const overhead = await web3.op.getOverhead({
  number: FMT_NUMBER.HEX,
  bytes: FMT_BYTES.HEX,
})

console.log(overhead) // 0xbc
```

### `getScalar`

Retrieves the current fee scalar

```typescript
async getScalar(returnFormat?: ReturnFormat)
```

#### Parameters

- `returnFormat?: ReturnFormat` - A web3.js [DataFormat][1] object that specifies how to format number and bytes values
  - If `returnFormat` is not provided, [DEFAULT_RETURN_FORMAT][2] is used which will format numbers to `BigInt`s

#### Returns

- `Promise<Numbers>` - The current scalar fee as a `BigInt` by default, but `returnFormat` determines type

#### Example

```typescript
const scalarFee = await web3.op.getScalar()

console.log(scalarFee) // 684000n
```

##### Formatting Response as a Hex String

```typescript
const scalarFee = await web3.op.getScalar({
  number: FMT_NUMBER.HEX,
  bytes: FMT_BYTES.HEX,
})

console.log(scalarFee) // 0xa6fe0
```

### `getVersion`

Retrieves the full semver version of GasPriceOracle

```typescript
async getVersion()
```

#### Returns

- `Promise<string>` - The semver version

#### Example

```typescript
const version = await web3.op.getVersion()

console.log(version) // 1.0.0
```

## Known Issues

- As of version `4.0.3` of web3.js, both `input` and `data` parameters are automatically added to a transaction objects causing the gas estimations to be inflated. This was corrected in [this](https://github.com/web3/web3.js/pull/6294) PR, but has yet to be released
- For the plugin function `getL2Fee`, you should be able to get the fee estimates using the state of the blockchain at a specified block, however, this doesn't seem to be working with web3.js and requires further investigation

[1]: https://docs.web3js.org/api/web3-types#DataFormat
[2]: https://docs.web3js.org/api/web3-types#DEFAULT_RETURN_FORMAT
[3]: https://docs.web3js.org/api/web3-types#DataFormat