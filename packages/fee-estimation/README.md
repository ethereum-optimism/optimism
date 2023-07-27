# @eth-optimism/fee-estimation

## Overview

This package is designed to provide an easy way to estimate gas on OP chains.

Fee estimation on OP-chains has both an l2 and l1 component. By default tools such as Viem, Wagmi, Ethers, and Web3.js do not support the l1 component. They will support this soon but in meantime, this library can help estimate fees for transactions, or act as a reference.
As these tools add support for gas estimation natively this README will be updated with framework specific instructions.

For more detailed information about gas fees on Optimism's Layer 2, you can visit their [official documentation](https://community.optimism.io/docs/developers/build/transaction-fees/#the-l2-execution-fee).

## GasPriceOracle contract

- The l2 contract that can estimate l1Fees is called [GasPriceOracle](../contracts-bedrock/contracts/l2/GasPriceOracle.sol) contract. This library provides utils for interacting with it at a high level.
- The GasPriceOracle is [deployed to Optimism](https://optimistic.etherscan.io/address/0x420000000000000000000000000000000000000F) and other OP chains at a predeployed address of `0x420000000000000000000000000000000000000F`

This library provides a higher level abstraction over the gasPriceOracle

## Installation

```bash
pnpm install @eth-optimism/fee-estimation
```

```bash
npm install @eth-optimism/fee-estimation
```

```bash
yarn add @eth-optimism/fee-estimation
```

## Usage

### Import the package

```ts
import { estimateFees } from '@eth-optimism/fee-estimation'
```

### Basic Usage

```ts
const fees = await estimateFees({
  client: {
    chainId: 10,
    rpcUrl: 'https://mainnet.optimism.io',
  },
  blockNumber: BigInt(106889079),
  account: '0xe371815c5f8a4f9acd1576879de288acd81269f1',
  to: '0xe35f24470730f5a488da9721548c1ab0b65b53d5',
  data: '0x5c19a95c00000000000000000000000046abfe1c972fca43766d6ad70e1c1df72f4bb4d1',
})
```

## API

### `estimateFees` function

```ts
estimateFees(params: EstimateFeeParams): Promise<bigint>
```

#### Parameters

`params`: An object with the following fields:

- `client`: An object with `rpcUrl` and `chainId` fields, or an instance of a Viem `PublicClient`.

- `blockNumber`: A BigInt representing the block number at which you want to estimate the fees.

- `blockTag`: A string representing the block tag to query from.

- `account`: A string representing the account making the transaction.

- `to`: A string representing the recipient of the transaction.

- `data`: A string representing the data being sent with the transaction. This should be a 0x-prefixed hex string.

#### Returns

A Promise that resolves to a BigInt representing the estimated fee in wei.

## FAQ

### How to encode function data?

You can use our package to encode the function data. Here is an example:

```ts
import { encodeFunctionData } from '@eth-optimism/fee-estimation'
import { optimistABI } from '@eth-optimism/contracts-ts'

const data = encodeFunctionData({
  functionName: 'burn',
  abi: optimistABI,
  args: [BigInt('0x77194aa25a06f932c10c0f25090f3046af2c85a6')],
})
```

This will return a 0x-prefixed hex string that represents the encoded function data.

## Testing

The package provides a set of tests that you can run to verify its operation. The tests are a great resource for examples. You can find them at `./src/estimateFees.spec.ts`.

## Other methods

This package also provides lower level methods for estimating gas

### getL2Client()

This method returns a Layer 2 (L2) client that communicates with an L2 network.

```ts
getL2Client(options: ClientOptions): PublicClient;
```

#### Parameters

- `options: ClientOptions` - The options required to initialize the L2 client.

#### Returns

- `PublicClient` - Returns a public client that can interact with the L2 network.

#### Example

```ts
const clientParams = {
  chainId: 10,
  rpcUrl: process.env.VITE_L2_RPC_URL ?? 'https://mainnet.optimism.io',
} as const

const client = getL2Client(clientParams)
```

---

### baseFee()

Returns the base fee.

```ts
baseFee({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - The options required to fetch the base fee.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the base fee.

#### Example

```ts
const blockNumber = BigInt(106889079)
const paramsWithClient = {
  client: clientParams,
  blockNumber,
}
const baseFeeValue = await baseFee(paramsWithClient)
```

---

### decimals()

Returns the decimals used in the scalar.

```ts
decimals({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - The options required to fetch the decimals.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the decimals used in the scalar.

#### Example

```ts
const decimalsValue = await decimals(paramsWithClient)
```

---

### gasPrice()

Returns the gas price.

```ts
gasPrice({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - The options required to fetch the gas price.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the gas price.

#### Example

```ts
const gasPriceValue = await gasPrice(paramsWithClient)
```

---

### getL1Fee()

Computes the L1 portion of the fee based on the size of the rlp encoded input transaction, the current L1 base fee, and the various dynamic parameters.

```ts
getL1Fee(data: Bytes, { client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `data: Bytes` - The transaction call data as a 0x-prefixed hex string.
- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the L1 portion of the fee.

#### Example

```ts
const data =
  '0x5c19a95c00000000000000000000000046abfe1c972fca43766d6ad70e1c1df72f4bb4d1'
const l1FeeValue = await getL1Fee(data, paramsWithClient)
```

### getL1GasUsed()

This method returns the amount of gas used on the L1 network for a given transaction.

```ts
getL1GasUsed(data: Bytes, { client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `data: Bytes` - The transaction call data as a 0x-prefixed hex string.
- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the amount of gas used on the L1 network for the given transaction.

#### Example

```ts
const data =
  '0x5c19a95c00000000000000000000000046abfe1c972fca43766d6ad70e1c1df72f4bb4d1'
const l1GasUsed = await getL1GasUsed(data, paramsWithClient)
```

---

### l1BaseFee()

Returns the base fee on the L1 network.

```ts
l1BaseFee({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the base fee on the L1 network.

#### Example

```ts
const l1BaseFeeValue = await l1BaseFee(paramsWithClient)
```

---

### overhead()

Returns the overhead for the given transaction.

```ts
overhead({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the overhead for the given transaction.

#### Example

```ts
const overheadValue = await overhead(paramsWithClient)
```

---

### scalar()

Returns the scalar value for the gas estimation.

```ts
scalar({ client, ...params }: GasPriceOracleOptions): Promise<bigint>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the scalar value for the gas estimation.

#### Example

```ts
const scalarValue = await scalar(paramsWithClient)
```

---

### version()

Returns the version of the fee estimation library.

```ts
version({ client, ...params }: GasPriceOracleOptions): Promise<string>;
```

#### Parameters

- `{ client, ...params }: GasPriceOracleOptions` - Optional lock options and provider options.

#### Returns

- `Promise<string>` - Returns a promise that resolves to the version of the fee estimation library.

#### Example

```ts
const libraryVersion = await version(paramsWithClient)
```

---

### encodeFunctionData()

Encodes function data based on a given function name and arguments.

```ts
encodeFunctionData({ functionName, abi, args }: EncodeFunctionDataParams): string;
```

#### Parameters

- `{ functionName, abi, args }: EncodeFunctionDataParams` - An object containing the function name, ABI (Application Binary Interface), and arguments.

#### Returns

- `string` - Returns the encoded function data as a string.

#### Example

```ts
const encodedData = encodeFunctionData({
  functionName: 'burn',
  abi: optimistABI,
  args: [BigInt(optimistOwnerAddress)],
})
```

---

### estimateFees()

Estimates the fee for a transaction given the input data and the address of the sender and recipient.

```ts
estimateFees({ client, data, account, to, blockNumber }: EstimateFeesParams): Promise<bigint>;
```

#### Parameters

- `{ client, data, account, to, blockNumber }: EstimateFeesParams` - An object containing the client, transaction data, sender's address, recipient's address, and block number.

#### Returns

- `Promise<bigint>` - Returns a promise that resolves to the estimated fee for the given transaction.

#### Example

```ts
const estimateFeesParams = {
  data: '0xd1e16f0a603acf1f8150e020434b096e408bafa429a7134fbdad2ae82a9b2b882bfcf5fe174162cf4b3d5f2ab46ff6433792fc99885d55ce0972d982583cc1e11b64b1d8d50121c0497642000000000000000000000000000000000000060a2c8052ed420000000000000000000000000000000000004234002c8052edba12222222228d8ba445958a75a0704d566bf2c84200000000000000000000000000000000000006420000000000000000000000000000000000004239965c9dab5448482cf7e002f583c812ceb53046000100000000000000000003',
  account: '0xe371815c5f8a4f9acd1576879de288acd81269f1',
  to: '0xe35f24470730f5a488da9721548c1ab0b65b53d5',
}
const estimatedFees = await estimateFees({
  ...paramsWithClient,
  ...estimateFeesParams,
})
```

I hope this information is helpful!
