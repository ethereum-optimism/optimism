import { Wallet, ContractFactory } from 'ethers'

import { actor, setupActor, run } from '../lib/convenience'
import { devWalletsL2 } from './utils'
import * as ERC20 from './contracts/ERC20.json'

actor('Deployer', () => {
  let wallets: Wallet[]

  setupActor(async () => {
    wallets = devWalletsL2()
  })

  run(async (b, ctx, logger) => {
    const sender = wallets[Math.floor(Math.random() * wallets.length)]
    const contract = new ContractFactory(ERC20.abi, ERC20.bytecode).connect(
      sender
    )
    logger.log(`Deploying contract with ${sender.address}.`)
    const deployment = await contract.deploy(
      Math.floor(1_000_000 * Math.random()),
      'Test Token',
      18,
      'OP'
    )
    logger.log(
      `Awaiting receipt for deployment tx ${deployment.deployTransaction.hash}.`
    )
    await deployment.deployed()
    const receipt = await sender.provider.getTransactionReceipt(
      deployment.deployTransaction.hash
    )
    logger.log(`Deployment completed in block ${receipt.blockNumber}.`)
  })
})
