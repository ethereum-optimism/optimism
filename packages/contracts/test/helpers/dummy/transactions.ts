import { ZERO_ADDRESS, NULL_BYTES32 } from '../constants'

export const DUMMY_OVM_TRANSACTIONS = [
  {
    timestamp: 0,
    number: 0,
    l1QueueOrigin: 0,
    l1TxOrigin: ZERO_ADDRESS,
    entrypoint: ZERO_ADDRESS,
    gasLimit: 0,
    data: NULL_BYTES32,
  },
]
