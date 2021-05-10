import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  NON_NULL_BYTES32,
  getEthTime,
  increaseEthTime,
} from '../../../helpers'

describe('OVM_StateCommitmentChain', () => {
  let sequencer: Signer
  let user: Signer
  before(async () => {
    ;[sequencer, user] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__OVM_CanonicalTransactionChain: MockContract
  let Mock__OVM_BondManager: MockContract
  before(async () => {
    Mock__OVM_CanonicalTransactionChain = await smockit(
      await ethers.getContractFactory('OVM_CanonicalTransactionChain')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_CanonicalTransactionChain',
      Mock__OVM_CanonicalTransactionChain
    )

    Mock__OVM_BondManager = await smockit(
      await ethers.getContractFactory('OVM_BondManager')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_BondManager',
      Mock__OVM_BondManager
    )

    Mock__OVM_BondManager.smocked.isCollateralized.will.return.with(true)

    await AddressManager.setAddress(
      'OVM_Proposer',
      await sequencer.getAddress()
    )
  })

  let Factory__OVM_StateCommitmentChain: ContractFactory
  let Factory__OVM_ChainStorageContainer: ContractFactory
  before(async () => {
    Factory__OVM_StateCommitmentChain = await ethers.getContractFactory(
      'OVM_StateCommitmentChain'
    )

    Factory__OVM_ChainStorageContainer = await ethers.getContractFactory(
      'OVM_ChainStorageContainer'
    )
  })

  let OVM_StateCommitmentChain: Contract
  beforeEach(async () => {
    OVM_StateCommitmentChain = await Factory__OVM_StateCommitmentChain.deploy(
      AddressManager.address,
      60 * 60 * 24 * 7, // 1 week fraud proof window
      60 * 30 // 30 minute sequencer publish window
    )

    const batches = await Factory__OVM_ChainStorageContainer.deploy(
      AddressManager.address,
      'OVM_StateCommitmentChain'
    )

    await AddressManager.setAddress(
      'OVM_ChainStorageContainer-SCC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'OVM_StateCommitmentChain',
      OVM_StateCommitmentChain.address
    )
  })

  describe('appendStateBatch', () => {
    describe('when the provided batch is empty', () => {
      const batch = []

      it('should revert', async () => {
        await expect(
          OVM_StateCommitmentChain.appendStateBatch(batch, 0)
        ).to.be.revertedWith('Cannot submit an empty state batch.')
      })
    })

    describe('when the provided batch is not empty', () => {
      const batch = [NON_NULL_BYTES32]

      describe('when start index does not match total elements', () => {
        it('should revert', async () => {
          await expect(
            OVM_StateCommitmentChain.appendStateBatch(batch, 1)
          ).to.be.revertedWith(
            'Actual batch start index does not match expected start index.'
          )
        })
      })

      describe('when submitting more elements than present in the OVM_CanonicalTransactionChain', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
            batch.length - 1
          )
        })

        it('should revert', async () => {
          await expect(
            OVM_StateCommitmentChain.appendStateBatch(batch, 0)
          ).to.be.revertedWith(
            'Number of state roots cannot exceed the number of canonical transactions.'
          )
        })
      })

      describe('when not submitting more elements than present in the OVM_CanonicalTransactionChain', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
            batch.length
          )
        })

        it('should append the state batch', async () => {
          await expect(OVM_StateCommitmentChain.appendStateBatch(batch, 0)).to
            .not.be.reverted
        })
      })

      describe('when a sequencer submits ', () => {
        beforeEach(async () => {
          Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
            batch.length * 2
          )

          await OVM_StateCommitmentChain.connect(sequencer).appendStateBatch(
            batch,
            0
          )
        })

        describe('when inside sequencer publish window', () => {
          it('should revert', async () => {
            await expect(
              OVM_StateCommitmentChain.connect(user).appendStateBatch(batch, 1)
            ).to.be.revertedWith(
              'Cannot publish state roots within the sequencer publication window.'
            )
          })
        })

        describe('when outside sequencer publish window', () => {
          beforeEach(async () => {
            const SEQUENCER_PUBLISH_WINDOW = await OVM_StateCommitmentChain.SEQUENCER_PUBLISH_WINDOW()
            await increaseEthTime(
              ethers.provider,
              SEQUENCER_PUBLISH_WINDOW.toNumber() + 1
            )
          })

          it('should succeed', async () => {
            await expect(
              OVM_StateCommitmentChain.connect(user).appendStateBatch(batch, 1)
            ).to.not.be.reverted
          })
        })
      })
    })
  })

  describe('deleteStateBatch', () => {
    const batch = [NON_NULL_BYTES32]
    const batchHeader = {
      batchIndex: 0,
      batchRoot: NON_NULL_BYTES32,
      batchSize: 1,
      prevTotalElements: 0,
      extraData: ethers.constants.HashZero,
    }

    beforeEach(async () => {
      Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
        batch.length
      )
      await OVM_StateCommitmentChain.appendStateBatch(batch, 0)
      batchHeader.extraData = ethers.utils.defaultAbiCoder.encode(
        ['uint256', 'address'],
        [await getEthTime(ethers.provider), await sequencer.getAddress()]
      )
    })

    describe('when the sender is not the OVM_FraudVerifier', () => {
      before(async () => {
        await AddressManager.setAddress(
          'OVM_FraudVerifier',
          constants.AddressZero
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_StateCommitmentChain.deleteStateBatch(batchHeader)
        ).to.be.revertedWith(
          'State batches can only be deleted by the OVM_FraudVerifier.'
        )
      })
    })

    describe('when the sender is the OVM_FraudVerifier', () => {
      before(async () => {
        await AddressManager.setAddress(
          'OVM_FraudVerifier',
          await sequencer.getAddress()
        )
      })

      describe('when the provided batch index is greater than the total submitted', () => {
        it('should revert', async () => {
          await expect(
            OVM_StateCommitmentChain.deleteStateBatch({
              ...batchHeader,
              batchIndex: 1,
            })
          ).to.be.revertedWith('Index out of bounds.')
        })
      })

      describe('when the provided batch index is not greater than the total submitted', () => {
        describe('when the provided batch header is invalid', () => {
          it('should revert', async () => {
            await expect(
              OVM_StateCommitmentChain.deleteStateBatch({
                ...batchHeader,
                extraData: '0x' + '22'.repeat(32),
              })
            ).to.be.revertedWith('Invalid batch header.')
          })
        })

        describe('when the provided batch header is valid', () => {
          it('should remove the batch and all following batches', async () => {
            await expect(OVM_StateCommitmentChain.deleteStateBatch(batchHeader))
              .to.not.be.reverted
          })
        })
      })
    })
  })

  describe('getTotalElements', () => {
    describe('when no batch elements have been inserted', () => {
      it('should return zero', async () => {
        expect(await OVM_StateCommitmentChain.getTotalElements()).to.equal(0)
      })
    })

    describe('when one batch element has been inserted', () => {
      beforeEach(async () => {
        const batch = [NON_NULL_BYTES32]
        Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
          batch.length
        )
        await OVM_StateCommitmentChain.appendStateBatch(batch, 0)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_StateCommitmentChain.getTotalElements()).to.equal(1)
      })
    })

    describe('when 64 batch elements have been inserted in one batch', () => {
      beforeEach(async () => {
        const batch = Array(64).fill(NON_NULL_BYTES32)
        Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
          batch.length
        )
        await OVM_StateCommitmentChain.appendStateBatch(batch, 0)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_StateCommitmentChain.getTotalElements()).to.equal(64)
      })
    })

    describe('when 32 batch elements have been inserted in each of two batches', () => {
      beforeEach(async () => {
        const batch = Array(32).fill(NON_NULL_BYTES32)
        Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
          batch.length * 2
        )
        await OVM_StateCommitmentChain.appendStateBatch(batch, 0)
        await OVM_StateCommitmentChain.appendStateBatch(batch, 32)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_StateCommitmentChain.getTotalElements()).to.equal(64)
      })
    })
  })

  describe('getTotalBatches()', () => {
    describe('when no batches have been inserted', () => {
      it('should return zero', async () => {
        expect(await OVM_StateCommitmentChain.getTotalBatches()).to.equal(0)
      })
    })

    describe('when one batch has been inserted', () => {
      beforeEach(async () => {
        const batch = [NON_NULL_BYTES32]
        Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
          batch.length
        )
        await OVM_StateCommitmentChain.appendStateBatch(batch, 0)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_StateCommitmentChain.getTotalBatches()).to.equal(1)
      })
    })

    describe('when 8 batches have been inserted', () => {
      beforeEach(async () => {
        const batch = [NON_NULL_BYTES32]
        Mock__OVM_CanonicalTransactionChain.smocked.getTotalElements.will.return.with(
          batch.length * 8
        )

        for (let i = 0; i < 8; i++) {
          await OVM_StateCommitmentChain.appendStateBatch(batch, i)
        }
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_StateCommitmentChain.getTotalBatches()).to.equal(8)
      })
    })
  })

  describe('verifyElement()', () => {
    it('should revert when given an invalid batch header', async () => {
      // TODO
    })

    it('should revert when given an invalid inclusion proof', async () => {
      // TODO
    })

    it('should return true when given a valid proof', async () => {
      // TODO
    })
  })
})
