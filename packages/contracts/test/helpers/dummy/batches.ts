import { NULL_BYTES32 } from '../constants'

export const DUMMY_BATCH_HEADERS = [
  {
    batchIndex: 0,
    batchRoot: NULL_BYTES32,
    batchSize: 0,
    prevTotalElements: 0,
    extraData: NULL_BYTES32,
  },
]

export const DUMMY_BATCH_PROOFS = [
  {
    index: 0,
    siblings: [NULL_BYTES32],
  },
]
