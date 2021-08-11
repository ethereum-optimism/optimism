import { expect } from 'chai'
import chai from 'chai';
import chaiAsPromised from 'chai-as-promised';
chai.use(chaiAsPromised);
import { Contract, ContractFactory, BigNumber, utils, ethers } from 'ethers'
import { Direction } from './shared/watcher-utils'
import { expectLogs } from './shared/utils'
import { getContractFactory } from '@eth-optimism/contracts';

import L1ERC20Json from '../contracts/L1ERC20.json'
import L1LiquidityPoolJson from '../contracts/L1LiquidityPool.json'
import L2LiquidityPoolJson from '../contracts/L2LiquidityPool.json'
import L2TokenPoolJson from '../contracts/TokenPool.json'

import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('Liquidity Pool Test', async () => {

  let Factory__L1ERC20: ContractFactory
  let Factory__L2ERC20: ContractFactory

  let L1LiquidityPool: Contract
  let L2LiquidityPool: Contract
  let L1ERC20: Contract
  let L2ERC20: Contract
  let L1StandardBridge: Contract
  let L2TokenPool: Contract

  let env: OptimismEnv

  const initialSupply = utils.parseEther('10000000000')
  const tokenName = 'JLKN'
  const tokenSymbol = 'JLKN'

  before(async () => {

    env = await OptimismEnv.new()

    Factory__L1ERC20 = new ContractFactory(
      L1ERC20Json.abi,
      L1ERC20Json.bytecode,
      env.bobl1Wallet
    )

    const L1StandardBridgeAddress = await env.addressManager.getAddress('Proxy__OVM_L1StandardBridge')

    L1StandardBridge = getContractFactory(
      "OVM_L1StandardBridge",
      env.bobl1Wallet
    ).attach(L1StandardBridgeAddress)

    const L2StandardBridgeAddress = await L1StandardBridge.l2TokenBridge()

    //we deploy a new erc20, so tests won't fail on a rerun on the same contracts
    L1ERC20 = await Factory__L1ERC20.deploy(
      initialSupply,
      tokenName,
      tokenSymbol
    )
    await L1ERC20.deployTransaction.wait()

    Factory__L2ERC20 = getContractFactory(
      "L2StandardERC20",
      env.bobl2Wallet,
      true
    )

    L2ERC20 = await Factory__L2ERC20.deploy(
      L2StandardBridgeAddress,
      L1ERC20.address,
      tokenName,
      tokenSymbol,
      {gasLimit: 85390000}
    )
    await L2ERC20.deployTransaction.wait()

    L1LiquidityPool = new Contract(
      env.addressesOMGX.Proxy__L1LiquidityPool,
      L1LiquidityPoolJson.abi,
      env.bobl1Wallet
    )

    L2LiquidityPool = new Contract(
      env.addressesOMGX.Proxy__L2LiquidityPool,
      L2LiquidityPoolJson.abi,
      env.bobl2Wallet
    )

    L2TokenPool = new Contract(
      env.addressesOMGX.L2TokenPool,
      L2TokenPoolJson.abi,
      env.bobl2Wallet
    )

  })

  it('should deposit 10000 TEST ERC20 token from L1 to L2', async () => {

    const depositL2ERC20Amount = utils.parseEther("10000");

    const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

    const approveL1ERC20TX = await L1ERC20.approve(
      L1StandardBridge.address,
      depositL2ERC20Amount
    )
    await approveL1ERC20TX.wait()

    await env.waitForXDomainTransaction(
      L1StandardBridge.depositERC20(
        L1ERC20.address,
        L2ERC20.address,
        depositL2ERC20Amount,
        9999999,
        ethers.utils.formatBytes32String((new Date().getTime()).toString())
      ),
      Direction.L1ToL2
    )

    const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

    expect(preL1ERC20Balance).to.deep.eq(
      postL1ERC20Balance.add(depositL2ERC20Amount)
    )

    expect(preL2ERC20Balance).to.deep.eq(
      postL2ERC20Balance.sub(depositL2ERC20Amount)
    )
  })

  it('should transfer L2 ERC20 TEST token from Bob to Alice and Kate', async () => {

    const transferL2ERC20Amount = utils.parseEther("150")

    const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const preAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)
    const preKateL2ERC20Balance = await L2ERC20.balanceOf(env.katel2Wallet.address)

    const tranferToAliceTX = await L2ERC20.transfer(
      env.alicel2Wallet.address,
      transferL2ERC20Amount,
      {gasLimit: 7000000}
    )
    await tranferToAliceTX.wait()

    const transferToKateTX = await L2ERC20.transfer(
      env.katel2Wallet.address,
      transferL2ERC20Amount,
      {gasLimit: 7000000}
    )
    await transferToKateTX.wait()

    const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const postAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)
    const postKateL2ERC20Balance = await L2ERC20.balanceOf(env.katel2Wallet.address)

    expect(preBobL2ERC20Balance).to.deep.eq(
      postBobL2ERC20Balance.add(transferL2ERC20Amount).add(transferL2ERC20Amount)
    )

    expect(preAliceL2ERC20Balance).to.deep.eq(
      postAliceL2ERC20Balance.sub(transferL2ERC20Amount)
    )

    expect(preKateL2ERC20Balance).to.deep.eq(
      postKateL2ERC20Balance.sub(transferL2ERC20Amount)
    )
  })

  it('should add 1000 ERC20 TEST tokens to the L2 token pool', async () => {

    const addL2TPAmount = utils.parseEther("1000")

    const approveL2TPTX = await L2ERC20.approve(
      L2TokenPool.address,
      addL2TPAmount,
      {gasLimit: 7000000}
    )
    await approveL2TPTX.wait()

    const transferL2TPTX = await L2ERC20.transfer(
      L2TokenPool.address,
      addL2TPAmount,
      {gasLimit: 7000000}
    );
    await transferL2TPTX.wait()

    const L2TPBalance = await L2ERC20.balanceOf(L2TokenPool.address)

    expect(L2TPBalance).to.deep.eq(addL2TPAmount)
  })

  it('should register L1 the pool', async () => {

    const registerPoolERC20TX = await L1LiquidityPool.registerPool(
      L1ERC20.address,
      L2ERC20.address,
    )
    await registerPoolERC20TX.wait()

    const poolERC20Info = await L1LiquidityPool.poolInfo(L1ERC20.address)

    expect(poolERC20Info.l1TokenAddress).to.deep.eq(L1ERC20.address)
    expect(poolERC20Info.l2TokenAddress).to.deep.eq(L2ERC20.address)

    const poolETHInfo = await L1LiquidityPool.poolInfo("0x0000000000000000000000000000000000000000")

    expect(poolETHInfo.l1TokenAddress).to.deep.eq("0x0000000000000000000000000000000000000000")
    expect(poolETHInfo.l2TokenAddress).to.deep.eq(env.l2ETHAddress)
  })

  it('should register L2 the pool', async () => {

    const registerPoolERC20TX = await L2LiquidityPool.registerPool(
      L1ERC20.address,
      L2ERC20.address
    )
    await registerPoolERC20TX.wait()

    const poolERC20Info = await L2LiquidityPool.poolInfo(L2ERC20.address)

    expect(poolERC20Info.l1TokenAddress).to.deep.eq(L1ERC20.address)
    expect(poolERC20Info.l2TokenAddress).to.deep.eq(L2ERC20.address)

    const poolETHInfo = await L2LiquidityPool.poolInfo(env.l2ETHAddress)

    expect(poolETHInfo.l1TokenAddress).to.deep.eq("0x0000000000000000000000000000000000000000")
    expect(poolETHInfo.l2TokenAddress).to.deep.eq(env.l2ETHAddress)
  })

  it('shouldn\'t update the pool', async () => {
    const registerPoolTX = await L2LiquidityPool.registerPool(
      L1ERC20.address,
      L2ERC20.address,
      {gasLimit: 7000000}
    )
    await expect(registerPoolTX.wait()).to.be.eventually.rejected;
  })

  it('should add L1 liquidity', async () => {
    const addLiquidityAmount = utils.parseEther("100")

    const preBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)

    const approveBobL1TX = await L1ERC20.approve(
      L1LiquidityPool.address,
      addLiquidityAmount,
    )
    await approveBobL1TX.wait()

    const BobAddLiquidity = await L1LiquidityPool.addLiquidity(
        addLiquidityAmount,
        L1ERC20.address
    )
    await BobAddLiquidity.wait()

    // ERC20 balance
    const postBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)

    expect(preBobL1ERC20Balance).to.deep.eq(
      postBobL1ERC20Balance.add(addLiquidityAmount)
    )

    // Pool Balance
    const L1LPERC20Balance = await L1ERC20.balanceOf(L1LiquidityPool.address)

    expect(L1LPERC20Balance).to.deep.eq(addLiquidityAmount)
  })

  it('should add L2 liquidity', async () => {

    const addLiquidityAmount = utils.parseEther("100")

    const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const preAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)

    const approveBobL2TX = await L2ERC20.approve(
      L2LiquidityPool.address,
      addLiquidityAmount,
      {gasLimit: 7000000}
    )
    await approveBobL2TX.wait()

    const BobAddLiquidity = await L2LiquidityPool.addLiquidity(
      addLiquidityAmount,
      L2ERC20.address,
      {gasLimit: 7000000}
    )
    await BobAddLiquidity.wait()

    const approveAliceL2TX = await L2ERC20.connect(env.alicel2Wallet).approve(
      L2LiquidityPool.address,
      addLiquidityAmount,
      {gasLimit: 7000000}
    )
    await approveAliceL2TX.wait()

    const AliceAddLiquidity = await L2LiquidityPool.connect(env.alicel2Wallet).addLiquidity(
      addLiquidityAmount,
      L2ERC20.address,
      {gasLimit: 7000000}
    )
    await AliceAddLiquidity.wait()

    // ERC20 balance
    const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const postAliceL2ERC20Balance = await L2ERC20.balanceOf(env.alicel2Wallet.address)

    expect(preBobL2ERC20Balance).to.deep.eq(
      postBobL2ERC20Balance.add(addLiquidityAmount)
    )
    expect(preAliceL2ERC20Balance).to.deep.eq(
      postAliceL2ERC20Balance.add(addLiquidityAmount)
    )

    // User deposit amount
    const BobPoolAmount = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address);
    const AlicePoolAmount = await L2LiquidityPool.userInfo(L2ERC20.address, env.alicel2Wallet.address);

    expect(BobPoolAmount.amount).to.deep.eq(addLiquidityAmount)
    expect(AlicePoolAmount.amount).to.deep.eq(addLiquidityAmount)

    // Pool Balance
    const L2LPERC20Balance = await L2ERC20.balanceOf(L2LiquidityPool.address)

    expect(L2LPERC20Balance).to.deep.eq(addLiquidityAmount.mul(2))
  })

  it("should fast exit L2", async () => {

    const fastExitAmount = utils.parseEther("10")

    const preKateL1ERC20Balance = await L1ERC20.balanceOf(env.katel1Wallet.address)

    const approveKateL2TX = await L2ERC20.connect(env.katel2Wallet).approve(
      L2LiquidityPool.address,
      fastExitAmount,
      {gasLimit: 7000000}
    )
    await approveKateL2TX.wait()

    const depositTx = await env.waitForXDomainTransactionFast(
      L2LiquidityPool.connect(env.katel2Wallet).clientDepositL2(
        fastExitAmount,
        L2ERC20.address,
        {gasLimit: 7000000}
      ),
      Direction.L2ToL1
    )

    const poolInfo = await L1LiquidityPool.poolInfo(L1ERC20.address)

    expect(poolInfo.accOwnerReward).to.deep.eq(fastExitAmount.mul(15).div(1000))
    expect(poolInfo.accUserReward).to.deep.eq(fastExitAmount.mul(35).div(1000))
    expect(poolInfo.userDepositAmount).to.deep.eq(utils.parseEther("100"))

    const postKateL1ERC20Balance = await L1ERC20.balanceOf(env.katel1Wallet.address)

    expect(postKateL1ERC20Balance).to.deep.eq(preKateL1ERC20Balance.add(fastExitAmount.mul(95).div(100)))

    // Update the user reward per share
    const updateRewardPerShareTX = await L1LiquidityPool.updateUserRewardPerShare(
      L1ERC20.address
    )
    await updateRewardPerShareTX.wait()

    // The user reward per share should be (10 * 0.035 / 200) * 10^12
    const updateRewardPerShare = await L1LiquidityPool.updateUserRewardPerShare(
      L1ERC20.address
    )
    await updateRewardPerShare.wait()
    const updatedPoolInfo = await L1LiquidityPool.poolInfo(
      L1ERC20.address
    )

    expect(updatedPoolInfo.lastAccUserReward).to.deep.eq(updatedPoolInfo.accUserReward)

    expect(updatedPoolInfo.accUserRewardPerShare).to.deep.eq(
      (fastExitAmount.mul(35).div(1000)).mul(BigNumber.from(10).pow(12)).div(poolInfo.userDepositAmount)
    )

    // check event ClientDepositL2 is emitted
    await expectLogs(depositTx.receipt,L2LiquidityPoolJson.abi,L2LiquidityPool.address, 'ClientDepositL2', {
      sender: env.katel2Wallet.address,
      receivedAmount: fastExitAmount,
      tokenAddress: L2ERC20.address,
    })

    // check event ClientPayL1 is emitted
    await expectLogs(depositTx.remoteReceipt,L1LiquidityPoolJson.abi,L1LiquidityPool.address, 'ClientPayL1', {
      sender: env.katel2Wallet.address,
      amount: fastExitAmount.mul(95).div(100),
      tokenAddress: L1ERC20.address
    })
  })

  it("should withdraw liquidity", async () => {

    const withdrawAmount = utils.parseEther("10")

    const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const preBobUserInfo = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address)

    const withdrawTX = await L2LiquidityPool.withdrawLiquidity(
      withdrawAmount,
      L2ERC20.address,
      env.bobl2Wallet.address,
      {gasLimit: 7000000}
    )
    await withdrawTX.wait()

    const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

    expect(preBobL2ERC20Balance).to.deep.eq(postBobL2ERC20Balance.sub(withdrawAmount))

    const postBobUserInfo = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address)
    const poolInfo = await L2LiquidityPool.poolInfo(L2ERC20.address)

    expect(preBobUserInfo.amount).to.deep.eq(postBobUserInfo.amount.add(withdrawAmount))

    expect(postBobUserInfo.rewardDebt).to.deep.eq(
      poolInfo.accUserRewardPerShare.mul(postBobUserInfo.amount).div(BigNumber.from(10).pow(12))
    )

    expect(postBobUserInfo.pendingReward).to.deep.eq(
      preBobUserInfo.amount.mul(poolInfo.accUserRewardPerShare).div(BigNumber.from(10).pow(12))
    )
  })

  it("shouldn't withdraw liquidity", async () => {
    const withdrawAmount = utils.parseEther("100")

    const withdrawTX = await L2LiquidityPool.withdrawLiquidity(
      withdrawAmount,
      L2ERC20.address,
      env.bobl2Wallet.address,
      {gasLimit: 7000000}
    )
    await expect(withdrawTX.wait()).to.be.eventually.rejected;
  })

  it("should withdraw reward from L2 pool", async () => {

    const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const preBobUserInfo = await L2LiquidityPool.userInfo(L2ERC20.address, env.bobl2Wallet.address)
    const pendingReward = BigNumber.from(preBobUserInfo.pendingReward).div(2)

    const withdrawRewardTX = await L2LiquidityPool.withdrawReward(
      pendingReward,
      L2ERC20.address,
      env.bobl2Wallet.address,
      {gasLimit: 7000000}
    )
    await withdrawRewardTX.wait()

    const postBobUserInfo = await L2LiquidityPool.userInfo(
      L2ERC20.address,
      env.bobl2Wallet.address,
      {gasLimit: 7000000}
    )
    const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

    expect(postBobUserInfo.pendingReward).to.deep.eq(preBobUserInfo.pendingReward.sub(pendingReward))
    expect(preL2ERC20Balance).to.deep.eq(postL2ERC20Balance.sub(pendingReward))
  })

  it("should withdraw reward from L1 pool", async () => {

    const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const preBobUserInfo = await L1LiquidityPool.userInfo(L1ERC20.address, env.bobl1Wallet.address)
    const prePoolInfo = await L1LiquidityPool.poolInfo(L1ERC20.address)
    const pendingReward = BigNumber.from(preBobUserInfo.pendingReward).add(
      BigNumber.from(preBobUserInfo.amount)
      .mul(BigNumber.from(prePoolInfo.accUserRewardPerShare))
      .div(BigNumber.from(10).pow(BigNumber.from(12)))
      .sub(BigNumber.from(preBobUserInfo.rewardDebt))
    )

    const withdrawRewardTX = await L1LiquidityPool.withdrawReward(
      pendingReward,
      L1ERC20.address,
      env.bobl1Wallet.address//,
      //{gasLimit: 800000}
    )
    await withdrawRewardTX.wait()

    const postBobUserInfo = await L1LiquidityPool.userInfo(
      L1ERC20.address,
      env.bobl1Wallet.address//,
      //{gasLimit: 800000}
    )
    const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)

    expect(postBobUserInfo.pendingReward).to.deep.eq(BigNumber.from(0))
    expect(preL1ERC20Balance).to.deep.eq(postL1ERC20Balance.sub(pendingReward))
  })

  it("shouldn't withdraw reward from L2 pool", async () => {
    const withdrawRewardAmount = utils.parseEther("100")

    const withdrawRewardTX = await L2LiquidityPool.withdrawReward(
      withdrawRewardAmount,
      L2ERC20.address,
      env.bobl2Wallet.address,
      {gasLimit: 7000000}
    )
    await expect(withdrawRewardTX.wait()).to.be.eventually.rejected;
  })

  it("should fast onramp", async () => {
    const depositAmount = utils.parseEther("10")

    const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const prePoolInfo = await L2LiquidityPool.poolInfo(L2ERC20.address)

    const approveL1LPTX = await L1ERC20.approve(
      L1LiquidityPool.address,
      depositAmount,
      {gasLimit: 9000000}
    )
    await approveL1LPTX.wait()

    const depositTx = await env.waitForXDomainTransaction(
      L1LiquidityPool.clientDepositL1(
        depositAmount,
        L1ERC20.address,
        {gasLimit: 9000000}
      ),
      Direction.L1ToL2
    )

    const postL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
    const postL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
    const postPoolInfo = await L2LiquidityPool.poolInfo(L2ERC20.address)

    expect(postL2ERC20Balance).to.deep.eq(preL2ERC20Balance.add(depositAmount.mul(95).div(100)))

    expect(postL1ERC20Balance).to.deep.eq(preL1ERC20Balance.sub(depositAmount))

    expect(prePoolInfo.accUserReward).to.deep.eq(
      postPoolInfo.accUserReward.sub(depositAmount.mul(35).div(1000))
    )

    expect(prePoolInfo.accOwnerReward).to.deep.eq(
      postPoolInfo.accOwnerReward.sub(depositAmount.mul(15).div(1000))
    )

    // check event ClientDepositL1 is emitted
    await expectLogs(depositTx.receipt,L1LiquidityPoolJson.abi,L1LiquidityPool.address, 'ClientDepositL1', {
      sender: env.bobl1Wallet.address,
      receivedAmount: depositAmount,
      tokenAddress: L1ERC20.address,
    })

    // check event ClientPayL2 is emitted
    await expectLogs(depositTx.remoteReceipt,L2LiquidityPoolJson.abi,L2LiquidityPool.address, 'ClientPayL2', {
      sender: env.bobl1Wallet.address,
      amount: depositAmount.mul(95).div(100),
      tokenAddress: L2ERC20.address
    })
  })

  it("should revert unfulfillable swap-offs", async () => {

     const preBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
     const preBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
     const requestedLiquidity = (await L1ERC20.balanceOf(L1LiquidityPool.address)).add(10)
     const fastExitAmount = requestedLiquidity.mul(100).div(95)

     const approveBobL2TX = await L2ERC20.connect(env.bobl2Wallet).approve(
       L2LiquidityPool.address,
       fastExitAmount,
       {gasLimit: 7000000}
     )
     await approveBobL2TX.wait()

    await env.waitForRevertXDomainTransactionFast(
       L2LiquidityPool.connect(env.bobl2Wallet).clientDepositL2(
         fastExitAmount,
         L2ERC20.address,
         {gasLimit: 7000000}
       ),
       Direction.L2ToL1
     )

     const postBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
     const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

     expect(preBobL1ERC20Balance).to.deep.eq(postBobL1ERC20Balance)

     const exitFees = fastExitAmount.mul(5).div(100)
     expect(postBobL2ERC20Balance).to.deep.eq(preBobL2ERC20Balance.sub(exitFees))
   })

   it("should revert unfulfillable swap-ons", async () => {

      const preL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)
      const preL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)

      const requestedLiquidity = (await L2ERC20.balanceOf(L2LiquidityPool.address)).add(10)
      const swapOnAmount = requestedLiquidity.mul(100).div(95)

      const approveBobL1TX = await L1ERC20.connect(env.bobl1Wallet).approve(
        L1LiquidityPool.address,
        swapOnAmount
      )
      await approveBobL1TX.wait()

      await env.waitForRevertXDomainTransaction(
        L1LiquidityPool.clientDepositL1(
          swapOnAmount,
          L1ERC20.address
        ),
        Direction.L1ToL2
      )

      const postBobL1ERC20Balance = await L1ERC20.balanceOf(env.bobl1Wallet.address)
      const postBobL2ERC20Balance = await L2ERC20.balanceOf(env.bobl2Wallet.address)

      const swapOnFees = swapOnAmount.mul(5).div(100)

      expect(preL2ERC20Balance).to.deep.eq(postBobL2ERC20Balance)
      expect(postBobL1ERC20Balance).to.deep.eq(preL1ERC20Balance.sub(swapOnFees))
   })
})
