import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
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
  NULL_BYTES32,
} from '../../../helpers'

describe('OVM_FraudVerifier', () => {
  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__OVM_StateCommitmentChain: MockContract
  let Mock__OVM_CanonicalTransactionChain: MockContract
  let Mock__OVM_StateTransitioner: MockContract
  let Mock__OVM_StateTransitionerFactory: MockContract
  before(async () => {
    Mock__OVM_StateCommitmentChain = smockit(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    Mock__OVM_CanonicalTransactionChain = smockit(
      await ethers.getContractFactory('OVM_CanonicalTransactionChain')
    )

    Mock__OVM_StateTransitioner = smockit(
      await ethers.getContractFactory('OVM_StateTransitioner')
    )

    Mock__OVM_StateTransitionerFactory = smockit(
      await ethers.getContractFactory('OVM_StateTransitionerFactory')
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
        Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
          false
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_FraudVerifier.initializeFraudVerification(
            NULL_BYTES32,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
            DUMMY_OVM_TRANSACTIONS[0],
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0]
          )
        ).to.be.revertedWith('Invalid pre-state root inclusion proof.')
      })
    })

    describe('when provided a valid pre-state root inclusion proof', () => {
      before(() => {
        Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
          true
        )
      })

      describe('when provided an invalid transaction inclusion proof', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.smocked.verifyElement.will.return.with(
            false
          )
        })

        it('should revert', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              NULL_BYTES32,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_OVM_TRANSACTIONS[0],
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0]
            )
          ).to.be.revertedWith('Invalid transaction inclusion proof.')
        })
      })

      describe('when provided a valid transaction inclusion proof', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.smocked.verifyElement.will.return.with(
            true
          )
        })

        it('should deploy a new state transitioner', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              NULL_BYTES32,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              DUMMY_OVM_TRANSACTIONS[0],
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0]
            )
          ).to.not.be.reverted

          expect(
            await OVM_FraudVerifier.getStateTransitioner(NULL_BYTES32)
          ).to.equal(Mock__OVM_StateTransitioner.address)
        })
      })
    })
  })

  describe('finalizeFraudVerification', () => {
    beforeEach(async () => {
      Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
        true
      )
      Mock__OVM_CanonicalTransactionChain.smocked.verifyElement.will.return.with(
        true
      )

      await OVM_FraudVerifier.initializeFraudVerification(
        NULL_BYTES32,
        DUMMY_BATCH_HEADERS[0],
        DUMMY_BATCH_PROOFS[0],
        DUMMY_OVM_TRANSACTIONS[0],
        DUMMY_BATCH_HEADERS[0],
        DUMMY_BATCH_PROOFS[0]
      )
    })

    describe('when the transition process is not complete', () => {
      before(async () => {
        Mock__OVM_StateTransitioner.smocked.isComplete.will.return.with(false)
      })

      it('should revert', async () => {
        await expect(
          OVM_FraudVerifier.finalizeFraudVerification(
            NULL_BYTES32,
            DUMMY_BATCH_HEADERS[0],
            DUMMY_BATCH_PROOFS[0],
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
              NULL_BYTES32,
              DUMMY_BATCH_HEADERS[0],
              DUMMY_BATCH_PROOFS[0],
              NON_NULL_BYTES32,
              DUMMY_BATCH_HEADERS[0],
              batchProof
            )
          ).to.be.revertedWith('Invalid post-state root index.')
        })
      })

      describe('when provided a valid post-state root index', () => {
        const batchProof = {
          ...DUMMY_BATCH_PROOFS[0],
          index: DUMMY_BATCH_PROOFS[0].index + 1,
        }

        describe('when provided an invalid pre-state root inclusion proof', () => {
          beforeEach(() => {
            Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
              false
            )
          })

          it('should revert', async () => {
            await expect(
              OVM_FraudVerifier.finalizeFraudVerification(
                NULL_BYTES32,
                DUMMY_BATCH_HEADERS[0],
                DUMMY_BATCH_PROOFS[0],
                NON_NULL_BYTES32,
                DUMMY_BATCH_HEADERS[0],
                batchProof
              )
            ).to.be.revertedWith('Invalid pre-state root inclusion proof.')
          })
        })

        describe('when provided a valid pre-state root inclusion proof', () => {
          before(() => {
            Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
              true
            )
          })

          describe('when provided an invalid post-state root inclusion proof', () => {
            beforeEach(() => {
              Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
                (stateRoot: string, ...args: any) => {
                  return stateRoot !== NON_NULL_BYTES32
                }
              )
            })

            it('should revert', async () => {
              await expect(
                OVM_FraudVerifier.finalizeFraudVerification(
                  NULL_BYTES32,
                  DUMMY_BATCH_HEADERS[0],
                  DUMMY_BATCH_PROOFS[0],
                  NON_NULL_BYTES32,
                  DUMMY_BATCH_HEADERS[0],
                  batchProof
                )
              ).to.be.revertedWith('Invalid post-state root inclusion proof.')
            })
          })

          describe('when provided a valid post-state root inclusion proof', () => {
            before(() => {
              Mock__OVM_StateCommitmentChain.smocked.verifyElement.will.return.with(
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
                    NULL_BYTES32,
                    DUMMY_BATCH_HEADERS[0],
                    DUMMY_BATCH_PROOFS[0],
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
                  NULL_BYTES32
                )
              })

              it('should succeed and attempt to delete a state batch', async () => {
                await OVM_FraudVerifier.finalizeFraudVerification(
                  NULL_BYTES32,
                  DUMMY_BATCH_HEADERS[0],
                  DUMMY_BATCH_PROOFS[0],
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
    })
  })
})
