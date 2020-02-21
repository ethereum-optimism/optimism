import { execSync, spawn } from 'child_process'

/**
 * Starts a local OVM node process for testing. It will be killed when the current process terminates.
 */
const startLocalNode = () => {
  const runText = `(async function(){ const {runFullnode} = require('@eth-optimism/rollup-full-node');runFullnode();})();`

  // Assumes this process was kicked off with node, but that's true for `truffle test` and `yarn ...`
  const sub = spawn(process.argv[0], [`-e`, `${runText}`], {
    stdio: ['ignore', 'ignore', 2],
  })

  sub.on('error', (e) => {
    // tslint:disable-next-line:no-console
    console.error(
      `Local server could not be started. Error details: ${e.message}, Stack: ${e.stack}`
    )
  })

  // This reliably stops the local server when the current process exits.
  process.on('exit', () => {
    try {
      sub.kill()
    } catch (e) {
      /*swallow any errors */
    }
  })

  // TODO: This is hacky. If host / port become configurable, spawn a node process to ping it or something better.
  execSync(`sleep 3`)
}

/**
 * Wraps the provided Truffle provider so it will work with the OVM.
 * @returns The wrapped provider.
 */
const wrapProvider = (provider: any) => {
  if (typeof provider !== 'object' || !provider['sendAsync']) {
    throw Error(
      'Invalid provider. Exepcted provider to conform to Truffle provider interface!'
    )
  }

  const chainId = process.env.OVM_CHAIN_ID || 108
  const sendAsync = provider.sendAsync

  provider.sendAsync = function(...args) {
    if (args[0].method === 'eth_sendTransaction') {
      // To properly set chainID for all transactions.
      args[0].params[0].chainId = chainId
    }
    sendAsync.apply(this, args)
  }
  return provider
}

let nodeStarted = false
/**
 * Wraps the provided Truffle provider so it will work with the OVM and starts a
 * local OVM node for the duration of the current process.
 * @returns The wrapped provider.
 */
const wrapProviderAndStartLocalNode = (provider: any) => {
  if (!nodeStarted) {
    nodeStarted = true
    startLocalNode()
  }

  return wrapProvider(provider)
}

module.exports = {
  wrapProvider,
  wrapProviderAndStartLocalNode,
}
