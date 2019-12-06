import '../../setup'

/* Internal Imports */
import {
  SignedTransaction,
  RollupTransaction,
  Transfer,
  abiEncodeSignedTransaction,
  abiEncodeTransaction,
  parseSignedTransactionFromABI,
  parseTransactionFromABI,
} from '../../../src/'

const sender: string = '423Ace7C343094Ed5EB34B0a1838c19adB2BAC92'
const recipient: string = 'ba3739e8B603cFBCe513C9A4f8b6fFD44312d75E'

describe('RollupEncoding', () => {
  describe('Transactions', () => {
    it('should encoded & decode Transfer without throwing', async () => {
      const tx: Transfer = {
        sender,
        recipient,
        tokenType: 1,
        amount: 15,
      }

      const abiEncoded: string = abiEncodeTransaction(tx)
      const transfer: RollupTransaction = parseTransactionFromABI(abiEncoded)

      transfer.should.deep.equal(tx)
    })

    it('should encoded & decode SignedTransactions without throwing', async () => {
      const transfer: Transfer = {
        sender,
        recipient,
        tokenType: 1,
        amount: 15,
      }
      const signedTransfer: SignedTransaction = {
        signature: '0x1234',
        transaction: transfer,
      }

      const abiEncodedTransfer: string = abiEncodeSignedTransaction(
        signedTransfer
      )
      const parsedTransfer: SignedTransaction = parseSignedTransactionFromABI(
        abiEncodedTransfer
      )
      parsedTransfer.should.deep.equal(signedTransfer)
    })
  })
})
