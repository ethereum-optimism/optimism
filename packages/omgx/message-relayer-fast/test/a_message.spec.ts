import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import { Direction } from './shared/watcher-utils'

import L1MessageJson from '../contracts/L1Message.json'
import L2MessageJson from '../contracts/L2Message.json'

import { OptimismEnv } from './shared/env'
import * as fs from 'fs'

describe('Fast Messenge Relayer Test', async () => {

  let Factory__L1Message: ContractFactory
  let Factory__L2Message: ContractFactory

  let L1Message: Contract
  let L2Message: Contract

  let env: OptimismEnv

  before(async () => {

    env = await OptimismEnv.new()

    Factory__L1Message = new ContractFactory(
      L1MessageJson.abi,
      L1MessageJson.bytecode,
      env.bobl1Wallet
    )

    Factory__L2Message = new ContractFactory(
      L2MessageJson.abi,
      L2MessageJson.bytecode,
      env.bobl2Wallet
    )

    L1Message = await Factory__L1Message.deploy(
      env.l1Messenger.address,
      env.l1MessengerFast.address
    )
    await L1Message.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1 Message deployed to:')} ${chalk.green(L1Message.address)}`)

    L2Message = await Factory__L2Message.deploy(
      env.l2Messenger.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2Message.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L2 Message deployed to:')} ${chalk.green(L2Message.address)}`)

    // Initialize L1 message
    const L1MessageTX = await L1Message.init(
      L2Message.address
    )
    await L1MessageTX.wait()
    console.log(`⭐️ ${chalk.blue('L1 Message initialized:')} ${chalk.green(L1MessageTX.hash)}`)

    // Initialize L2 message
    const L2MessageTX = await L2Message.init(
      L1Message.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2MessageTX.wait()
    console.log(`⭐️ ${chalk.blue('L2 Message initialized:')} ${chalk.green(L2MessageTX.hash)}`)

    })

    it('should send message from L1 to L2', async () => {
      await env.waitForXDomainTransaction(
        L1Message.sendMessageL1ToL2(),
        Direction.L1ToL2
      )
    })
    
    it('should QUICKLY send message from L2 to L1 using the fast relayer', async () => {
      await env.waitForXDomainTransactionFast(
        L2Message.sendMessageL2ToL1({ gasLimit: 800000, gasPrice: 0 }),
        Direction.L2ToL1
      )
    })

})
