import { expect } from 'chai'
import { Contract, ContractFactory, BigNumber, Wallet, utils, providers } from 'ethers'
import { Direction } from './shared/watcher-utils'

import L1ERC20Json from '../artifacts/contracts/ERC20.sol/ERC20.json'
import L1ERC20GatewayJson from '../artifacts/contracts/L1ERC20Gateway.sol/L1ERC20Gateway.json'
import L2DepositedERC20Json from '../artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'

import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('System setup', async () => {

  let Factory__L1ERC20: ContractFactory
  let Factory__L2DepositedERC20: ContractFactory
  let Factory__L1ERC20Gateway: ContractFactory

  let L1ERC20: Contract
  let L2DepositedERC20: Contract
  let L1ERC20Gateway: Contract
  
  let env: OptimismEnv

  //Test ERC20 
  const initialAmount = utils.parseEther("10000000000")
  const tokenName = 'OMGX Test'
  const tokenDecimals = 18
  const tokenSymbol = 'OMG'

  before(async () => {

    env = await OptimismEnv.new()

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

  })

  before(async () => {

    //Mint a new token on L1 and set up the L1 and L2 infrastructure
    // [initialSupply, name, decimals, symbol]
    // this is owned by bobl1Wallet
    L1ERC20 = await Factory__L1ERC20.deploy(
      initialAmount,
      tokenName,
      tokenDecimals,
      tokenSymbol
    )
    await L1ERC20.deployTransaction.wait()
    console.log("L1ERC20 deployed to:", L1ERC20.address)

    //Set up things on L2 for this new token
    // [l2MessengerAddress, name, symbol]
    L2DepositedERC20 = await Factory__L2DepositedERC20.deploy(
      env.watcher.l2.messengerAddress,
      tokenName,
      tokenSymbol
    )
    await L2DepositedERC20.deployTransaction.wait()
    console.log("L2DepositedERC20 deployed to:", L2DepositedERC20.address)
    
    //Deploy a gateway for the new token
    // [L1_ERC20.address, OVM_L2DepositedERC20.address, l1MessengerAddress]
    L1ERC20Gateway = await Factory__L1ERC20Gateway.deploy(
      L1ERC20.address,
      L2DepositedERC20.address,
      env.watcher.l1.messengerAddress,
    )
    await L1ERC20Gateway.deployTransaction.wait()
    console.log("L1ERC20Gateway deployed to:", L1ERC20Gateway.address)

    //Initialize the contracts for the new token
    const initL2 = await L2DepositedERC20.init(L1ERC20Gateway.address);
    await initL2.wait();
    console.log('L2 ERC20 initialized:',initL2.hash);
    
  })

  it('Bob Approve and Deposit ERC20 from L1 to L2', async () => {

    const depositL2ERC20Amount = utils.parseEther('150')
    const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const preL2ERC20Balance = await L2DepositedERC20.balanceOf(env.bobl2Wallet.address)
    
    console.log(" Bob Depositing L1 ERC20 to L2...")
    console.log(" On L1, Bob has:", preL1ERC20Balance)
    console.log(" On L2, Bob has:", preL2ERC20Balance)

    const approveL1ERC20TX = await L1ERC20.approve(
      L1ERC20Gateway.address,
      depositL2ERC20Amount
    )
    await approveL1ERC20TX.wait()

    const { tx, receipt } = await env.waitForXDomainTransaction(
      L1ERC20Gateway.deposit(depositL2ERC20Amount),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address);
    const postL2ERC20Balance = await L2DepositedERC20.balanceOf(env.bobl2Wallet.address)

    console.log(" On L1, Bob now has:", postL1ERC20Balance)
    console.log(" On L2, Bob now has:", postL2ERC20Balance)

    expect(preL1ERC20Balance).to.deep.eq(
      postL1ERC20Balance.add(depositL2ERC20Amount)
    )
    expect(preL2ERC20Balance).to.deep.eq(
      postL2ERC20Balance.sub(depositL2ERC20Amount)
    )

  })

  it('should transfer ERC20 token to Alice and Fraud', async () => {

    const transferL2ERC20Amount = utils.parseEther('10')

    const preBobL2ERC20Balance = await L2DepositedERC20.balanceOf(env.bobl2Wallet.address)
    const preAliceL2ERC20Balance = await L2DepositedERC20.balanceOf(env.alicel2Wallet.address)
    const preFraudL2ERC20Balance = await L2DepositedERC20.balanceOf(env.fraudl2Wallet.address)

    const tranferToAliceTX = await L2DepositedERC20.transfer(env.alicel2Wallet.address, transferL2ERC20Amount)
    await tranferToAliceTX.wait()

    const tranferToFraudTX = await L2DepositedERC20.transfer(env.fraudl2Wallet.address, transferL2ERC20Amount)
    await tranferToFraudTX.wait()

    const postBobL2ERC20Balance = await L2DepositedERC20.balanceOf(env.bobl2Wallet.address)
    const postAliceL2ERC20Balance = await L2DepositedERC20.balanceOf(env.alicel2Wallet.address)
    const postFraudL2ERC20Balance = await L2DepositedERC20.balanceOf(env.fraudl2Wallet.address)

    //because i'm sending the same amount out, twice....
    expect(preBobL2ERC20Balance).to.deep.eq(
      postBobL2ERC20Balance.add(transferL2ERC20Amount).add(transferL2ERC20Amount)
    )

    expect(preAliceL2ERC20Balance).to.deep.eq(
      postAliceL2ERC20Balance.sub(transferL2ERC20Amount)
    )

    expect(preFraudL2ERC20Balance).to.deep.eq(
      postFraudL2ERC20Balance.sub(transferL2ERC20Amount)
    )
  })

  it('should transfer ERC20 token from Alice to Fraud', async () => {

    const transferL2ERC20Amount = utils.parseEther('3')

    const preAliceL2ERC20Balance = await L2DepositedERC20.balanceOf(env.alicel2Wallet.address)
    const preFraudL2ERC20Balance = await L2DepositedERC20.balanceOf(env.fraudl2Wallet.address)

    console.log(" On L2, Alice has:", preAliceL2ERC20Balance.toString())
    console.log(" On L2, Fraud has:", preFraudL2ERC20Balance.toString())

    const tranferToFraudTX = await L2DepositedERC20.connect(env.alicel2Wallet).transfer(
      env.fraudl2Wallet.address, 
      transferL2ERC20Amount
    )
    await tranferToFraudTX.wait()

    const postAliceL2ERC20Balance = await L2DepositedERC20.balanceOf(env.alicel2Wallet.address)
    const postFraudL2ERC20Balance = await L2DepositedERC20.balanceOf(env.fraudl2Wallet.address)

    console.log(" On L2, Alice now has:", postAliceL2ERC20Balance.toString())
    console.log(" On L2, Fraud now has:", postFraudL2ERC20Balance.toString())

    expect(postAliceL2ERC20Balance).to.deep.eq(
      preAliceL2ERC20Balance.sub(transferL2ERC20Amount)
    )

    expect(postFraudL2ERC20Balance).to.deep.eq(
      preFraudL2ERC20Balance.add(transferL2ERC20Amount)
    )
  })

  it('should transfer ERC20 token from Fraud to Bob and commit fraud', async () => {

    const transferL2ERC20Amount = utils.parseEther('1')

    const preBobL2ERC20Balance = await L2DepositedERC20.balanceOf(env.bobl2Wallet.address)
    const preFraudL2ERC20Balance = await L2DepositedERC20.balanceOf(env.fraudl2Wallet.address)

    console.log(" On L2, Bob has:", preBobL2ERC20Balance.toString())
    console.log(" On L2, Fraud has:", preFraudL2ERC20Balance.toString())

    const tranferToFraudTX = await L2DepositedERC20.connect(env.fraudl2Wallet).transfer(
      env.bobl2Wallet.address, 
      transferL2ERC20Amount
    )
    await tranferToFraudTX.wait()

    const postBobL2ERC20Balance = await L2DepositedERC20.balanceOf(env.bobl2Wallet.address)
    const postFraudL2ERC20Balance = await L2DepositedERC20.balanceOf(env.fraudl2Wallet.address)

    console.log(" On L2, Bob now has:", postBobL2ERC20Balance.toString())
    console.log(" On L2, Fraud now has:", postFraudL2ERC20Balance.toString())

    expect(postBobL2ERC20Balance).to.deep.eq(
      preBobL2ERC20Balance.add(transferL2ERC20Amount)
    )

    expect(postFraudL2ERC20Balance).to.deep.eq(
      preFraudL2ERC20Balance.sub(transferL2ERC20Amount)
    )
  })

})