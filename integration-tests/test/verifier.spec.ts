/* Imports: External */
import { TransactionReceipt } from '@ethersproject/abstract-provider'
import { sleep } from '@eth-optimism/core-utils'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import {
  defaultTransactionFactory,
  gasPriceForL2,
  envConfig,
} from './shared/utils'

describe('Verifier Tests', () => {
  let env: OptimismEnv

  before(async function () {
    if (!envConfig.RUN_VERIFIER_TESTS) {
      this.skip()
      return
    }

    env = await OptimismEnv.new()
  })

  describe('Matching blocks', () => {
    it('should sync a transaction', async () => {
      const tx = defaultTransactionFactory()
      tx.gasPrice = await gasPriceForL2()
      const result = await env.l2Wallet.sendTransaction(tx)

      let receipt: TransactionReceipt
      while (!receipt) {
        receipt = await env.verifierProvider.getTransactionReceipt(result.hash)
        await sleep(200)
      }

      const sequencerBlock = (await env.l2Provider.getBlock(
        result.blockNumber
      )) as any

      const verifierBlock = (await env.verifierProvider.getBlock(
        result.blockNumber
      )) as any

      expect(sequencerBlock.stateRoot).to.deep.eq(verifierBlock.stateRoot)
      expect(sequencerBlock.hash).to.deep.eq(verifierBlock.hash)
    })

    it('sync an unprotected tx (eip155)', async () => {
      const tx = {
        ...defaultTransactionFactory(),
        nonce: await env.l2Wallet.getTransactionCount(),
        gasPrice: await gasPriceForL2(),
        chainId: null, // Disables EIP155 transaction signing.
      }
      const signed = await env.l2Wallet.signTransaction(tx)
      const result = await env.l2Provider.sendTransaction(signed)

      let receipt: TransactionReceipt
      while (!receipt) {
        receipt = await env.verifierProvider.getTransactionReceipt(result.hash)
        await sleep(200)
      }

      const sequencerBlock = (await env.l2Provider.getBlock(
        result.blockNumber
      )) as any

      const verifierBlock = (await env.verifierProvider.getBlock(
        result.blockNumber
      )) as any

      expect(sequencerBlock.stateRoot).to.deep.eq(verifierBlock.stateRoot)
      expect(sequencerBlock.hash).to.deep.eq(verifierBlock.hash)
    })

    it('should forward tx to sequencer', async () => {
      const tx = {
        ...defaultTransactionFactory(),
        nonce: await env.l2Wallet.getTransactionCount(),
        gasPrice: await gasPriceForL2(),
      }
      const signed = await env.l2Wallet.signTransaction(tx)
      const result = await env.verifierProvider.sendTransaction(signed)

      let receipt: TransactionReceipt
      while (!receipt) {
        receipt = await env.verifierProvider.getTransactionReceipt(result.hash)
        await sleep(200)
      }

      const sequencerBlock = (await env.l2Provider.getBlock(
        result.blockNumber
      )) as any

      const verifierBlock = (await env.verifierProvider.getBlock(
        result.blockNumber
      )) as any

      expect(sequencerBlock.stateRoot).to.deep.eq(verifierBlock.stateRoot)
      expect(sequencerBlock.hash).to.deep.eq(verifierBlock.hash)
    })
  })
})
