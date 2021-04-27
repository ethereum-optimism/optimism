import { expect } from 'chai'

/* Imports: External */
import { Contract, ContractFactory, BigNumber } from 'ethers'
import { Direction } from './shared/watcher-utils'

import L1LiquidityPoolJson from '../artifacts/contracts/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../artifacts-ovm/contracts/L2LiquidityPool.sol/L2LiquidityPool.json'
import { OptimismEnv } from './shared/env'

/*
import { OVM_CrossDomainEnabled } from "enyalabs_contracts/build/contracts/libraries/bridge/OVM_CrossDomainEnabled.sol"
test/swap-pool-integration.spec.ts:7:33 - error TS2307: Cannot find module '../artifacts/contracts/L1LiquidityPool.sol/L1LiquidityPool.json' or its corresponding type declarations.
7 import L1LiquidityPoolJson from '../artifacts/contracts/L1LiquidityPool.sol/L1LiquidityPool.json'
*/

describe('Swap Pool Integration Tests', async () => {
  let Factory__L1LiquidityPool: ContractFactory
  let Factory__L2LiquidityPool: ContractFactory
  let L1LiquidityPool: Contract
  let L2LiquidityPool: Contract
  let env: OptimismEnv

  const getBalances = async (_address: string, _L1LiquidityPool=L1LiquidityPool, _L2LiquidityPool=L2LiquidityPool, _env=env) => {

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
    Factory__L1LiquidityPool = new ContractFactory(
      L1LiquidityPoolJson.abi,
      L1LiquidityPoolJson.bytecode,
      env.bobl1Wallet
    )
    Factory__L2LiquidityPool = new ContractFactory(
      L2LiquidityPoolJson.abi,
      L2LiquidityPoolJson.bytecode,
      env.alicel2Wallet
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

    const L2LiquidityPoolTX = await L2LiquidityPool.init(L1LiquidityPool.address, "3")
    await L2LiquidityPoolTX.wait()
  })

  it('should add initial funds in L1 Liquidity Pool', async () => {
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

  it('should add initial funds in L2 Liquidity Pool', async () => {
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

    const depositTX = await L2LiquidityPool.initiateDepositTo(
      addAmount,
      env.L2ETHGateway.address,
    );
    await depositTX.wait()

    const postBalance = await getBalances(env.L2ETHGateway.address)

    expect(postBalance.L2LPBalance).to.deep.eq(
      preBalances.L2LPBalance.add(addAmount)
    )
  })

  it('should swap on funds from L1 LP to L2', async () => {
    const swapAmount = BigNumber.from(100)
    const preBalances = await getBalances(env.L2ETHGateway.address)

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
  
  it('should swap off funds from L2 LP to L1', async () => {
    const swapAmount = BigNumber.from(100)
    const preBalances = await getBalances("0x0000000000000000000000000000000000000000")

    const approveTX = await env.L2ETHGateway.approve(
      L2LiquidityPool.address,
      swapAmount,
    );
    await approveTX.wait()

    await env.waitForXDomainTransaction(
      L2LiquidityPool.depositTo(
        swapAmount,
        env.L2ETHGateway.address,
        "0x0000000000000000000000000000000000000000",
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