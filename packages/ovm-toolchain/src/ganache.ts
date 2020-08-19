import * as eGanache from 'ganache-core'
// tslint:disable-next-line
const VM = require('ethereumjs-vm').default

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
