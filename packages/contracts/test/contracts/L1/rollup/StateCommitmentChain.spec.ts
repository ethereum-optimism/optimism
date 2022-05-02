import { ethers } from 'hardhat'
import { Contract, constants } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { smock, FakeContract } from '@defi-wonderland/smock'

import { expect } from '../../../setup'
import {
  deploy,
  NON_NULL_BYTES32,
  getEthTime,
  increaseEthTime,
} from '../../../helpers'

describe('StateCommitmentChain', () => {
  let sequencer: SignerWithAddress
  let user: SignerWithAddress
  const batch = [NON_NULL_BYTES32]
  before(async () => {
    ;[sequencer, user] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let Fake__CanonicalTransactionChain: FakeContract
  let Fake__BondManager: FakeContract
  before(async () => {
    AddressManager = await deploy('Lib_AddressManager')

    Fake__CanonicalTransactionChain = await smock.fake<Contract>(
      'CanonicalTransactionChain'
    )

    await AddressManager.setAddress(
      'CanonicalTransactionChain',
      Fake__CanonicalTransactionChain.address
    )

    Fake__BondManager = await smock.fake<Contract>('BondManager')

    await AddressManager.setAddress('BondManager', Fake__BondManager.address)

    Fake__BondManager.isCollateralized.returns(true)

    await AddressManager.setAddress('OVM_Proposer', sequencer.address)
  })

  let StateCommitmentChain: Contract
  beforeEach(async () => {
    StateCommitmentChain = await deploy('StateCommitmentChain', {
      signer: sequencer,
      args: [
        AddressManager.address,
        60 * 60 * 24 * 7, // 1 week fraud proof window
        60 * 30, // 30 minute sequencer publish window
      ],
    })

    const batches = await deploy('ChainStorageContainer', {
      args: [AddressManager.address, 'StateCommitmentChain'],
    })

    await AddressManager.setAddress(
      'ChainStorageContainer-SCC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'StateCommitmentChain',
      StateCommitmentChain.address
    )
  })

  describe('appendStateBatch', () => {
    describe('when the provided batch is empty', () => {
      it('should revert', async () => {
        await expect(
          StateCommitmentChain.appendStateBatch([], 0)
        ).to.be.revertedWith('Cannot submit an empty state batch.')
      })
    })

    describe('when the provided batch is not empty', () => {
      describe('when start index does not match total elements', () => {
        it('should revert', async () => {
          await expect(
            StateCommitmentChain.appendStateBatch(batch, 1)
          ).to.be.revertedWith(
            'Actual batch start index does not match expected start index.'
          )
        })
      })

      describe('when submitting more elements than present in the CanonicalTransactionChain', () => {
        before(() => {
          Fake__CanonicalTransactionChain.getTotalElements.returns(
            batch.length - 1
          )
        })

        it('should revert', async () => {
          await expect(
            StateCommitmentChain.appendStateBatch(batch, 0)
          ).to.be.revertedWith(
            'Number of state roots cannot exceed the number of canonical transactions.'
          )
        })
      })

      describe('when not submitting more elements than present in the CanonicalTransactionChain', () => {
        before(() => {
          Fake__CanonicalTransactionChain.getTotalElements.returns(batch.length)
        })

        it('should append the state batch', async () => {
          await expect(StateCommitmentChain.appendStateBatch(batch, 0)).to.not
            .be.reverted
        })
      })

      describe('when a sequencer submits ', () => {
        beforeEach(async () => {
          Fake__CanonicalTransactionChain.getTotalElements.returns(
            batch.length * 2
          )

          await StateCommitmentChain.appendStateBatch(batch, 0)
        })

        describe('when inside sequencer publish window', () => {
          it('should revert', async () => {
            await expect(
              StateCommitmentChain.connect(user).appendStateBatch(batch, 1)
            ).to.be.revertedWith(
              'Cannot publish state roots within the sequencer publication window.'
            )
          })
        })

        describe('when outside sequencer publish window', () => {
          beforeEach(async () => {
            const SEQUENCER_PUBLISH_WINDOW =
              await StateCommitmentChain.SEQUENCER_PUBLISH_WINDOW()
            await increaseEthTime(
              ethers.provider,
              SEQUENCER_PUBLISH_WINDOW.toNumber() + 1
            )
          })

          it('should succeed', async () => {
            await expect(
              StateCommitmentChain.connect(user).appendStateBatch(batch, 1)
            ).to.not.be.reverted
          })
        })
      })
      describe('when the proposer has not previously staked at the BondManager', () => {
        before(() => {
          Fake__BondManager.isCollateralized.returns(false)
        })

        it('should revert', async () => {
          await expect(
            StateCommitmentChain.appendStateBatch(batch, 0)
          ).to.be.revertedWith(
            'Proposer does not have enough collateral posted'
          )
        })
      })
    })
  })

  describe('deleteStateBatch', () => {
    const batchHeader = {
      batchIndex: 0,
      batchRoot: NON_NULL_BYTES32,
      batchSize: 1,
      prevTotalElements: 0,
      extraData: ethers.constants.HashZero,
    }

    before(() => {
      Fake__BondManager.isCollateralized.returns(true)
    })

    beforeEach(async () => {
      Fake__CanonicalTransactionChain.getTotalElements.returns(batch.length)
      await StateCommitmentChain.appendStateBatch(batch, 0)
      batchHeader.extraData = ethers.utils.defaultAbiCoder.encode(
        ['uint256', 'address'],
        [await getEthTime(ethers.provider), sequencer.address]
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
          StateCommitmentChain.deleteStateBatch(batchHeader)
        ).to.be.revertedWith(
          'State batches can only be deleted by the OVM_FraudVerifier.'
        )
      })
    })

    describe('when the sender is the OVM_FraudVerifier', () => {
      before(async () => {
        await AddressManager.setAddress('OVM_FraudVerifier', sequencer.address)
      })

      describe('when the provided batch index is greater than the total submitted', () => {
        it('should revert', async () => {
          await expect(
            StateCommitmentChain.deleteStateBatch({
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
              StateCommitmentChain.deleteStateBatch({
                ...batchHeader,
                extraData: '0x' + '22'.repeat(32),
              })
            ).to.be.revertedWith('Invalid batch header.')
          })
        })

        describe('when outside fraud proof window', () => {
          beforeEach(async () => {
            const FRAUD_PROOF_WINDOW =
              await StateCommitmentChain.FRAUD_PROOF_WINDOW()
            await increaseEthTime(
              ethers.provider,
              FRAUD_PROOF_WINDOW.toNumber() + 1
            )
          })

          it('should revert', async () => {
            await expect(
              StateCommitmentChain.deleteStateBatch(batchHeader)
            ).to.be.revertedWith(
              'State batches can only be deleted within the fraud proof window.'
            )
          })
        })

        describe('when the provided batch header is valid', () => {
          it('should remove the batch and all following batches', async () => {
            await expect(StateCommitmentChain.deleteStateBatch(batchHeader)).to
              .not.be.reverted
          })
        })
      })
    })
  })

  describe('insideFraudProofWindow', () => {
    const batchHeader = {
      batchIndex: 0,
      batchRoot: NON_NULL_BYTES32,
      batchSize: 1,
      prevTotalElements: 0,
      extraData: ethers.constants.HashZero,
    }

    it('should revert when timestamp is zero', async () => {
      await expect(
        StateCommitmentChain.insideFraudProofWindow({
          ...batchHeader,
          extraData: ethers.utils.defaultAbiCoder.encode(
            ['uint256', 'address'],
            [0, sequencer.address]
          ),
        })
      ).to.be.revertedWith('Batch header timestamp cannot be zero')
    })
  })

  describe('getTotalElements', () => {
    describe('when no batch elements have been inserted', () => {
      it('should return zero', async () => {
        expect(await StateCommitmentChain.getTotalElements()).to.equal(0)
      })
    })

    describe('when one batch element has been inserted', () => {
      beforeEach(async () => {
        Fake__CanonicalTransactionChain.getTotalElements.returns(batch.length)
        await StateCommitmentChain.appendStateBatch(batch, 0)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await StateCommitmentChain.getTotalElements()).to.equal(1)
      })
    })

    describe('when 64 batch elements have been inserted in one batch', () => {
      beforeEach(async () => {
        const batchArray = Array(64).fill(NON_NULL_BYTES32)
        Fake__CanonicalTransactionChain.getTotalElements.returns(
          batchArray.length
        )
        await StateCommitmentChain.appendStateBatch(batchArray, 0)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await StateCommitmentChain.getTotalElements()).to.equal(64)
      })
    })

    describe('when 32 batch elements have been inserted in each of two batches', () => {
      beforeEach(async () => {
        const batchArray = Array(32).fill(NON_NULL_BYTES32)
        Fake__CanonicalTransactionChain.getTotalElements.returns(
          batchArray.length * 2
        )
        await StateCommitmentChain.appendStateBatch(batchArray, 0)
        await StateCommitmentChain.appendStateBatch(batchArray, 32)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await StateCommitmentChain.getTotalElements()).to.equal(64)
      })
    })
  })

  describe('getTotalBatches()', () => {
    describe('when no batches have been inserted', () => {
      it('should return zero', async () => {
        expect(await StateCommitmentChain.getTotalBatches()).to.equal(0)
      })
    })

    describe('when one batch has been inserted', () => {
      beforeEach(async () => {
        Fake__CanonicalTransactionChain.getTotalElements.returns(batch.length)
        await StateCommitmentChain.appendStateBatch(batch, 0)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await StateCommitmentChain.getTotalBatches()).to.equal(1)
      })
    })

    describe('when 8 batches have been inserted', () => {
      beforeEach(async () => {
        Fake__CanonicalTransactionChain.getTotalElements.returns(
          batch.length * 8
        )

        for (let i = 0; i < 8; i++) {
          await StateCommitmentChain.appendStateBatch(batch, i)
        }
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await StateCommitmentChain.getTotalBatches()).to.equal(8)
      })
    })
  })

  describe('getLastSequencerTimestamp', () => {
    describe('when no batches have been inserted', () => {
      it('should return zero', async () => {
        expect(await StateCommitmentChain.getLastSequencerTimestamp()).to.equal(
          0
        )
      })
    })

    describe('when one batch element has been inserted', () => {
      let timestamp
      beforeEach(async () => {
        Fake__CanonicalTransactionChain.getTotalElements.returns(batch.length)
        await StateCommitmentChain.appendStateBatch(batch, 0)
        timestamp = await getEthTime(ethers.provider)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await StateCommitmentChain.getLastSequencerTimestamp()).to.equal(
          timestamp
        )
      })
    })
  })

  describe('verifyStateCommitment()', () => {
    const batchHeader = {
      batchIndex: 0,
      batchRoot: NON_NULL_BYTES32,
      batchSize: 1,
      prevTotalElements: 0,
      extraData: ethers.constants.HashZero,
    }

    const inclusionProof = {
      index: 0,
      siblings: [],
    }

    const element = NON_NULL_BYTES32

    before(async () => {
      Fake__BondManager.isCollateralized.returns(true)
    })

    beforeEach(async () => {
      Fake__CanonicalTransactionChain.getTotalElements.returns(batch.length)
      await StateCommitmentChain.appendStateBatch(batch, 0)
      batchHeader.extraData = ethers.utils.defaultAbiCoder.encode(
        ['uint256', 'address'],
        [await getEthTime(ethers.provider), sequencer.address]
      )
    })

    it('should revert when given an invalid batch header', async () => {
      await expect(
        StateCommitmentChain.verifyStateCommitment(
          element,
          {
            ...batchHeader,
            extraData: '0x' + '22'.repeat(32),
          },
          inclusionProof
        )
      ).to.be.revertedWith('Invalid batch header.')
    })

    it('should revert when given an invalid inclusion proof', async () => {
      await expect(
        StateCommitmentChain.verifyStateCommitment(
          ethers.constants.HashZero,
          batchHeader,
          inclusionProof
        )
      ).to.be.revertedWith('Invalid inclusion proof.')
    })

    it('should return true when given a valid proof', async () => {
      expect(
        await StateCommitmentChain.verifyStateCommitment(
          element,
          batchHeader,
          inclusionProof
        )
      ).to.equal(true)
    })
  })
})
