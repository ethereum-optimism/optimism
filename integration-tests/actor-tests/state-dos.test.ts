import { utils, Wallet, Contract, ContractFactory, BigNumber } from 'ethers'
import { actor, setupActor, run, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import StateDOS from '../artifacts/contracts/StateDOS.sol/StateDOS.json'
import { expect } from 'chai'

interface Context {
  wallet: Wallet
}

actor('Trie DoS accounts', () => {
  let env: OptimismEnv

  let contract: Contract

  setupActor(async () => {
    env = await OptimismEnv.new()

    const factory = new ContractFactory(
      StateDOS.abi,
      StateDOS.bytecode,
      env.l2Wallet
    )
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
        gasLimit: 10000000,
      })
      const receipt = await tx.wait()
      expect(receipt.gasUsed.gte(9970000)).to.be.true
    })
  })
})
