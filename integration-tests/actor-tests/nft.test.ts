import { utils, Wallet, Contract, ContractFactory } from 'ethers'
import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import ERC721 from '../artifacts/contracts/NFT.sol/NFT.json'
import { expect } from 'chai'

interface Context {
  wallet: Wallet
  contract: Contract
}

actor('NFT claimer', () => {
  let env: OptimismEnv

  let contract: Contract

  setupActor(async () => {
    env = await OptimismEnv.new()

    const factory = new ContractFactory(
      ERC721.abi,
      ERC721.bytecode,
      env.l2Wallet
    )
    contract = await factory.deploy()
    await contract.deployed()
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom().connect(env.l2Wallet.provider)
    await env.l2Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('0.01'),
    })
    return {
      wallet,
      contract: contract.connect(wallet),
    }
  })

  run(async (b, ctx: Context) => {
    let receipt: any
    await b.bench('mint', async () => {
      const tx = await ctx.contract.give()
      receipt = await tx.wait()
    })
    expect(receipt.events[0].event).to.equal('Transfer')
    expect(receipt.events[0].args[1]).to.equal(ctx.wallet.address)
  })
})
