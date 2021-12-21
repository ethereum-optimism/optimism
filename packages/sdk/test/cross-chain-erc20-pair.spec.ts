/* eslint-disable @typescript-eslint/no-empty-function */
import './setup'

describe('CrossChainERC20Pair', () => {
  describe('construction', () => {
    it('should have a messenger', () => {})

    describe('when the token is a standard bridge token', () => {
      it('should resolve the correct bridge', () => {})
    })

    describe('when the token is SNX', () => {
      it('should resolve the correct bridge', () => {})
    })

    describe('when the token is DAI', () => {
      it('should resolve the correct bridge', () => {})
    })

    describe('when a custom adapter is provided', () => {
      it('should use the custom adapter', () => {})
    })
  })

  describe('deposit', () => {
    describe('when the user has enough balance and allowance', () => {
      describe('when the token is a standard bridge token', () => {
        it('should trigger a token deposit', () => {})
      })

      describe('when the token is ETH', () => {
        it('should trigger a token deposit', () => {})
      })

      describe('when the token is SNX', () => {
        it('should trigger a token deposit', () => {})
      })

      describe('when the token is DAI', () => {
        it('should trigger a token deposit', () => {})
      })
    })

    describe('when the user does not have enough balance', () => {
      it('should throw an error', () => {})
    })

    describe('when the user has not given enough allowance to the bridge', () => {
      it('should throw an error', () => {})
    })
  })

  describe('withdraw', () => {
    describe('when the user has enough balance', () => {
      describe('when the token is a standard bridge token', () => {
        it('should trigger a token withdrawal', () => {})
      })

      describe('when the token is ETH', () => {
        it('should trigger a token withdrawal', () => {})
      })

      describe('when the token is SNX', () => {
        it('should trigger a token withdrawal', () => {})
      })

      describe('when the token is DAI', () => {
        it('should trigger a token withdrawal', () => {})
      })
    })

    describe('when the user does not have enough balance', () => {
      it('should throw an error', () => {})
    })
  })

  describe('populateTransaction', () => {
    describe('deposit', () => {
      it('should populate the transaction with the correct values', () => {})
    })

    describe('withdraw', () => {
      it('should populate the transaction with the correct values', () => {})
    })
  })

  describe('estimateGas', () => {
    describe('deposit', () => {
      it('should estimate gas required for the transaction', () => {})
    })

    describe('withdraw', () => {
      it('should estimate gas required for the transaction', () => {})
    })
  })
})
