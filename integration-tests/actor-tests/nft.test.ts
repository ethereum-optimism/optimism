import { utils, Wallet, Contract } from 'ethers'
import { expect } from 'chai'

import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import ERC721 from '../artifacts/contracts/NFT.sol/NFT.json'

interface Context {
  wallet: Wallet
  contract: Contract
}

actor('NFT claimer', () => {
  let env: OptimismEnv

  let contract: Contract

  setupActor(async () => {
    env = await OptimismEnv.new()
    contract = new Contract(process.env.ERC_721_ADDRESS, ERC721.abi)
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
