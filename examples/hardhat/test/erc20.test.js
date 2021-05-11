/* External Imports */
const { ethers, network } = require('hardhat')
const chai = require('chai')
const { solidity } = require('ethereum-waffle')
const { expect } = chai

chai.use(solidity)

describe(`ERC20`, () => {
  const INITIAL_SUPPLY = 1000000
  const TOKEN_NAME = 'An Optimistic ERC20'

  let account1
  let account2
  before(`load accounts`, async () => {
    ;[ account1, account2 ] = await ethers.getSigners()
  })

  let ERC20
  beforeEach(`deploy ERC20 contract`, async () => {
    const Factory__ERC20 = await ethers.getContractFactory('ERC20')
    ERC20 = await Factory__ERC20.connect(account1).deploy(
      INITIAL_SUPPLY,
      TOKEN_NAME
    )

    await ERC20.deployTransaction.wait()
  })

  it(`should have a name`, async () => {
    const tokenName = await ERC20.name()
    expect(tokenName).to.equal(TOKEN_NAME)
  })

  it(`should have a total supply equal to the initial supply`, async () => {
    const tokenSupply = await ERC20.totalSupply()
    expect(tokenSupply).to.equal(INITIAL_SUPPLY)
  })

  it(`should give the initial supply to the creator's address`, async () => {
    const balance = await ERC20.balanceOf(await account1.getAddress())
    expect(balance).to.equal(INITIAL_SUPPLY)
  })

  describe(`transfer(...)`, () => {
    it(`should revert when the sender does not have enough balance`, async () => {
      const tx = ERC20.connect(account1).transfer(
        await account2.getAddress(),
        INITIAL_SUPPLY + 1
      )
      await expect(tx).to.be.revertedWith("You don't have enough balance to make this transfer!")
    })

    it(`should succeed when the sender has enough balance`, async () => {
      const tx = await ERC20.connect(account1).transfer(
        await account2.getAddress(),
        INITIAL_SUPPLY
      )
      await tx.wait()

      expect(
        (await ERC20.balanceOf(
          await account1.getAddress()
        )).toNumber()
      ).to.equal(0)
      expect(
        (await ERC20.balanceOf(
          await account2.getAddress()
        )).toNumber()
      ).to.equal(INITIAL_SUPPLY)
    })
  })

  describe(`transferFrom(...)`, () => {
    it(`should revert when the sender does not have enough of an allowance`, async () => {
      const tx = ERC20.connect(account2).transferFrom(
        await account1.getAddress(),
        await account2.getAddress(),
        INITIAL_SUPPLY
      )
      await expect(tx).to.be.revertedWith("Can't transfer from the desired account because you don't have enough of an allowance.")
    })

    it(`should succeed when the owner has enough balance and the sender has a large enough allowance`, async () => {
      const tx1 = await ERC20.connect(account1).approve(
        await account2.getAddress(),
        INITIAL_SUPPLY
      )
      await tx1.wait()

      const tx2 = await ERC20.connect(account2).transferFrom(
        await account1.getAddress(),
        await account2.getAddress(),
        INITIAL_SUPPLY
      )
      await tx2.wait()

      expect(
        (await ERC20.balanceOf(
          await account1.getAddress()
        )).toNumber()
      ).to.equal(0)
      expect(
        (await ERC20.balanceOf(
          await account2.getAddress()
        )).toNumber()
      ).to.equal(INITIAL_SUPPLY)
    })
  })
})
