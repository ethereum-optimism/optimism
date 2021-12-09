/* eslint-disable @typescript-eslint/no-empty-function */
import './setup'

describe('CrossChainERC20Pair', () => {
  describe('construction', () => {
    it('should have a messenger', () => {})

    describe('when only an L1 token is provided', () => {
      describe('when the token is a standard bridge token', () => {
        it('should resolve an L2 token from the token list', () => {})
      })

      describe('when the token is ETH', () => {
        it('should resolve the L2 ETH token address', () => {})
      })

      describe('when the token is SNX', () => {
        it('should resolve the L2 SNX token address', () => {})
      })

      describe('when the token is DAI', () => {
        it('should resolve the L2 DAI token address', () => {})
      })

      describe('when the token is not a standard token or a special token', () => {
        it('should throw an error', () => {})
      })
    })

    describe('when only an L2 token is provided', () => {
      describe('when the token is a standard bridge token', () => {
        it('should resolve an L1 token from the token list', () => {})
      })

      describe('when the token is ETH', () => {
        it('should resolve the L1 ETH token address', () => {})
      })

      describe('when the token is SNX', () => {
        it('should resolve the L1 SNX token address', () => {})
      })

      describe('when the token is DAI', () => {
        it('should resolve the L1 DAI token address', () => {})
      })

      describe('when the token is not a standard token or a special token', () => {
        it('should throw an error', () => {})
      })
    })

    describe('when both an L1 token and an L2 token are provided', () => {
      it('should attach both instances', () => {})
    })

    describe('when neither an L1 token or an L2 token are provided', () => {
      it('should throw an error', () => {})
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
    describe('estimateGas', () => {
      it('should estimate gas required for the transaction', () => {})
    })

    describe('estimateGas', () => {
      it('should estimate gas required for the transaction', () => {})
    })
  })
})
