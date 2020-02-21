import {execSync, spawn} from 'child_process'

const startLocalNode = () => {
  const runText = `(async function(){ const {runFullnode} = require('@eth-optimism/rollup-full-node');runFullnode();})();`

  // Assumes this process was kicked off with node, but that's true for `truffle test` and `yarn ...`
  const sub = spawn(process.argv[0], [`-e`, `${runText}`])

  sub.on('error', (e) => {
    console.error(`Local server could not be started. Error details: ${e.message}, Stack: ${e.stack}`)
  })

  // This reliably stops the local server when the current process exits.
  process.on('exit', () => {
    try {sub.kill()} catch (e) {/*swallow any errors */}
  });

  // TODO: This is hacky. If host / port become configurable, spawn a node process to ping it or something better.
  execSync(`sleep 3`)
}

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
const wrapProviderAndStartLocalNode = (provider: any) => {
  if (!nodeStarted) {
    nodeStarted = true
    startLocalNode()
  }

  return wrapProvider(provider)
}

module.exports = {
  wrapProvider,
  wrapProviderAndStartLocalNode
}
