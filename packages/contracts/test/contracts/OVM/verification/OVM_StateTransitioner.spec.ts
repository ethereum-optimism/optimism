/* tslint:disable:no-empty */
import { expect } from '../../../setup'

describe('OVM_StateTransitioner', () => {
  describe('proveContractState', () => {
    describe('when provided an invalid code hash', () => {
      it('should revert', async () => {})
    })

    describe('when provided a valid code hash', () => {
      describe('when provided an invalid account inclusion proof', () => {
        it('should revert', async () => {})
      })

      describe('when provided a valid account inclusion proof', () => {})
    })
  })

  describe('proveStorageSlot', () => {
    describe('when the corresponding account is not proven', () => {
      it('should revert', async () => {})
    })

    describe('when the corresponding account is proven', () => {
      describe('when provided an invalid slot inclusion proof', () => {
        it('should revert', async () => {})
      })

      describe('when provided a valid slot inclusion proof', () => {})
    })
  })

  describe('applyTransaction', () => {
    // TODO
  })

  describe('commitContractState', () => {
    describe('when the account was not changed', () => {
      it('should revert', async () => {})
    })

    describe('when the account was changed', () => {
      describe('when the account has not been committed', () => {
        it('should commit the account and update the state', async () => {})
      })

      describe('when the account was already committed', () => {
        it('should revert', () => {})
      })
    })
  })

  describe('commitStorageSlot', () => {
    describe('when the slot was not changed', () => {
      it('should revert', async () => {})
    })

    describe('when the slot was changed', () => {
      describe('when the slot has not been committed', () => {
        it('should commit the slot and update the state', async () => {})
      })

      describe('when the slot was already committed', () => {
        it('should revert', () => {})
      })
    })
  })

  describe('completeTransition', () => {
    describe('when there are uncommitted accounts', () => {
      it('should revert', async () => {})
    })

    describe('when there are uncommitted storage slots', () => {
      it('should revert', async () => {})
    })

    describe('when all state changes are committed', () => {})
  })
})
