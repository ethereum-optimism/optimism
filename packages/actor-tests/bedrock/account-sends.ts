import { Wallet, utils } from 'ethers'
import { expect } from 'chai'

import { actor, setupActor, run, setupRun } from '../lib/convenience'
import { devWalletsL2 } from './utils'

interface Context {
  wallet: Wallet
}

actor('Sender', () => {
  let sourceWallet: Wallet

  let destWallet: Wallet

  setupActor(async () => {
    const devWallets = devWalletsL2()
    sourceWallet = devWallets[0]
    destWallet = devWallets[1]
  })

  setupRun(async () => {
    const newWallet = Wallet.createRandom().connect(sourceWallet.provider)
    const tx = await sourceWallet.sendTransaction({
      to: newWallet.address,
      value: utils.parseEther('0.1'),
    })
    await tx.wait()

    return {
      wallet: newWallet,
    }
  })

  run(async (b, ctx: Context, logger) => {
    const { wallet } = ctx
    logger.log(`Sending funds to ${destWallet.address}.`)
    const tx = await wallet.sendTransaction({
      to: destWallet.address,
      value: 0x42,
    })
    logger.log(`Awaiting receipt for send tx ${tx.hash}.`)
    const receipt = await tx.wait()
    expect(receipt.status).to.eq(1)
    logger.log(`Send completed in block ${receipt.blockNumber}.`)
  })
})
