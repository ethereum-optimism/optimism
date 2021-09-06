import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import L1MessageJson from '../artifacts/contracts/test-helpers/Message/L1Message.sol/L1Message.json'
import L2MessageJson from '../artifacts-ovm/contracts/test-helpers/Message/L2Message.sol/L2Message.json'


import { OptimismEnv } from './shared/env'

import { promises as fs } from 'fs'

describe('System setup', async () => {

  let Factory__L1Message: ContractFactory
  let Factory__L2Message: ContractFactory

  let L1Message: Contract
  let L2Message: Contract

  let env: OptimismEnv

  /************* BOB owns all the pools, and ALICE mints a new token ***********/
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
  })

  it('should deploy contracts', async () => {

    L1Message = await Factory__L1Message.deploy(
      env.watcher.l1.messengerAddress,
      env.fastWatcher.l1.messengerAddress
    )
    await L1Message.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1 Message deployed to:')} ${chalk.green(L1Message.address)}`)

    L2Message = await Factory__L2Message.deploy(
      env.watcher.l2.messengerAddress,
      {gasPrice: 0}
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
      {gasPrice: 0}
    )
    await L2MessageTX.wait()
    console.log(`⭐️ ${chalk.blue('L2 Message initialized:')} ${chalk.green(L2MessageTX.hash)}`)
  })

  it('should write addresses to file', async () => {
    //keep track of where things are for future use by the front end
    console.log(`${chalk.yellow('\n\n********************************')}`)

    const addresses = {
      L1Message: L1Message.address,
      L2Message: L2Message.address
    }

    console.log(chalk.green(JSON.stringify(addresses, null, 2)))

    try{
      await fs.writeFile('./deployment/local/addresses.json', JSON.stringify(addresses, null, 2))
      console.log(`\n🚨 ${chalk.red('Successfully wrote addresses to file\n')}`)
    } catch (err) {
      console.log(`\n📬 ${chalk.red(`Error writing addresses to file: ${err}\n`)}`)
    }

    console.log(`${chalk.yellow('********************************')}`)
  })
})