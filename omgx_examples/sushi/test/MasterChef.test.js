
const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { expect } = require("chai");
const { gasOptions } = require("./utilities/index")
const { advanceBlockTo } = require("./utilities/time");
const { bob, alice, carol, dev, minter } = require('./utilities/wallet');

const MasterChefJSON = require('../artifacts-ovm/contracts/MasterChef.sol/MasterChef.ovm.json');
const SushiTokenJSON = require('../artifacts-ovm/contracts/SushiToken.sol/SushiToken.ovm.json');
const ERC20MockJSON = require('../artifacts-ovm/contracts/mocks/ERC20Mock.sol/ERC20Mock.ovm.json');

/******************************************************************/
/******************** evm_mint is not supported *******************/
/******************************************************************/

describe("MasterChef", function () {
  before(async function () {

    this.Factory__MasterChef = new ContractFactory(
      MasterChefJSON.abi,
      MasterChefJSON.bytecode,
      bob,
    )

    this.Factory__SushiToken = new ContractFactory(
      SushiTokenJSON.abi,
      SushiTokenJSON.bytecode,
      bob,
    )

    this.Factory__ERC20Mock = new ContractFactory(
      ERC20MockJSON.abi,
      ERC20MockJSON.bytecode,
      minter,
    )
  })

  beforeEach(async function () {
    this.sushi = await this.Factory__SushiToken.deploy(gasOptions)
    await this.sushi.deployTransaction.wait()
  })

  it("should set correct state variables", async function () {
    this.chef = await this.Factory__MasterChef.deploy()
    await this.chef.deployTransaction.wait()

    const chefInitializeTx = await this.chef.initialize(this.sushi.address, dev.address, "1000", "0", "1000", gasOptions)
    await chefInitializeTx.wait()

    const ownershipTX = await this.sushi.transferOwnership(this.chef.address, gasOptions)
    await ownershipTX.wait()

    const sushi = await this.chef.sushi()
    const devaddr = await this.chef.devaddr()
    const owner = await this.sushi.owner()

    expect(sushi).to.equal(this.sushi.address)
    expect(devaddr).to.equal(dev.address)
    expect(owner).to.equal(this.chef.address)
  })

  it("should allow dev and only dev to update dev", async function () {
    this.chef = await this.Factory__MasterChef.deploy()
    await this.chef.deployTransaction.wait()

    const chefInitializeTx = await this.chef.initialize(this.sushi.address, dev.address, "1000", "0", "1000", gasOptions)
    await chefInitializeTx.wait()

    expect(await this.chef.devaddr()).to.equal(dev.address)

    let ownershipTX = await this.chef.connect(dev).dev(bob.address, { from: dev.address, ...gasOptions})
    await ownershipTX.wait()
    expect(await this.chef.devaddr()).to.equal(bob.address)

    ownershipTX = await this.chef.connect(bob).dev(alice.address, { from: bob.address, ...gasOptions})
    await ownershipTX.wait()
    expect(await this.chef.devaddr()).to.equal(alice.address)
  })

  /******************************************************************/
  /******************** evm_mint is not supported *******************/
  /******************************************************************/

  context("With ERC/LP token added to the field", function () {
    beforeEach(async function () {
      this.lp = await this.Factory__ERC20Mock.deploy("LPToken", "LP", "10000000000",gasOptions)

      let tx
      tx = await this.lp.transfer(alice.address, "1000", gasOptions)
      await tx.wait()

      tx = await this.lp.transfer(bob.address, "1000", gasOptions)
      await tx.wait()

      tx = await this.lp.transfer(carol.address, "1000", gasOptions)
      await tx.wait()

      this.lp2 = await this.Factory__ERC20Mock.deploy("LPToken2", "LP2", "10000000000", gasOptions)

      tx = await this.lp2.transfer(alice.address, "1000", gasOptions)
      await tx.wait()

      tx = await this.lp2.transfer(bob.address, "1000", gasOptions)
      await tx.wait()

      tx = await this.lp2.transfer(carol.address, "1000", gasOptions)
      await tx.wait()
    })

    it("should allow emergency withdraw", async function () {
      // 100 per block farming rate starting at block 100 with bonus until block 1000
      this.chef = await this.Factory__MasterChef.deploy()
      await this.chef.deployTransaction.wait()

      const chefInitializeTx = await this.chef.initialize(this.sushi.address, dev.address, "100", "100", "1000",gasOptions)
      await chefInitializeTx.wait()

      const addTX = await this.chef.add("100", this.lp.address, true,gasOptions)
      await addTX.wait()

      const approveTX = await this.lp.connect(bob).approve(this.chef.address, "1000",gasOptions)
      await approveTX.wait()

      const depositTX = await this.chef.connect(bob).deposit(0, "100",gasOptions)
      await depositTX.wait()

      expect(await this.lp.balanceOf(bob.address)).to.equal("900")

      const withdrawTX = await this.chef.connect(bob).emergencyWithdraw(0,gasOptions)
      await withdrawTX.wait()

      expect(await this.lp.balanceOf(bob.address)).to.equal("1000")
    })

  /******************************************************************/
  /******************** evm_mint is not supported *******************/
  /******************************************************************/
    // it("should give out SUSHIs only after farming time", async function () {
    //   // 100 per block farming rate starting at block 100 with bonus until block 1000
    //   this.chef = await this.Factory__MasterChef.deploy(this.sushi.address, dev.address, "100", "100", "1000")
    //   await this.chef.deployTransaction.wait()

    //   const tx = await this.sushi.transferOwnership(this.chef.address)
    //   await tx.wait()

    //   const add = await this.chef.add("100", this.lp.address, true)
    //   await add.wait()

    //   const approve = await this.lp.connect(bob).approve(this.chef.address, "1000")
    //   await approve.wait()

    //   let deposit
    //   deposit = await this.chef.connect(bob).deposit(0, "100")
    //   await deposit.wait()
    //   await advanceBlockTo("89")

    //   deposit = await this.chef.connect(bob).deposit(0, "0") // block 90
    //   await deposit.wait()
    //   expect(await this.sushi.balanceOf(bob.address)).to.equal("0")
    //   await advanceBlockTo("94")

    //   deposit = await this.chef.connect(bob).deposit(0, "0") // block 95
    //   await deposit.wait()
    //   expect(await this.sushi.balanceOf(bob.address)).to.equal("0")
    //   await advanceBlockTo("99")

    //   deposit = await this.chef.connect(bob).deposit(0, "0") // block 100
    //   await deposit.wait()
    //   expect(await this.sushi.balanceOf(bob.address)).to.equal("0")
    //   await advanceBlockTo("100")

    //   deposit = await this.chef.connect(bob).deposit(0, "0") // block 101
    //   await deposit.wait()
    //   // expect(await this.sushi.balanceOf(this.bob.address)).to.equal("1000")

    //   await advanceBlockTo("104")
    //   deposit = await this.chef.connect(bob).deposit(0, "0") // block 105
    //   await deposit.wait()

    //   const bobBalance = await this.sushi.balanceOf(bob.address);
    //   const devBalance = await this.sushi.balanceOf(dev.address);
    //   const supply = await this.sushi.totalSupply();
    //   console.log({
    //     bobBalance: bobBalance.toString(),
    //     devBalance: devBalance.toString(),
    //     supply: supply.toString(),
    //   })
    //   // expect(await this.sushi.balanceOf(this.bob.address)).to.equal("5000")
    //   // expect(await this.sushi.balanceOf(this.dev.address)).to.equal("500")
    //   // expect(await this.sushi.totalSupply()).to.equal("5500")
    // })

    // it("should not distribute SUSHIs if no one deposit", async function () {
    //   // 100 per block farming rate starting at block 200 with bonus until block 1000
    //   this.chef = await this.Factory__MasterChef.deploy(this.sushi.address, dev.address, "100", "200", "1000")
    //   await this.chef.deployTransaction.wait()

    //   const transferOwnershipTX = await this.sushi.transferOwnership(this.chef.address)
    //   await transferOwnershipTX.wait()

    //   const addTX = await this.chef.add("100", this.lp.address, true)
    //   await addTX.wait()

    //   const approveTX = await this.lp.connect(bob).approve(this.chef.address, "1000")
    //   await approveTX.wait()

    //   await advanceBlockTo("199")
    //   expect(await this.sushi.totalSupply()).to.equal("0")
    //   await advanceBlockTo("204")
    //   expect(await this.sushi.totalSupply()).to.equal("0")
    //   await advanceBlockTo("209")

    //   const depositTX = await this.chef.connect(bob).deposit(0, "10") // block 210
    //   await depositTX.wait()

    //   expect(await this.sushi.totalSupply()).to.equal("0")
    //   expect(await this.sushi.balanceOf(bob.address)).to.equal("0")
    //   expect(await this.sushi.balanceOf(dev.address)).to.equal("0")
    //   expect(await this.lp.balanceOf(bob.address)).to.equal("990")

    //   await advanceBlockTo("219")

    //   const withdrawTX = await this.chef.connect(bob).withdraw(0, "10") // block 220
    //   await withdrawTX.wait()

    //   expect(await this.sushi.totalSupply()).to.equal("11000")
    //   expect(await this.sushi.balanceOf(this.bob.address)).to.equal("10000")
    //   expect(await this.sushi.balanceOf(this.dev.address)).to.equal("1000")
    //   expect(await this.lp.balanceOf(this.bob.address)).to.equal("1000")
    // })

    // it("should distribute SUSHIs properly for each staker", async function () {
    //   // 100 per block farming rate starting at block 300 with bonus until block 1000
    //   this.chef = await this.MasterChef.deploy(this.sushi.address, this.dev.address, "100", "300", "1000")
    //   await this.chef.deployed()
    //   await this.sushi.transferOwnership(this.chef.address)
    //   await this.chef.add("100", this.lp.address, true)
    //   await this.lp.connect(this.alice).approve(this.chef.address, "1000", {
    //     from: this.alice.address,
    //   })
    //   await this.lp.connect(this.bob).approve(this.chef.address, "1000", {
    //     from: this.bob.address,
    //   })
    //   await this.lp.connect(this.carol).approve(this.chef.address, "1000", {
    //     from: this.carol.address,
    //   })
    //   // Alice deposits 10 LPs at block 310
    //   await advanceBlockTo("309")
    //   await this.chef.connect(this.alice).deposit(0, "10", { from: this.alice.address })
    //   // Bob deposits 20 LPs at block 314
    //   await advanceBlockTo("313")
    //   await this.chef.connect(this.bob).deposit(0, "20", { from: this.bob.address })
    //   // Carol deposits 30 LPs at block 318
    //   await advanceBlockTo("317")
    //   await this.chef.connect(this.carol).deposit(0, "30", { from: this.carol.address })
    //   // Alice deposits 10 more LPs at block 320. At this point:
    //   //   Alice should have: 4*1000 + 4*1/3*1000 + 2*1/6*1000 = 5666
    //   //   MasterChef should have the remaining: 10000 - 5666 = 4334
    //   await advanceBlockTo("319")
    //   await this.chef.connect(this.alice).deposit(0, "10", { from: this.alice.address })
    //   expect(await this.sushi.totalSupply()).to.equal("11000")
    //   expect(await this.sushi.balanceOf(this.alice.address)).to.equal("5666")
    //   expect(await this.sushi.balanceOf(this.bob.address)).to.equal("0")
    //   expect(await this.sushi.balanceOf(this.carol.address)).to.equal("0")
    //   expect(await this.sushi.balanceOf(this.chef.address)).to.equal("4334")
    //   expect(await this.sushi.balanceOf(this.dev.address)).to.equal("1000")
    //   // Bob withdraws 5 LPs at block 330. At this point:
    //   //   Bob should have: 4*2/3*1000 + 2*2/6*1000 + 10*2/7*1000 = 6190
    //   await advanceBlockTo("329")
    //   await this.chef.connect(this.bob).withdraw(0, "5", { from: this.bob.address })
    //   expect(await this.sushi.totalSupply()).to.equal("22000")
    //   expect(await this.sushi.balanceOf(this.alice.address)).to.equal("5666")
    //   expect(await this.sushi.balanceOf(this.bob.address)).to.equal("6190")
    //   expect(await this.sushi.balanceOf(this.carol.address)).to.equal("0")
    //   expect(await this.sushi.balanceOf(this.chef.address)).to.equal("8144")
    //   expect(await this.sushi.balanceOf(this.dev.address)).to.equal("2000")
    //   // Alice withdraws 20 LPs at block 340.
    //   // Bob withdraws 15 LPs at block 350.
    //   // Carol withdraws 30 LPs at block 360.
    //   await advanceBlockTo("339")
    //   await this.chef.connect(this.alice).withdraw(0, "20", { from: this.alice.address })
    //   await advanceBlockTo("349")
    //   await this.chef.connect(this.bob).withdraw(0, "15", { from: this.bob.address })
    //   await advanceBlockTo("359")
    //   await this.chef.connect(this.carol).withdraw(0, "30", { from: this.carol.address })
    //   expect(await this.sushi.totalSupply()).to.equal("55000")
    //   expect(await this.sushi.balanceOf(this.dev.address)).to.equal("5000")
    //   // Alice should have: 5666 + 10*2/7*1000 + 10*2/6.5*1000 = 11600
    //   expect(await this.sushi.balanceOf(this.alice.address)).to.equal("11600")
    //   // Bob should have: 6190 + 10*1.5/6.5 * 1000 + 10*1.5/4.5*1000 = 11831
    //   expect(await this.sushi.balanceOf(this.bob.address)).to.equal("11831")
    //   // Carol should have: 2*3/6*1000 + 10*3/7*1000 + 10*3/6.5*1000 + 10*3/4.5*1000 + 10*1000 = 26568
    //   expect(await this.sushi.balanceOf(this.carol.address)).to.equal("26568")
    //   // All of them should have 1000 LPs back.
    //   expect(await this.lp.balanceOf(this.alice.address)).to.equal("1000")
    //   expect(await this.lp.balanceOf(this.bob.address)).to.equal("1000")
    //   expect(await this.lp.balanceOf(this.carol.address)).to.equal("1000")
    // })

    // it("should give proper SUSHIs allocation to each pool", async function () {
    //   // 100 per block farming rate starting at block 400 with bonus until block 1000
    //   this.chef = await this.MasterChef.deploy(this.sushi.address, this.dev.address, "100", "400", "1000")
    //   await this.sushi.transferOwnership(this.chef.address)
    //   await this.lp.connect(this.alice).approve(this.chef.address, "1000", { from: this.alice.address })
    //   await this.lp2.connect(this.bob).approve(this.chef.address, "1000", { from: this.bob.address })
    //   // Add first LP to the pool with allocation 1
    //   await this.chef.add("10", this.lp.address, true)
    //   // Alice deposits 10 LPs at block 410
    //   await advanceBlockTo("409")
    //   await this.chef.connect(this.alice).deposit(0, "10", { from: this.alice.address })
    //   // Add LP2 to the pool with allocation 2 at block 420
    //   await advanceBlockTo("419")
    //   await this.chef.add("20", this.lp2.address, true)
    //   // Alice should have 10*1000 pending reward
    //   expect(await this.chef.pendingSushi(0, this.alice.address)).to.equal("10000")
    //   // Bob deposits 10 LP2s at block 425
    //   await advanceBlockTo("424")
    //   await this.chef.connect(this.bob).deposit(1, "5", { from: this.bob.address })
    //   // Alice should have 10000 + 5*1/3*1000 = 11666 pending reward
    //   expect(await this.chef.pendingSushi(0, this.alice.address)).to.equal("11666")
    //   await advanceBlockTo("430")
    //   // At block 430. Bob should get 5*2/3*1000 = 3333. Alice should get ~1666 more.
    //   expect(await this.chef.pendingSushi(0, this.alice.address)).to.equal("13333")
    //   expect(await this.chef.pendingSushi(1, this.bob.address)).to.equal("3333")
    // })

    // it("should stop giving bonus SUSHIs after the bonus period ends", async function () {
    //   // 100 per block farming rate starting at block 500 with bonus until block 600
    //   this.chef = await this.MasterChef.deploy(this.sushi.address, this.dev.address, "100", "500", "600")
    //   await this.sushi.transferOwnership(this.chef.address)
    //   await this.lp.connect(this.alice).approve(this.chef.address, "1000", { from: this.alice.address })
    //   await this.chef.add("1", this.lp.address, true)
    //   // Alice deposits 10 LPs at block 590
    //   await advanceBlockTo("589")
    //   await this.chef.connect(this.alice).deposit(0, "10", { from: this.alice.address })
    //   // At block 605, she should have 1000*10 + 100*5 = 10500 pending.
    //   await advanceBlockTo("605")
    //   expect(await this.chef.pendingSushi(0, this.alice.address)).to.equal("10500")
    //   // At block 606, Alice withdraws all pending rewards and should get 10600.
    //   await this.chef.connect(this.alice).deposit(0, "0", { from: this.alice.address })
    //   expect(await this.chef.pendingSushi(0, this.alice.address)).to.equal("0")
    //   expect(await this.sushi.balanceOf(this.alice.address)).to.equal("10600")
    // })
  })
})
