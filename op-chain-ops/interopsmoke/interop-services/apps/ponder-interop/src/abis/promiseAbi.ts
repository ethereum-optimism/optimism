/**
 * ABI for the Promise contract (events indexed by ponder).
 * @category ABI
 */
export const promiseAbi = [
  {
    type: 'event',
    name: 'PromiseCreated',
    inputs: [
      {
        name: 'promiseId',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
      {
        name: 'resolver',
        type: 'address',
        indexed: true,
        internalType: 'address',
      },
    ],
    anonymous: false,
  },
  {
    type: 'event',
    name: 'PromiseResolved',
    inputs: [
      {
        name: 'promiseId',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
      {
        name: 'returnData',
        type: 'bytes',
        indexed: false,
        internalType: 'bytes',
      },
    ],
    anonymous: false,
  },
  {
    type: 'event',
    name: 'PromiseRejected',
    inputs: [
      {
        name: 'promiseId',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
      {
        name: 'errorData',
        type: 'bytes',
        indexed: false,
        internalType: 'bytes',
      },
    ],
    anonymous: false,
  },
  {
    type: 'event',
    name: 'ResolutionTransferred',
    inputs: [
      {
        name: 'promiseId',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
      {
        name: 'destinationChainId',
        type: 'uint256',
        indexed: true,
        internalType: 'uint256',
      },
      {
        name: 'resolver',
        type: 'address',
        indexed: true,
        internalType: 'address',
      },
    ],
    anonymous: false,
  },
] as const
