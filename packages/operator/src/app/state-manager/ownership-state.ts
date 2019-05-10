/* External Imports */
import level from 'level'

/* Internal Imports */

/* Logging */
import debug from 'debug'
const log = debug('test:info:state-ownership')

export class OwnershipState {
  constructor (readonly db: level) {
  }

  public applyTransaction(transaction: Buffer) {
    log('Applying transaction:', transaction)
  }
}
