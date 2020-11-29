# @eth-optimism/ovm-toolchain

`@eth-optimism/ovm-toolchain` provides "OVM-ified" wrappers or plugins for common Ethereum developer tools. Currently, this package directly or indirectly enables OVM execution within the following tools:

* [buidler](https://buidler.dev)
* [waffle](https://ethereum-waffle.readthedocs.io/en/latest/)
* [ganache](https://github.com/trufflesuite/ganache-core)
* [ethers](https://docs.ethers.io)

## Usage

### ganache

`ovm-toolchain` exports a `ganache` object which behaves identically to the one exported by [`ganache-core`](https://github.com/trufflesuite/ganache-core). However, we hijack the `ganache` instance such that the resulting `provider` object is backed by our own [`ethereumjs-vm` fork](https://github.com/ethereum-optimism/ethereumjs-vm) instead of the canonical version.

Import our `ganache` object as follows:

```typescript
import { ganache } from '@eth-optimism/ovm-toolchain'

const provider = ganache.provider(options) // Same options as `ganache-core`.
```

Please refer to the [`ganache-core` README](https://github.com/trufflesuite/ganache-core/blob/develop/README.md) for information about using and configuring `ganache`.

### waffleV2/waffleV3

`ovm-toolchain` exports two `waffle` objects, `waffleV2` and `waffleV3`, one for each major version of `waffle`. Each object has a single field (`MockProvider`) that can replace the `MockProvider` import from `ethereum-waffle`.

Import these objects as follows:

```typescript
import { waffleV2, waffleV3 } from '@eth-optimism/ovm-toolchain'

const providerV2 = new waffleV2.MockProvider(options) // Same options as V2 waffle MockProvider.
const providerV3 = new waffleV3.MockProvider({
    ganacheOptions: options,                          // Same options as V3 waffle MockProvider.
})
```

Alternatively:

```typescript
import { MockProvider } from '@eth-optimism/ovm-toolchain/build/src/waffle/waffle-v2'

const provider = new MockProvider(options)
```

Please refer to the [`waffle` docs](https://ethereum-waffle.readthedocs.io/en/latest/index.html) for more information.

### buidler

`ovm-toolchain` provides two `builder` plugins, `buidler-ovm-compiler` and `buidler-ovm-node`.

#### buidler-ovm-compiler
`buidler-ovm-compiler` allows users to specify a custom compiler `path` within `buidler.config.ts`. This makes it possible to compile your contracts with our [custom Solidity compiler](https://github.com/ethereum-optimism/solidity).

Import `buidler-ovm-compiler` as follows:

```typescript
// buidler.config.ts

import '@eth-optimism/ovm-toolchain/build/src/buidler-plugins/buidler-ovm-compiler'

const config = {
  solc: {
    path: '@eth-optimism/solc',
  },
}

export default config
```

#### buidler-ovm-node
`buidler-ovm-node` performs a hijack similar to the one performed for `ganache` in order to replace the VM object with our own custom `ethereumjs-vm` fork. Add `useOvm` to your buidler config object to enable OVM execution.

Import `buidler-ovm-node` as follows:

```typescript
// buidler.config.ts

import '@eth-optimism/ovm-toolchain/build/src/buidler-plugins/buidler-ovm-node'

const config = {
  useOvm: true,
}

export default config
```