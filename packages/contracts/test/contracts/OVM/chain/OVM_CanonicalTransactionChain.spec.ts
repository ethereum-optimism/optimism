import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  getEthTime,
  setEthTime,
  NON_NULL_BYTES32,
  FORCE_INCLUSION_PERIOD_SECONDS,
  ZERO_ADDRESS,
} from '../../../helpers'

describe('OVM_CanonicalTransactionChain', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__OVM_L1ToL2TransactionQueue: MockContract
  before(async () => {
    Mock__OVM_L1ToL2TransactionQueue = smockit(
      await ethers.getContractFactory('OVM_L1ToL2TransactionQueue')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_L1ToL2TransactionQueue',
      Mock__OVM_L1ToL2TransactionQueue
    )
  })

  let Factory__OVM_CanonicalTransactionChain: ContractFactory
  before(async () => {
    Factory__OVM_CanonicalTransactionChain = await ethers.getContractFactory(
      'OVM_CanonicalTransactionChain'
    )
  })

  let OVM_CanonicalTransactionChain: Contract
  beforeEach(async () => {
    OVM_CanonicalTransactionChain = await Factory__OVM_CanonicalTransactionChain.deploy(
      AddressManager.address,
      FORCE_INCLUSION_PERIOD_SECONDS
    )
  })

  describe('appendQueueBatch()', () => {
    describe('when the L1ToL2TransactionQueue queue is empty', () => {
      before(() => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
      })

      it('should revert', async () => {
        await expect(
          OVM_CanonicalTransactionChain.appendQueueBatch()
        ).to.be.revertedWith('No batches are currently queued to be appended.')
      })
    })

    describe('when the L1ToL2TransactionQueue queue is not empty', () => {
      before(() => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(1)
      })

      describe('when the inclusion delay period has not elapsed', () => {
        beforeEach(async () => {
          const timestamp = await getEthTime(ethers.provider)
          Mock__OVM_L1ToL2TransactionQueue.smocked.peek.will.return.with([
            {
              timestamp,
              batchRoot: NON_NULL_BYTES32,
              isL1ToL2Batch: true,
            },
          ])

          await setEthTime(
            ethers.provider,
            timestamp + FORCE_INCLUSION_PERIOD_SECONDS / 2
          )
        })

        it('should revert', async () => {
          await expect(
            OVM_CanonicalTransactionChain.appendQueueBatch()
          ).to.be.revertedWith(
            'Cannot append until the inclusion delay period has elapsed.'
          )
        })
      })

      describe('when the inclusion delay period has elapsed', () => {
        beforeEach(async () => {
          const timestamp = await getEthTime(ethers.provider)
          Mock__OVM_L1ToL2TransactionQueue.smocked.dequeue.will.return()
          Mock__OVM_L1ToL2TransactionQueue.smocked.peek.will.return.with([
            {
              timestamp,
              batchRoot: NON_NULL_BYTES32,
              isL1ToL2Batch: true,
            },
          ])

          await setEthTime(
            ethers.provider,
            timestamp + FORCE_INCLUSION_PERIOD_SECONDS
          )
        })

        it('should append the top element of the queue and attempt to dequeue', async () => {
          await expect(OVM_CanonicalTransactionChain.appendQueueBatch()).to.not
            .be.reverted

          // TODO: Check that the batch root was inserted.

          expect(
            Mock__OVM_L1ToL2TransactionQueue.smocked.dequeue.calls.length
          ).to.equal(1)
        })
      })
    })
  })

  describe('appendSequencerBatch()', () => {
    describe('when the sender is not the sequencer', () => {
      before(async () => {
        await AddressManager.setAddress('Sequencer', ZERO_ADDRESS)
      })

      it('should revert', async () => {
        await expect(
          OVM_CanonicalTransactionChain.appendSequencerBatch([], 0)
        ).to.be.revertedWith('Function can only be called by the Sequencer.')
      })
    })

    describe('when the sender is the sequencer', () => {
      before(async () => {
        await AddressManager.setAddress('Sequencer', await signer.getAddress())
      })

      describe('when the given batch is empty', () => {
        const batch = []

        it('should revert', async () => {
          await expect(
            OVM_CanonicalTransactionChain.appendSequencerBatch(batch, 0)
          ).to.be.revertedWith('Cannot submit an empty batch.')
        })
      })

      describe('when the given batch is not empty', () => {
        const batch = [NON_NULL_BYTES32]

        describe('when the timestamp is not greater than the previous OVM timestamp', () => {
          const timestamp = 0

          it('should revert', async () => {
            await expect(
              OVM_CanonicalTransactionChain.appendSequencerBatch(
                batch,
                timestamp
              )
            ).to.be.revertedWith(
              'Batch timestamp must be later than the last OVM timestamp.'
            )
          })
        })

        describe('when the timestamp is greater than the previous OVM timestamp', () => {
          const timestamp = 1000

          describe('when the queue is not empty', () => {
            before(() => {
              Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(1)
            })

            describe('when the first element in the queue is older than the provided batch', () => {
              before(() => {
                Mock__OVM_L1ToL2TransactionQueue.smocked.peek.will.return.with([
                  {
                    timestamp: timestamp / 2,
                    batchRoot: NON_NULL_BYTES32,
                    isL1ToL2Batch: true,
                  },
                ])
              })

              it('should revert', async () => {
                await expect(
                  OVM_CanonicalTransactionChain.appendSequencerBatch(
                    batch,
                    timestamp
                  )
                ).to.be.revertedWith(
                  'Older queue batches must be processed before a newer sequencer batch.'
                )
              })
            })

            describe('when the first element in the queue is not older than the provided batch', () => {
              before(() => {
                Mock__OVM_L1ToL2TransactionQueue.smocked.peek.will.return.with([
                  {
                    timestamp,
                    batchRoot: NON_NULL_BYTES32,
                    isL1ToL2Batch: true,
                  },
                ])
              })

              it('should insert the sequencer batch', async () => {
                await expect(
                  OVM_CanonicalTransactionChain.appendSequencerBatch(
                    batch,
                    timestamp
                  )
                ).to.not.be.reverted

                // TODO: Check that the batch was inserted correctly.
              })
            })
          })

          describe('when the queue is empty', async () => {
            before(() => {
              Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
            })

            it('should insert the sequencer batch', async () => {
              await expect(
                OVM_CanonicalTransactionChain.appendSequencerBatch(
                  batch,
                  timestamp
                )
              ).to.not.be.reverted

              // TODO: Check that the batch was inserted correctly.
            })
          })
        })
      })
    })
  })

  describe('getTotalElements()', () => {
    describe('when no batch elements have been inserted', () => {
      it('should return zero', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalElements()).to.equal(
          0
        )
      })
    })

    describe('when one batch element has been inserted', () => {
      beforeEach(async () => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
        await OVM_CanonicalTransactionChain.appendSequencerBatch(
          [NON_NULL_BYTES32],
          1000
        )
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalElements()).to.equal(
          1
        )
      })
    })

    describe('when 64 batch elements have been inserted in one batch', () => {
      const batch = Array(64).fill(NON_NULL_BYTES32)
      beforeEach(async () => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
        await OVM_CanonicalTransactionChain.appendSequencerBatch(batch, 1000)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalElements()).to.equal(
          64
        )
      })
    })

    describe('when 32 batch elements have been inserted in each of two batches', () => {
      const batch = Array(32).fill(NON_NULL_BYTES32)
      beforeEach(async () => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
        await OVM_CanonicalTransactionChain.appendSequencerBatch(batch, 1000)
        await OVM_CanonicalTransactionChain.appendSequencerBatch(batch, 2000)
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalElements()).to.equal(
          64
        )
      })
    })
  })

  describe('getTotalBatches()', () => {
    describe('when no batches have been inserted', () => {
      it('should return zero', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalBatches()).to.equal(
          0
        )
      })
    })

    describe('when one batch has been inserted', () => {
      beforeEach(async () => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
        await OVM_CanonicalTransactionChain.appendSequencerBatch(
          [NON_NULL_BYTES32],
          1000
        )
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalBatches()).to.equal(
          1
        )
      })
    })

    describe('when 8 batches have been inserted', () => {
      beforeEach(async () => {
        Mock__OVM_L1ToL2TransactionQueue.smocked.size.will.return.with(0)
        for (let i = 0; i < 8; i++) {
          await OVM_CanonicalTransactionChain.appendSequencerBatch(
            [NON_NULL_BYTES32],
            1000 * (i + 1)
          )
        }
      })

      it('should return the number of inserted batch elements', async () => {
        expect(await OVM_CanonicalTransactionChain.getTotalBatches()).to.equal(
          8
        )
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
