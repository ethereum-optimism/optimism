const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { bob, alice, carol, dev, minter } = require('./utilities/wallet');
const { deploy, getBigNumber, gasOptions, createSLP } = require('./utilities/index');

const ERC20MockJSON = require('../artifacts-ovm/contracts/mocks/ERC20Mock.sol/ERC20Mock.ovm.json');
const brokenRewarderJSON = require('../artifacts-ovm/contracts/mocks/RewarderBrokenMock.sol/RewarderBrokenMock.ovm.json');
const SushiTokenJSON = require('../artifacts-ovm/contracts/SushiToken.sol/SushiToken.ovm.json');
const MasterChefJSON = require('../artifacts-ovm/contracts/MasterChef.sol/MasterChef.ovm.json');
const MasterChefV2JSON = require('../artifacts-ovm/contracts/MasterChefV2.sol/MasterChefV2.ovm.json');
const RewarderMockJSON = require('../artifacts-ovm/contracts/mocks/RewarderMock.sol/RewarderMock.ovm.json');

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

describe("MasterChefV2", function () {
  before(async function () {
    await deploy(this, [
      ["brokenRewarder", brokenRewarderJSON]
    ])
  })

  beforeEach(async function () {
    await deploy(this, [
      ["sushi", SushiTokenJSON],
    ])

    await deploy(this,
      [["lp", ERC20MockJSON, ["LP Token", "LPT", getBigNumber(10)]],
      ["dummy", ERC20MockJSON, ["Dummy", "DummyT", getBigNumber(10)]],
      ['chef', MasterChefJSON, []]
    ])


    chefInitializeTx = await this.chef.initialize(this.sushi.address, alice.address, getBigNumber(100), "0", "0", gasOptions)
    await chefInitializeTx.wait()
    let transferTX, addTX, approveTX, depositTX, initTX
    transferTX = await this.sushi.transferOwnership(this.chef.address, gasOptions)
    await transferTX.wait()
    addTX = await this.chef.add(100, this.lp.address, true, gasOptions)
    await addTX.wait()
    addTX = await this.chef.add(100, this.dummy.address, true, gasOptions)
    await addTX.wait()
    approveTX = await this.lp.approve(this.chef.address, getBigNumber(10), gasOptions)
    await approveTX.wait()
    depositTX = await this.chef.deposit(0, getBigNumber(10), gasOptions)
    await depositTX.wait()

    await deploy(this, [
        ['chef2', MasterChefV2JSON, []],
        ["rlp", ERC20MockJSON, ["LP", "rLPT", getBigNumber(10)]],
        ["r", ERC20MockJSON, ["Reward", "RewardT", getBigNumber(100000)]],
    ])
    chef2InitTx = await this.chef2.initialize(this.chef.address, this.sushi.address, 1, gasOptions)
    await chef2InitTx.wait()
    await deploy(this, [["rewarder", RewarderMockJSON, []]])
    rewarderInitTx = await this.rewarder.initialize(getBigNumber(1), this.r.address, this.chef2.address, gasOptions)
    await rewarderInitTx.wait()
    approveTX = await this.dummy.approve(this.chef2.address, getBigNumber(10), gasOptions)
    await approveTX.wait()
    initTX = await this.chef2.init(this.dummy.address, gasOptions)
    await initTX.wait()
    transferTX = await this.rlp.transfer(alice.address, getBigNumber(1), gasOptions)
    await transferTX.wait()
  })

  describe("Init", function () {
    it("Balance of dummyToken should be 0 after init(), repeated execution should fail", async function () {
      await expect(this.chef2.init(this.dummy.address)).to.be.eventually.rejected;
    })
  })

  describe("PoolLength", function () {
    it("PoolLength should execute", async function () {
      const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions)
      await addTX.wait()
      expect((await this.chef2.poolLength())).to.equal(1);
    })
  })

  describe("Set", function() {
    it("Should emit event LogSetPool", async function () {
      const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions)
      await addTX.wait()
      await expect(this.chef2.set(0, 10, this.dummy.address, false, gasOptions))
            .to.emit(this.chef2, "LogSetPool")
            .withArgs(0, 10, this.rewarder.address, false)
      await expect(this.chef2.set(0, 10, this.dummy.address, true, gasOptions))
            .to.emit(this.chef2, "LogSetPool")
            .withArgs(0, 10, this.dummy.address, true)
      })

    it("Should revert if invalid pool", async function () {
      await expect(this.chef2.set(0, 10, this.rewarder.address, false)).to.be.eventually.rejected;
    })
  })

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

