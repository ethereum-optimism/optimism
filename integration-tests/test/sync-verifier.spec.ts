import chai, { expect } from 'chai'
import { Wallet, BigNumber, Contract } from 'ethers'

import { OptimismEnv } from './shared/env'

describe('Syncing a verifier', () => {
  let env: OptimismEnv
  let wallet: Wallet

  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
  })

  it('should sync ERC20 deployment and transfer', async () => {
    const tx = ''
    const result = await wallet.sendTransaction(tx)

  })

})