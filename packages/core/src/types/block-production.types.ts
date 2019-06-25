/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import { AbiStateUpdate } from '../app'

export interface MerkleIntervalTreeNode {
  hash: Buffer // Hash of the sibling or leaf data
  start: Buffer // The start interval value for this node
  data: Buffer // concatenation of (hash, index)
}

export interface SubtreeContents {
  assetId: Buffer
  stateUpdates: AbiStateUpdate[]
}
