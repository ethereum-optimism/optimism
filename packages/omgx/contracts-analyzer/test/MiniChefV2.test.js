const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { bob, alice, carol, dev, minter } = require('./utilities/wallet');
const { deploy, getBigNumber, createSLP } = require('./utilities/index');

const ERC20MockJSON = require('../artifacts/contracts/mocks/ERC20Mock.sol/ERC20Mock.ovm.json');
const brokenRewarderJSON = require('../artifacts/contracts/mocks/RewarderBrokenMock.sol/RewarderBrokenMock.ovm.json');
const SushiTokenJSON = require('../artifacts/contracts/SushiToken.sol/SushiToken.ovm.json');
const MiniChefV2JSON = require('../artifacts/contracts/MiniChefV2.sol/MiniChefV2.ovm.json');
const RewarderMockJSON = require('../artifacts/contracts/mocks/RewarderMock.sol/RewarderMock.ovm.json');

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

describe("MiniChefV2", function () {
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
      ['chef', MiniChefV2JSON, [this.sushi.address]],
      ["rlp", ERC20MockJSON, ["LP", "rLPT", getBigNumber(10)]],
      ["r", ERC20MockJSON, ["Reward", "RewardT", getBigNumber(100000)]],
    ])

    await deploy(this, [["rewarder", RewarderMockJSON, [getBigNumber(1), this.r.address, this.chef.address]]])

    const mintTX = await this.sushi.mint(this.chef.address, getBigNumber(10000))
    await mintTX.wait()
    const approveTX = await this.lp.approve(this.chef.address, getBigNumber(10))
    await approveTX.wait()
    const setupTX = await this.chef.setSushiPerSecond("10000000000000000")
    await setupTX.wait()
    const transferTX = await this.rlp.transfer(alice.address, getBigNumber(1))
    await transferTX.wait()
  })

  describe("PoolLength", function () {
    it("PoolLength should execute", async function () {
      const addTX = await this.chef.add(10, this.rlp.address, this.rewarder.address)
      await addTX.wait()
      expect((await this.chef.poolLength())).to.be.equal(1);
    })
  })

  describe("Set", function() {
    it("Should emit event LogSetPool", async function () {
      const addTX = await this.chef.add(10, this.rlp.address, this.rewarder.address)
      await addTX.wait()
      await expect(this.chef.set(0, 10, this.dummy.address, false))
            .to.emit(this.chef, "LogSetPool")
            .withArgs(0, 10, this.rewarder.address, false)
      await expect(this.chef.set(0, 10, this.dummy.address, true))
            .to.emit(this.chef, "LogSetPool")
            .withArgs(0, 10, this.dummy.address, true)
      })

    it("Should revert if invalid pool", async function () {
      const setTX = await this.chef.set(0, 10, this.rewarder.address, false)
      await expect(setTX.wait()).to.be.eventually.rejected;
    })
  })

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

//   describe("PendingSushi", function() {
//     it("PendingSushi should equal ExpectedSushi", async function () {
//       await this.chef.add(10, this.rlp.address, this.rewarder.address)
//       await this.rlp.approve(this.chef.address, getBigNumber(10))
//       let log = await this.chef.deposit(0, getBigNumber(1), this.alice.address)
//       await advanceTime(86400)
//       let log2 = await this.chef.updatePool(0)
//       let timestamp2 = (await ethers.provider.getBlock(log2.blockNumber)).timestamp
//       let timestamp = (await ethers.provider.getBlock(log.blockNumber)).timestamp
//       let expectedSushi = BigNumber.from("10000000000000000").mul(timestamp2 - timestamp)
//       let pendingSushi = await this.chef.pendingSushi(0, this.alice.address)
//       expect(pendingSushi).to.be.equal(expectedSushi)
//     })
//     it("When time is lastRewardTime", async function () {
//       await this.chef.add(10, this.rlp.address, this.rewarder.address)
//       await this.rlp.approve(this.chef.address, getBigNumber(10))
//       let log = await this.chef.deposit(0, getBigNumber(1), this.alice.address)
//       await advanceBlockTo(3)
//       let log2 = await this.chef.updatePool(0)
//       let timestamp2 = (await ethers.provider.getBlock(log2.blockNumber)).timestamp
//       let timestamp = (await ethers.provider.getBlock(log.blockNumber)).timestamp
//       let expectedSushi = BigNumber.from("10000000000000000").mul(timestamp2 - timestamp)
//       let pendingSushi = await this.chef.pendingSushi(0, this.alice.address)
//       expect(pendingSushi).to.be.equal(expectedSushi)
//     })
//   })

//   describe("MassUpdatePools", function () {
//     it("Should call updatePool", async function () {
//       await this.chef.add(10, this.rlp.address, this.rewarder.address)
//       await advanceBlockTo(1)
//       await this.chef.massUpdatePools([0])
//       //expect('updatePool').to.be.calledOnContract(); //not suported by heardhat
//       //expect('updatePool').to.be.calledOnContractWith(0); //not suported by heardhat

//     })

//     it("Updating invalid pools should fail", async function () {
//       let err;
//       try {
//         await this.chef.massUpdatePools([0, 10000, 100000])
//       } catch (e) {
//         err = e;
//       }

//       assert.equal(err.toString(), "Error: VM Exception while processing transaction: invalid opcode")
//     })
// })

  describe("Add", function () {
    it("Should add pool with reward token multiplier", async function () {
      await expect(this.chef.add(10, this.rlp.address, this.rewarder.address))
            .to.emit(this.chef, "LogPoolAddition")
            .withArgs(0, 10, this.rlp.address, this.rewarder.address)
      })
  })

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

