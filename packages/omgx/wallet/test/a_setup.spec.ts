import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, utils } from 'ethers'
import chalk from 'chalk';

import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'
import L1ERC20GatewayJson from '../artifacts/contracts/L1ERC20Gateway.sol/L1ERC20Gateway.json'
import L2DepositedERC20Json from '../artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'

import L1LiquidityPoolJson from '../artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'

import L2TokenPoolJson from '../artifacts-ovm/contracts/TokenPool.sol/TokenPool.json'

import AtomicSwapJson from '../artifacts-ovm/contracts/AtomicSwap.sol/AtomicSwap.json';

import L1MessageJson from '../artifacts/contracts/Message/L1Message.sol/L1Message.json'
import L2MessageJson from '../artifacts-ovm/contracts/Message/L2Message.sol/L2Message.json'


import { OptimismEnv } from './shared/env'

import { promises as fs } from 'fs'

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

  //Test ERC20 
  const initialSupply = utils.parseEther("10000000000")
  const tokenName = 'JLKN'
  const tokenSymbol = 'JLKN'

  /************* BOB owns all the pools, and ALICE mints a new token ***********/
  before(async () => {

    env = await OptimismEnv.new()

    Factory__L1LiquidityPool = new ContractFactory(
      L1LiquidityPoolJson.abi,
      L1LiquidityPoolJson.bytecode,
      env.bobl1Wallet
    )

    Factory__L2LiquidityPool = new ContractFactory(
      L2LiquidityPoolJson.abi,
      L2LiquidityPoolJson.bytecode,
      env.bobl2Wallet
    )

    Factory__L1ERC20 = new ContractFactory(
      L1ERC20Json.abi,
      L1ERC20Json.bytecode,
      env.bobl1Wallet
    )

    Factory__L2DepositedERC20 = new ContractFactory(
      L2DepositedERC20Json.abi,
      L2DepositedERC20Json.bytecode,
      env.bobl2Wallet
    )

    Factory__L1ERC20Gateway = new ContractFactory(
      L1ERC20GatewayJson.abi,
      L1ERC20GatewayJson.bytecode,
      env.bobl1Wallet
    )

    Factory__L2TokenPool = new ContractFactory(
      L2TokenPoolJson.abi,
      L2TokenPoolJson.bytecode,
      env.bobl2Wallet
    )

    Factory__AtomicSwap = new ContractFactory(
      AtomicSwapJson.abi,
      AtomicSwapJson.bytecode,
      env.bobl2Wallet
    )

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
    
    // Deploy L2 liquidity pool
    L2LiquidityPool = await Factory__L2LiquidityPool.deploy(
      env.watcher.l2.messengerAddress,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2LiquidityPool.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L2LiquidityPool deployed to:')} ${chalk.green(L2LiquidityPool.address)}`)
    
    // Deploy L1 liquidity pool
    L1LiquidityPool = await Factory__L1LiquidityPool.deploy(
      env.watcher.l1.messengerAddress
    )
    await L1LiquidityPool.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1LiquidityPool deployed to:')} ${chalk.green(L1LiquidityPool.address)}`)
    
    // Initialize L1 liquidity pool
    const L1LiquidityPoolTX = await L1LiquidityPool.init(
      /* userRewardFeeRate 3.5% */ 35, 
      /* ownerRewardFeeRate 1.5% */ 15, 
      L2LiquidityPool.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L1LiquidityPoolTX.wait()
    console.log(`⭐️ ${chalk.blue('L1 LP initialized:')} ${chalk.green(L1LiquidityPoolTX.hash)}`)

    // Initialize L2 liquidity pool
    const L2LiquidityPoolTX = await L2LiquidityPool.init(
      /* userRewardFeeRate 3.5% */ 35, 
      /* ownerRewardFeeRate 1.5% */ 15, 
      L1LiquidityPool.address,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2LiquidityPoolTX.wait()
    console.log(`⭐️ ${chalk.blue('L2 LP initialized:')} ${chalk.green(L2LiquidityPoolTX.hash)}`)

    //Mint a new token on L1 and set up the L1 and L2 infrastructure
    // [initialSupply, name, symbol]
    // this is owned by bobl1Wallet
    L1ERC20 = await Factory__L1ERC20.deploy(
      initialSupply,
      tokenName,
      tokenSymbol
    )
    await L1ERC20.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1ERC20 deployed to:')} ${chalk.green(L1ERC20.address)}`)

    //Set up things on L2 for this new token
    // [l2MessengerAddress, name, symbol]
    L2DepositedERC20 = await Factory__L2DepositedERC20.deploy(
      env.watcher.l2.messengerAddress,
      tokenName,
      tokenSymbol,
      {gasLimit: 800000, gasPrice: 0}
    )
    await L2DepositedERC20.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L2DepositedERC20 deployed to:')} ${chalk.green(L2DepositedERC20.address)}`)

    //Deploy a gateway for the new token
    // [L1_ERC20.address, OVM_L2DepositedERC20.address, l1MessengerAddress]
    L1ERC20Gateway = await Factory__L1ERC20Gateway.deploy(
      L1ERC20.address,
      L2DepositedERC20.address,
      env.watcher.l1.messengerAddress
    )
    await L1ERC20Gateway.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1ERC20Gateway deployed to:')} ${chalk.green(L1ERC20Gateway.address)}`)

    //Initialize the ERC20 for the new token
    const initL2ERC20TX = await L2DepositedERC20.init(
      L1ERC20Gateway.address,
      {gasLimit: 800000, gasPrice: 0}
    );
    await initL2ERC20TX.wait();
    console.log(`⭐️ ${chalk.blue('L2DepositedERC20 initialized:')} ${chalk.green(initL2ERC20TX.hash)}`)

    //Deploy L2 token pool for the new token
    L2TokenPool = await Factory__L2TokenPool.deploy({gasLimit: 1000000, gasPrice: 0})
    await L2TokenPool.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L2TokenPool deployed to:')} ${chalk.green(L2TokenPool.address)}`)

    //Register ERC20 token address in L2 token pool
    const registerL2TokenPoolTX = await L2TokenPool.registerTokenAddress(
      L2DepositedERC20.address,
      {gasLimit: 800000, gasPrice: 0}
    );
    await registerL2TokenPoolTX.wait()
    console.log(`⭐️ ${chalk.blue('L2TokenPool registered:')} ${chalk.green(registerL2TokenPoolTX.hash)}`)

    // Deploy atomic swap
    AtomicSwap = await Factory__AtomicSwap.deploy({gasLimit: 1500000, gasPrice: 0})
    await AtomicSwap.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('AtomicSwap deployed to:')} ${chalk.green(AtomicSwap.address)}`)

    L1Message = await Factory__L1Message.deploy(
      env.watcher.l1.messengerAddress
    )
    await L1Message.deployTransaction.wait()
    console.log(`🌕 ${chalk.red('L1 Message deployed to:')} ${chalk.green(L1Message.address)}`)
    
    L2Message = await Factory__L2Message.deploy(
      env.watcher.l2.messengerAddress,
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

  it('should write addresses to file', async () => {
    //keep track of where things are for future use by the front end
    console.log(`${chalk.yellow('\n\n********************************')}`)

    const addresses = {
      L1LiquidityPool: L1LiquidityPool.address,
      L2LiquidityPool: L2LiquidityPool.address,
      L1ERC20: L1ERC20.address,
      L2DepositedERC20: L2DepositedERC20.address,
      L1ERC20Gateway: L1ERC20Gateway.address,
      l1ETHGatewayAddress: env.L1ETHGateway.address,
      l1MessengerAddress: env.l1MessengerAddress,
      L2TokenPool: L2TokenPool.address,
      AtomicSwap: AtomicSwap.address,
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