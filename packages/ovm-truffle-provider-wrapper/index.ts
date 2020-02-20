module.exports = (provider: any) => {
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
