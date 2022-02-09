import { utils, Wallet, BigNumber } from 'ethers'
import { expect } from 'chai'

import { setupActor, setupRun, actor, run } from './lib/convenience'
import { OptimismEnv } from '../test/shared/env'

interface BenchContext {
  l1Wallet: Wallet
  l2Wallet: Wallet
}

const DEFAULT_TEST_GAS_L1 = 330_000
const DEFAULT_TEST_GAS_L2 = 1_300_000

actor('Funds depositor', () => {
  let env: OptimismEnv

  setupActor(async () => {
    env = await OptimismEnv.new()
  })

  setupRun(async () => {
    const wallet = Wallet.createRandom()
    await env.l1Wallet.sendTransaction({
      to: wallet.address,
      value: utils.parseEther('0.01'),
    })
    return {
      l1Wallet: wallet.connect(env.l1Wallet.provider),
      l2Wallet: wallet.connect(env.l2Wallet.provider),
    }
  })

  run(async (b, ctx: BenchContext) => {
    const { l1Wallet, l2Wallet } = ctx
    const balBefore = await l2Wallet.getBalance()
    await b.bench('deposit', async () => {
      await env.waitForXDomainTransaction(
        env.l1Bridge
          .connect(l1Wallet)
          .depositETH(DEFAULT_TEST_GAS_L2, '0xFFFF', {
            value: 0x42,
            gasLimit: DEFAULT_TEST_GAS_L1,
          })
      )
    })
    expect((await l2Wallet.getBalance()).sub(balBefore)).to.deep.equal(
      BigNumber.from(0x42)
    )
  })
})