//   describe("PendingSushi", function() {
//     it("PendingSushi should equal ExpectedSushi", async function () {
//       await this.chef2.add(10, this.rlp.address, this.rewarder.address)
//       await this.rlp.approve(this.chef2.address, getBigNumber(10))
//       let log = await this.chef2.deposit(0, getBigNumber(1), this.alice.address)
//       await advanceBlock()
//       let log2 = await this.chef2.updatePool(0)
//       await advanceBlock()
//       let expectedSushi = getBigNumber(100).mul(log2.blockNumber + 1 - log.blockNumber).div(2)
//       let pendingSushi = await this.chef2.pendingSushi(0, this.alice.address)
//       expect(pendingSushi).to.be.equal(expectedSushi)
//     })
//     it("When block is lastRewardBlock", async function () {
//       await this.chef2.add(10, this.rlp.address, this.rewarder.address)
//       await this.rlp.approve(this.chef2.address, getBigNumber(10))
//       let log = await this.chef2.deposit(0, getBigNumber(1), this.alice.address)
//       await advanceBlockTo(3)
//       let log2 = await this.chef2.updatePool(0)
//       let expectedSushi = getBigNumber(100).mul(log2.blockNumber - log.blockNumber).div(2)
//       let pendingSushi = await this.chef2.pendingSushi(0, this.alice.address)
//       expect(pendingSushi).to.be.equal(expectedSushi)
//     })
//   })

  describe("MassUpdatePools", function () {
    it("Should call updatePool", async function () {
      const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions)
      await addTX.wait()
      const massUpdatePoolsTX = await this.chef2.massUpdatePools([0], gasOptions)
      await massUpdatePoolsTX.wait()
      //expect('updatePool').to.be.calledOnContract(); //not suported by heardhat
      //expect('updatePool').to.be.calledOnContractWith(0); //not suported by heardhat

    })

    it("Updating invalid pools should fail", async function () {
      await expect(this.chef2.set(0, 10, this.rewarder.address, false)).to.be.eventually.rejected;
    })
})

  describe("Add", function () {
    it("Should add pool with reward token multiplier", async function () {
      await expect(this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions))
            .to.emit(this.chef2, "LogPoolAddition")
            .withArgs(0, 10, this.rlp.address, this.rewarder.address)
      })
  })

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/
  // describe("UpdatePool", function () {
  //   it("Should emit event LogUpdatePool", async function () {
  //     const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address)
  //     await addTX.wait()
  //     await advanceBlockTo(1)
  //     await expect(this.chef2.updatePool(0))
  //           .to.emit(this.chef2, "LogUpdatePool")
  //           .withArgs(0, (await this.chef2.poolInfo(0)).lastRewardBlock,
  //             (await this.rlp.balanceOf(this.chef2.address)),
  //             (await this.chef2.poolInfo(0)).accSushiPerShare)
  //   })

  //   it("Should take else path", async function () {
  //     const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address)
  //     await addTX.wait()
  //     await advanceBlockTo(1)
  //     const batchTX = await this.chef2.batch(
  //         [
  //             this.chef2.interface.encodeFunctionData("updatePool", [0]),
  //             this.chef2.interface.encodeFunctionData("updatePool", [0]),
  //         ],
  //         true
  //     )
  //     await batchTX.wait()
  //   })
  // })

  describe("Deposit", function () {
    it("Depositing 0 amount", async function () {
      const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions)
      await addTX.wait()
      const approveTX = await this.rlp.approve(this.chef2.address, getBigNumber(10), gasOptions)
      await approveTX.wait()
      await expect(this.chef2.deposit(0, getBigNumber(0), bob.address, gasOptions))
            .to.emit(this.chef2, "Deposit")
            .withArgs(bob.address, 0, 0, bob.address)
    })

    it("Depositing into non-existent pool should fail", async function () {
      await expect(this.chef2.deposit(1001, getBigNumber(0), alice.address)).to.be.eventually.rejected;
    })
  })

  describe("Withdraw", function () {
    it("Withdraw 0 amount", async function () {
      const addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions)
      await addTX.wait()
      await expect(this.chef2.withdraw(0, getBigNumber(0), bob.address, gasOptions))
            .to.emit(this.chef2, "Withdraw")
            .withArgs(bob.address, 0, 0, bob.address)
    })
  })

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/
//   describe("Harvest", function () {
//     it("Should give back the correct amount of SUSHI and reward", async function () {
//         await this.r.transfer(this.rewarder.address, getBigNumber(100000))
//         await this.chef2.add(10, this.rlp.address, this.rewarder.address)
//         await this.rlp.approve(this.chef2.address, getBigNumber(10))
//         expect(await this.chef2.lpToken(0)).to.be.equal(this.rlp.address)
//         let log = await this.chef2.deposit(0, getBigNumber(1), this.alice.address)
//         await advanceBlockTo(20)
//         await this.chef2.harvestFromMasterChef()
//         let log2 = await this.chef2.withdraw(0, getBigNumber(1), this.alice.address)
//         let expectedSushi = getBigNumber(100).mul(log2.blockNumber - log.blockNumber).div(2)
//         expect((await this.chef2.userInfo(0, this.alice.address)).rewardDebt).to.be.equal("-"+expectedSushi)
//         await this.chef2.harvest(0, this.alice.address)
//         expect(await this.sushi.balanceOf(this.alice.address)).to.be.equal(await this.r.balanceOf(this.alice.address)).to.be.equal(expectedSushi)
//     })
//     it("Harvest with empty user balance", async function () {
//       await this.chef2.add(10, this.rlp.address, this.rewarder.address)
//       await this.chef2.harvest(0, this.alice.address)
//     })

