import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
const expect = chai.expect;

import { Contract, ContractFactory, utils } from 'ethers';
import chalk from 'chalk';
import { getContractFactory } from '@eth-optimism/contracts';
import { Direction } from './shared/watcher-utils'

import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'

import { OptimismEnv } from './shared/env'
import { promises as fs } from 'fs'

describe('System setup', async () => {

  let L1ERC20: Contract
  let L2ERC20: Contract
  let Factory__L2ERC20: ContractFactory
  let L1StandardBridgeAddress: string
  let L2StandardBridgeAddress: string
  let L1StandardBridge: Contract

  let env: OptimismEnv

  before(async () => {

    env = await OptimismEnv.new()

    L1StandardBridgeAddress = await env.addressManager.getAddress('Proxy__OVM_L1StandardBridge')

    L1StandardBridge = getContractFactory(
      "OVM_L1StandardBridge",
      env.bobl1Wallet
    ).attach(L1StandardBridgeAddress)

    L2StandardBridgeAddress = await L1StandardBridge.l2TokenBridge()

    //let's tap into the contract we just deployed
    L1ERC20 = new Contract(
      env.addressesOMGX.TOKENS.TEST.L1,
      L1ERC20Json.abi,
      env.bobl1Wallet
    )

    Factory__L2ERC20 = getContractFactory(
      "L2StandardERC20",
      env.bobl2Wallet,
      true,
    )

    //let's tap into the contract we just deployed
    L2ERC20 = new Contract(
      env.addressesOMGX.TOKENS.TEST.L2,
      Factory__L2ERC20.interface,
      env.bobl2Wallet
    )
  })

  it('should use the recently deployed ERC20 TEST token and send some from L1 to L2', async () => {

    const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

    console.log(`🌕 ${chalk.red('L1ERC20 TEST token balance for Deployer PK:')} ${chalk.green(preL1ERC20Balance.toString())}`)
    console.log(`🌕 ${chalk.red('L2ERC20 TEST token balance for Deployer PK:')} ${chalk.green(preL2ERC20Balance.toString())}`)

    const depositL2ERC20Amount = utils.parseEther("12345")

    const approveL1ERC20TX = await L1ERC20.approve(
      L1StandardBridgeAddress,
      depositL2ERC20Amount
    )
    await approveL1ERC20TX.wait()

    await env.waitForXDomainTransaction(
      L1StandardBridge.depositERC20(
        L1ERC20.address,
        L2ERC20.address,
        depositL2ERC20Amount,
        9999999,
        utils.formatBytes32String((new Date().getTime()).toString())
      ),
      Direction.L1ToL2
    )

    const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

    console.log(`🌕 ${chalk.red('L1ERC20 TEST token balance for Deployer PK now:')} ${chalk.green(postL1ERC20Balance.toString())}`)
    console.log(`🌕 ${chalk.red('L2ERC20 TEST token balance for Deployer PK now:')} ${chalk.green(postL2ERC20Balance.toString())}`)

    expect(preL1ERC20Balance).to.deep.eq(
      postL1ERC20Balance.add(depositL2ERC20Amount)
    )

    expect(preL2ERC20Balance).to.deep.eq(
      postL2ERC20Balance.sub(depositL2ERC20Amount)
    )

  })

  it('should transfer ERC20 TEST token to Kate', async () => {

    const transferL2ERC20Amount = utils.parseEther("999")

    const preKateL2ERC20Balance = await L2ERC20.balanceOf(env.katel2Wallet.address)

    const transferToKateTX = await L2ERC20.transfer(
      env.katel2Wallet.address,
      transferL2ERC20Amount,
      {gasLimit: 6440000}
    )
    await transferToKateTX.wait()

    const postKateL2ERC20Balance = await L2ERC20.balanceOf(env.katel2Wallet.address)

    expect(postKateL2ERC20Balance).to.deep.eq(
      preKateL2ERC20Balance.add(transferL2ERC20Amount)
    )
  })

  it('should transfer ERC20 TEST token to Alice', async () => {

    const transferL2ERC20Amount = utils.parseEther("1111")

    let preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const preAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)

    const tranferToAliceTX = await L2ERC20.transfer(
      env.alicel2Wallet.address,
      transferL2ERC20Amount,
      {gasLimit: 6440000}
    )
    await tranferToAliceTX.wait()

    let postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const postAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)

    expect(postBobL2ERC20Balance).to.deep.eq(
      preBobL2ERC20Balance.sub(transferL2ERC20Amount)
    )

  })

})