import { expect } from '../../../setup'

describe('OVM_ECDSAContractAccount', () => {
  describe('execute', () => {
    describe('when provided an invalid signature', () => {
      it('should revert', async () => {

      })
    })

    describe('when provided a valid signature', () => {
      describe('when provided an invalid nonce', () => {
        it('should revert', async () => {

        })
      })

      describe('when provided a valid nonce', () => {
        describe('when executing ovmCREATE', () => {
          it('should return the address of the created contract', async () => {

          })
        })

        describe('when executing ovmCALL', () => {
          it('should return the result of the call', async () => {

          })
        })
      })
    })
  })
})
