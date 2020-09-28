// tslint:disable
const BN = require('bn.js')
import * as eGanache from 'ganache-core'

/* Internal Imports */
import Common from 'ethjs-common-v1'
import { to } from './utils/to'
import { makeOVM } from '../utils/ovm'

function createVMFromStateTrie(
  state: any,
  activatePrecompiles: any,
  ovm?: any
): any {
  const self = this
  const common = Common.forCustomChain(
    'mainnet', // TODO needs to match chain id
    {
      name: 'ganache',
      networkId: self.options.network_id || self.forkVersion,
      chainId: self.options._chainId,
      comment: 'Local test network',
      bootstrapNodes: [],
    },
    self.options.hardfork
  )

  const vm = makeOVM({
    evmOpts: {
      state: state,
      common,
      blockchain: {
        // EthereumJS VM needs a blockchain object in order to get block information.
        // When calling getBlock() it will pass a number that's of a Buffer type.
        // Unfortunately, it uses a 64-character buffer (when converted to hex) to
        // represent block numbers as well as block hashes. Since it's very unlikely
        // any block number will get higher than the maximum safe Javascript integer,
        // we can convert this buffer to a number ahead of time before calling our
        // own getBlock(). If the conversion succeeds, we have a block number.
        // If it doesn't, we have a block hash. (Note: Our implementation accepts both.)
        getBlock: function(number, done) {
          try {
            number = typeof to.number(number)
          } catch (e) {
            // Do nothing; must be a block hash.
          }

          self.getBlock(number, done)
        },
      },
      activatePrecompiles: activatePrecompiles || false,
      allowUnlimitedContractSize: self.options.allowUnlimitedContractSize,
    },
    ovmOpts: ovm
      ? {
          initialized:
            ovm.contracts &&
            ovm.contracts.ovmExecutionManager.address.length !== 0,
          contracts: ovm.contracts,
        }
      : {},
  })

  if (self.options.debug === true) {
    // log executed opcodes, including args as hex
    vm.on('step', function(info) {
      var name = info.opcode.name
      var argsNum = info.opcode.in
      if (argsNum) {
        var args = info.stack
          .slice(-argsNum)
          .map((arg) => to.hex(arg))
          .join(' ')

        self.logger.log(`${name} ${args}`)
      } else {
        self.logger.log(name)
      }
    })
  }

  return vm
}

const wrap = (provider: any, opts: any = {}) => {
  const gasLimit = opts.gasLimit || 100_000_000
  const blockchain = provider.engine.manager.state.blockchain

  blockchain.blockGasLimit = '0x' + new BN(gasLimit).toString('hex')

  let ovm: any
  blockchain.createVMFromStateTrie = function(
    state: any,
    activatePrecompiles: any
  ) {
    const vm = createVMFromStateTrie.call(this, state, activatePrecompiles, ovm)

    if (!ovm) {
      ovm = vm
    }

    return vm
  }

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
