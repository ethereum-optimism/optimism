# Optimism Provider

The `OptimismProvider` extends the ethers.js `JsonRpcProvider` and
implements all of the same methods. It will submit transactions
to the Optimism Sequencer and needs a `Web3Provider` based provider
to manage keys for any transaction signing.

## Usage

```js
import { OptimismProvider } from '@eth-optimism/provider'
import { Web3Provider } from '@ethersproject/providers'

// Uses a Web3Provider to manage keys, pass in `window.ethereum` or
// another key management backend.
const web3 = new Web3Provider()

// Accepts either a URL or a network name (main, kovan)
const provider = new OptimismProvider('http://localhost:8545', web3)
```

## Goerli Testnet

To connect to the Goerli testnet:

```js
const provider = new OptimismProvider('goerli')
```
