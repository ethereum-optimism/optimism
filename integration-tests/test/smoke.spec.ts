import { Wallet, BigNumber } from 'ethers'
import { OptimismEnv } from './shared/env'
import { expect } from 'chai'
import { gasPriceForL2 } from './shared/utils'

describe('Transfers', () => {
  let env: OptimismEnv

  before(async () => {
    env = await OptimismEnv.new()
  })

  it('should support transfers', async () => {
    const amount = 42000
    const other = Wallet.createRandom().connect(env.l2Wallet.provider)
    const tx = await env.l2Wallet.sendTransaction({
      to: other.address,
      value: amount,
      gasPrice: await gasPriceForL2(env),
    })
    await tx.wait()
    expect(await other.getBalance()).to.deep.equal(BigNumber.from(amount))
  })
})
