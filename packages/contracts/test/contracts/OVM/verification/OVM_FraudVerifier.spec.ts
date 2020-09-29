import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Contract, Signer, BigNumber } from 'ethers'

/* Internal Imports */

/*
import {
  getProxyManager,
  getMockContract,
  MockContract,
  ZERO_ADDRESS,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  setProxyTarget,
} from '../../../helpers'

const DUMMY_BATCH_HEADER = {
  batchIndex: 0,
  batchRoot: NULL_BYTES32,
  batchSize: 0,
  prevTotalElements: 0,
  extraData: NULL_BYTES32,
}

const DUMMY_BATCH_PROOF = {
  index: 0,
  siblings: [NULL_BYTES32],
}

const DUMMY_OVM_TRANSACTION = {
  timestamp: 0,
  queueOrigin: 0,
  entrypoint: ZERO_ADDRESS,
  origin: ZERO_ADDRESS,
  msgSender: ZERO_ADDRESS,
  gasLimit: 0,
  data: NULL_BYTES32,
}

describe('OVM_FraudVerifier', () => {
  let Proxy_Manager: Contract
  before(async () => {
    Proxy_Manager = await getProxyManager()
  })

  let Mock__OVM_StateCommitmentChain: MockContract
  let Mock__OVM_CanonicalTransactionChain: MockContract
  let Mock__OVM_StateTransitioner: MockContract
  let Mock__OVM_StateTransitionerFactory: MockContract
  before(async () => {
    Mock__OVM_StateCommitmentChain = await getMockContract(
      await ethers.getContractFactory('OVM_StateCommitmentChain')
    )

    Mock__OVM_CanonicalTransactionChain = await getMockContract(
      await ethers.getContractFactory('OVM_CanonicalTransactionChain')
    )

    Mock__OVM_StateTransitioner = await getMockContract(
      await ethers.getContractFactory('OVM_StateTransitioner')
    )

    Mock__OVM_StateTransitionerFactory = await getMockContract(
      await ethers.getContractFactory('OVM_StateTransitionerFactory')
    )

    await setProxyTarget(
      Proxy_Manager,
      'OVM_StateCommitmentChain',
      Mock__OVM_StateCommitmentChain
    )

    await setProxyTarget(
      Proxy_Manager,
      'OVM_CanonicalTransactionChain',
      Mock__OVM_CanonicalTransactionChain
    )

    await setProxyTarget(
      Proxy_Manager,
      'OVM_StateTransitioner',
      Mock__OVM_StateTransitioner
    )

    await setProxyTarget(
      Proxy_Manager,
      'OVM_StateTransitionerFactory',
      Mock__OVM_StateTransitionerFactory
    )

    Mock__OVM_StateTransitionerFactory.setReturnValues('create', [
      Mock__OVM_StateTransitioner.address,
    ])
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
      Proxy_Manager.address
    )
  })

  describe('initializeFraudVerification', () => {
    describe('when provided an invalid pre-state root inclusion proof', () => {
      before(() => {
        Mock__OVM_StateCommitmentChain.setReturnValues('verifyElement', [false])
      })

      it('should revert', async () => {
        await expect(
          OVM_FraudVerifier.initializeFraudVerification(
            NULL_BYTES32,
            DUMMY_BATCH_HEADER,
            DUMMY_BATCH_PROOF,
            DUMMY_OVM_TRANSACTION,
            DUMMY_BATCH_HEADER,
            DUMMY_BATCH_PROOF
          )
        ).to.be.revertedWith('Invalid pre-state root inclusion proof.')
      })
    })

    describe('when provided a valid pre-state root inclusion proof', () => {
      before(() => {
        Mock__OVM_StateCommitmentChain.setReturnValues('verifyElement', [true])
      })

      describe('when provided an invalid transaction inclusion proof', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.setReturnValues('verifyElement', [
            false,
          ])
        })

        it('should revert', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              NULL_BYTES32,
              DUMMY_BATCH_HEADER,
              DUMMY_BATCH_PROOF,
              DUMMY_OVM_TRANSACTION,
              DUMMY_BATCH_HEADER,
              DUMMY_BATCH_PROOF
            )
          ).to.be.revertedWith('Invalid transaction inclusion proof.')
        })
      })

      describe('when provided a valid transaction inclusion proof', () => {
        before(() => {
          Mock__OVM_CanonicalTransactionChain.setReturnValues('verifyElement', [
            true,
          ])
        })

        it('should deploy a new state transitioner', async () => {
          await expect(
            OVM_FraudVerifier.initializeFraudVerification(
              NULL_BYTES32,
              DUMMY_BATCH_HEADER,
              DUMMY_BATCH_PROOF,
              DUMMY_OVM_TRANSACTION,
              DUMMY_BATCH_HEADER,
              DUMMY_BATCH_PROOF
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
      Mock__OVM_StateCommitmentChain.setReturnValues('verifyElement', [true])
      Mock__OVM_CanonicalTransactionChain.setReturnValues('verifyElement', [
        true,
      ])
      await OVM_FraudVerifier.initializeFraudVerification(
        NULL_BYTES32,
        DUMMY_BATCH_HEADER,
        DUMMY_BATCH_PROOF,
        DUMMY_OVM_TRANSACTION,
        DUMMY_BATCH_HEADER,
        DUMMY_BATCH_PROOF
      )
    })

    describe('when the transition process is not complete', () => {
      before(() => {
        Mock__OVM_StateTransitioner.setReturnValues('isComplete', [false])
      })

      it('should revert', async () => {
        await expect(
          OVM_FraudVerifier.finalizeFraudVerification(
            NULL_BYTES32,
            DUMMY_BATCH_HEADER,
            DUMMY_BATCH_PROOF,
            NON_NULL_BYTES32,
            DUMMY_BATCH_HEADER,
            DUMMY_BATCH_PROOF
          )
        ).to.be.revertedWith(
          'State transition process must be completed prior to finalization.'
        )
      })
    })

    describe('when the transition process is complete', () => {
      before(() => {
        Mock__OVM_StateTransitioner.setReturnValues('isComplete', [true])
      })

      describe('when provided an invalid post-state root index', () => {
        const batchProof = {
          ...DUMMY_BATCH_PROOF,
          index: DUMMY_BATCH_PROOF.index + 2,
        }

        it('should revert', async () => {
          await expect(
            OVM_FraudVerifier.finalizeFraudVerification(
              NULL_BYTES32,
              DUMMY_BATCH_HEADER,
              DUMMY_BATCH_PROOF,
              NON_NULL_BYTES32,
              DUMMY_BATCH_HEADER,
              batchProof
            )
          ).to.be.revertedWith('Invalid post-state root index.')
        })
      })

      describe('when provided a valid post-state root index', () => {
        const batchProof = {
          ...DUMMY_BATCH_PROOF,
          index: DUMMY_BATCH_PROOF.index + 1,
        }

        describe('when provided an invalid pre-state root inclusion proof', () => {
          beforeEach(() => {
            Mock__OVM_StateCommitmentChain.setReturnValues('verifyElement', [
              false,
            ])
          })

          it('should revert', async () => {
            await expect(
              OVM_FraudVerifier.finalizeFraudVerification(
                NULL_BYTES32,
                DUMMY_BATCH_HEADER,
                DUMMY_BATCH_PROOF,
                NON_NULL_BYTES32,
                DUMMY_BATCH_HEADER,
                batchProof
              )
            ).to.be.revertedWith('Invalid pre-state root inclusion proof.')
          })
        })

        describe('when provided a valid pre-state root inclusion proof', () => {
          before(() => {
            Mock__OVM_StateCommitmentChain.setReturnValues('verifyElement', [
              true,
            ])
          })

          describe('when provided an invalid post-state root inclusion proof', () => {
            beforeEach(() => {
              Mock__OVM_StateCommitmentChain.setReturnValues(
                'verifyElement',
                (stateRoot: string, ...args: any) => {
                  return [stateRoot !== NON_NULL_BYTES32]
                }
              )
            })

            it('should revert', async () => {
              await expect(
                OVM_FraudVerifier.finalizeFraudVerification(
                  NULL_BYTES32,
                  DUMMY_BATCH_HEADER,
                  DUMMY_BATCH_PROOF,
                  NON_NULL_BYTES32,
                  DUMMY_BATCH_HEADER,
                  batchProof
                )
              ).to.be.revertedWith('Invalid post-state root inclusion proof.')
            })
          })

          describe('when provided a valid post-state root inclusion proof', () => {
            before(() => {
              Mock__OVM_StateCommitmentChain.setReturnValues('verifyElement', [
                true,
              ])
            })

            describe('when the provided post-state root does not differ from the computed one', () => {
              before(() => {
                Mock__OVM_StateTransitioner.setReturnValues(
                  'getPostStateRoot',
                  [NON_NULL_BYTES32]
                )
              })

              it('should revert', async () => {
                await expect(
                  OVM_FraudVerifier.finalizeFraudVerification(
                    NULL_BYTES32,
                    DUMMY_BATCH_HEADER,
                    DUMMY_BATCH_PROOF,
                    NON_NULL_BYTES32,
                    DUMMY_BATCH_HEADER,
                    batchProof
                  )
                ).to.be.revertedWith(
                  'State transition has not been proven fraudulent.'
                )
              })
            })

            describe('when the provided post-state root differs from the computed one', () => {
              before(() => {
                Mock__OVM_StateTransitioner.setReturnValues(
                  'getPostStateRoot',
                  [NULL_BYTES32]
                )
              })

              it('should succeed and attempt to delete a state batch', async () => {
                await OVM_FraudVerifier.finalizeFraudVerification(
                  NULL_BYTES32,
                  DUMMY_BATCH_HEADER,
                  DUMMY_BATCH_PROOF,
                  NON_NULL_BYTES32,
                  DUMMY_BATCH_HEADER,
                  batchProof
                )

                expect(
                  Mock__OVM_StateCommitmentChain.getCallData(
                    'deleteStateBatch',
                    0
                  )
                ).to.deep.equal([
                  Object.values(DUMMY_BATCH_HEADER).map((value) => {
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
*/
