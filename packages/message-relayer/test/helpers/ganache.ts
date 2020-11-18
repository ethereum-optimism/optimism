import * as eGanache from 'ganache-core'
import { Wallet } from 'ethers'
import { defaultAccounts } from 'ethereum-waffle'

import { fromHexString } from './buffer-utils'
import { getEthTrieProofInternal, EthTrieProof } from './trie-proof'

export const wallets = defaultAccounts.map((account) => {
  return new Wallet(account.secretKey)
})

const getEthTrieProof = async (
  vm: any,
  address: Buffer | string,
  slots: Array<Buffer | string> = []
): Promise<EthTrieProof> => {
  const addressBuf =
    typeof address === 'string' ? fromHexString(address) : address
  const slotsBuf: Buffer[] = slots.map(
    (slot): Buffer => {
      return typeof slot === 'string' ? fromHexString(slot) : slot
    }
  )

  return getEthTrieProofInternal(vm, addressBuf, slotsBuf)
}

const wrap = (provider: any) => {
  const blockchain = provider.engine.manager.state.blockchain

  const _send = provider.send.bind(provider)
  const _wrappedSend = (payload: any, cb: any) => {
    if (payload.method === 'eth_getProof') {
      getEthTrieProof(blockchain.vm, payload.params[0], payload.params[1])
        .then((ethTrieProof: any) => {
          cb(null, {
            id: payload.id,
            jsonrpc: '2.0',
            result: ethTrieProof,
          })
        })
        .catch((err: any) => {
          console.log(err)
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

const provider = (opts: any = {}) => {
  opts.accounts = defaultAccounts
  const gProvider = (eGanache as any).provider(opts)
  return wrap(gProvider)
}

const server = (opts: any = {}) => {
  opts.accounts = defaultAccounts
  const gServer = (eGanache as any).server(opts)
  gServer.provider = wrap(gServer.provider)
  return gServer
}

export const ganache = {
  provider,
  server,
}
