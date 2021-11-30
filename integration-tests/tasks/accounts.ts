import dotenv from 'dotenv'

dotenv.config()

import { Wallet, utils } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import {
  fundUser,
  getAddressManager,
  getL1Bridge,
  l1Provider,
  l2Provider,
} from '../test/shared/utils'
import { writeStderr } from './util'
import { initWatcher } from '../test/shared/watcher-utils'

task('fund-l1')
  .addParam('recipient', 'Recipient of the deposit on L2.', null, types.string)
  .addParam('amount', 'Amount to deposit, in Ether.', null, types.string)
  .setAction(async (args) => {
    const l1Wallet = new Wallet(process.env.PRIVATE_KEY).connect(l1Provider)
    writeStderr(`Transferring ${args.amount} ETH to ${args.recipient}...`)
    const value = utils.parseEther(args.amount)
    await l1Wallet.sendTransaction({
      to: args.recipient,
      value,
    })
    writeStderr('Done.')
  })

task('deposit-l2')
  .addParam('recipient', 'Recipient of the deposit on L2.', null, types.string)
  .addParam('amount', 'Amount to deposit, in Ether.', null, types.string)
  .setAction(async (args) => {
    const l1Wallet = new Wallet(process.env.PRIVATE_KEY).connect(l1Provider)
    const l2Wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    const addressManager = getAddressManager(l1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const l1Bridge = await getL1Bridge(l1Wallet, addressManager)

    const value = utils.parseEther(args.amount)
    writeStderr(`Depositing ${args.amount} ETH onto L2...`)
    await fundUser(watcher, l1Bridge, value)
    writeStderr(`Transferring funds to ${args.recipient}...`)
    await l2Wallet.sendTransaction({
      to: args.recipient,
      value,
    })
    writeStderr('Done.')
  })

task('balance-l2')
  .addParam('address', 'Address to get the balance for.', null, types.string)
  .setAction(async (args) => {
    const balance = await l2Provider.getBalance(args.address)
    console.log(utils.formatEther(balance))
  })
