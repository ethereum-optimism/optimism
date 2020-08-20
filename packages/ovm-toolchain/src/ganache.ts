import * as eGanache from 'ganache-core'
// tslint:disable-next-line
const VM = require('ethereumjs-ovm').default

// tslint:disable-next-line:no-shadowed-variable
const wrap = (provider: any, opts: any) => {
  const blockchain = provider.engine.manager.state.blockchain
  let ovm: any

  const _original = blockchain.createVMFromStateTrie.bind(blockchain)
  // tslint:disable-next-line:only-arrow-functions
  blockchain.createVMFromStateTrie = function(
    state: any,
    activatePrecompiles: any
  ) {
    if (ovm === undefined) {
      const vm = _original(state, activatePrecompiles)
      ovm = new VM({
        ...vm.opts,
        stateManager: vm.stateManager,
        emGasLimit: opts.gasLimit || 100_000_000,
      })
      return ovm
    } else {
      return new VM({
        ...ovm.opts,
        state,
        stateManager: undefined,
        activatePrecompiles,
        emOpts: ovm._emOpts,
        initialized: ovm._initialized,
        contracts: ovm._contracts,
      })
    }
  }

  const _send = provider.send.bind(provider)
  const _wrappedSend = (payload: any, cb: any) => {
    if (payload.method === 'eth_getProof') {
      ovm
        .getEthTrieProof(payload.params[0], payload.params[1])
        .then((ethTrieProof: any) => {
          cb(null, {
            id: payload.id,
            jsonrpc: '2.0',
            result: ethTrieProof,
          })
        })
        .catch((err: any) => {
          cb(err, null)
        })
    } else if (payload.method === 'eth_getAccount') {
      ovm
        .getEthAccount(payload.params[0])
        .then((account: any) => {
          cb(null, {
            id: payload.id,
            jsonrpc: '2.0',
            result: account,
          })
        })
        .catch((err: any) => {
          cb(err, null)
        })
    } else {
      _send(payload, cb)
    }
  }

  provider.send = _wrappedSend
  provider.sendAsync = _wrappedSend

  return provider
}

const provider = (opts: any) => {
  const gProvider = (eGanache as any).provider(opts)
  return wrap(gProvider, opts)
}

const server = (opts: any) => {
  const gServer = (eGanache as any).server(opts)
  gServer.provider = wrap(gServer.provider, opts)
  return gServer
}

export const ganache = {
  provider,
  server,
}
