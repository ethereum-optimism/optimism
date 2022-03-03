import { utils, Wallet, Contract } from 'ethers'
import { ethers } from 'hardhat'
import { expect } from 'chai'

import { actor, setupActor, run, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'

interface Context {
  wallet: Wallet
}

actor('Trie DoS accounts', () => {
  let env: OptimismEnv

  let contract: Contract

  setupActor(async () => {
    env = await OptimismEnv.new()

    const factory = await ethers.getContractFactory('StateDOS', env.l2Wallet)
    contract = await factory.deploy()
    await contract.deployed()
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom()
    await env.l2Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('1'),
    })
    return {
      wallet: wallet.connect(env.l2Wallet.provider),
    }
  })

  run(async (b, ctx: Context) => {
    await b.bench('DOS transactions', async () => {
      const tx = await contract.connect(ctx.wallet).attack({
        gasLimit: 9000000 + Math.floor(1000000 * Math.random()),
      })
      const receipt = await tx.wait()
      // make sure that this was an actual transaction in a block
      expect(receipt.blockNumber).to.be.gt(1)
      expect(receipt.gasUsed.gte(8000000)).to.be.true
    })
  })
})
