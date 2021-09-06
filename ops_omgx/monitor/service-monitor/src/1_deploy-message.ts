import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import L1MessageJson from './artifacts/contracts/test-helpers/Message/L1Message.sol/L1Message.json'
import L2MessageJson from './artifacts-ovm/contracts/test-helpers/Message/L2Message.sol/L2Message.json'


import { OptimismEnv } from './libs/env'

import { promises as fs } from 'fs'

async function deploy() {
  let Factory__L1Message: ContractFactory
  let Factory__L2Message: ContractFactory

  let L1Message: Contract
  let L2Message: Contract

  let env: OptimismEnv

  /************* PREFPARE ***********/

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


  /*** DEPLOY CONTRACT *****/

  L1Message = await Factory__L1Message.deploy(
    env.watcher.l1.messengerAddress,
    env.fastWatcher.l1.messengerAddress
  )
  await L1Message.deployTransaction.wait()
  console.log(`🌕 ${chalk.red('L1 Message deployed to:')} ${chalk.green(L1Message.address)}`)

  L2Message = await Factory__L2Message.deploy(
    env.watcher.l2.messengerAddress,
    { gasLimit: 3000000, gasPrice: 0 }
  )
  console.log(`${chalk.red('L2 message')} ${L2Message.address}`)

  let result = await L2Message.deployTransaction.wait()
  console.log(`${chalk.red('L2 message deployed result')} ${result}`)
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
    { gasLimit: 3000000, gasPrice: 0 }
  )
  await L2MessageTX.wait()
  console.log(`⭐️ ${chalk.blue('L2 Message initialized:')} ${chalk.green(L2MessageTX.hash)}`)



  /********* WRITE TO FILE  **************/
  //keep track of where things are for future use by the front end
  console.log(`${chalk.yellow('\n\n********************************')}`)

  const addresses = {
    L1Message: L1Message.address,
    L2Message: L2Message.address
  }

  console.log(chalk.green(JSON.stringify(addresses, null, 2)))

  try {
    await fs.writeFile('./deployment/rinkeby/addresses.json', JSON.stringify(addresses, null, 2))
    console.log(`\n🚨 ${chalk.red('Successfully wrote addresses to file\n')}`)
  } catch (err) {
    console.log(`\n📬 ${chalk.red(`Error writing addresses to file: ${err}\n`)}`)
  }

  console.log(`${chalk.yellow('********************************')}`)
}


deploy().catch((err) => {
  console.log(err.message)
})
