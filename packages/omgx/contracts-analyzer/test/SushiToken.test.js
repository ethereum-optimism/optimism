const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { bob, alice, carol } = require('./utilities/wallet');

const SushiTokenJSON = require('../artifacts/contracts/SushiToken.sol/SushiToken.ovm.json');

describe("SushiToken", function () {
  before(async function () {
    this.Factory__SushiTokenPool = new ContractFactory(
      SushiTokenJSON.abi,
      SushiTokenJSON.bytecode,
      bob,
    )
  })

  beforeEach(async function () {
    this.sushi = await this.Factory__SushiTokenPool.deploy()
    await this.sushi.deployTransaction.wait()
  })

  it("should have correct name and symbol and decimal", async function () {
    const name = await this.sushi.name()
    const symbol = await this.sushi.symbol()
    const decimals = await this.sushi.decimals()
    expect(name, "SushiToken")
    expect(symbol, "SUSHI")
    expect(decimals, "18")
  })

  it("should only allow owner to mint token", async function () {
    const bobMint = await this.sushi.mint(bob.address, "1000")
    await bobMint.wait()
    const aliceMint = await this.sushi.mint(alice.address, "100")
    await aliceMint.wait()

    // not the owner
    const carolMint = await this.sushi.connect(alice).mint(carol.address, "1000")
    await expect(carolMint.wait()).to.be.eventually.rejected;

    const totalSupply = await this.sushi.totalSupply()
    const aliceBal = await this.sushi.balanceOf(alice.address)
    const bobBal = await this.sushi.balanceOf(bob.address)
    const carolBal = await this.sushi.balanceOf(carol.address)
    expect(totalSupply).to.equal("1100")
    expect(aliceBal).to.equal("100")
    expect(bobBal).to.equal("1000")
    expect(carolBal).to.equal("0")
  })

  it("should supply token transfers properly", async function () {
    const aliceMint = await this.sushi.mint(alice.address, "100")
    await aliceMint.wait()
    const bobMint = await this.sushi.mint(bob.address, "1000")
    await bobMint.wait()
    const carolTX = await this.sushi.transfer(carol.address, "10")
    await carolTX.wait()
    const bobTX = await this.sushi.connect(bob).transfer(carol.address, "100", {
      from: bob.address,
    })
    await bobTX.wait()

    const totalSupply = await this.sushi.totalSupply()
    const aliceBal = await this.sushi.balanceOf(alice.address)
    const bobBal = await this.sushi.balanceOf(bob.address)
    const carolBal = await this.sushi.balanceOf(carol.address)
    expect(totalSupply, "1100")
    expect(aliceBal, "90")
    expect(bobBal, "900")
    expect(carolBal, "110")
  })

  it("should fail if you try to do bad transfers", async function () {
    const aliceMint = await this.sushi.mint(alice.address, "100")
    await aliceMint.wait()
    //ERC20: transfer amount exceeds balance
    const carolTX = await this.sushi.transfer(carol.address, "110");
    await expect(carolTX.wait()).to.be.eventually.rejected;
    //ERC20: transfer amount exceeds balance
    const bobTX = await this.sushi.connect(bob).transfer(carol.address, "1", { from: bob.address });
    await expect(bobTX.wait()).to.be.eventually.rejected;
  })
})