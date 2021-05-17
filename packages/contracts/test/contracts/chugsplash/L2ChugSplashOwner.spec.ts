import { expect } from '../../setup'

/* Imports: External */
import hre from 'hardhat'
import { ethers, Contract, Signer, ContractFactory } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'

/* Imports: Internal */
import { predeploys } from '../../../src'
import { NON_ZERO_ADDRESS } from '../../helpers'

describe('L2ChugSplashDeployer', () => {
  let signer1: Signer
  let signer2: Signer
  before(async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  let Mock__OVM_ExecutionManager: MockContract
  let Mock__OVM_L2CrossDomainMessenger: MockContract
  let Mock__OVM_L2ChugSplashDeployer: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit('OVM_ExecutionManager', {
      address: predeploys.OVM_ExecutionManagerWrapper,
    })
    Mock__OVM_L2CrossDomainMessenger = await smockit(
      'OVM_L2CrossDomainMessenger',
      {
        address: predeploys.OVM_L2CrossDomainMessenger,
      }
    )
    Mock__OVM_L2ChugSplashDeployer = await smockit('L2ChugSplashDeployer', {
      address: predeploys.L2ChugSplashDeployer,
    })
  })

  let Factory__L2ChugSplashOwner: ContractFactory
  before(async () => {
    Factory__L2ChugSplashOwner = await hre.ethers.getContractFactory(
      'L2ChugSplashOwner'
    )
  })

  let L2ChugSplashOwner: Contract
  beforeEach(async () => {
    L2ChugSplashOwner = await Factory__L2ChugSplashOwner.connect(
      signer1
    ).deploy(
      await signer1.getAddress() // _owner
    )
  })

  describe('owner', () => {
    it('should have an owner', async () => {
      expect(await L2ChugSplashOwner.owner()).to.equal(
        await signer1.getAddress()
      )
    })
  })

  describe('renounceOwnership', () => {
    it('should revert if called directly by the owner', async () => {
      await expect(L2ChugSplashOwner.connect(signer1).renounceOwnership()).to.be
        .reverted
    })

    it('should revert if called directly by someone other than the owner', async () => {
      await expect(L2ChugSplashOwner.connect(signer2).renounceOwnership()).to.be
        .reverted
    })

    it('should revert if called by an L1 => L2 message from someone other than the owner', async () => {
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        await signer2.getAddress()
      )

      await expect(
        L2ChugSplashOwner.connect(
          hre.ethers.provider
        ).callStatic.renounceOwnership({
          from: Mock__OVM_L2CrossDomainMessenger.address,
        })
      ).to.be.reverted
    })

    it('should succeed if called via an L1 => L2 message from the owner', async () => {
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        await signer1.getAddress()
      )

      await expect(
        L2ChugSplashOwner.connect(
          hre.ethers.provider
        ).callStatic.renounceOwnership({
          from: Mock__OVM_L2CrossDomainMessenger.address,
        })
      ).to.not.be.reverted
    })
  })

  describe('transferOwnership', () => {
    it('should revert if called directly by the owner', async () => {
      await expect(
        L2ChugSplashOwner.connect(signer1).transferOwnership(NON_ZERO_ADDRESS)
      ).to.be.reverted
    })

    it('should revert if called directly by someone other than the owner', async () => {
      await expect(
        L2ChugSplashOwner.connect(signer2).transferOwnership(NON_ZERO_ADDRESS)
      ).to.be.reverted
    })

    it('should revert if called by an L1 => L2 message from someone other than the owner', async () => {
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        await signer2.getAddress()
      )

      await expect(
        L2ChugSplashOwner.connect(
          hre.ethers.provider
        ).callStatic.transferOwnership(NON_ZERO_ADDRESS, {
          from: Mock__OVM_L2CrossDomainMessenger.address,
        })
      ).to.be.reverted
    })

    it('should succeed if called via an L1 => L2 message from the owner', async () => {
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        await signer1.getAddress()
      )

      await expect(
        L2ChugSplashOwner.connect(
          hre.ethers.provider
        ).callStatic.transferOwnership(NON_ZERO_ADDRESS, {
          from: Mock__OVM_L2CrossDomainMessenger.address,
        })
      ).to.not.be.reverted
    })
  })

  describe('fallback function', () => {
    it('should revert if called directly by the owner', async () => {
      await expect(
        signer1.sendTransaction({
          to: L2ChugSplashOwner.address,
        })
      ).to.be.reverted
    })

    it('should revert if called directly by someone other than the owner', async () => {
      await expect(
        signer2.sendTransaction({
          to: L2ChugSplashOwner.address,
        })
      ).to.be.reverted
    })

    it('should revert if called by an L1 => L2 message from someone other than the owner', async () => {
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        await signer2.getAddress()
      )

      await expect(
        hre.ethers.provider.call({
          to: L2ChugSplashOwner.address,
          from: Mock__OVM_L2CrossDomainMessenger.address,
        })
      ).to.be.reverted
    })

    describe('when called by an L1 => L2 message from the owner', async () => {
      beforeEach(async () => {
        Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
          await signer1.getAddress()
        )
      })

      it('should be able to trigger approveTransactionBundle', async () => {
        await expect(
          hre.ethers.provider.call({
            to: L2ChugSplashOwner.address,
            from: Mock__OVM_L2CrossDomainMessenger.address,
            data: Mock__OVM_L2ChugSplashDeployer.interface.encodeFunctionData(
              'approveTransactionBundle',
              [ethers.constants.HashZero, ethers.BigNumber.from(0)]
            ),
          })
        ).to.not.be.reverted

        expect(
          Mock__OVM_L2ChugSplashDeployer.smocked.approveTransactionBundle
            .calls[0]
        ).to.deep.equal([ethers.constants.HashZero, ethers.BigNumber.from(0)])
      })

      it('should be able to trigger cancelTransactionBundle', async () => {
        await expect(
          hre.ethers.provider.call({
            to: L2ChugSplashOwner.address,
            from: Mock__OVM_L2CrossDomainMessenger.address,
            data: Mock__OVM_L2ChugSplashDeployer.interface.encodeFunctionData(
              'cancelTransactionBundle'
            ),
          })
        ).to.not.be.reverted

        expect(
          Mock__OVM_L2ChugSplashDeployer.smocked.cancelTransactionBundle
            .calls[0]
        ).to.not.be.undefined
      })

      it('should be able to trigger overrideTransactionBundle', async () => {
        await expect(
          hre.ethers.provider.call({
            to: L2ChugSplashOwner.address,
            from: Mock__OVM_L2CrossDomainMessenger.address,
            data: Mock__OVM_L2ChugSplashDeployer.interface.encodeFunctionData(
              'overrideTransactionBundle',
              [ethers.constants.HashZero, ethers.BigNumber.from(0)]
            ),
          })
        ).to.not.be.reverted

        expect(
          Mock__OVM_L2ChugSplashDeployer.smocked.overrideTransactionBundle
            .calls[0]
        ).to.deep.equal([ethers.constants.HashZero, ethers.BigNumber.from(0)])
      })
    })
  })
})
