const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { bob, alice, carol, dev, minter } = require('./utilities/wallet');

const MasterChefJSON = require('../artifacts/contracts/MasterChef.sol/MasterChef.ovm.json');
const SushiTokenJSON = require('../artifacts/contracts/SushiToken.sol/SushiToken.ovm.json');
const MigratorJSON = require('../artifacts/contracts/Migrator.sol/Migrator.ovm.json');
const ERC20MockJSON = require('../artifacts/contracts/mocks/ERC20Mock.sol/ERC20Mock.ovm.json');
const UniswapV2FactoryJSON = require('../artifacts/contracts/uniswapv2/UniswapV2Factory.sol/UniswapV2Factory.ovm.json');
const UniswapV2PairJSON = require('../artifacts/contracts/uniswapv2/UniswapV2Pair.sol/UniswapV2Pair.ovm.json');

describe("Migrator", function () {
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

    this.Factory__Migrator = new ContractFactory(
      MigratorJSON.abi,
      MigratorJSON.bytecode,
      bob,
    )

    this.Factory__UniswapV2Factory = new ContractFactory(
      UniswapV2FactoryJSON.abi,
      UniswapV2FactoryJSON.bytecode,
      bob,
    )

    this.Factory__UniswapV2Pair = new ContractFactory(
      UniswapV2PairJSON.abi,
      UniswapV2PairJSON.bytecode,
      bob,
    )
  })

  beforeEach(async function () {
    this.factory1 = await this.Factory__UniswapV2Factory.deploy(bob.address)
    await this.factory1.deployTransaction.wait()

    this.factory2 = await this.Factory__UniswapV2Factory.deploy(bob.address)
    await this.factory2.deployTransaction.wait()

    this.sushi = await this.Factory__SushiToken.deploy()
    await this.sushi.deployTransaction.wait()

    this.weth = await this.Factory__ERC20Mock.deploy("WETH", "WETH", "100000000")
    await this.weth.deployTransaction.wait()

    this.token = await this.Factory__ERC20Mock.deploy("TOKEN", "TOKEN", "100000000")
    await this.token.deployTransaction.wait()

    const pair1 = await this.factory1.createPair(this.weth.address, this.token.address)
    const pair1TX = await pair1.wait()

    this.lp1 = await this.Factory__UniswapV2Pair.attach(pair1TX.events[1].args.pair)

    const pair2 = await this.factory2.createPair(this.weth.address, this.token.address)
    const pair2TX = await pair2.wait()

    this.lp2 = await this.Factory__UniswapV2Pair.attach(pair2TX.events[1].args.pair)

    this.chef = await this.Factory__MasterChef.deploy(this.sushi.address, dev.address, "1000", "0", "100000")
    await this.chef.deployTransaction.wait()

    this.migrator = await this.Factory__Migrator.deploy(this.chef.address, this.factory1.address, this.factory2.address, "0")
    await this.migrator.deployTransaction.wait()

    const transfer = await this.sushi.transferOwnership(this.chef.address)
    await transfer.wait()

    const add = await this.chef.add("100", this.lp1.address, true)
    await add.wait()
  })

  it("should do the migration successfully", async function () {
    let transfer
    transfer = await this.token.transfer(this.lp1.address, "10000000")
    await transfer.wait()
    transfer = await this.weth.transfer(this.lp1.address, "500000")
    await transfer.wait()
    const mint = await this.lp1.mint(minter.address)
    await mint.wait()
    expect(await this.lp1.balanceOf(minter.address)).to.equal("2235067")

    // Add some fake revenue
    transfer = await this.token.transfer(this.lp1.address, "100000")
    await transfer.wait()
    transfer = await this.weth.transfer(this.lp1.address, "5000")
    await transfer.wait()
    const sync = await this.lp1.sync()
    await sync.wait()
    const approve = await this.lp1.connect(minter).approve(this.chef.address, "100000000000", { from: minter.address })
    await approve.wait()
    const deposit = await this.chef.connect(minter).deposit("0", "2000000", { from: minter.address })
    await deposit.wait()
    expect(await this.lp1.balanceOf(this.chef.address), "2000000")
    let migrate
    migrate = await this.chef.migrate(0)
    await expect(migrate.wait()).to.be.eventually.rejected;

    let setMigrate
    setMigrate = await this.chef.setMigrator(this.migrator.address)
    await setMigrate.wait()
    migrate = await this.chef.migrate(0)
    await expect(migrate.wait()).to.be.eventually.rejected;

    setMigrate = await this.factory2.setMigrator(this.migrator.address)
    await setMigrate.wait()
    migrate = await this.chef.migrate(0)
    await migrate.wait()
    expect(await this.lp1.balanceOf(this.chef.address)).to.equal("0")
    expect(await this.lp2.balanceOf(this.chef.address)).to.equal("2000000")

    const withdraw = await this.chef.connect(minter).withdraw("0", "2000000")
    await withdraw.wait()
    transfer = await this.lp2.connect(minter).transfer(this.lp2.address, "2000000")
    await transfer.wait()
    const burn = await this.lp2.burn(bob.address)
    await burn.wait()
    expect(await this.token.balanceOf(bob.address)).to.equal("9033718")
    expect(await this.weth.balanceOf(bob.address)).to.equal("451685")
  })

  it("should allow first minting from public only after migrator is gone", async function () {
    const setMigrate = await this.factory2.setMigrator(this.migrator.address)
    await setMigrate.wait()

    this.tokenx = await this.Factory__ERC20Mock.deploy("TOKENX", "TOKENX", "100000000")
    await this.tokenx.deployTransaction.wait()

    const pair = await this.factory2.createPair(this.weth.address, this.tokenx.address)
    const pairTX = await pair.wait()

    this.lpx = await this.Factory__UniswapV2Pair.attach(pairTX.events[1].args.pair)

    let transfer 
    transfer = await this.weth.connect(minter).transfer(this.lpx.address, "10000000")
    await transfer.wait()
    transfer = await this.tokenx.connect(minter).transfer(this.lpx.address, "500000")
    await transfer.wait()
    const mint = await this.lpx.mint(minter.address)
    await expect(mint.wait()).to.be.eventually.rejected;
  })
})