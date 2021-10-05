import { KECCAK256_RLP_S, KECCAK256_NULL_S } from 'ethereumjs-util'

/* Imports: Internal */
import { expect } from '../setup'
import { handlers } from '../../scripts/handlers'
import { Account, AccountType } from '../../scripts/types'

describe('Handlers', () => {
  const dummyAccount: Account = {
    address: '0x0000000000000000000000000000000000000420',
    nonce: 69,
    balance: '0',
    codeHash: '420e69',
    root: '420e69',
    code: '608060405',
    storage: {
      '0x0000000000000000000000000000000000000420':
        '0000000000000000000000000000000000000420',
    },
  }

  describe('EOA', () => {
    it('returns the account without code', async () => {
      const output = await handlers[AccountType.EOA](dummyAccount, null)
      expect(output.address).to.eq(dummyAccount.address)
      expect(output.nonce).to.eq(dummyAccount.nonce)
      expect(output.balance).to.eq(dummyAccount.balance)
      expect(output.codeHash).to.eq(KECCAK256_NULL_S)
      expect(output.root).to.eq(KECCAK256_RLP_S)
      expect(output.code).to.be.undefined
      expect(output.storage).to.be.undefined
    })
  })
})
