import attestationArtifacts from '@eth-optimism/contracts-periphery/artifacts/contracts/universal/op-nft/AttestationStation.sol/AttestationStation.json'

// TODO hardcoding this type now because importing this directly loses type information
// This will fix itseslf when I add wagmi cli to this package to generate the abi
// I'm still using the attestationArtifact as a source of truth so if we somehow
// forget to update this we are still always using the latest abi
export const abi = attestationArtifacts.abi as unknown as typeof hardcodedAbi

const hardcodedAbi = [
  {
    inputs: [],
    stateMutability: 'nonpayable',
    type: 'constructor',
  },
  {
    anonymous: false,
    inputs: [
      {
        indexed: true,
        internalType: 'address',
        name: 'creator',
        type: 'address',
      },
      {
        indexed: true,
        internalType: 'address',
        name: 'about',
        type: 'address',
      },
      {
        indexed: true,
        internalType: 'bytes32',
        name: 'key',
        type: 'bytes32',
      },
      {
        indexed: false,
        internalType: 'bytes',
        name: 'val',
        type: 'bytes',
      },
    ],
    name: 'AttestationCreated',
    type: 'event',
  },
  {
    inputs: [
      {
        components: [
          {
            internalType: 'address',
            name: 'about',
            type: 'address',
          },
          {
            internalType: 'bytes32',
            name: 'key',
            type: 'bytes32',
          },
          {
            internalType: 'bytes',
            name: 'val',
            type: 'bytes',
          },
        ],
        internalType: 'struct AttestationStation.AttestationData[]',
        name: '_attestations',
        type: 'tuple[]',
      },
    ],
    name: 'attest',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: '_about',
        type: 'address',
      },
      {
        internalType: 'bytes32',
        name: '_key',
        type: 'bytes32',
      },
      {
        internalType: 'bytes',
        name: '_val',
        type: 'bytes',
      },
    ],
    name: 'attest',
    outputs: [],
    stateMutability: 'nonpayable',
    type: 'function',
  },
  {
    inputs: [
      {
        internalType: 'address',
        name: '',
        type: 'address',
      },
      {
        internalType: 'address',
        name: '',
        type: 'address',
      },
      {
        internalType: 'bytes32',
        name: '',
        type: 'bytes32',
      },
    ],
    name: 'attestations',
    outputs: [
      {
        internalType: 'bytes',
        name: '',
        type: 'bytes',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
  {
    inputs: [],
    name: 'version',
    outputs: [
      {
        internalType: 'string',
        name: '',
        type: 'string',
      },
    ],
    stateMutability: 'view',
    type: 'function',
  },
] as const
