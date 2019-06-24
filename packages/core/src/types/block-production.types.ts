/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import { AbiStateUpdate } from '../app'

export interface SubtreeContents {
  address: Buffer
  stateUpdates: AbiStateUpdate[]
}
