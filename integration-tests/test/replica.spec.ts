import { OptimismEnv } from './shared/env'
import {
  defaultTransactionFactory,
  gasPriceForL2,
  sleep,
  isLiveNetwork,
} from './shared/utils'
import { expect } from 'chai'
import { TransactionReceipt } from '@ethersproject/abstract-provider'

describe('Replica Tests', () => {
  let env: OptimismEnv

  before(async () => {
    env = await OptimismEnv.new()
  })

  describe('Matching blocks', () => {
    if (isLiveNetwork()) {
      console.log('Skipping replica tests on live network')
      return
    }

    it('should sync a transaction', async () => {
      const tx = defaultTransactionFactory()
      tx.gasPrice = await gasPriceForL2(env)
      const result = await env.l2Wallet.sendTransaction(tx)

      let receipt: TransactionReceipt
      while (!receipt) {
        receipt = await env.replicaProvider.getTransactionReceipt(result.hash)
        await sleep(200)
      }

      const sequencerBlock = (await env.l2Provider.getBlock(
        result.blockNumber
      )) as any

      const replicaBlock = (await env.replicaProvider.getBlock(
        result.blockNumber
      )) as any

      expect(sequencerBlock.stateRoot).to.deep.eq(replicaBlock.stateRoot)
      expect(sequencerBlock.hash).to.deep.eq(replicaBlock.hash)
    })

    it('sync an unprotected tx (eip155)', async () => {
      const tx = {
        ...defaultTransactionFactory(),
        nonce: await env.l2Wallet.getTransactionCount(),
        gasPrice: await gasPriceForL2(env),
        chainId: null, // Disables EIP155 transaction signing.
      }
      const signed = await env.l2Wallet.signTransaction(tx)
      const result = await env.l2Provider.sendTransaction(signed)

      let receipt: TransactionReceipt
      while (!receipt) {
        receipt = await env.replicaProvider.getTransactionReceipt(result.hash)
        await sleep(200)
      }

      const sequencerBlock = (await env.l2Provider.getBlock(
        result.blockNumber
      )) as any

      const replicaBlock = (await env.replicaProvider.getBlock(
        result.blockNumber
      )) as any

      expect(sequencerBlock.stateRoot).to.deep.eq(replicaBlock.stateRoot)
      expect(sequencerBlock.hash).to.deep.eq(replicaBlock.hash)
    })
  })
})
