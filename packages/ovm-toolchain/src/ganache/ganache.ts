// tslint:disable-next-line
const BN = require('bn.js')
import * as eGanache from 'ganache-core'

/* Internal Imports */
import { makeOVM } from '../utils/ovm'

// tslint:disable-next-line:no-shadowed-variable
const wrap = (provider: any, opts: any = {}) => {
  const gasLimit = opts.gasLimit || 100_000_000
  const blockchain = provider.engine.manager.state.blockchain

  blockchain.blockGasLimit = '0x' + new BN(gasLimit).toString('hex')

  let ovm: any
  const _original = blockchain.createVMFromStateTrie.bind(blockchain)
  // tslint:disable-next-line:only-arrow-functions
  blockchain.createVMFromStateTrie = function(
    state: any,
    activatePrecompiles: any
  ) {
    if (ovm === undefined) {
      const vm = _original(state, activatePrecompiles)
      ovm = makeOVM({
        evmOpts: vm.opts,
        ovmOpts: {
          emGasLimit: gasLimit,
        },
      })

      return ovm
    } else {
      return makeOVM({
        evmOpts: {
          ...ovm.opts,
          state,
          stateManager: undefined,
          activatePrecompiles,
        },
        ovmOpts: {
          ...ovm.opts.ovmOpts,
          emGasLimit: ovm.emGasLimit,
          initialized: ovm.initialized,
          contracts: ovm.contracts,
        },
      })
    }
  }

  // tslint:disable-next-line:only-arrow-functions
  blockchain.estimateGas = function(tx: any, blockNumber: any, callback: any) {
    callback(null, {
      gasEstimate: new BN(gasLimit),
    })
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
