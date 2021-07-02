import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';
import { getOMGXDeployerAddresses } from './shared/utils'

import { OptimismEnv } from './shared/env'

import { promises as fs } from 'fs'

let walletAddresses;

describe('System setup', async () => {

  let Factory__L1LiquidityPool: ContractFactory
  let Factory__L2LiquidityPool: ContractFactory
  let Factory__L1ERC20: ContractFactory
  let Factory__L2DepositedERC20: ContractFactory
  let Factory__L1ERC20Gateway: ContractFactory
  let Factory__L2TokenPool: ContractFactory
  let Factory__AtomicSwap: ContractFactory
  let Factory__L1Message: ContractFactory
  let Factory__L2Message: ContractFactory

  let L1LiquidityPool: Contract
  let L2LiquidityPool: Contract
  let L1ERC20: Contract
  let L2DepositedERC20: Contract
  let L1ERC20Gateway: Contract
  let L2TokenPool: Contract
  let AtomicSwap: Contract
  let L1Message: Contract
  let L2Message: Contract

  let env: OptimismEnv

  /************* BOB owns all the pools, and ALICE mints a new token ***********/
  before(async () => {
    env = await OptimismEnv.new()
  })

  it('should write addresses to file', async () => {

    walletAddresses = await getOMGXDeployerAddresses()
    //keep track of where things are for future use by the front end
    console.log(`${chalk.yellow('\n\n********************************')}`)

    const addresses = {
      L2LiquidityPool: walletAddresses.L2LiquidityPool,
      L1LiquidityPool: walletAddresses.L1LiquidityPool,
      L1Message: walletAddresses.L1Message,
      L2Message: walletAddresses.L2Message,
    }

    console.log(chalk.green(JSON.stringify(addresses, null, 2)))

    try{
      await fs.mkdir('./deployment/local/', { recursive: true })
      await fs.writeFile('./deployment/local/addresses.json', JSON.stringify(addresses, null, 2))
      console.log(`\n🚨 ${chalk.red('Successfully wrote addresses to file\n')}`)
    } catch (err) {
      console.log(`\n📬 ${chalk.red(`Error writing addresses to file: ${err}\n`)}`)
    }

    console.log(`${chalk.yellow('********************************')}`)
  })
})