/* Imports: External */
import hre from 'hardhat'
import { expect } from 'chai'
import { Contract } from 'ethers'

/* Imports: Internal */
import { smockit } from '../../src'

describe('[smock]: sending transactions from smock contracts', () => {
  const ethers = (hre as any).ethers

  let TestHelpers_SenderAssertions: Contract
  before(async () => {
    TestHelpers_SenderAssertions = await (
      await ethers.getContractFactory('TestHelpers_SenderAssertions')
    ).deploy()
  })

  it('should attach a signer for a mock with a random address', async () => {
    const mock = await smockit('TestHelpers_BasicReturnContract')

    expect(
      await TestHelpers_SenderAssertions.connect(mock.wallet).getSender()
    ).to.equal(mock.address)
  })

  it('should attach a signer for a mock with a fixed address', async () => {
    const mock = await smockit('TestHelpers_BasicReturnContract', {
      address: '0x1234123412341234123412341234123412341234',
    })

    expect(
      await TestHelpers_SenderAssertions.connect(mock.wallet).getSender()
    ).to.equal(mock.address)
  })
})
