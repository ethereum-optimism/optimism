import { expect } from '../../setup'

/* Imports: External */
import hre from 'hardhat'
import { ethers, Contract, Signer, ContractFactory, Wallet } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import {
  ChugSplashActionBundle,
  getChugSplashActionBundle,
} from '@eth-optimism/chugsplash'

/* Imports: Internal */
import { predeploys } from '../../../src'
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../helpers'
import { toPlainObject } from 'lodash'

describe('L2ChugSplashDeployer', () => {
  let wallet: Wallet
  before(async () => {
    ;[wallet] = hre.waffle.provider.getWallets()
  })

  let signer1: Signer
  let signer2: Signer
  before(async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  let Mock__OVM_ExecutionManager: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit('OVM_ExecutionManager', {
      address: predeploys.OVM_ExecutionManagerWrapper,
    })

    Mock__OVM_ExecutionManager.smocked.ovmCHAINID.will.return.with(420)
  })

  let Factory__L2ChugSplashDeployer: ContractFactory
  before(async () => {
    Factory__L2ChugSplashDeployer = await hre.ethers.getContractFactory(
      'L2ChugSplashDeployer'
    )
  })

  let L2ChugSplashDeployer: Contract
  beforeEach(async () => {
    L2ChugSplashDeployer = await Factory__L2ChugSplashDeployer.connect(
      signer1
    ).deploy(
      await signer1.getAddress() // _owner
    )
  })

  describe('owner', () => {
    it('should have an owner', async () => {
      expect(await L2ChugSplashDeployer.owner()).to.equal(
        await signer1.getAddress()
      )
    })
  })

  describe('approveTransactionBundle', () => {
    it('should revert if caller is not the owner', async () => {
      await expect(
        L2ChugSplashDeployer.connect(signer2).approveTransactionBundle(
          ethers.constants.HashZero,
          0
        )
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should allow the owner to approve a new transaction bundle', async () => {
      await expect(
        L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
          NON_NULL_BYTES32,
          1234
        )
      ).to.not.be.reverted

      expect(await L2ChugSplashDeployer.currentBundleHash()).to.equal(
        NON_NULL_BYTES32
      )

      expect(await L2ChugSplashDeployer.currentBundleSize()).to.equal(1234)
    })

    it('should revert if trying to approve a bundle with the empty hash', async () => {
      await expect(
        L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
          ethers.constants.HashZero,
          1234
        )
      ).to.be.revertedWith(
        'ChugSplashDeployer: bundle hash must not be the empty hash'
      )
    })

    it('should revert if trying to approve a bundle with no actions', async () => {
      await expect(
        L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
          NON_NULL_BYTES32,
          0
        )
      ).to.be.revertedWith(
        'ChugSplashDeployer: bundle must include at least one action'
      )
    })

    it('should revert if trying to approve a bundle when another bundle is already active', async () => {
      await L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
        NON_NULL_BYTES32,
        1234
      )

      await expect(
        L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
          NON_NULL_BYTES32,
          1234
        )
      ).to.be.revertedWith(
        'ChugSplashDeployer: previous bundle is still active'
      )
    })
  })

  describe('executeAction', () => {
    const dummyAction = {
      actionType: 0,
      target: NON_ZERO_ADDRESS,
      data: '0x1234',
    }

    const dummyActionProof = {
      actionIndex: 0,
      siblings: [],
    }

    it('should revert if there is no active upgrade bundle', async () => {
      await expect(
        L2ChugSplashDeployer.executeAction(dummyAction, dummyActionProof)
      ).to.be.revertedWith('ChugSplashDeployer: there is no active bundle')
    })

    describe('while there is an active upgrade bundle', () => {
      const actions = [
        {
          target: NON_ZERO_ADDRESS,
          code: '0x1234',
        },
        {
          target: NON_ZERO_ADDRESS,
          key: `0x${'11'.repeat(32)}`,
          value: `0x${'22'.repeat(32)}`,
        },
      ]
      const bundle: ChugSplashActionBundle = getChugSplashActionBundle(actions)

      beforeEach(async () => {
        await L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
          bundle.root,
          bundle.actions.length
        )
      })

      it('should revert if the given action proof is invalid (1: bad action index)', async () => {
        await expect(
          L2ChugSplashDeployer.executeAction(bundle.actions[0].action, {
            ...bundle.actions[0].proof,
            actionIndex: 1, // Bad action index
          })
        ).to.be.revertedWith('ChugSplashDeployer: invalid action proof')
      })

      it('should revert if the given action proof is invalid (2: bad siblings)', async () => {
        await expect(
          L2ChugSplashDeployer.executeAction(bundle.actions[0].action, {
            ...bundle.actions[0].proof,
            siblings: [ethers.constants.HashZero], // Bad siblings
          })
        ).to.be.revertedWith('ChugSplashDeployer: invalid action proof')
      })

      it('should revert if the given action proof is invalid (2: bad action)', async () => {
        await expect(
          L2ChugSplashDeployer.executeAction(
            bundle.actions[0].action,
            bundle.actions[1].proof // Good proof but for the wrong action
          )
        ).to.be.revertedWith('ChugSplashDeployer: invalid action proof')
      })

      it('should be able to trigger a SETCODE action', async () => {
        await expect(
          L2ChugSplashDeployer.executeAction(
            bundle.actions[0].action,
            bundle.actions[0].proof
          )
        ).to.not.be.reverted

        expect(
          toPlainObject(Mock__OVM_ExecutionManager.smocked.ovmSETCODE.calls[0])
        ).to.deep.include({
          _address: actions[0].target,
          _code: actions[0].code,
        })
      })

      it('should be able to trigger a SETSTORAGE action', async () => {
        await expect(
          L2ChugSplashDeployer.executeAction(
            bundle.actions[1].action,
            bundle.actions[1].proof
          )
        ).to.not.be.reverted

        expect(
          toPlainObject(
            Mock__OVM_ExecutionManager.smocked.ovmSETSTORAGE.calls[0]
          )
        ).to.deep.include({
          _address: actions[1].target,
          _key: actions[1].key,
          _value: actions[1].value,
        })
      })

      it('should revert if trying to execute the same action more than once', async () => {
        await expect(
          L2ChugSplashDeployer.executeAction(
            bundle.actions[0].action,
            bundle.actions[0].proof
          )
        ).to.not.be.reverted

        await expect(
          L2ChugSplashDeployer.executeAction(
            bundle.actions[0].action,
            bundle.actions[0].proof
          )
        ).to.be.revertedWith(
          'ChugSplashDeployer: action has already been executed'
        )
      })

      it('should change the upgrade status when the bundle is complete', async () => {
        expect(await L2ChugSplashDeployer.isUpgrading()).to.equal(true)

        for (const action of bundle.actions) {
          await L2ChugSplashDeployer.executeAction(action.action, action.proof)
        }

        expect(await L2ChugSplashDeployer.isUpgrading()).to.equal(false)
      })

      it('should allow the upgrader to submit a new bundle when the previous bundle is complete', async () => {
        for (const action of bundle.actions) {
          await L2ChugSplashDeployer.executeAction(action.action, action.proof)
        }

        await expect(
          L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
            bundle.root,
            bundle.actions.length
          )
        ).to.not.be.reverted
      })
    })
  })

  describe('fallback', () => {
    it('should revert if not provided a valid EIP155 tx', async () => {
      await expect(
        signer1.sendTransaction({
          to: L2ChugSplashDeployer.address,
          data: '0x',
        })
      ).to.be.reverted
    })

    it('should revert if the target is not the L2ChugSplashDeployer', async () => {
      await expect(
        signer1.sendTransaction({
          to: L2ChugSplashDeployer.address,
          data: await wallet.signTransaction({
            chainId: 420,
            to: await signer1.getAddress(),
            data: '0x',
          }),
        })
      ).to.be.reverted
    })

    it('should revert if trying to call approveTransactionBundle', async () => {
      await expect(
        signer1.sendTransaction({
          to: L2ChugSplashDeployer.address,
          data: await wallet.signTransaction({
            chainId: 420,
            to: L2ChugSplashDeployer.address,
            data: L2ChugSplashDeployer.interface.encodeFunctionData(
              'approveTransactionBundle',
              [ethers.constants.HashZero, 1234]
            ),
          }),
        })
      ).to.be.reverted
    })

    it('should be able to trigger executeAction', async () => {
      const bundle: ChugSplashActionBundle = getChugSplashActionBundle([
        {
          target: NON_ZERO_ADDRESS,
          code: '0x1234',
        },
      ])

      await L2ChugSplashDeployer.connect(signer1).approveTransactionBundle(
        bundle.root,
        bundle.actions.length
      )

      await expect(
        signer1.sendTransaction({
          to: L2ChugSplashDeployer.address,
          data: await wallet.signTransaction({
            chainId: 420,
            to: L2ChugSplashDeployer.address,
            data: L2ChugSplashDeployer.interface.encodeFunctionData(
              'executeAction',
              [bundle.actions[0].action, bundle.actions[0].proof]
            ),
          }),
        })
      ).to.not.be.reverted
    })
  })
})
