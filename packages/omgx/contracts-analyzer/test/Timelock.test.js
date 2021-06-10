const { expect } = require("chai");
const chai = require('chai');
const chaiAsPromised = require('chai-as-promised');
chai.use(chaiAsPromised);
const { Contract, Wallet, ContractFactory, BigNumber, providers } = require('ethers');
const { encodeParameters } = require('./utilities/index');
const { latest, duration, increase } = require('./utilities/time');
const { bob, alice, carol, dev, minter } = require('./utilities/wallet');

const MasterChefJSON = require('../artifacts/contracts/MasterChef.sol/MasterChef.ovm.json');
const SushiTokenJSON = require('../artifacts/contracts/SushiToken.sol/SushiToken.ovm.json');
const ERC20MockJSON = require('../artifacts/contracts/mocks/ERC20Mock.sol/ERC20Mock.ovm.json');
const TimelockJSON = require('../artifacts/contracts/governance/Timelock.sol/Timelock.ovm.json');

/******************************************************************/
/*************   evm_increaseTime is not supported ****************/
/**** this could be something Optimism adds in the near future! ***/
/******************************************************************/

describe("Timelock", function () {
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
      
    this.Factory__Timelock = new ContractFactory(
      TimelockJSON.abi,
      TimelockJSON.bytecode,
      bob,
    )
  })

  beforeEach(async function () {
    this.sushi = await this.Factory__SushiToken.deploy()
    await this.sushi.deployTransaction.wait()
    this.timelock = await this.Factory__Timelock.deploy(bob.address, "259200")
    await this.timelock.deployTransaction.wait()
  })

  it("should not allow non-owner to do operation", async function () {
    let transferOwnership
    transferOwnership = await this.sushi.transferOwnership(this.timelock.address);
    await transferOwnership.wait()

    transferOwnership = await this.sushi.transferOwnership(carol.address);
    await expect(transferOwnership.wait()).to.be.eventually.rejected;

    transferOwnership = await this.sushi.connect(alice).transferOwnership(carol.address);
    await expect(transferOwnership.wait()).to.be.eventually.rejected;

    const timelock = await this.timelock.connect(alice).queueTransaction(
      this.sushi.address,
      "0",
      "transferOwnership(address)",
      encodeParameters(["address"], [carol.address]),
      (await latest()).add(duration.days(4))
    )
    await expect(timelock.wait()).to.be.eventually.rejected;
  })

  /******************************************************************/
  /*************   evm_increaseTime is not supported ****************/
  /**** this could be something Optimism adds in the near future! ***/
  /******************************************************************/

  // it("should do the timelock thing", async function () {
  //   let transferOwnership, queueTransaction
  //   transferOwnership = await this.sushi.connect(bob).transferOwnership(this.timelock.address);
  //   await transferOwnership.wait()

  //   const eta = (await latest()).add(duration.days(4))
  //   queueTransaction = await this.timelock
  //     .connect(bob)
  //     .queueTransaction(this.sushi.address, "0", "transferOwnership(address)", encodeParameters(["address"], [carol.address]), eta)
  //   await queueTransaction.wait()

  //   await increase(duration.days(1))

  //   queueTransaction = await this.timelock
  //     .connect(bob)
  //     .executeTransaction(this.sushi.address, "0", "transferOwnership(address)", encodeParameters(["address"], [carol.address]), eta)
  //   await expect(queueTransaction.wait()).to.be.eventually.rejected;

  //   await increase(duration.days(4))
  //   await this.timelock
  //     .connect(alice)
  //     .executeTransaction(this.sushi.address, "0", "transferOwnership(address)", encodeParameters(["address"], [carol.address]), eta)
  //   expect(await this.sushi.owner()).to.equal(carol.address)
  // })

  // it("should also work with MasterChef", async function () {
  //   let transferOwner
  //   this.lp1 = await this.Factory__ERC20Mock.deploy("LPToken", "LP", "10000000000")
  //   await this.lp1.deployTransaction.wait()
  //   this.lp2 = await this.Factory__ERC20Mock.deploy("LPToken", "LP", "10000000000")
  //   await this.lp2.deployTransaction.wait()
  //   this.chef = await this.Factory__MasterChef.deploy(this.sushi.address, dev.address, "1000", "0", "1000")
  //   await this.chef.deployTransaction.wait()
  //   transferOwner = await this.sushi.transferOwnership(this.chef.address)
  //   await transferOwner.wait()
  //   const deposit = await this.chef.add("100", this.lp1.address, true)
  //   await deposit.wait()
  //   transferOwner = await this.chef.transferOwnership(this.timelock.address)
  //   await transferOwner.wait()
  //   const eta = (await latest()).add(duration.days(4))
  //   await this.timelock
  //     .connect(this.bob)
  //     .queueTransaction(
  //       this.chef.address,
  //       "0",
  //       "set(uint256,uint256,bool)",
  //       encodeParameters(["uint256", "uint256", "bool"], ["0", "200", false]),
  //       eta
  //     )
  //   await this.timelock
  //     .connect(this.bob)
  //     .queueTransaction(
  //       this.chef.address,
  //       "0",
  //       "add(uint256,address,bool)",
  //       encodeParameters(["uint256", "address", "bool"], ["100", this.lp2.address, false]),
  //       eta
  //     )
  //   await increase(duration.days(4))
  //   await this.timelock
  //     .connect(this.bob)
  //     .executeTransaction(
  //       this.chef.address,
  //       "0",
  //       "set(uint256,uint256,bool)",
  //       encodeParameters(["uint256", "uint256", "bool"], ["0", "200", false]),
  //       eta
  //     )
  //   await this.timelock
  //     .connect(this.bob)
  //     .executeTransaction(
  //       this.chef.address,
  //       "0",
  //       "add(uint256,address,bool)",
  //       encodeParameters(["uint256", "address", "bool"], ["100", this.lp2.address, false]),
  //       eta
  //     )
  //   expect((await this.chef.poolInfo("0")).allocPoint).to.equal("200")
  //   expect(await this.chef.totalAllocPoint()).to.equal("300")
  //   expect(await this.chef.poolLength()).to.equal("2")
  // })
})