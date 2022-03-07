/* External Imports */
import { ethers } from 'hardhat'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import { expect } from '../../../setup'
import { NON_ZERO_ADDRESS } from '../../../helpers'

describe('AddressDictator', () => {
  let signer: Signer
  let otherSigner: Signer
  let signerAddress: string
  let Factory__AddressDictator: ContractFactory
  let Factory__Lib_AddressManager: ContractFactory
  before(async () => {
    ;[signer, otherSigner] = await ethers.getSigners()

    Factory__AddressDictator = await ethers.getContractFactory(
      'AddressDictator'
    )

    Factory__Lib_AddressManager = await ethers.getContractFactory(
      'Lib_AddressManager'
    )

    signerAddress = await signer.getAddress()
  })

  let AddressDictator: Contract
  let Lib_AddressManager: Contract
  beforeEach(async () => {
    Lib_AddressManager = await Factory__Lib_AddressManager.connect(
      signer
    ).deploy()

    AddressDictator = await Factory__AddressDictator.connect(signer).deploy(
      Lib_AddressManager.address,
      signerAddress,
      ['addr1'],
      [NON_ZERO_ADDRESS]
    )

    Lib_AddressManager.transferOwnership(AddressDictator.address)
  })

  describe('initialize', () => {
    it('should revert when providing wrong arguments', async () => {
      await expect(
        Factory__AddressDictator.connect(signer).deploy(
          Lib_AddressManager.address,
          signerAddress,
          ['addr1', 'addr2'],
          [NON_ZERO_ADDRESS]
        )
      ).to.be.revertedWith(
        'AddressDictator: Must provide an equal number of names and addresses.'
      )
    })
  })

  describe('setAddresses', async () => {
    it('should change the addresses associated with a name', async () => {
      await AddressDictator.setAddresses()
      expect(await Lib_AddressManager.getAddress('addr1')).to.be.equal(
        NON_ZERO_ADDRESS
      )
    })
  })

  describe('getNamedAddresses', () => {
    it('should return all the addresses and their names', async () => {
      expect(await AddressDictator.getNamedAddresses()).to.be.deep.equal([
        ['addr1', NON_ZERO_ADDRESS],
      ])
    })
  })

  describe('returnOwnership', () => {
    it('should transfer contract ownership to finalOwner', async () => {
      await expect(AddressDictator.connect(signer).returnOwnership()).to.not.be
        .reverted
    })

    it('should revert when called by non-owner', async () => {
      await expect(
        AddressDictator.connect(otherSigner).returnOwnership()
      ).to.be.revertedWith('AddressDictator: only callable by finalOwner')
    })
  })
})