//   describe("UpdatePool", function () {
//     it("Should emit event LogUpdatePool", async function () {
//       await this.chef.add(10, this.rlp.address, this.rewarder.address)
//       await advanceBlockTo(1)
//       await expect(this.chef.updatePool(0))
//             .to.emit(this.chef, "LogUpdatePool")
//             .withArgs(0, (await this.chef.poolInfo(0)).lastRewardTime,
//               (await this.rlp.balanceOf(this.chef.address)),
//               (await this.chef.poolInfo(0)).accSushiPerShare)
//     })

//     it("Should take else path", async function () {
//       await this.chef.add(10, this.rlp.address, this.rewarder.address)
//       await advanceBlockTo(1)
//       await this.chef.batch(
//           [
//               this.chef.interface.encodeFunctionData("updatePool", [0]),
//               this.chef.interface.encodeFunctionData("updatePool", [0]),
//           ],
//           true
//       )
//     })
//   })

  describe("Deposit", function () {
    it("Depositing 0 amount", async function () {
      const addTX = await this.chef.add(10, this.rlp.address, this.rewarder.address)
      await addTX.wait()
      const approveTX = await this.rlp.approve(this.chef.address, getBigNumber(10))
      await approveTX.wait()
      await expect(this.chef.deposit(0, getBigNumber(0), bob.address))
            .to.emit(this.chef, "Deposit")
            .withArgs(bob.address, 0, 0, bob.address)
    })

    it("Depositing into non-existent pool should fail", async function () {
      const depositTX = await this.chef.deposit(1001, getBigNumber(0), bob.address)
      await expect(depositTX.wait()).to.be.eventually.rejected;
    })
  })

  describe("Withdraw", function () {
    it("Withdraw 0 amount", async function () {
      const addTX = await this.chef.add(10, this.rlp.address, this.rewarder.address)
      await addTX.wait()
      await expect(this.chef.withdraw(0, getBigNumber(0), bob.address))
            .to.emit(this.chef, "Withdraw")
            .withArgs(bob.address, 0, 0, bob.address)
    })
  })

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

//   describe("Harvest", function () {
//     it("Should give back the correct amount of SUSHI and reward", async function () {
//         await this.r.transfer(this.rewarder.address, getBigNumber(100000))
//         await this.chef.add(10, this.rlp.address, this.rewarder.address)
//         await this.rlp.approve(this.chef.address, getBigNumber(10))
//         expect(await this.chef.lpToken(0)).to.be.equal(this.rlp.address)
//         let log = await this.chef.deposit(0, getBigNumber(1), this.alice.address)
//         await advanceTime(86400)
//         let log2 = await this.chef.withdraw(0, getBigNumber(1), this.alice.address)
//         let timestamp2 = (await ethers.provider.getBlock(log2.blockNumber)).timestamp
//         let timestamp = (await ethers.provider.getBlock(log.blockNumber)).timestamp
//         let expectedSushi = BigNumber.from("10000000000000000").mul(timestamp2 - timestamp)
//         expect((await this.chef.userInfo(0, this.alice.address)).rewardDebt).to.be.equal("-"+expectedSushi)
//         await this.chef.harvest(0, this.alice.address)
//         expect(await this.sushi.balanceOf(this.alice.address)).to.be.equal(await this.r.balanceOf(this.alice.address)).to.be.equal(expectedSushi)
//     })
//     it("Harvest with empty user balance", async function () {
//       await this.chef.add(10, this.rlp.address, this.rewarder.address)
//       await this.chef.harvest(0, this.alice.address)
//     })

//     it("Harvest for SUSHI-only pool", async function () {
//       await this.chef.add(10, this.rlp.address, ADDRESS_ZERO)
//       await this.rlp.approve(this.chef.address, getBigNumber(10))
//       expect(await this.chef.lpToken(0)).to.be.equal(this.rlp.address)
//       let log = await this.chef.deposit(0, getBigNumber(1), this.alice.address)
//       await advanceBlock()
//       let log2 = await this.chef.withdraw(0, getBigNumber(1), this.alice.address)
//       let timestamp2 = (await ethers.provider.getBlock(log2.blockNumber)).timestamp
//       let timestamp = (await ethers.provider.getBlock(log.blockNumber)).timestamp
//       let expectedSushi = BigNumber.from("10000000000000000").mul(timestamp2 - timestamp)
//       expect((await this.chef.userInfo(0, this.alice.address)).rewardDebt).to.be.equal("-"+expectedSushi)
//       await this.chef.harvest(0, this.alice.address)
//       expect(await this.sushi.balanceOf(this.alice.address)).to.be.equal(expectedSushi)
//     })
//   })

  describe("EmergencyWithdraw", function() {
    it("Should emit event EmergencyWithdraw", async function () {
      const transferTX = await this.r.transfer(this.rewarder.address, getBigNumber(100000))
      await transferTX.wait()
      const addTX = await this.chef.add(10, this.rlp.address, this.rewarder.address)
      await addTX.wait()
      const approveTX = await this.rlp.approve(this.chef.address, getBigNumber(10))
      await approveTX.wait()
      const depositTX = await this.chef.deposit(0, getBigNumber(1), alice.address)
      await depositTX.wait()
      //await this.chef.emergencyWithdraw(0, this.alice.address)
      await expect(this.chef.connect(alice).emergencyWithdraw(0, alice.address))
      .to.emit(this.chef, "EmergencyWithdraw")
      .withArgs(alice.address, 0, getBigNumber(1), alice.address)
    })
  })
})