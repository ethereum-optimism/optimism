import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import { Direction, Relayer } from './shared/watcher-utils'

import L1MessageJson from '../artifacts/contracts/Message/L1Message.sol/L1Message.json'
import L2MessageJson from '../artifacts-ovm/contracts/Message/L2Message.sol/L2Message.json'

import { OptimismEnv } from './shared/env'

describe('Messenge Relayer Test', async () => {

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

    const accountNonceBob1 = await env.l1Provider.getTransactionCount(env.bobl1Wallet.address)
    console.log(`accountNonceBob1:`,accountNonceBob1)

    const accountNonceBob2 = await env.l2Provider.getTransactionCount(env.bobl2Wallet.address)
    console.log(`accountNonceBob2:`,accountNonceBob2)

  })

  it('should deploy contracts', async () => {
    
    L1Message = await Factory__L1Message.deploy(
      env.watcher.l1.messengerAddress,
      env.watcherMessengerFast.l1.messengerAddress
    )
    await L1Message.deployTransaction.wait()
    console.log(`🌕 ${chalk.green('L1 Message deployed to:')} ${chalk.white(L1Message.address)}`)
    
    L2Message = await Factory__L2Message.deploy(
      env.watcher.l2.messengerAddress,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2Message.deployTransaction.wait()
    console.log(`🌕 ${chalk.green('L2 Message deployed to:')} ${chalk.white(L2Message.address)}`)

    // Initialize L1 message
    const L1MessageTX = await L1Message.init(
      L2Message.address
    )
    await L1MessageTX.wait()
    console.log(`⭐️ ${chalk.green('L1 Message initialized:')} ${chalk.white(L1MessageTX.hash)}`)

    // Initialize L2 message
    const L2MessageTX = await L2Message.init(
      L1Message.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2MessageTX.wait()
    console.log(`⭐️ ${chalk.green('L2 Message initialized:')} ${chalk.white(L2MessageTX.hash)}`)
  })

  it('should send message from L2 to L1', async () => {
    await env.waitForXDomainTransactionMessengerFast(
      L2Message.sendMessageL2ToL1({
        gasLimit: 800000, 
        gasPrice: 0
      }),
      Direction.L2ToL1
    )
  })

  it('should send message from L1 to L2', async () => {
    await env.waitForXDomainTransaction(
      L1Message.sendMessageL1ToL2(),
      Direction.L1ToL2
    )
  })
})