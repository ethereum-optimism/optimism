import { expect } from 'chai'

import { Contract, ContractFactory, BigNumber, Wallet } from 'ethers'
import { Direction } from './shared/watcher-utils'

import L1LiquidityPoolJson from '../artifacts/contracts/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../artifacts-ovm/contracts/L2LiquidityPool.sol/L2LiquidityPool.json'
import L1ERC20Json from '../artifacts/contracts/ERC20.sol/ERC20.json'
import L2DepositedERC20Json from '../artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'
import L1ERC20GatewayJson from '../artifacts/contracts/L1ERC20Gateway.sol/L1ERC20Gateway.json'

import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('Token, Bridge, and Swap Pool Setup and Test', async () => {

  let Factory__L1LiquidityPool: ContractFactory
  let Factory__L2LiquidityPool: ContractFactory
  let Factory__L1ERC20: ContractFactory
  let Factory__L2DepositedERC20: ContractFactory
  let Factory__L1ERC20Gateway: ContractFactory

  let L1LiquidityPool: Contract
  let L2LiquidityPool: Contract
  let L1ERC20: Contract
  let L2DepositedERC20: Contract
  let L1ERC20Gateway: Contract
  
  let env: OptimismEnv

  let wallet: Wallet
  let other: Wallet

  const initialAmount = 1000
  const tokenName = 'JLKN Test'
  const tokenDecimals = 18
  const TokenSymbol = 'JLKN'

  const getBalances = async (
    _address: string, 
    _L1LiquidityPool=L1LiquidityPool, 
    _L2LiquidityPool=L2LiquidityPool, 
    _env=env
   ) => {

    const L1LPBalance = await _L1LiquidityPool.balanceOf(_address)
    const L2LPBalance = await _L2LiquidityPool.balanceOf(_address)

    const L1LPFeeBalance = await _L1LiquidityPool.feeBalanceOf(_address)
    const L2LPFeeBalance = await _L2LiquidityPool.feeBalanceOf(_address)

    const aliceL1Balance = await _env.alicel1Wallet.getBalance()
    const aliceL2Balance = await _env.alicel2Wallet.getBalance()

    return {
      L1LPBalance,      
      L2LPBalance,
      L1LPFeeBalance,
      L2LPFeeBalance,
      aliceL1Balance,
      aliceL2Balance,
    }
  }

  before(async () => {

    env = await OptimismEnv.new()

    //this is going on to the L1
    Factory__L1LiquidityPool = new ContractFactory(
      L1LiquidityPoolJson.abi,
      L1LiquidityPoolJson.bytecode,
      env.bobl1Wallet
    )

    //this is going on to the L2
    Factory__L2LiquidityPool = new ContractFactory(
      L2LiquidityPoolJson.abi,
      L2LiquidityPoolJson.bytecode,
      env.alicel2Wallet
    )

    //this is going on to the L1
    Factory__L1ERC20 = new ContractFactory(
      L1ERC20Json.abi,
      L1ERC20Json.bytecode,
      env.bobl1Wallet
    )

    //this is going on to the L2
    Factory__L2DepositedERC20 = new ContractFactory(
      L2DepositedERC20Json.abi,
      L2DepositedERC20Json.bytecode,
      env.alicel2Wallet
    )

    Factory__L1ERC20Gateway = new ContractFactory(
      L1ERC20GatewayJson.abi,
      L1ERC20GatewayJson.bytecode,
      env.bobl1Wallet
    )


  })

  before(async () => {

    L2LiquidityPool = await Factory__L2LiquidityPool.deploy(
      env.watcher.l2.messengerAddress,
    )
    await L2LiquidityPool.deployTransaction.wait()
    
    L1LiquidityPool = await Factory__L1LiquidityPool.deploy(
      L2LiquidityPool.address,
      env.watcher.l1.messengerAddress,
      env.L2ETHGateway.address,
      3
    )
    await L1LiquidityPool.deployTransaction.wait()

    // const initL2LP = await L2_LP.init(L1_LP.address, "3");
    // await initL2LP.wait();
    // console.log(' L2 LP initialized:',initL2LP.hash);
    
    const L2LiquidityPoolTX = await L2LiquidityPool.init(L1LiquidityPool.address, "3")
    await L2LiquidityPoolTX.wait()
    console.log(' L2 LP initialized:',L2LiquidityPoolTX.hash);

    // [initialSupply, name, decimals, symbol]
    L1ERC20 = await Factory__L1ERC20.deploy(
      initialAmount,
      tokenName,
      tokenDecimals,
      TokenSymbol
    )
    await L1ERC20.deployTransaction.wait()

    // [l2MessengerAddress, name, symbol]
    L2DepositedERC20 = await Factory__L2DepositedERC20.deploy(
      env.watcher.l2.messengerAddress,
      tokenName,
      TokenSymbol
    )
    await L2DepositedERC20.deployTransaction.wait()

    //are we ready?
    //await L1ERC20.deployed()
    //await L2DepositedERC20.deployed()
    
    // Ok, let's go for it
    // [L1_ERC20.address, OVM_L2DepositedERC20.address, l1MessengerAddress]
    L1ERC20Gateway = await Factory__L1ERC20Gateway.deploy(
      L1ERC20.address,
      L2DepositedERC20.address,
      env.watcher.l1.messengerAddress,
    )
    await L1ERC20Gateway.deployTransaction.wait()

    // initialize the contracts
    const initL2 = await L2DepositedERC20.init(L1ERC20Gateway.address);
    await initL2.wait();
    console.log(' L2 ERC20 initialized:',initL2.hash);
    
  })

  before(async () => {

    //keep track of where things are for future use by the front end
    console.log("\n\nSaving all key addresses")

    await L1LiquidityPool.deployed()
    console.log("L1LiquidityPool deployed to:", L1LiquidityPool.address)

    await L2LiquidityPool.deployed()
    console.log("L2LiquidityPool deployed to:", L2LiquidityPool.address)

    await L1ERC20.deployed()
    console.log("L1ERC20 deployed to:", L1ERC20.address)

    await L2DepositedERC20.deployed()
    console.log("L2DepositedERC20 deployed to:", L2DepositedERC20.address)

    await L1ERC20Gateway.deployed()
    console.log("L1ERC20Gateway deployed to:", L1ERC20Gateway.address)

    const addresses = {
      L1LiquidityPool: L1LiquidityPool.address,
      L2LiquidityPool: L2LiquidityPool.address,
      L1ERC20: L1ERC20.address,
      L2DepositedERC20: L2DepositedERC20.address,
      L1ERC20Gateway: L1ERC20Gateway.address,
      l1ETHGatewayAddress: env.L1ETHGateway.address,
      l1MessengerAddress: env.l1MessengerAddress
    }

    console.log(JSON.stringify(addresses, null, 2))

    fs.writeFile('./deployment/addresses.json', JSON.stringify(addresses, null, 2), err => {
      if (err) {
        console.log('Error writing addresses to file:', err)
      } else {
        console.log('Successfully wrote addresses to file')
      }
    })

  })

  it('should add initial ETH to the L1 Liquidity Pool', async () => {

    // **************************************************
    // Only the contract owner can deposit ETH into L1 LP
    // **************************************************
    const addAmount = BigNumber.from(250)
    const preBalances = await getBalances("0x0000000000000000000000000000000000000000")

    const depositTX = await env.bobl1Wallet.sendTransaction({
      from: env.bobl1Wallet.address,
      to: L1LiquidityPool.address,
      value: addAmount
    })
    await depositTX.wait()

    const postBalance = await getBalances("0x0000000000000000000000000000000000000000")

    expect(postBalance.L1LPBalance).to.deep.eq(
      preBalances.L1LPBalance.add(addAmount)
    )
  })

  it('should add initial funds to the L2 Liquidity Pool', async () => {

    const depositL2Amount = BigNumber.from(350)
    const addAmount = BigNumber.from(200)
    const preBalances = await getBalances(env.L2ETHGateway.address)

    await env.waitForXDomainTransaction(
      env.L1ETHGateway.deposit({ value: depositL2Amount }),
      Direction.L1ToL2
    )
    
    const approveTX = await env.L2ETHGateway.approve(
      L2LiquidityPool.address,
      addAmount,
    );
    await approveTX.wait()

    const depositTX = await L2LiquidityPool.ownerAddERC20Liquidity(
      addAmount,
      env.L2ETHGateway.address,
    );
    await depositTX.wait()

    const postBalance = await getBalances(env.L2ETHGateway.address)

    expect(postBalance.L2LPBalance).to.deep.eq(
      preBalances.L2LPBalance.add(addAmount)
    )
  })

  it('should move ETH from L1 LP to L2', async () => {

    const swapAmount = BigNumber.from(100)
    const preBalances = await getBalances(env.L2ETHGateway.address)

    //this triggers the receive
    await env.waitForXDomainTransaction(
      env.alicel1Wallet.sendTransaction({
        from: env.alicel1Wallet.address,
        to: L1LiquidityPool.address,
        value: swapAmount
      }),
      Direction.L1ToL2
    )

    const postBalance = await getBalances(env.L2ETHGateway.address)

    expect(postBalance.aliceL2Balance).to.deep.eq(
      preBalances.aliceL2Balance.add(swapAmount.mul(97).div(100))
    )
    expect(postBalance.L2LPFeeBalance).to.deep.eq(
      preBalances.L2LPFeeBalance.add(swapAmount.mul(3).div(100))
    )
  })
  
  it('should swap wETH from L2 LP to ETH in L1 user wallet', async () => {
    
    const swapAmount = BigNumber.from(100)
    const preBalances = await getBalances("0x0000000000000000000000000000000000000000")

    const approveTX = await env.L2ETHGateway.approve(
      L2LiquidityPool.address,
      swapAmount
    )
    await approveTX.wait()

    //this is for wETH 
    const depositTX = await L2LiquidityPool.clientDepositL2(
      swapAmount,
      L2ERC20.address, //should be _erc20L2ContractAddress
      L1ERC20.address  //should be _erc20L1ContractAddress
    )
    await depositTX.wait()

    await env.waitForXDomainTransaction(
      L1LiquidityPool.clientPayL1(
        env.alicel1Wallet.address,
        swapAmount,
        env.L1ETHGateway.address
      ),
      Direction.L2ToL1
    )

    const postBalance = await getBalances("0x0000000000000000000000000000000000000000")

    expect(postBalance.aliceL1Balance).to.deep.eq(
      preBalances.aliceL1Balance.add(swapAmount.mul(97).div(100))
    )
    expect(postBalance.L1LPFeeBalance).to.deep.eq(
      preBalances.L1LPFeeBalance.add(swapAmount.mul(3).div(100))
    )
  })
})