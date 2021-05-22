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
  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
  })

  describe('ERC20 interactions', () => {
    const initialAmount = 1000
    const tokenName = 'OVM Test'
    const tokenDecimals = 8
    const TokenSymbol = 'OVM'

    let other: Wallet
    let Factory__ERC20: ContractFactory
    let ERC20: Contract

    before(async () => {
      const env = await OptimismEnv.new()
      wallet = env.l2Wallet
      other = Wallet.createRandom().connect(ethers.provider)
      Factory__ERC20 = await ethers.getContractFactory('ERC20', wallet)
    })

    // TODO(annieke): this currently brings down the sequencer too ugh
    // afterEach(async () => {
    //   verifier.stop('verifier')
    // })

    it('should sync ERC20 deployment and transfer', async () => {
      const preTxTotalEl = (await env.ctc.getTotalElements()) as BigNumber
      ERC20 = await Factory__ERC20.deploy(
        initialAmount,
        tokenName,
        tokenDecimals,
        TokenSymbol
      )

      const transfer = await ERC20.transfer(other.address, 100)
      await transfer.wait()

      // Wait for batch submission to happen by watching the CTC
      let newTotalEl = (await env.ctc.getTotalElements()) as BigNumber
      while (preTxTotalEl.eq(newTotalEl)) {
        await sleep(500)
        console.log(
          `still equal`,
          preTxTotalEl.toNumber(),
          newTotalEl.toNumber()
        )
        newTotalEl = (await env.ctc.getTotalElements()) as BigNumber
      }
      console.log(preTxTotalEl.toNumber())
      console.log(newTotalEl.toNumber())

      expect(newTotalEl.gt(preTxTotalEl))

      const latestSequencerBlock = (await provider.getBlock('latest')) as any
      console.log(latestSequencerBlock)

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
      let latestVerifierBlock = (await verifierProvider.getBlock(
        'latest'
      )) as any
      while (latestVerifierBlock.number < latestSequencerBlock.number) {
        await sleep(500)
        latestVerifierBlock = (await verifierProvider.getBlock('latest')) as any
      }

      expect(latestVerifierBlock.stateRoot).to.eq(
        latestSequencerBlock.stateRoot
      )
    })
  })
})
