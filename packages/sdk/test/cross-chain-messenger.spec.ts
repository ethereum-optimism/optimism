/* eslint-disable @typescript-eslint/no-empty-function */
import './setup'

describe('CrossChainMessenger', () => {
  describe('sendMessage', () => {
    describe('when no l2GasLimit is provided', () => {
      it('should send a message with an estimated l2GasLimit', () => {})
    })

    describe('when an l2GasLimit is provided', () => {
      it('should send a message with the provided l2GasLimit', () => {})
    })
  })

  describe('resendMessage', () => {
    describe('when the message being resent exists', () => {
      it('should resend the message with the new gas limit', () => {})
    })

    describe('when the message being resent does not exist', () => {
      it('should throw an error', () => {})
    })
  })

  describe('finalizeMessage', () => {
    describe('when the message being finalized exists', () => {
      describe('when the message is ready to be finalized', () => {
        it('should finalize the message', () => {})
      })

      describe('when the message is not ready to be finalized', () => {
        it('should throw an error', () => {})
      })

      describe('when the message has already been finalized', () => {
        it('should throw an error', () => {})
      })
    })

    describe('when the message being finalized does not exist', () => {
      it('should throw an error', () => {})
    })
  })
})
