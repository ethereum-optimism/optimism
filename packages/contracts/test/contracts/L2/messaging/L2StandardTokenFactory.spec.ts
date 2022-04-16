import { ethers } from 'hardhat'
import { ContractFactory, Contract } from 'ethers'
import {
  smock,
  MockContractFactory,
  MockContract,
} from '@defi-wonderland/smock'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'
import { predeploys } from '../../../../src'

describe('L2StandardTokenFactory', () => {
  let Factory__L1ERC20: MockContractFactory<ContractFactory>
  let L1ERC20: MockContract<Contract>
  let L2StandardTokenFactory: Contract
  before(async () => {
    Factory__L1ERC20 = await smock.mock('ERC20')
    L1ERC20 = await Factory__L1ERC20.deploy('L1ERC20', 'ERC')
    L2StandardTokenFactory = await deploy('L2StandardTokenFactory')
  })

  describe('Standard token factory', () => {
    it('should be able to create a standard token', async () => {
      const tx = await L2StandardTokenFactory.createStandardL2Token(
        L1ERC20.address,
        'L2ERC20',
        'ERC'
      )

      // Pull the token creation event from the receipt
      const receipt = await tx.wait()
      const tokenCreatedEvent = receipt.events[0]

      // Expect there to be an event emitted for the standard token creation
      expect(tokenCreatedEvent.event).to.be.eq('StandardL2TokenCreated')

      // Get the L2 token address from the emitted event and check it was created correctly
      const l2Token = await ethers.getContractAt(
        'L2StandardERC20',
        tokenCreatedEvent.args._l2Token
      )

      expect(await l2Token.l2Bridge()).to.equal(predeploys.L2StandardBridge)
      expect(await l2Token.l1Token()).to.equal(L1ERC20.address)
      expect(await l2Token.name()).to.equal('L2ERC20')
      expect(await l2Token.symbol()).to.equal('ERC')
    })

    it('should not be able to create a standard token with a 0 address for l1 token', async () => {
      await expect(
        L2StandardTokenFactory.createStandardL2Token(
          ethers.constants.AddressZero,
          'L2ERC20',
          'ERC'
        )
      ).to.be.revertedWith('Must provide L1 token address')
    })
  })
})
