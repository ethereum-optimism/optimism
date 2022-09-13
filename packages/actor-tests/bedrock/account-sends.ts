import { Wallet } from 'ethers'
import { expect } from 'chai'

import { actor, setupActor, run, setupRun } from '../lib/convenience'
import { devWalletsL2, l2Provider } from './utils'
import { Faucet } from '../lib/faucet'

interface Context {
  wallet: Wallet
}

actor('Sender', () => {
  let destWallet: Wallet

  setupActor(async () => {
    const devWallets = devWalletsL2()
    destWallet = devWallets[0]
  })

  setupRun(async () => {
    const faucet = new Faucet(process.env.FAUCET_URL, l2Provider)
    const wallet = Wallet.createRandom().connect(l2Provider)
    await faucet.drip(wallet.address)
    return {
      wallet,
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
