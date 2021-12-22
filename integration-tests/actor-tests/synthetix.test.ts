import { BigNumber, Contract, utils, Wallet, ContractFactory } from 'ethers'
import { actor, run, setupActor, setupRun } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'
import * as path from 'path'

interface Context {
  contracts: { [name: string]: Contract }
  wallet: Wallet
}

actor('Synthetix Trader', () => {
  let env: OptimismEnv

  setupActor(async () => {
    env = await OptimismEnv.new()

    // TODO: how to know what the addresses are...
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom().connect(env.l2Provider)

    await env.l2Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('0.1'),
    })

    return {
      contracts: {},
      wallet,
    }
  })

  run(async (b, ctx: Context) => {
    await b.bench('swap', async () => {
      let a = ctx
    })
  })
})
