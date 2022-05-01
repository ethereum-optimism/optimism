import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy, NON_ZERO_ADDRESS } from '../../../helpers'

describe('AddressDictator', () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let AddressDictator: Contract
  let Lib_AddressManager: Contract
  beforeEach(async () => {
    Lib_AddressManager = await deploy('Lib_AddressManager', {
      signer: signer1,
    })

    AddressDictator = await deploy('AddressDictator', {
      signer: signer1,
      args: [
        Lib_AddressManager.address,
        signer1.address,
        ['addr1'],
        [NON_ZERO_ADDRESS],
      ],
    })

    Lib_AddressManager.transferOwnership(AddressDictator.address)
  })

  describe('initialize', () => {
    it('should revert when providing wrong arguments', async () => {
      await expect(
        deploy('AddressDictator', {
          signer: signer1,
          args: [
            Lib_AddressManager.address,
            signer1.address,
            ['addr1', 'addr2'],
            [NON_ZERO_ADDRESS],
          ],
        })
      ).to.be.revertedWith(
        'AddressDictator: Must provide an equal number of names and addresses.'
      )
    })
  })

  describe('setAddresses', async () => {
    it('should change the addresses associated with a name', async () => {
      await AddressDictator.setAddresses()
      expect(await Lib_AddressManager.getAddress('addr1')).to.equal(
        NON_ZERO_ADDRESS
      )
    })
  })

  describe('getNamedAddresses', () => {
    it('should return all the addresses and their names', async () => {
      expect(await AddressDictator.getNamedAddresses()).to.deep.equal([
        ['addr1', NON_ZERO_ADDRESS],
      ])
    })
  })

  describe('returnOwnership', () => {
    it('should transfer contract ownership to finalOwner', async () => {
      await expect(AddressDictator.returnOwnership()).to.not.be.reverted
    })

    it('should revert when called by non-owner', async () => {
      await expect(
        AddressDictator.connect(signer2).returnOwnership()
      ).to.be.revertedWith('AddressDictator: only callable by finalOwner')
    })
  })
})
