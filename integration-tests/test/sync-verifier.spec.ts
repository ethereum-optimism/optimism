import chai, { expect } from 'chai'
import { Wallet, BigNumber, Contract, ContractFactory, providers } from 'ethers'
import { ethers } from 'hardhat'
import { injectL2Context } from '@eth-optimism/core-utils'

import { sleep, l2Provider } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { DockerComposeNetwork } from './shared/docker-compose'

describe('Syncing a verifier', () => {
  let env: OptimismEnv
  let wallet: Wallet
  let verifier: DockerComposeNetwork

  const provider = injectL2Context(l2Provider)

  /* Helper functions */

  const waitForBatchSubmission = async (totalElementsBefore: BigNumber) => {
    // Wait for batch submission to happen by watching the CTC
    let totalElementsAfter = (await env.ctc.getTotalElements()) as BigNumber
    while (totalElementsBefore.eq(totalElementsAfter)) {
      await sleep(500)
      console.log(
        `still equal`,
        totalElementsBefore.toNumber(),
        totalElementsAfter.toNumber()
      )
      totalElementsAfter = (await env.ctc.getTotalElements()) as BigNumber
    }
    return totalElementsAfter
  }

  const startAndSyncVerifier = async (sequencerBlockNumber: number) => {
    // Bring up new verifier
    verifier = new DockerComposeNetwork(['verifier'])
    await verifier.up({ commandOptions: ['--scale', 'verifier=1'] })

    // Wait for verifier to be looping
    let logs = await verifier.logs()
    while (!logs.out.includes('Starting Sequencer Loop')) {
      console.log('Retrieving more logs')
      await sleep(500)
      logs = await verifier.logs()
    }

    const verifierProvider = injectL2Context(
      new providers.JsonRpcProvider('http://localhost:8547')
    )
    console.log(await verifierProvider.getBlock('latest'))

    // Wait until verifier has caught up to the sequencer
    let latestVerifierBlock = (await verifierProvider.getBlock('latest')) as any
    while (latestVerifierBlock.number < sequencerBlockNumber) {
      console.log('waiting for new verifier blocks')
      await sleep(500)
      latestVerifierBlock = (await verifierProvider.getBlock('latest')) as any
    }

    return latestVerifierBlock
  }

  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
  })

  describe('Basic transactions and ERC20s', () => {
    const initialAmount = 1000
    const tokenName = 'OVM Test'
    const tokenDecimals = 8
    const TokenSymbol = 'OVM'

    let other: Wallet
    let Factory__ERC20: ContractFactory
    let ERC20: Contract

    before(async () => {
      other = Wallet.createRandom().connect(ethers.provider)
      Factory__ERC20 = await ethers.getContractFactory('ERC20', wallet)
    })

    afterEach(async () => {
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

      const latestSequencerBlock = (await provider.getBlock('latest')) as any
      console.log(latestSequencerBlock)

      const latestVerifierBlock = await startAndSyncVerifier(
        latestSequencerBlock.number
      )

      expect(latestVerifierBlock.stateRoot).to.eq(
        latestSequencerBlock.stateRoot
      )
    })

    it('should sync ERC20 deployment and transfer', async () => {
      const totalElementsBefore = (await env.ctc.getTotalElements()) as BigNumber
      ERC20 = await Factory__ERC20.deploy(
        initialAmount,
        tokenName,
        tokenDecimals,
        TokenSymbol
      )

      const transfer = await ERC20.transfer(other.address, 100)
      await transfer.wait()

      const totalElementsAfter = await waitForBatchSubmission(
        totalElementsBefore
      )
      expect(totalElementsAfter.gt(totalElementsAfter))

      const latestSequencerBlock = (await provider.getBlock('latest')) as any
      console.log(latestSequencerBlock)

      const latestVerifierBlock = await startAndSyncVerifier(
        latestSequencerBlock.number
      )

      expect(latestVerifierBlock.stateRoot).to.eq(
        latestSequencerBlock.stateRoot
      )
    })
  })
})
