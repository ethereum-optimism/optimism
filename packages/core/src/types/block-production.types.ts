/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import { AbiStateUpdate } from '../app'

export interface SubtreeContents {
  assetId: Buffer
  stateUpdates: AbiStateUpdate[]
}
