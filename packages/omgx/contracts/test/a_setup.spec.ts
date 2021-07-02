import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers';
import chalk from 'chalk';
import { getContractFactory } from '@eth-optimism/contracts';

import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'

import { OptimismEnv } from './shared/env'
import { promises as fs } from 'fs'

describe('System setup', async () => {

  let Factory__L1ERC20: ContractFactory
  let Factory__L2ERC20: ContractFactory

  let L1ERC20: Contract
  let L2ERC20: Contract

  let env: OptimismEnv

  //Test ERC20
  const initialSupply = utils.parseEther("10000000000")
  const tokenName = 'JLKN'
  const tokenSymbol = 'JLKN'

  /************* BOB owns all the pools, and ALICE mints a new token ***********/
  before(async () => {

    env = await OptimismEnv.new()

    Factory__L1ERC20 = new ContractFactory(
      L1ERC20Json.abi,
      L1ERC20Json.bytecode,
      env.bobl1Wallet
    )

    Factory__L2ERC20 = getContractFactory(
      "L2StandardERC20",
      env.bobl2Wallet,
      true,
    )

  })

  it('should deploy ERC20', async () => {

    L1ERC20 = await Factory__L1ERC20.deploy(
      initialSupply,
      tokenName,
      tokenSymbol
    )
    await L1ERC20.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1ERC20 deployed to:')} ${chalk.green(L1ERC20.address)}`)
    
  })

})