/**
 * ABI for the Callback contract (events indexed by ponder).
 * @category ABI
 */
export const callbackAbi = [
  {
    type: 'event',
    name: 'CallbackRegistered',
    inputs: [
      {
        name: 'callbackPromiseId',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
      {
        name: 'parentPromiseId',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
      {
        name: 'callbackType',
        type: 'uint8',
        indexed: false,
        internalType: 'uint8',
      },
    ],
    anonymous: false,
  },
  {
    type: 'function',
    name: 'getCallback',
    stateMutability: 'view',
    inputs: [{ name: 'callbackPromiseId', type: 'bytes32' }],
    outputs: [
      {
        name: 'callbackData',
        type: 'tuple',
        components: [
          { name: 'parentPromiseId', type: 'bytes32' },
          { name: 'target', type: 'address' },
          { name: 'selector', type: 'bytes4' },
          { name: 'callbackType', type: 'uint8' },
          { name: 'registrant', type: 'address' },
          { name: 'sourceChain', type: 'uint256' },
        ],
      },
    ],
  },
] as const
