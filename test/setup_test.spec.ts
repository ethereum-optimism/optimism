import { expect } from 'chai'

import { Contract, ContractFactory, BigNumber, Wallet, utils, providers } from 'ethers'
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

  const initialAmount = utils.parseEther("10000000000")
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

    const bobL1Balance = await _env.bobl1Wallet.getBalance()
    const bobL2Balance = await _env.bobl2Wallet.getBalance()
/*
    console.log("\nbobL1Balance:", bobL1Balance.toString())
    console.log("bobL2Balance:", bobL2Balance.toString())
    console.log("aliceL1Balance:", aliceL1Balance.toString())
    console.log("aliceL2Balance:", aliceL2Balance.toString())
    console.log("L1LPBalance:", L1LPBalance.toString())
    console.log("L2LPBalance:", L2LPBalance.toString())
    console.log("L1LPFeeBalance:", L1LPFeeBalance.toString())
    console.log("L2LPFeeBalance:", L2LPFeeBalance.toString())
*/
    return {
      L1LPBalance,      
      L2LPBalance,
      L1LPFeeBalance,
      L2LPFeeBalance,
      aliceL1Balance,
      aliceL2Balance,
      bobL1Balance,
      bobL2Balance,
    }
  }

  /************* BOB owns all the pools, and ALICE Mints a new token ***********/
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

  })

  before(async () => {

    //Who? sets up the L2LP
    L2LiquidityPool = await Factory__L2LiquidityPool.deploy(
      env.watcher.l2.messengerAddress,
    )
    await L2LiquidityPool.deployTransaction.wait()
    console.log("L2LiquidityPool deployed to:", L2LiquidityPool.address)

    L1LiquidityPool = await Factory__L1LiquidityPool.deploy(
      L2LiquidityPool.address,
      env.watcher.l1.messengerAddress,
      env.L2ETHGateway.address,
      3
    )
    await L1LiquidityPool.deployTransaction.wait()
    console.log("L1LiquidityPool deployed to:", L1LiquidityPool.address)
    
    const L2LiquidityPoolTX = await L2LiquidityPool.init(L1LiquidityPool.address, /*this is the fee = */ "3")
    //The '3' here denotes the fee to charge, i.e. fee = 3%
    await L2LiquidityPoolTX.wait()
    console.log('L2 LP initialized with the L1LiquidityPool.address:',L2LiquidityPoolTX.hash);

    //Who? mints a new token and sets up the L1 and L2 infrastructure
    // Mint a new token on L1
    // [initialSupply, name, decimals, symbol]
    // this is owned by bobl1Wallet
    L1ERC20 = await Factory__L1ERC20.deploy(
      initialAmount,
      tokenName,
      tokenDecimals,
      TokenSymbol
    )
    await L1ERC20.deployTransaction.wait()
    console.log("L1ERC20 deployed to:", L1ERC20.address)

    // Who? sets up things on L2 for this new token
    // [l2MessengerAddress, name, symbol]
    L2DepositedERC20 = await Factory__L2DepositedERC20.deploy(
      env.watcher.l2.messengerAddress,
      tokenName,
      TokenSymbol
    )
    await L2DepositedERC20.deployTransaction.wait()
    console.log("L2DepositedERC20 deployed to:", L2DepositedERC20.address)
    
    // Who? deploys a gateway for this new token
    // [L1_ERC20.address, OVM_L2DepositedERC20.address, l1MessengerAddress]
    L1ERC20Gateway = await Factory__L1ERC20Gateway.deploy(
      L1ERC20.address,
      L2DepositedERC20.address,
      env.watcher.l1.messengerAddress,
    )
    await L1ERC20Gateway.deployTransaction.wait()
    console.log("L1ERC20Gateway deployed to:", L1ERC20Gateway.address)

    // Who initializes the contracts for the new token
    const initL2 = await L2DepositedERC20.init(L1ERC20Gateway.address);
    await initL2.wait();
    console.log('L2 ERC20 initialized:',initL2.hash);
    
  })

  before(async () => {
    //keep track of where things are for future use by the front end
    console.log("\n\n********************************\nSaving all key addresses")

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

    console.log('********************************\n\n')

  })

  it('should transfer ERC20 from Bob to Alice', async () => {
    const transferAmount = utils.parseEther("50")

    //L1ERC20 is owned by Bob
    const preERC20Balances = await L1ERC20.balanceOf(env.alicel1Wallet.address);

    const transferERC20TX = await L1ERC20.transfer(
      env.alicel1Wallet.address,
      transferAmount,
    )
    await transferERC20TX.wait()

    const postERC20Balance = await L1ERC20.balanceOf(env.alicel1Wallet.address);
    
    expect(postERC20Balance).to.deep.eq(
      preERC20Balances.add(transferAmount)
    )
  })

  it('should add initial ETH and ERC20 to the L1 Liquidity Pool', async () => {

    // **************************************************
    // Only the contract owner (Bob) can deposit ETH into L1 LP
    // **************************************************
    const addAmount = utils.parseEther("50")

    // Add ETH
    const preETHBalances = await getBalances("0x0000000000000000000000000000000000000000")

    //const fee = BigNumber.from(189176000000000)
    const chainID = await env.bobl1Wallet.getChainId()
    const gasPrice = await env.bobl1Wallet.getGasPrice()

    const gasEstimate = await env.bobl1Wallet.estimateGas({
      from: env.bobl1Wallet.address,
      to: L1LiquidityPool.address,
      value: addAmount
    })
    
    //Bob, the owner of the L1LiquidityPool, sends funds into the L1LP
    const depositETHTX = await env.bobl1Wallet.sendTransaction({
      from: env.bobl1Wallet.address,
      to: L1LiquidityPool.address,
      value: addAmount
    })
    await depositETHTX.wait()

    const postETHBalances = await getBalances("0x0000000000000000000000000000000000000000")

    const l1ProviderLL = providers.getDefaultProvider(process.env.L1_NODE_WEB3_URL)
    const receipt = await l1ProviderLL.getTransactionReceipt(depositETHTX.hash);
    //console.log("transaction receipt:",receipt)

    //add fee calculation
    const feeSimple = depositETHTX.gasLimit.mul(depositETHTX.gasPrice)
    
    console.log('\nFEE DEBUG INFORMATION')
    console.log("ChainID:",chainID)
    console.log("GasPrice (gWei):",utils.formatUnits(gasPrice, "gwei"))
    console.log("Fee actually paid:",utils.formatUnits(preETHBalances.bobL1Balance.sub(addAmount).sub(postETHBalances.bobL1Balance), "gwei"))
    console.log("Fee gasLimit*gasPrice:",utils.formatUnits(feeSimple, "gwei"))
    console.log("GasEstimate (gWei):",utils.formatUnits(gasEstimate, "gwei"))
    console.log("GasUsed (gWei):",utils.formatUnits(receipt.gasUsed, "gwei"))
    console.log('\n')
    
    /*
    console.log("Fee actually paid:",postETHBalances.bobL1Balance.sub(addAmount).sub(postETHBalances.bobL1Balance).toString())
    console.log("Fee gasLimit*gasPrice:",feeSimple.toString())
    console.log("GasEstimate:",gasEstimate.toString())
    console.log("GasUsed:",gasUsed.toString())
    */
    
    /*
    IF YOU SEND ZERO AMOUNT, these numbers will be off. For example,
    
    Fee actually paid: 189176.0
    Fee gasLimit*gasPrice: 202464.0
    GasEstimate (gWei): 0.000025308
    GasUsed (gWei): 0.000023647

    IF YOU SEND NONZERO AMOUNT

    Fee actually paid: 342776.0
    Fee gasLimit*gasPrice: 342776.0
    GasEstimate (gWei): 0.000042847
    GasUsed (gWei): 0.000042847
    */

    //Bob should have less money now - this breaks for zero value transfer
    expect(postETHBalances.bobL1Balance).to.deep.eq(
      preETHBalances.bobL1Balance.sub(addAmount).sub(feeSimple)
    )

    //He paid into the L1LP
    expect(postETHBalances.L1LPBalance).to.deep.eq(
      preETHBalances.L1LPBalance.add(addAmount)
    )

    //Alice did not pay, so no change
    expect(postETHBalances.aliceL1Balance).to.deep.eq(
      preETHBalances.aliceL1Balance
    )
    
    // Add ERC20 Token
    const preERC20Balances = await getBalances(L1ERC20.address)

    const approveERC20TX = await L1ERC20.approve(
      L1LiquidityPool.address,
      addAmount,
    )
    await approveERC20TX.wait()

    const depositERC20TX = await L1LiquidityPool.ownerAddERC20Liquidity(
      addAmount,
      L1ERC20.address,
    );
    await depositERC20TX.wait();

    const postERC20Balance = await getBalances(L1ERC20.address)
    
    expect(postERC20Balance.L1LPBalance).to.deep.eq(
      preERC20Balances.L1LPBalance.add(addAmount)
    )
  })

  it('should add initial oWETH and ERC20 to the L2 Liquidity Pool', async () => {
    const depositL2Amount = utils.parseEther("50")
    const addAmount = utils.parseEther("45")

    // Add ETH
    const preETHBalances = await getBalances(env.L2ETHGateway.address)

    await env.waitForXDomainTransaction(
      env.L1ETHGateway.deposit({ value: depositL2Amount }),
      Direction.L1ToL2
    )
    
    const approveETHTX = await env.L2ETHGateway.approve(
      L2LiquidityPool.address,
      addAmount,
    );
    await approveETHTX.wait()

    const depositETHTX = await L2LiquidityPool.ownerAddERC20Liquidity(
      addAmount,
      env.L2ETHGateway.address,
    );
    await depositETHTX.wait()

    const postETHBalance = await getBalances(env.L2ETHGateway.address)

    expect(postETHBalance.L2LPBalance).to.deep.eq(
      preETHBalances.L2LPBalance.add(addAmount)
    )
    // Add ERC20
    const preERC20Balances = await getBalances(L2DepositedERC20.address)

    const approveL1ERC20TX = await L1ERC20.approve(
      L1ERC20Gateway.address,
      depositL2Amount,
    )
    await approveL1ERC20TX.wait()

    await env.waitForXDomainTransaction(
      L1ERC20Gateway.deposit(depositL2Amount),
      Direction.L1ToL2
    )

    const approveL2ERC20TX = await L2DepositedERC20.approve(
      L2LiquidityPool.address,
      addAmount,
    )
    await approveL2ERC20TX.wait()

    const depositERC20TX = await L2LiquidityPool.ownerAddERC20Liquidity(
      addAmount,
      L2DepositedERC20.address,
    );
    await depositERC20TX.wait()

    const postERC20Balances = await getBalances(L2DepositedERC20.address)

    expect(postERC20Balances.L2LPBalance).to.deep.eq(
      preERC20Balances.L2LPBalance.add(addAmount)
    )
  })

  it('should move ETH from L1 LP to L2', async () => {

    const swapAmount = utils.parseEther("1")
    const preBalances = await getBalances("0x0000000000000000000000000000000000000000")

    //this triggers the receive
    await env.waitForXDomainTransaction(
      env.alicel1Wallet.sendTransaction({
        from: env.alicel1Wallet.address,
        to: L1LiquidityPool.address,
        value: swapAmount
      }),
      Direction.L1ToL2
    )

    const postBalance = await getBalances("0x0000000000000000000000000000000000000000")

    expect(postBalance.aliceL2Balance).to.deep.eq(
      preBalances.aliceL2Balance.add(swapAmount.mul(97).div(100))
    )
    expect(postBalance.L1LPFeeBalance).to.deep.eq(
      preBalances.L1LPFeeBalance.add(swapAmount.mul(3).div(100))
    )
  })
  
  it('should swap wETH from L2 LP to ETH in L1 user wallet', async () => {
    
    const swapAmount = utils.parseEther("1")
    const preBalances = await getBalances(env.L2ETHGateway.address)

    const approveTX = await env.L2ETHGateway.approve(
      L2LiquidityPool.address,
      swapAmount
    )
    await approveTX.wait()

    await env.waitForXDomainTransaction(
      L2LiquidityPool.clientDepositL2(
        swapAmount,
        env.L2ETHGateway.address,
        "0x0000000000000000000000000000000000000000" // ETH Address
      ),
      Direction.L2ToL1
    )

    const postBalance = await getBalances(env.L2ETHGateway.address)

    expect(postBalance.bobL1Balance).to.deep.eq(
      preBalances.bobL1Balance.add(swapAmount.mul(97).div(100))
    )
    expect(postBalance.L2LPFeeBalance).to.deep.eq(
      preBalances.L2LPFeeBalance.add(swapAmount.mul(3).div(100))
    )
  })
})