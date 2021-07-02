import { expect } from 'chai'
import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, BigNumber, utils, ethers } from 'ethers'
import { Direction } from './shared/watcher-utils'
import { getContractFactory } from '@eth-optimism/contracts';

/*
import L1ERC20Json from '../artifacts/contracts/L1ERC20.sol/L1ERC20.json'
import L1LiquidityPoolJson from '../artifacts/contracts/LP/L1LiquidityPool.sol/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../artifacts-ovm/contracts/LP/L2LiquidityPool.sol/L2LiquidityPool.json'
import L2TokenPoolJson from '../artifacts-ovm/contracts/TokenPool.sol/TokenPool.json'
*/
import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('Liquidity Pool Test', async () => {

  let L1LiquidityPool: Contract
  let L2LiquidityPool: Contract
  let L1ERC20: Contract
  let L2ERC20: Contract
  let L1StandardBridge: Contract
  let L2TokenPool: Contract

  let env: OptimismEnv

  /************* BOB owns all the pools, and ALICE mints a new token ***********/
  before(async () => {

    env = await OptimismEnv.new()

    /****************************
    //  * THIS NEEDS TO BE CHANGED/UPDATED TO TEST THE DEPLOYED CONTRACTS
    //  * The addresses are at

    //  export const getOMGXDeployerAddresses = async () => {
    //    var options = {
    //        uri: OMGX_URL,
    //    }
    //    const result = await request.get(options)
    //    return JSON.parse(result)
    // }
    *****************************/

    // console.log(env.addressesOMGX)

    // L1LiquidityPool = new Contract(
    //   env.addressesOMGX.L1LiquidityPool,
    //   L1LiquidityPoolJson.abi,
    //   env.bobl1Wallet
    // )

    // L2LiquidityPool = new Contract(
    //   env.addressesOMGX.L2LiquidityPool,
    //   L2LiquidityPoolJson.abi,
    //   env.bobl2Wallet
    // )

    // L1ERC20 = new Contract(
    //   env.addressesOMGX.L1ERC20,
    //   L1ERC20Json.abi,
    //   env.bobl1Wallet
    // )

    // L2ERC20 = getContractFactory(
    //   "L2StandardERC20",
    //   env.bobl2Wallet,
    //   true,
    // ).attach(env.addressesOMGX.L2ERC20)

    // L1StandardBridge = env.L1StandardBridge

    // L2TokenPool = new Contract(
    //   env.addressesOMGX.L2TokenPool,
    //   L2TokenPoolJson.abi,
    //   env.bobl2Wallet,
    // )
  })

  it('should deposit ERC20 token to L2', async () => {

  //   const depositL2ERC20Amount = utils.parseEther("10000");

  //   const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
  //   const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

  //   const approveL1ERC20TX = await L1ERC20.approve(
  //     L1StandardBridge.address,
  //     depositL2ERC20Amount
  //   )
  //   await approveL1ERC20TX.wait()

  //   await env.waitForXDomainTransaction(
  //     L1StandardBridge.depositERC20(
  //       L1ERC20.address,
  //       L2ERC20.address,
  //       depositL2ERC20Amount,
  //       9999999,
  //       ethers.utils.formatBytes32String((new Date().getTime()).toString())
  //     ),
  //     Direction.L1ToL2
  //   )

  //   const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
  //   const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

  //   expect(preL1ERC20Balance).to.deep.eq(
  //     postL1ERC20Balance.add(depositL2ERC20Amount)
  //   )

  //   expect(preL2ERC20Balance).to.deep.eq(
  //     postL2ERC20Balance.sub(depositL2ERC20Amount)
  //   )
  })

  it('should transfer ERC20 token to Alice and Kate', async () => {

  //   const transferL2ERC20Amount = utils.parseEther("150")

  //   const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const preAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)
  //   const preKateL2ERC20Balance = await L2ERC20.balanceOf(env.katel2Wallet.address)

  //   const tranferToAliceTX = await L2ERC20.transfer(
  //     env.alicel2Wallet.address,
  //     transferL2ERC20Amount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await tranferToAliceTX.wait()

  //   const transferToKateTX = await L2ERC20.transfer(
  //     env.katel2Wallet.address,
  //     transferL2ERC20Amount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await transferToKateTX.wait()

  //   const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const postAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)
  //   const postKateL2ERC20Balance = await L2ERC20.balanceOf(env.katel2Wallet.address)

  //   expect(preBobL2ERC20Balance).to.deep.eq(
  //     postBobL2ERC20Balance.add(transferL2ERC20Amount).add(transferL2ERC20Amount)
  //   )

  //   expect(preAliceL2ERC20Balance).to.deep.eq(
  //     postAliceL2ERC20Balance.sub(transferL2ERC20Amount)
  //   )

  //   expect(preKateL2ERC20Balance).to.deep.eq(
  //     postKateL2ERC20Balance.sub(transferL2ERC20Amount)
  //   )
  })

  it('should add ERC20 token to token pool', async () => {

  //   const addL2TPAmount = utils.parseEther("1000")

  //   const approveL2TPTX = await L2ERC20.approve(
  //     L2TokenPool.address,
  //     addL2TPAmount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await approveL2TPTX.wait()

  //   const transferL2TPTX = await L2ERC20.transfer(
  //     L2TokenPool.address,
  //     addL2TPAmount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   );
  //   await transferL2TPTX.wait()

  //   const L2TPBalance = await L2ERC20.balanceOf(L2TokenPool.address)

  //   expect(L2TPBalance).to.deep.eq(addL2TPAmount)
  })

  it('should register L1 the pool', async () => {

  //   const registerPoolERC20TX = await L1LiquidityPool.registerPool(
  //     L1ERC20.address,
  //     L2ERC20.address,
  //   )
  //   await registerPoolERC20TX.wait()

  //   const poolERC20Info = await L1LiquidityPool.poolInfo(L1ERC20.address)

  //   expect(poolERC20Info.l1TokenAddress).to.deep.eq(L1ERC20.address)
  //   expect(poolERC20Info.l2TokenAddress).to.deep.eq(L2ERC20.address)

  //   const registerPoolETHTX = await L1LiquidityPool.registerPool(
  //     "0x0000000000000000000000000000000000000000",
  //     env.l2ETHAddress,
  //   )
  //   await registerPoolETHTX.wait()

  //   const poolETHInfo = await L1LiquidityPool.poolInfo("0x0000000000000000000000000000000000000000")

  //   expect(poolETHInfo.l1TokenAddress).to.deep.eq("0x0000000000000000000000000000000000000000")
  //   expect(poolETHInfo.l2TokenAddress).to.deep.eq(env.l2ETHAddress)
  })

  it('should register L2 the pool', async () => {

  //   const registerPoolERC20TX = await L2LiquidityPool.registerPool(
  //     L1ERC20.address,
  //     L2ERC20.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await registerPoolERC20TX.wait()

  //   const poolERC20Info = await L2LiquidityPool.poolInfo(L2ERC20.address)

  //   expect(poolERC20Info.l1TokenAddress).to.deep.eq(L1ERC20.address)
  //   expect(poolERC20Info.l2TokenAddress).to.deep.eq(L2ERC20.address)

  //   const registerPoolETHTX = await L2LiquidityPool.registerPool(
  //     "0x0000000000000000000000000000000000000000",
  //     env.l2ETHAddress,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await registerPoolETHTX.wait()

  //   const poolETHInfo = await L2LiquidityPool.poolInfo(env.l2ETHAddress)

  //   expect(poolETHInfo.l1TokenAddress).to.deep.eq("0x0000000000000000000000000000000000000000")
  //   expect(poolETHInfo.l2TokenAddress).to.deep.eq(env.l2ETHAddress)
  })

  it('shouldn\'t update the pool', async () => {
  //   const registerPoolTX = await L2LiquidityPool.registerPool(
  //     L1ERC20.address,
  //     L2ERC20.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await expect(registerPoolTX.wait()).to.be.eventually.rejected;
  })

  it('should add L1 liquidity', async () => {
  //   const addLiquidityAmount = utils.parseEther("100")

  //   const preBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)

  //   const approveBobL1TX = await L1ERC20.approve(
  //     L1LiquidityPool.address,
  //     addLiquidityAmount,
  //   )
  //   await approveBobL1TX.wait()

  //   const BobAddLiquidity = await L1LiquidityPool.addLiquidity(
  //       addLiquidityAmount,
  //       L1ERC20.address
  //   )
  //   await BobAddLiquidity.wait()

  //   // ERC20 balance
  //   const postBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)

  //   expect(preBobL1ERC20Balance).to.deep.eq(
  //     postBobL1ERC20Balance.add(addLiquidityAmount)
  //   )

  //   // Pool Balance
  //   const L1LPERC20Balance = await L1ERC20.balanceOf(L1LiquidityPool.address)

  //   expect(L1LPERC20Balance).to.deep.eq(addLiquidityAmount)
  })

  it('should add L2 liquidity', async () => {

  //   const addLiquidityAmount = utils.parseEther("100")

  //   const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const preAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)

  //   const approveBobL2TX = await L2ERC20.approve(
  //     L2LiquidityPool.address,
  //     addLiquidityAmount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await approveBobL2TX.wait()

  //   const BobAddLiquidity = await L2LiquidityPool.addLiquidity(
  //     addLiquidityAmount,
  //     L2ERC20.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await BobAddLiquidity.wait()

  //   const approveAliceL2TX = await L2ERC20.connect(env.alicel2Wallet).approve(
  //     L2LiquidityPool.address,
  //     addLiquidityAmount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await approveAliceL2TX.wait()

  //   const AliceAddLiquidity = await L2LiquidityPool.connect(env.alicel2Wallet).addLiquidity(
  //     addLiquidityAmount,
  //     L2ERC20.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await AliceAddLiquidity.wait()

  //   // ERC20 balance
  //   const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const postAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)

  //   expect(preBobL2ERC20Balance).to.deep.eq(
  //     postBobL2ERC20Balance.add(addLiquidityAmount)
  //   )
  //   expect(preAliceL2ERC20Balance).to.deep.eq(
  //     postAliceL2ERC20Balance.add(addLiquidityAmount)
  //   )

  //   // User deposit amount
  //   const BobPoolAmount = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address);
  //   const AlicePoolAmount = await L2LiquidityPool.userInfo(L2ERC20.address, env.alicel2Wallet.address);

  //   expect(BobPoolAmount.amount).to.deep.eq(addLiquidityAmount)
  //   expect(AlicePoolAmount.amount).to.deep.eq(addLiquidityAmount)

  //   // Pool Balance
  //   const L2LPERC20Balance = await L2ERC20.balanceOf(L2LiquidityPool.address)

  //   expect(L2LPERC20Balance).to.deep.eq(addLiquidityAmount.mul(2))
  })

  it("should fast exit L2", async () => {

  //   const fastExitAmount = utils.parseEther("10")

  //   const preKateL1ERC20Balance = await L1ERC20.balanceOf(env.katel1Wallet.address)

  //   const approveKateL2TX = await L2ERC20.connect(env.katel2Wallet).approve(
  //     L2LiquidityPool.address,
  //     fastExitAmount,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await approveKateL2TX.wait()

  //   //await env.waitForXDomainTransactionFast(
  //   await env.waitForXDomainTransaction(
  //     L2LiquidityPool.connect(env.katel2Wallet).clientDepositL2(
  //       fastExitAmount,
  //       L2ERC20.address,
  //       {gasLimit: 800000, gasPrice: 0}
  //     ),
  //     Direction.L2ToL1
  //   )

  //   const poolInfo = await L1LiquidityPool.poolInfo(L1ERC20.address)

  //   expect(poolInfo.accOwnerReward).to.deep.eq(fastExitAmount.mul(15).div(1000))
  //   expect(poolInfo.accUserReward).to.deep.eq(fastExitAmount.mul(35).div(1000))
  //   expect(poolInfo.userDepositAmount).to.deep.eq(utils.parseEther("100"))

  //   const postKateL1ERC20Balance = await L1ERC20.balanceOf(env.katel1Wallet.address)

  //   expect(postKateL1ERC20Balance).to.deep.eq(preKateL1ERC20Balance.add(fastExitAmount.mul(95).div(100)))

  //   // Update the user reward per share
  //   const updateRewardPerShareTX = await L1LiquidityPool.updateUserRewardPerShare(
  //     L1ERC20.address
  //   )
  //   await updateRewardPerShareTX.wait()

  //   // The user reward per share should be (10 * 0.035 / 200) * 10^12
  //   const updateRewardPerShare = await L1LiquidityPool.updateUserRewardPerShare(
  //     L1ERC20.address
  //   )
  //   await updateRewardPerShare.wait()
  //   const updatedPoolInfo = await L1LiquidityPool.poolInfo(
  //     L1ERC20.address
  //   )

  //   expect(updatedPoolInfo.lastAccUserReward).to.deep.eq(updatedPoolInfo.accUserReward)

  //   expect(updatedPoolInfo.accUserRewardPerShare).to.deep.eq(
  //     (fastExitAmount.mul(35).div(1000)).mul(BigNumber.from(10).pow(12)).div(poolInfo.userDepositAmount)
  //   )
  })

  it("should withdraw liquidity", async () => {

  //   const withdrawAmount = utils.parseEther("10")

  //   const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const preBobUserInfo = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address)

  //   const withdrawTX = await L2LiquidityPool.withdrawLiquidity(
  //     withdrawAmount,
  //     L2ERC20.address,
  //     env.bobl2Wallet.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await withdrawTX.wait()

  //   const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

  //   expect(preBobL2ERC20Balance).to.deep.eq(postBobL2ERC20Balance.sub(withdrawAmount))

  //   const postBobUserInfo = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address)
  //   const poolInfo = await L2LiquidityPool.poolInfo(L2ERC20.address)

  //   expect(preBobUserInfo.amount).to.deep.eq(postBobUserInfo.amount.add(withdrawAmount))

  //   expect(postBobUserInfo.rewardDebt).to.deep.eq(
  //     poolInfo.accUserRewardPerShare.mul(postBobUserInfo.amount).div(BigNumber.from(10).pow(12))
  //   )

  //   expect(postBobUserInfo.pendingReward).to.deep.eq(
  //     preBobUserInfo.amount.mul(poolInfo.accUserRewardPerShare).div(BigNumber.from(10).pow(12))
  //   )
  })

  it("shouldn't withdraw liquidity", async () => {
  //   const withdrawAmount = utils.parseEther("100")

  //   const withdrawTX = await L2LiquidityPool.withdrawLiquidity(
  //     withdrawAmount,
  //     L2ERC20.address,
  //     env.bobl2Wallet.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await expect(withdrawTX.wait()).to.be.eventually.rejected;
  })

  it("should withdraw reward", async () => {
  //   const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const preBobUserInfo = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address)
  //   const pendingReward = BigNumber.from(preBobUserInfo.pendingReward).div(2)

  //   const withdrawRewardTX = await L2LiquidityPool.withdrawReward(
  //     pendingReward,
  //     L2ERC20.address,
  //     env.bobl2Wallet.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await withdrawRewardTX.wait()

  //   const postBobUserInfo = await L2LiquidityPool.userInfo(
  //     L2ERC20.address,
  //     env.bobl2Wallet.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

  //   expect(postBobUserInfo.pendingReward).to.deep.eq(preBobUserInfo.pendingReward.sub(pendingReward))
  //   expect(preL2ERC20Balance).to.deep.eq(postL2ERC20Balance.sub(pendingReward))
  })

  it("shouldn't withdraw reward", async () => {
  //   const withdrawRewardAmount = utils.parseEther("100")

  //   const withdrawRewardTX = await L2LiquidityPool.withdrawReward(
  //     withdrawRewardAmount,
  //     L2ERC20.address,
  //     env.bobl2Wallet.address,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await expect(withdrawRewardTX.wait()).to.be.eventually.rejected;
  })

  it("should fast onramp", async () => {
  //   const depositAmount = utils.parseEther("10")

  //   const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
  //   const prePoolInfo = await L2LiquidityPool.poolInfo(L2ERC20.address)

  //   const approveL1LPTX = await L1ERC20.approve(
  //     L1LiquidityPool.address,
  //     depositAmount,
  //   )
  //   await approveL1LPTX.wait()

  //   await env.waitForXDomainTransaction(
  //     L1LiquidityPool.clientDepositL1(
  //       depositAmount,
  //       L1ERC20.address
  //     ),
  //     Direction.L1ToL2
  //   )

  //   const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //   const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
  //   const postPoolInfo = await L2LiquidityPool.poolInfo(L2ERC20.address)

  //   expect(postL2ERC20Balance).to.deep.eq(preL2ERC20Balance.add(depositAmount.mul(95).div(100)))

  //   expect(postL1ERC20Balance).to.deep.eq(preL1ERC20Balance.sub(depositAmount))

  //   expect(prePoolInfo.accUserReward).to.deep.eq(
  //     postPoolInfo.accUserReward.sub(depositAmount.mul(35).div(1000))
  //   )

  //   expect(prePoolInfo.accOwnerReward).to.deep.eq(
  //     postPoolInfo.accOwnerReward.sub(depositAmount.mul(15).div(1000))
  //   )
  })

  it("should revert unfulfillable swaps", async () => {

  //    const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
  //    const preBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
  //    const requestedLiquidity = (await L1ERC20.balanceOf(L1LiquidityPool.address)).add(1)
  //    const fastExitAmount = requestedLiquidity.mul(1000).div(950)

  //    const approveBobL2TX = await L2ERC20.connect(env.bobl2Wallet).approve(
  //      L2LiquidityPool.address,
  //      fastExitAmount,
  //      {gasLimit: 800000, gasPrice: 0}
  //    )
  //    await approveBobL2TX.wait()

  //    //await env.waitForRevertXDomainTransactionFast(
  //    await env.waitForRevertXDomainTransaction(
  //      L2LiquidityPool.connect(env.bobl2Wallet).clientDepositL2(
  //        fastExitAmount,
  //        L2ERC20.address,
  //        {gasLimit: 800000, gasPrice: 0}
  //      ),
  //      Direction.L2ToL1
  //    )

  //    const postBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
  //    const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

  //    expect(preBobL1ERC20Balance).to.deep.eq(postBobL1ERC20Balance)

  //    const exitFees = fastExitAmount.mul(50).div(1000)
  //    expect(postBobL2ERC20Balance).to.deep.eq(preBobL2ERC20Balance.sub(exitFees))
   })
})
