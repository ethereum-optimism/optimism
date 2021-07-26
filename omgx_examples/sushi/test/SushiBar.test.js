const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { ContractFactory } = require('ethers');
const { bob, alice, carol } = require('./utilities/wallet');
const { gasOptions } = require("./utilities/index")

const SushiTokenJSON = require('../artifacts-ovm/contracts/SushiToken.sol/SushiToken.ovm.json');
const SushiBarJSON = require('../artifacts-ovm/contracts/SushiBar.sol/SushiBar.ovm.json');

describe("SushiBar", function () {
  before(async function () {
    this.Factory__SushiToken = new ContractFactory(
      SushiTokenJSON.abi,
      SushiTokenJSON.bytecode,
      bob,
    )

    this.Factory__SushiBar = new ContractFactory(
      SushiBarJSON.abi,
      SushiBarJSON.bytecode,
      bob,
    )
  })

  beforeEach(async function () {
    this.sushi = await this.Factory__SushiToken.deploy(gasOptions)
    await this.sushi.deployTransaction.wait()
    this.bar = await this.Factory__SushiBar.deploy(gasOptions)
    await this.bar.deployTransaction.wait()
    const barInitTx = await this.bar.initialize(this.sushi.address, gasOptions)
    await barInitTx.wait()
    let mint
    mint = await this.sushi.mint(alice.address, "100", gasOptions)
    await mint.wait()
    mint = await this.sushi.mint(bob.address, "100", gasOptions)
    await mint.wait()
    mint = await this.sushi.mint(carol.address, "100", gasOptions)
    await mint.wait()
  })

  it("should not allow enter if not enough approve", async function () {
    let barTransfer, approve
    await expect(this.bar.enter("100")).to.be.eventually.rejected;

    approve = await this.sushi.approve(this.bar.address, "50", gasOptions)
    await approve.wait()
    await expect(this.bar.enter("100")).to.be.eventually.rejected;

    approve = await this.sushi.approve(this.bar.address, "100", gasOptions)
    await approve.wait()
    barTransfer = await this.bar.enter("100", gasOptions)
    await barTransfer.wait()
    expect(await this.bar.balanceOf(bob.address)).to.equal("100")
  })

  it("should not allow withraw more than what you have", async function () {
    const approve = await this.sushi.approve(this.bar.address, "100", gasOptions)
    await approve.wait()
    const barTransfer = await this.bar.enter("100", gasOptions)
    await barTransfer.wait()
    await expect(this.bar.leave("200")).to.be.eventually.rejected;
  })

  it("should work with more than one participant", async function () {
    let approve, barTransfer, barDeposit, barWithdraw
    approve = await this.sushi.approve(this.bar.address, "100", gasOptions)
    await approve.wait()
    approve = await this.sushi.connect(alice).approve(this.bar.address, "100", gasOptions)
    await approve.wait()
    // Bob enters and gets 20 shares. Alice enters and gets 10 shares.
    barTransfer = await this.bar.enter("20", gasOptions)
    await barTransfer.wait()
    barTransfer = await this.bar.connect(alice).enter("10", gasOptions)
    await barTransfer.wait()
    expect(await this.bar.balanceOf(bob.address)).to.equal("20")
    expect(await this.bar.balanceOf(alice.address)).to.equal("10")
    expect(await this.sushi.balanceOf(this.bar.address)).to.equal("30")
    // SushiBar get 20 more SUSHIs from an external source.
    barTransfer = await this.sushi.connect(carol).transfer(this.bar.address, "20", gasOptions)
    await barTransfer.wait()
    // Alice deposits 10 more SUSHIs. She should receive 10*30/50 = 6 shares.
    barDeposit = await this.bar.enter("10", gasOptions)
    await barDeposit.wait()
    expect(await this.bar.balanceOf(bob.address)).to.equal("26")
    expect(await this.bar.balanceOf(alice.address)).to.equal("10")
    // Bob withdraws 5 shares. He should receive 5*60/36 = 8 shares
    barWithdraw = await this.bar.connect(alice).leave("5", gasOptions)
    await barWithdraw.wait()
    expect(await this.bar.balanceOf(bob.address)).to.equal("26")
    expect(await this.bar.balanceOf(alice.address)).to.equal("5")
    expect(await this.sushi.balanceOf(this.bar.address)).to.equal("52")
    expect(await this.sushi.balanceOf(bob.address)).to.equal("70")
    expect(await this.sushi.balanceOf(alice.address)).to.equal("98")
  })
})
