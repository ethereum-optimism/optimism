# AttestationStation sdk docs

Typescript sdk for interacting with the ATST based on [@wagmi/core](https://wagmi.sh/core/getting-started)


## Table of content 

- [Installation](#installation)
- [Basic usage](#basic-usage)
  - [Setup](#setup)
  - [Reading an attestation](#reading-an-attestation)
  - [Reading multiple attestation](#reading-multiple-attestations)
  - [Writing an attestation](#writing-an-attestation)
- [API](#api)
  - [High level API](#high-level-api)
  - [Low level API](#low-level-api)
  - [React API](#react-api)
- [Tutorial](#tutorial)



## Installation

Install atst and its peer dependencies.

npm

```bash
npm i @eth-optimism/atst @wagmi/core ethers@5.7.0
```

pnpm

```bash
pnpm i @eth-optimism/atst @wagmi/core ethers@5.7.0
```

yarn

```bash
yarn add @eth-optimism/atst @wagmi/core ethers@5.7.0
```

**Note:** As ethers v6 is not yet stable we only support ethers v5 at this time


## Basic usage

### Setup

atst uses `@wagmi/core` under the hood. 
[See their documentation for more information](https://wagmi.sh/core/getting-started).

```javascript
import ethers from "ethers"
const wagmiCore = await import("@wagmi/core")
const wagmiAlchemy = await import("@wagmi/core/providers/alchemy")
const wagmiChains = await import("@wagmi/core/chains")

const { chains, provider, webSocketProvider } = wagmiCore.configureChains(
  [wagmiChains.optimismGoerli],
  [wagmiAlchemy.alchemyProvider({ apiKey: process.env.ALCHEMY_API_KEY })],
)

wagmiCore.createClient({
  provider,
  webSocketProvider
})
```


### Reading an attestation

Here is an example of reading an attestation used by the optimist nft

```javascript
import { readAttestation } from '@eth-optimism/atst'

const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
const key = 'optimist.base-uri'

const str = await readAttestationString(creator, about, key)
console.log(attestation) 
// https://assets.optimism.io/4a609661-6774-441f-9fdb-453fdbb89931-bucket/optimist-nft/attributes
```

### Reading multiple attestations

If reading more than one attestation you can use [`readAttestations`](#readattestations) to read them with [multicall](https://www.npmjs.com/package/ethereum-multicall).

### Writing an attestation

To write to an attestation you must [connect](https://wagmi.sh/core/connectors/metaMask) your wagmi client if not already connected. 
If using Node.js use the [mock connector](https://wagmi.sh/core/connectors/mock).

```typescript
import { prepareWriteAttestation, writeAttestation } from '@eth-optimism/atst'

const preparedTx = await prepareWriteAttestation(about, key, 'hello world')
console.log(preparedTx.gasLimit)
await writeAttestation(preparedTx)
```

## API


### High level API

These functions are the easiest way to interact with the AttestationStation contract.
For a more detailed explanation, [see the tutorial](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/ecosystem/attestation-station/using-sdk).

#### `getEvents`

Use `getEvents` to get attestation events using a provider and filters. 

```typescript
const events = await getEvents({
  creator,
  about,
  key,
  value,
  provider: new ethers.providers.JsonRpcProvider('http://localhost:8545'),
  fromBlockOrBlockhash,
  toBlock,
})
```

Set `key`, `about`, `creator`, or `value` to `null` to not filter that value.

#### `prepareWriteAttestation`

[Prepares](https://wagmi.sh/core/actions/prepareWriteContract) an attestation to be written.
This function creates the transaction data, estimates the gas cost, etc.

```typescript
const preparedTx = await prepareWriteAttestation(about, key, 'hello world')
console.log(preparedTx.gasFee)
```

**Return Value:** A prepared transaction, ready for [`writeAttestation`](#writeattestation).

#### `readAttestation`

[Reads](https://wagmi.sh/core/actions/readContract) and parses an attestation based on its data type.

```typescript
const attestation = await readAttestation(
  creator,        // Address: The creator of the attestation
  about,          // Address: The about topic of the attestation
  key,            // string: The key of the attestation
  dataType,       // Optional, the data type of the attestation, 'string' | 'bytes' | 'number' | 'bool' | 'address'
  contractAddress // Optional address: the contract address of AttestationStation
)
```

**Return Value:** The value attested by the `creator` on the `about` address concerning the `key`, when interpreted as the `dataType`.
If there is no such attestation the result is zero, `false`, or an empty string.

#### `readAttestations`

Similar to `readAttestation` but reads multiple attestations at once. 
Pass in a variadic amount of attestations to read.
The parameters are all the same structure type, the one shown below.

```typescript
const attestationList = await readAttestations({
  creator: <creator address>,
  about: <about address>,
  key: <attestation key (string)>,
  [dataType: <data type - 'string' | 'bytes' | 'number' | 'bool' | 'address'>,]
  [contractAddress: <contract address, if not the default>]
}, {...}, {...})
```

**Return Value:** A list of values attested by the `creator` on the `about` address concerning the `key`, when interpreted as the `dataType`.



#### `writeAttestation`

[Writes the prepared tx](https://wagmi.sh/core/actions/writeContract).

```typescript
const preparedTx = await prepareWriteAttestation(about, key, 'hello world')
await writeAttestation(preparedTx)
```

### Low level API

These definitions allow you to communicate with AttestationStation, but are not as easy to use as the high level API.

#### `ATTESTATION_STATION_ADDRESS`


The deployment address for AttestationStation currently deployed with create2 on Optimism and Optimism Goerli `0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77`.

```typescript
import { ATTESTATION_STATION_ADDRESS } from '@eth-optimism/atst'
```

#### `abi`

The abi of the AttestationStation contract

```typescript
import { abi } from '@eth-optimism/atst'
```

#### `createKey`


`createKey` hashes keys longer than 31 bytes, because the atst key size is limited to 32 bytes.

```typescript
const key = await createKey(
  'i.am.a.key.much.longer.than.32.bytes.long'
)
```

createKey will keep the key as is if it is shorter than 32 bytes and otherwise run it through keccak256.


#### `parseAddress`

Turn bytes into an address.

**Note:** `readAttestation` and `readAttestations` already do this for you.
This is only needed if talking to the contracts directly, or through a different library.

#### `parseBool`

Turn bytes into a boolean value.

**Note:** `readAttestation` and `readAttestations` already do this for you.
This is only needed if talking to the contracts directly, or through a different library.


#### `parseNumber`

Turn bytes into a number.

**Note:** `readAttestation` and `readAttestations` already do this for you.
This is only needed if talking to the contracts directly, or through a different library.


#### `parseString`

Turn bytes into a string.

**Note:** `readAttestation` and `readAttestations` already do this for you.
This is only needed if talking to the contracts directly, or through a different library.


#### `stringifyAttestationBytes`

Stringifys an attestation into raw bytes.

```typescript
const stringAttestation = stringifyAttestationBytes('hello world')
const numberAttestation = stringifyAttestationBytes(500)
const hexAttestation = stringifyAttestationBytes('0x1')
const bigNumberAttestation = stringifyAttestationBytes(
  BigNumber.from('9999999999999999999999999')
)
```

**Note:** `writeAttestation` already does this for you so this is only needed if using a library other than `atst`.

### React API

For react hooks we recomend using the [wagmi cli](https://wagmi.sh/cli/getting-started) with the [etherscan plugin](https://wagmi.sh/cli/plugins/etherscan) and [react plugin](https://wagmi.sh/cli/plugins/react) to automatically generate react hooks around AttestationStation.

Use `createKey` and `createValue` to convert your raw keys and values into bytes that can be used in AttestationStation contract calls.
Use `parseString`, `parseBool`, `parseAddress` and `parseNumber` to convert values returned by AttestationStation to their correct data type.

For convenience we also [export the hooks here](../src/react.ts):
- `useAttestationStationAttestation` - Reads attestations with useContractRead
- `useAttestationStationVersion` - Reads attestation version
- `useAttestationStationAttest` - Wraps useContractWrite with AttestationStation abi calling attest
- `usePrepareAttestationStationAttest` - Wraps usePrepare with AttestationStation abi calling attest
- `useAttestationStationAttestationCreatedEvent` - Wraps useContractEvents for Created events



## Tutorial

- [General atst tutorial](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/ecosystem/attestation-station).
- [React atst starter](https://github.com/ethereum-optimism/optimism-starter). 

