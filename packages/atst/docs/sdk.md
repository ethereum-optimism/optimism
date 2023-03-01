# atst sdk docs

Typescript sdk for interacting with the ATST based on [@wagmi/core](https://wagmi.sh/core/getting-started)

TODO add a table of contents like [zod](https://github.com/colinhacks/zod/blob/master/README.md)

## Installation

Install atst and it's peer dependencies.

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

**Note** as ethers v6 is not yet stable we only support ethers v5 at this time

## Basic usage

Note: all functions are fully tested. The tests are a great example to see usage examples.

### Basic Setup

ATST uses `@wagmi/core` under the hood. See their documentation for more information.

```typescript
import { connect, createClient } from '@wagmi/core'
import { providers, Wallet } from 'ethers'

const provider = new providers.JsonRpcProvider({
  url: parsedOptions.rpcUrl,
  headers: {
    'User-Agent': '@eth-optimism/atst',
  },
})

createClient({
  provider,
})
```

### Reading an attestation

To read an attestation use `readString`, `readAddress`, `readNumber`, `readBool`

Use `readAttestationString` with `bytes` passed in for the types parameter to read raw bytes

Here is an example of reading an attestation used by the optimist nft

```typescript
import { readAttestationString } from '@eth-optimism/atst'

const creator = '0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3'
const about = '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5'
const key = 'optimist.base-uri'

const str = await readAttestationString(creator, about, key)

console.log(attestation) // https://assets.optimism.io/4a609661-6774-441f-9fdb-453fdbb89931-bucket/optimist-nft/attributes
```

### Reading multiple Attestations

If reading more than one attestation you can use readAttestations to read them with multicall

### Writing an attestation

To write to an attestation you must [connect](https://wagmi.sh/core/connectors/metaMask) your wagmi client if not already connected. If using Node.js use the [mock connector](https://wagmi.sh/core/connectors/mock)

```typescript
import { prepareWriteAttestation, writeAttestation } from '@eth-optimism/sdk'

const preparedTx = await prepareWriteAttestation(about, key, 'hello world')

console.log(preparedTx.gasLimit)

await writeAttestation(preparedTx)
```

## API

### ATTESTATION_STATION_ADDRESS

The deterministic deployment address for the attestation station currently deployed with create2 on Optimism and Optimism Goerli `0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77`

```typescript
import { ATTESTATION_STATION_ADDRESS } from '@eth-optimism/atst'
```

### abi

The abi of the attestation station

```typescript
import { abi } from '@eth-optimism/atst'
```

### readAttestation

[Reads](https://wagmi.sh/core/actions/readContract) and parses an attestation based on it's data type.

```typescript
const attestation = await readAttestation(
  /**
   * Address: The creator of the attestation
   */
  creator,
  /**
   * Address: The about topic of the attestation
   */
  about,
  /**
   * string: The key of the attestation
   */
  key,
  /**
   * 'string' | 'bytes' | 'number' | 'bool' | 'address'
   * The data type of the attestation
   * @defaults defaults to 'string'
   */
  dataType,
  /**
   * Address: the contract address of the attestation station
   * @defaults defaults to the create2 address
   */
  contractAddress
)
```

`Return Value` The data returned from invoking the contract method.

### readAttestations

Similar to read attestation but reads multiple attestations at once. Pass in a variadic amount of attestations to read.

```typescript
const attestation = await readAttestations({
  /**
   * Address: The creator of the attestation
   */
  creator,
  /**
   * Address: The about topic of the attestation
   */
  about,
  /**
   * string: The key of the attestation
   */
  key,
  /**
   * 'string' | 'bytes' | 'number' | 'bool' | 'address'
   * The data type of the attestation
   * @defaults defaults to 'string'
   */
  dataType,
  /**
   * Address: the contract address of the attestation station
   * @defaults defaults to the create2 address
   */
  contractAddress,
  /**
   * Boolean: Whether to allow some of the calls to fail
   * Defaults to false
   */
  allowFailures,
})
```

### Parsing bytes

These utilities for parsing bytes are provided:

`parseAddress`
`parseNumber`
`parseBool`
`parseString`

Note: `readAttestation` and `readAttestations` already parse the bytes so this is only necessary if reading attestations directly from chain instead of through this utility

```typescript
const attestation = parseAttestationBytes(
  /**
   * HexString: The raw bytes returned from reading an attestation
   */
  bytes,
  /**
   * 'string' | 'bytes' | 'number' | 'bool' | 'address'
   * The data type of the attestation
   * @defaults defaults to 'string'
   */
  dataType
)
```

### attestation keys

Attestation keys are limited to 32 bytes. To support keys longer than 32 bytes, you can use the `encodeRawKey` function

```typescript
const key = await encodeRawKey(
  about,
  key,
  'i.am.a.key.much.longer.than.32.bytes.long'
)
await writeAttestation(preparedTx)
```

encodeRawKey will keep the key as is if it is shorter than 32 bytes and otherwise run it through kekkak256

### prepareWriteAttestation

[Prepares](https://wagmi.sh/core/actions/prepareWriteContract) an attestation to be written.

```typescript
const preparedTx = await prepareWriteAttestation(about, key, 'hello world')
console.log(preparedTx.gasFee)
```

### stringifyAttestationBytes

Stringifys an attestation into raw bytes.

Note: `writeAttestation` already does this for you so this is only needed if using a library other than the attestation station

```typescript
const stringAttestatoin = stringifyAttestationBytes('hello world')
const numberAttestation = stringifyAttestationBytes(500)
const hexAttestation = stringifyAttestationBytes('0x1')
const bigNumberAttestation = stringifyAttestationBytes(
  BigNumber.from('9999999999999999999999999')
)
```

### writeAttestation

[Writes the prepared tx](https://wagmi.sh/core/actions/writeContract)

```typescript
const preparedTx = await prepareWriteAttestation(about, key, 'hello world')
await writeAttestation(preparedTx)
```

### getEvents

To getEvents use getEvents with a provider and any filters to filter the event

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

Set key, about, creator, or value to `null` to not include that filter

## Tutorial

For a tutorial on using the attestation station in general, see out tutorial as well as other Optimism related tutorials in our [optimism-tutorial](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/ecosystem/attestation-station#key-values) repo
