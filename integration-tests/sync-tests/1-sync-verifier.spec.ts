import chai, { expect } from 'chai'
import { Wallet, BigNumber, providers } from 'ethers'
import { injectL2Context } from '@eth-optimism/core-utils'

import {
  sleep,
  l2Provider,
  verifierProvider,
  waitForL2Geth,
} from '../test/shared/utils'
import { OptimismEnv } from '../test/shared/env'
import { DockerComposeNetwork } from '../test/shared/docker-compose'

describe('Syncing a verifier', () => {
  let env: OptimismEnv
  let wallet: Wallet
  let verifier: DockerComposeNetwork
  let provider: providers.JsonRpcProvider

  const sequencerProvider = injectL2Context(l2Provider)

  /* Helper functions */

  const waitForBatchSubmission = async (
    totalElementsBefore: BigNumber
  ): Promise<BigNumber> => {
    // Wait for batch submission to happen by watching the CTC
    let totalElementsAfter = (await env.ctc.getTotalElements()) as BigNumber
    while (totalElementsBefore.eq(totalElementsAfter)) {
      await sleep(500)
      totalElementsAfter = (await env.ctc.getTotalElements()) as BigNumber
    }
    return totalElementsAfter
  }

  const startVerifier = async () => {
    // Bring up new verifier
    verifier = new DockerComposeNetwork(['verifier'])
    await verifier.up({ commandOptions: ['--scale', 'verifier=1'] })

    provider = await waitForL2Geth(verifierProvider)
  }

  const syncVerifier = async (sequencerBlockNumber: number) => {
    // Wait until verifier has caught up to the sequencer
    let latestVerifierBlock = (await provider.getBlock('latest')) as any
    while (latestVerifierBlock.number < sequencerBlockNumber) {
      await sleep(500)
      latestVerifierBlock = (await provider.getBlock('latest')) as any
    }

    return provider.getBlock(sequencerBlockNumber)
  }

  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
  })

  describe('Basic transactions', () => {
    after(async () => {
      await verifier.stop('verifier')
      await verifier.rm()
    })

    it('should sync dummy transaction', async () => {
      const totalElementsBefore = (await env.ctc.getTotalElements()) as BigNumber

      const tx = {
        to: '0x' + '1234'.repeat(10),
        gasLimit: 4000000,
        gasPrice: 0,
        data: '0x',
        value: 0,
      }
      const result = await wallet.sendTransaction(tx)
      await result.wait()

      const totalElementsAfter = await waitForBatchSubmission(
        totalElementsBefore
      )
      expect(totalElementsAfter.gt(totalElementsAfter))

      const latestSequencerBlock = (await sequencerProvider.getBlock(
        'latest'
      )) as any

      await startVerifier()

      const matchingVerifierBlock = (await syncVerifier(
        latestSequencerBlock.number
      )) as any

      expect(matchingVerifierBlock.stateRoot).to.eq(
        latestSequencerBlock.stateRoot
      )
    })

    it('should have matching block data', async () => {
      const sequencerTip = await sequencerProvider.getBlock('latest')
      const verifierTip = await provider.getBlock('latest')

      expect(sequencerTip.number).to.deep.eq(verifierTip.number)
      expect(sequencerTip.hash).to.deep.eq(verifierTip.hash)
    })
  })
})
