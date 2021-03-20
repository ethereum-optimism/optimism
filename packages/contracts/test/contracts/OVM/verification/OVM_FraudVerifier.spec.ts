import { expect } from '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract, BigNumber } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  setProxyTarget,
  DUMMY_BATCH_HEADERS,
  DUMMY_BATCH_PROOFS,
  DUMMY_OVM_TRANSACTIONS,
  NON_NULL_BYTES32,
  hashTransaction,
} from '../../../helpers'

const DUMMY_TX_CHAIN_ELEMENTS = [...Array(10).keys()].map((i) => {
  return {
    isSequenced: false,
    queueIndex: BigNumber.from(0),
    timestamp: BigNumber.from(i),
    blockNumber: BigNumber.from(0),
    txData: ethers.constants.HashZero,
  }
})

const DUMMY_HASH = hashTransaction(DUMMY_OVM_TRANSACTIONS[0])

const DUMMY_BATCH_PROOFS_WITH_INDEX = [
  {
    index: 11,
    siblings: [ethers.constants.HashZero],
  },
]

describe('OVM_FraudVerifier', () => {
  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__OVM_StateCommitmentChain: MockContract
  let Mock__OVM_CanonicalTransactionChain: MockContract
  let Mock__OVM_StateTransitioner: MockContract
  let Mock__OVM_StateTransitionerFactory: MockContract
  let Mock__OVM_BondManager: MockContract
  before(async () => {
    Mock__OVM_StateCommitmentChain = await smockit(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    Mock__OVM_CanonicalTransactionChain = await smockit(
      await ethers.getContractFactory('OVM_CanonicalTransactionChain')
    )

    Mock__OVM_StateTransitioner = await smockit(
      await ethers.getContractFactory('OVM_StateTransitioner')
    )

    Mock__OVM_StateTransitionerFactory = await smockit(
      await ethers.getContractFactory('OVM_StateTransitionerFactory')
    )

    Mock__OVM_BondManager = await smockit(
      await ethers.getContractFactory('OVM_BondManager')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain
    )

    await setProxyTarget(
      AddressManager,
      'OVM_CanonicalTransactionChain',
      Mock__OVM_CanonicalTransactionChain
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateTransitionerFactory',
      Mock__OVM_StateTransitionerFactory
    )

    await setProxyTarget(
      AddressManager,
      'OVM_BondManager',
      Mock__OVM_BondManager
    )

    Mock__OVM_StateTransitionerFactory.smocked.create.will.return.with(
      Mock__OVM_StateTransitioner.address
    )
  })

  let Factory__OVM_FraudVerifier: ContractFactory
  before(async () => {
    Factory__OVM_FraudVerifier = await ethers.getContractFactory(
      'OVM_FraudVerifier'
    )
  })

  let OVM_FraudVerifier: Contract
  beforeEach(async () => {
    OVM_FraudVerifier = await Factory__OVM_FraudVerifier.deploy(
      AddressManager.address
    )
  })

  describe('initializeFraudVerification', () => {
    describe('when provided an invalid pre-state root inclusion proof', () => {
      before(() => {
        Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
          false
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_FraudVerifier.initializeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_OVM_TRANSACTIONS[0],
            DUMMY_TX_CHAIN_ELEMENTS[0],
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0]
          )
        ).to.be.revertedWith('Invalid pre-state root inclusion proof.')
      })
    })

    describe('when provided a valid pre-state root inclusion proof', () => {
      before(() => {
        Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
          true
        )
      })

      describe('when provided an invalid transaction inclusion proof', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.smocked.verifyTransaction.will.return.with(
            false
          )
        })

        it('should revert', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              ethers.constants.HashZero,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_OVM_TRANSACTIONS[0],
              DUMMY_TX_CHAIN_ELEMENTS[0],
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0]
            )
          ).to.be.revertedWith('Invalid transaction inclusion proof.')
        })
      })

      describe('when provided a valid transaction inclusion proof', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.smocked.verifyTransaction.will.return.with(
            true
          )
        })

        it('should deploy a new state transitioner', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              ethers.constants.HashZero,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_OVM_TRANSACTIONS[0],
              DUMMY_TX_CHAIN_ELEMENTS[0],
              DUMMY_BATCH_HEADERS[0],
              {
                ...DUMMY_BATCH_PROOFS[0],
                index: DUMMY_BATCH_PROOFS[0].index + 1,
              }
            )
          ).to.not.be.reverted

          expect(
            await OVM_FraudVerifier.getStateTransitioner(
              ethers.constants.HashZero,
              DUMMY_HASH
            )
          ).to.equal(Mock__OVM_StateTransitioner.address)
        })

        it('should revert when provided with a incorrect transaction root global index', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              ethers.constants.HashZero,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_OVM_TRANSACTIONS[0],
              DUMMY_TX_CHAIN_ELEMENTS[0],
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS_WITH_INDEX[0]
            )
          ).to.be.revertedWith(
            'Pre-state root global index must equal to the transaction root global index.'
          )
        })
      })
    })
  })

  describe('finalizeFraudVerification', () => {
    beforeEach(async () => {
      Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
        true
      )
      Mock__OVM_CanonicalTransactionChain.smocked.verifyTransaction.will.return.with(
        true
      )

      await OVM_FraudVerifier.initializeFraudVerification(
        ethers.constants.HashZero,
        DUMMY_BATCH_HEADERS[0],
        DUMMY_BATCH_PROOFS[0],
        DUMMY_OVM_TRANSACTIONS[0],
        DUMMY_TX_CHAIN_ELEMENTS[0],
        DUMMY_BATCH_HEADERS[0],
        {
          ...DUMMY_BATCH_PROOFS[0],
          index: DUMMY_BATCH_PROOFS[0].index + 1,
        }
      )
    })

    describe('when the transition process is not complete', () => {
      before(async () => {
        Mock__OVM_StateTransitioner.smocked.isComplete.will.return.with(false)
      })

      it('should revert', async () => {
        await expect(
          OVM_FraudVerifier.finalizeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_HASH,
            NON_NULL_BYTES32,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0]
          )
        ).to.be.revertedWith(
          'State transition process must be completed prior to finalization.'
        )
      })
    })

    describe('when the transition process is complete', () => {
      before(() => {
        Mock__OVM_StateTransitioner.smocked.isComplete.will.return.with(true)
      })

      describe('when provided an invalid post-state root index', () => {
        const batchProof = {
          ...DUMMY_BATCH_PROOFS[0],
          index: DUMMY_BATCH_PROOFS[0].index + 2,
        }

        it('should revert', async () => {
          await expect(
            OVM_FraudVerifier.finalizeFraudVerification(
              ethers.constants.HashZero,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_HASH,
              NON_NULL_BYTES32,
              DUMMY_BATCH_HEADERS[0],
              batchProof
            )
          ).to.be.revertedWith(
            'Post-state root global index must equal to the pre state root global index plus one.'
          )
        })
      })

      describe('when provided a valid post-state root index', () => {
        const batchProof = {
          ...DUMMY_BATCH_PROOFS[0],
          index: DUMMY_BATCH_PROOFS[0].index + 1,
        }

        describe('when provided an invalid pre-state root inclusion proof', () => {
          beforeEach(() => {
            Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
              false
            )
          })

          it('should revert', async () => {
            await expect(
              OVM_FraudVerifier.finalizeFraudVerification(
                ethers.constants.HashZero,
                DUMMY_BATCH_HEADERS[0],
                DUMMY_BATCH_PROOFS[0],
                DUMMY_HASH,
                NON_NULL_BYTES32,
                DUMMY_BATCH_HEADERS[0],
                batchProof
              )
            ).to.be.revertedWith('Invalid pre-state root inclusion proof.')
          })
        })

        describe('when provided a valid pre-state root inclusion proof', () => {
          before(() => {
            Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
              true
            )
          })

          describe('when provided an invalid post-state root inclusion proof', () => {
            beforeEach(() => {
              Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
                (stateRoot: string, ...args: any) => {
                  return stateRoot !== NON_NULL_BYTES32
                }
              )
            })

            it('should revert', async () => {
              await expect(
                OVM_FraudVerifier.finalizeFraudVerification(
                  ethers.constants.HashZero,
                  DUMMY_BATCH_HEADERS[0],
                  DUMMY_BATCH_PROOFS[0],
                  DUMMY_HASH,
                  NON_NULL_BYTES32,
                  DUMMY_BATCH_HEADERS[0],
                  batchProof
                )
              ).to.be.revertedWith('Invalid post-state root inclusion proof.')
            })
          })

          describe('when provided a valid post-state root inclusion proof', () => {
            before(() => {
              Mock__OVM_StateCommitmentChain.smocked.verifyStateCommitment.will.return.with(
                true
              )
            })

            describe('when the provided post-state root does not differ from the computed one', () => {
              before(() => {
                Mock__OVM_StateTransitioner.smocked.getPostStateRoot.will.return.with(
                  NON_NULL_BYTES32
                )
              })

              it('should revert', async () => {
                await expect(
                  OVM_FraudVerifier.finalizeFraudVerification(
                    ethers.constants.HashZero,
                    DUMMY_BATCH_HEADERS[0],
                    DUMMY_BATCH_PROOFS[0],
                    DUMMY_HASH,
                    NON_NULL_BYTES32,
                    DUMMY_BATCH_HEADERS[0],
                    batchProof
                  )
                ).to.be.revertedWith(
                  'State transition has not been proven fraudulent.'
                )
              })
            })

            describe('when the provided post-state root differs from the computed one', () => {
              before(() => {
                Mock__OVM_StateTransitioner.smocked.getPostStateRoot.will.return.with(
                  ethers.constants.HashZero
                )
              })

              it('should succeed and attempt to delete a state batch', async () => {
                await OVM_FraudVerifier.finalizeFraudVerification(
                  ethers.constants.HashZero,
                  DUMMY_BATCH_HEADERS[0],
                  DUMMY_BATCH_PROOFS[0],
                  DUMMY_HASH,
                  NON_NULL_BYTES32,
                  DUMMY_BATCH_HEADERS[0],
                  batchProof
                )

                expect(
                  Mock__OVM_StateCommitmentChain.smocked.deleteStateBatch
                    .calls[0]
                ).to.deep.equal([
                  Object.values(DUMMY_BATCH_HEADERS[0]).map((value) => {
                    return Number.isInteger(value)
                      ? BigNumber.from(value)
                      : value
                  }),
                ])
              })
            })
          })
        })
      })

      describe('multiple fraud proofs for the same pre-execution state', () => {
        let state2: any
        const DUMMY_HASH_2 = hashTransaction(DUMMY_OVM_TRANSACTIONS[1])
        beforeEach(async () => {
          state2 = await smockit(
            await ethers.getContractFactory('OVM_StateTransitioner')
          )

          Mock__OVM_StateTransitionerFactory.smocked.create.will.return.with(
            state2.address
          )

          Mock__OVM_StateTransitioner.smocked.getPostStateRoot.will.return.with(
            ethers.constants.HashZero
          )

          state2.smocked.getPostStateRoot.will.return.with(
            ethers.constants.HashZero
          )
        })

        it('creates multiple state transitioners per tx hash', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              ethers.constants.HashZero,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_OVM_TRANSACTIONS[1],
              DUMMY_TX_CHAIN_ELEMENTS[0],
              DUMMY_BATCH_HEADERS[0],
              {
                ...DUMMY_BATCH_PROOFS[0],
                index: DUMMY_BATCH_PROOFS[0].index + 1,
              }
            )
          ).to.not.be.reverted

          expect(
            await OVM_FraudVerifier.getStateTransitioner(
              ethers.constants.HashZero,
              DUMMY_HASH
            )
          ).to.equal(Mock__OVM_StateTransitioner.address)
          expect(
            await OVM_FraudVerifier.getStateTransitioner(
              ethers.constants.HashZero,
              DUMMY_HASH_2
            )
          ).to.equal(state2.address)
        })

        const batchProof = {
          ...DUMMY_BATCH_PROOFS[0],
          index: DUMMY_BATCH_PROOFS[0].index + 1,
        }

        // TODO: Appears to be failing because of a bug in smock.
        it.skip('Case 1: allows proving fraud on the same pre-state root twice', async () => {
          // finalize previous fraud
          await OVM_FraudVerifier.finalizeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_HASH,
            NON_NULL_BYTES32,
            DUMMY_BATCH_HEADERS[0],
            batchProof
          )

          // start new fraud
          await OVM_FraudVerifier.initializeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_OVM_TRANSACTIONS[1],
            DUMMY_TX_CHAIN_ELEMENTS[1],
            DUMMY_BATCH_HEADERS[1],
            {
              ...DUMMY_BATCH_PROOFS[0],
              index: DUMMY_BATCH_PROOFS[0].index + 1,
            }
          )

          // finalize it as well
          await OVM_FraudVerifier.finalizeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_HASH_2,
            NON_NULL_BYTES32,
            DUMMY_BATCH_HEADERS[1],
            batchProof
          )

          // the new batch was deleted
          expect(
            Mock__OVM_StateCommitmentChain.smocked.deleteStateBatch.calls[0]
          ).to.deep.equal([
            Object.values(DUMMY_BATCH_HEADERS[1]).map((value) => {
              return Number.isInteger(value) ? BigNumber.from(value) : value
            }),
          ])
        })

        // TODO: Appears to be failing because of a bug in smock.
        it.skip('Case 2: does not get blocked by the first transitioner', async () => {
          // start new fraud
          await OVM_FraudVerifier.initializeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_OVM_TRANSACTIONS[1],
            DUMMY_TX_CHAIN_ELEMENTS[1],
            DUMMY_BATCH_HEADERS[1],
            {
              ...DUMMY_BATCH_PROOFS[0],
              index: DUMMY_BATCH_PROOFS[0].index + 1,
            }
          )

          // finalize the new fraud first
          await OVM_FraudVerifier.finalizeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_HASH_2,
            NON_NULL_BYTES32,
            DUMMY_BATCH_HEADERS[1],
            batchProof
          )

          // the new fraud's batch was deleted
          expect(
            Mock__OVM_StateCommitmentChain.smocked.deleteStateBatch.calls[0]
          ).to.deep.equal([
            Object.values(DUMMY_BATCH_HEADERS[1]).map((value) => {
              return Number.isInteger(value) ? BigNumber.from(value) : value
            }),
          ])

          // finalize previous fraud
          await OVM_FraudVerifier.finalizeFraudVerification(
            ethers.constants.HashZero,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_HASH,
            NON_NULL_BYTES32,
            DUMMY_BATCH_HEADERS[0],
            batchProof
          )

          // the old fraud's batch was deleted
          expect(
            Mock__OVM_StateCommitmentChain.smocked.deleteStateBatch.calls[0]
          ).to.deep.equal([
            Object.values(DUMMY_BATCH_HEADERS[0]).map((value) => {
              return Number.isInteger(value) ? BigNumber.from(value) : value
            }),
          ])
        })
      })
    })
  })
})