//     it("Harvest for SUSHI-only pool", async function () {
//       await this.chef2.add(10, this.rlp.address, ADDRESS_ZERO)
//       await this.rlp.approve(this.chef2.address, getBigNumber(10))
//       expect(await this.chef2.lpToken(0)).to.be.equal(this.rlp.address)
//       let log = await this.chef2.deposit(0, getBigNumber(1), this.alice.address)
//       await advanceBlock()
//       await this.chef2.harvestFromMasterChef()
//       let log2 = await this.chef2.withdraw(0, getBigNumber(1), this.alice.address)
//       let expectedSushi = getBigNumber(100).mul(log2.blockNumber - log.blockNumber).div(2)
//       expect((await this.chef2.userInfo(0, this.alice.address)).rewardDebt).to.be.equal("-"+expectedSushi)
//       await this.chef2.harvest(0, this.alice.address)
//       expect(await this.sushi.balanceOf(this.alice.address)).to.be.equal(expectedSushi)
//     })
//   })

  describe("EmergencyWithdraw", function() {
    it("Should emit event EmergencyWithdraw", async function () {
      let transferTX, addTX, approveTX, depositTX
      transferTX = await this.r.transfer(this.rewarder.address, getBigNumber(100000), gasOptions)
      await transferTX.wait()
      addTX = await this.chef2.add(10, this.rlp.address, this.rewarder.address, gasOptions)
      await addTX.wait()
      approveTX = await this.rlp.approve(this.chef2.address, getBigNumber(10), gasOptions)
      await approveTX.wait()
      depositTX = await this.chef2.deposit(0, getBigNumber(1), alice.address, gasOptions)
      await depositTX.wait()
      //await this.chef2.emergencyWithdraw(0, this.alice.address)
      await expect(this.chef2.connect(alice).emergencyWithdraw(0, alice.address, gasOptions))
      .to.emit(this.chef2, "EmergencyWithdraw")
      .withArgs(alice.address, 0, getBigNumber(1), alice.address)
    })
  })
})
