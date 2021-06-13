const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { bob, alice, carol, dev, minter } = require('./utilities/wallet');
const { deploy, getBigNumber, createSLP } = require('./utilities/index');

const ERC20MockJSON = require('../artifacts/contracts/mocks/ERC20Mock.sol/ERC20Mock.ovm.json');
const UniswapV2FactoryJSON = require('../artifacts/contracts/uniswapv2/UniswapV2Factory.sol/UniswapV2Factory.ovm.json');
const UniswapV2PairJSON = require('../artifacts/contracts/uniswapv2/UniswapV2Pair.sol/UniswapV2Pair.ovm.json');
const SushiMakerExploitMockJSON = require('../artifacts/contracts/mocks/SushiMakerExploitMock.sol/SushiMakerExploitMock.ovm.json')
const SushiMakerJSON = require('../artifacts/contracts/SushiMaker.sol/SushiMaker.ovm.json');
const SushiBarJSON = require('../artifacts/contracts/SushiBar.sol/SushiBar.ovm.json');

describe("SushiMaker", function () {
  before(async function () {
    this.Factory__UniswapV2Pair = new ContractFactory(
      UniswapV2PairJSON.abi,
      UniswapV2PairJSON.bytecode,
      bob,
    )
  })
  beforeEach(async function () {
    await deploy(this, [
      ["sushi", ERC20MockJSON, ["SUSHI", "SUSHI", getBigNumber("10000000")]],
      ["dai", ERC20MockJSON, ["DAI", "DAI", getBigNumber("10000000")]],
      ["mic", ERC20MockJSON, ["MIC", "MIC", getBigNumber("10000000")]],
      ["usdc", ERC20MockJSON, ["USDC", "USDC", getBigNumber("10000000")]],
      ["weth", ERC20MockJSON, ["WETH", "ETH", getBigNumber("10000000")]],
      ["strudel", ERC20MockJSON, ["$TRDL", "$TRDL", getBigNumber("10000000")]],
      ["factory", UniswapV2FactoryJSON, [bob.address]],
    ])
    await deploy(this, [["bar", SushiBarJSON, [this.sushi.address]]])
    await deploy(this, [["sushiMaker", SushiMakerJSON, [this.factory.address, this.bar.address, this.sushi.address, this.weth.address]]])
    await deploy(this, [["exploiter", SushiMakerExploitMockJSON, [this.sushiMaker.address]]])
    await createSLP(this, "sushiEth", this.sushi, this.weth, getBigNumber(10))
    await createSLP(this, "strudelEth", this.strudel, this.weth, getBigNumber(10))
    await createSLP(this, "daiEth", this.dai, this.weth, getBigNumber(10))
    await createSLP(this, "usdcEth", this.usdc, this.weth, getBigNumber(10))
    await createSLP(this, "micUSDC", this.mic, this.usdc, getBigNumber(10))
    await createSLP(this, "sushiUSDC", this.sushi, this.usdc, getBigNumber(10))
    await createSLP(this, "daiUSDC", this.dai, this.usdc, getBigNumber(10))
    await createSLP(this, "daiMIC", this.dai, this.mic, getBigNumber(10))
  })
  describe("setBridge", function () {
    it("does not allow to set bridge for Sushi", async function () {
      const setBridgeTX = await this.sushiMaker.setBridge(this.sushi.address, this.weth.address)
      await expect(setBridgeTX.wait()).to.be.eventually.rejected;
    })

    it("does not allow to set bridge for WETH", async function () {
      const setBridgeTX = await this.sushiMaker.setBridge(this.weth.address, this.sushi.address)
      await expect(setBridgeTX.wait()).to.be.eventually.rejected;
    })

    it("does not allow to set bridge to itself", async function () {
      const setBridgeTX = await this.sushiMaker.setBridge(this.dai.address, this.dai.address)
      await expect(setBridgeTX.wait()).to.be.eventually.rejected;
    })

    it("emits correct event on bridge", async function () {
      await expect(this.sushiMaker.setBridge(this.dai.address, this.sushi.address))
        .to.emit(this.sushiMaker, "LogBridgeSet")
        .withArgs(this.dai.address, this.sushi.address)
    })
  })
  describe("convert", function () {
    it("should convert SUSHI - ETH", async function () {
      let transferTX, convertTX
      transferTX = await this.sushiEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convert(this.sushi.address, this.weth.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushiEth.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1897569270781234370")
    })

    it("should convert USDC - ETH", async function () {
      let transferTX, convertTX
      transferTX = await this.usdcEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convert(this.usdc.address, this.weth.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.usdcEth.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1590898251382934275")
    })

    it("should convert $TRDL - ETH", async function () {
      let transferTX, convertTX
      transferTX = await this.strudelEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convert(this.strudel.address, this.weth.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.strudelEth.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1590898251382934275")
    })

    it("should convert USDC - SUSHI", async function () {
      let transferTX, convertTX
      transferTX = await this.sushiUSDC.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convert(this.usdc.address, this.sushi.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushiUSDC.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1897569270781234370")
    })

    it("should convert using standard ETH path", async function () {
      let transferTX, convertTX
      transferTX = await this.daiEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convert(this.dai.address, this.weth.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.daiEth.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1590898251382934275")
    })

    it("converts MIC/USDC using more complex path", async function () {
      let transferTX, setBridgeTX, convertTX
      transferTX = await this.micUSDC.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.usdc.address, this.sushi.address)
      await setBridgeTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.mic.address, this.usdc.address)
      await setBridgeTX.wait()
      convertTX = await this.sushiMaker.convert(this.mic.address, this.usdc.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.micUSDC.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1590898251382934275")
    })

    it("converts DAI/USDC using more complex path", async function () {
      let transferTX, setBridgeTX, convertTX
      transferTX = await this.daiUSDC.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.usdc.address, this.sushi.address)
      await setBridgeTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.dai.address, this.usdc.address)
      await setBridgeTX.wait()
      convertTX = await this.sushiMaker.convert(this.dai.address, this.usdc.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.daiUSDC.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1590898251382934275")
    })

    it("converts DAI/MIC using two step path", async function () {
      let transferTX, setBridgeTX, convertTX
      transferTX = await this.daiMIC.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.dai.address, this.usdc.address)
      await setBridgeTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.mic.address, this.dai.address)
      await setBridgeTX.wait()
      convertTX = await this.sushiMaker.convert(this.dai.address, this.mic.address)
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.daiMIC.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("1200963016721363748")
    })

    it("reverts if it loops back", async function () {
      let transferTX, setBridgeTX, convertTX
      transferTX = await this.daiMIC.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.dai.address, this.mic.address)
      await setBridgeTX.wait()
      setBridgeTX = await this.sushiMaker.setBridge(this.mic.address, this.dai.address)
      await setBridgeTX.wait()
      convertTX = await this.sushiMaker.convert(this.dai.address, this.mic.address)
      await expect(convertTX.wait()).to.be.eventually.rejected;
    })

    it("reverts if caller is not EOA", async function () {
      let transferTX, convertTX
      transferTX = await this.sushiEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.exploiter.convert(this.sushi.address, this.weth.address)
      await expect(convertTX.wait()).to.be.eventually.rejected;
    })

    it("reverts if pair does not exist", async function () {
      let convertTX
      convertTX = await this.sushiMaker.convert(this.mic.address, this.micUSDC.address)
      await expect(convertTX.wait()).to.be.eventually.rejected;
    })

    it("reverts if no path is available", async function () {
      let transferTX, convertTX
      transferTX = await this.micUSDC.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convert(this.mic.address, this.usdc.address)
      await expect(convertTX.wait()).to.be.eventually.rejected;
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.micUSDC.balanceOf(this.sushiMaker.address)).to.equal(getBigNumber(1))
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal(0)
    })
  })

  describe("convertMultiple", function () {
    it("should allow to convert multiple", async function () {
      let transferTX, convertTX
      transferTX = await this.daiEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      transferTX = await this.sushiEth.transfer(this.sushiMaker.address, getBigNumber(1))
      await transferTX.wait()
      convertTX = await this.sushiMaker.convertMultiple([this.dai.address, this.sushi.address], [this.weth.address, this.weth.address])
      await convertTX.wait()
      expect(await this.sushi.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.daiEth.balanceOf(this.sushiMaker.address)).to.equal(0)
      expect(await this.sushi.balanceOf(this.bar.address)).to.equal("3186583558687783097")
    })
  })
})