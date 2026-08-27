/**
 * ABI for the OP Stack contract `L2ToL2CrossDomainMessenger` deployed on
 * devnet.
 * @category ABI
 */
export const l2ToL2CrossDomainMessengerAbi = [
  {
    type: 'function',
    name: 'crossDomainMessageContext',
    inputs: [],
    outputs: [
      {
        name: 'sender_',
        type: 'address',
        internalType: 'address',
      },
      {
        name: 'source_',
        type: 'uint256',
        internalType: 'uint256',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'function',
    name: 'crossDomainMessageSender',
    inputs: [],
    outputs: [
      {
        name: 'sender_',
        type: 'address',
        internalType: 'address',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'function',
    name: 'crossDomainMessageSource',
    inputs: [],
    outputs: [
      {
        name: 'source_',
        type: 'uint256',
        internalType: 'uint256',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'function',
    name: 'messageNonce',
    inputs: [],
    outputs: [
      {
        name: '',
        type: 'uint256',
        internalType: 'uint256',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'function',
    name: 'messageVersion',
    inputs: [],
    outputs: [
      {
        name: '',
        type: 'uint16',
        internalType: 'uint16',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'function',
    name: 'relayMessage',
    inputs: [
      {
        name: '_id',
        type: 'tuple',
        internalType: 'struct Identifier',
        components: [
          {
            name: 'origin',
            type: 'address',
            internalType: 'address',
          },
          {
            name: 'blockNumber',
            type: 'uint256',
            internalType: 'uint256',
          },
          {
            name: 'logIndex',
            type: 'uint256',
            internalType: 'uint256',
          },
          {
            name: 'timestamp',
            type: 'uint256',
            internalType: 'uint256',
          },
          {
            name: 'chainId',
            type: 'uint256',
            internalType: 'uint256',
          },
        ],
      },
      {
        name: '_sentMessage',
        type: 'bytes',
        internalType: 'bytes',
      },
    ],
    outputs: [
      {
        name: 'returnData_',
        type: 'bytes',
        internalType: 'bytes',
      },
    ],
    stateMutability: 'payable',
  },
  {
    type: 'function',
    name: 'sendMessage',
    inputs: [
      {
        name: '_destination',
        type: 'uint256',
        internalType: 'uint256',
      },
      {
        name: '_target',
        type: 'address',
        internalType: 'address',
      },
      {
        name: '_message',
        type: 'bytes',
        internalType: 'bytes',
      },
    ],
    outputs: [
      {
        name: '',
        type: 'bytes32',
        internalType: 'bytes32',
      },
    ],
    stateMutability: 'nonpayable',
  },
  {
    type: 'function',
    name: 'successfulMessages',
    inputs: [
      {
        name: '',
        type: 'bytes32',
        internalType: 'bytes32',
      },
    ],
    outputs: [
      {
        name: '',
        type: 'bool',
        internalType: 'bool',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'function',
    name: 'version',
    inputs: [],
    outputs: [
      {
        name: '',
        type: 'string',
        internalType: 'string',
      },
    ],
    stateMutability: 'view',
  },
  {
    type: 'event',
    name: 'RelayedMessage',
    inputs: [
      {
        name: 'source',
        type: 'uint256',
        indexed: true,
        internalType: 'uint256',
      },
      {
        name: 'messageNonce',
        type: 'uint256',
        indexed: true,
        internalType: 'uint256',
      },
      {
        name: 'messageHash',
        type: 'bytes32',
        indexed: true,
        internalType: 'bytes32',
      },
    ],
    anonymous: false,
  },
  {
    type: 'event',
    name: 'SentMessage',
    inputs: [
      {
        name: 'destination',
        type: 'uint256',
        indexed: true,
        internalType: 'uint256',
      },
      {
        name: 'target',
        type: 'address',
        indexed: true,
        internalType: 'address',
      },
      {
        name: 'messageNonce',
        type: 'uint256',
        indexed: true,
        internalType: 'uint256',
      },
      {
        name: 'sender',
        type: 'address',
        indexed: false,
        internalType: 'address',
      },
      {
        name: 'message',
        type: 'bytes',
        indexed: false,
        internalType: 'bytes',
      },
    ],
    anonymous: false,
  },
  {
    type: 'error',
    name: 'EventPayloadNotSentMessage',
    inputs: [],
  },
  {
    type: 'error',
    name: 'IdOriginNotL2ToL2CrossDomainMessenger',
    inputs: [],
  },
  {
    type: 'error',
    name: 'MessageAlreadyRelayed',
    inputs: [],
  },
  {
    type: 'error',
    name: 'MessageDestinationNotRelayChain',
    inputs: [],
  },
  {
    type: 'error',
    name: 'MessageDestinationSameChain',
    inputs: [],
  },
  {
    type: 'error',
    name: 'MessageTargetL2ToL2CrossDomainMessenger',
    inputs: [],
  },
  {
    type: 'error',
    name: 'NotEntered',
    inputs: [],
  },
  {
    type: 'error',
    name: 'ReentrantCall',
    inputs: [],
  },
] as const
