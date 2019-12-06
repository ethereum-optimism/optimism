import '../setup'

/* External Imports */
import { DB, newInMemoryDB } from '@pigi/core-db'

/* Internal Imports */
import { DefaultRollupStateMachine } from '../../src/app'
import { SignedTransaction, Transfer } from '../../src/types'

const sender: string = '423Ace7C343094Ed5EB34B0a1838c19adB2BAC92'
const recipient: string = 'ba3739e8B603cFBCe513C9A4f8b6fFD44312d75E'

const transfer: Transfer = {
  sender,
  recipient,
  tokenType: 1,
  amount: 10,
}

const mockSignedTransfer: SignedTransaction = {
  signature: 'derp derp derp',
  transaction: transfer,
}

describe('RollupStateMachine', () => {
  let rollupStateMachine: DefaultRollupStateMachine
  let db: DB

  beforeEach(async () => {
    db = newInMemoryDB()
    rollupStateMachine = await DefaultRollupStateMachine.create(db)
  })

  // TODO: Add tests when logic exists
})
