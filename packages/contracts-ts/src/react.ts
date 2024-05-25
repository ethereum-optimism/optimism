import {
  useNetwork,
  useChainId,
  useContractRead,
  UseContractReadConfig,
  useContractWrite,
  Address,
  UseContractWriteConfig,
  useContractEvent,
  UseContractEventConfig,
} from 'wagmi'
import {
  ReadContractResult,
  WriteContractMode,
  PrepareWriteContractResult,
} from 'wagmi/actions'

/* eslint-disable */

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// AddressManager
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111)
 */
export const addressManagerABI = [
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: '_name', internalType: 'string', type: 'string', indexed: true },
      {
        name: '_newAddress',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
      {
        name: '_oldAddress',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'AddressSet',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'getAddress',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_address', internalType: 'address', type: 'address' },
    ],
    name: 'setAddress',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111)
 */
export const addressManagerAddress = {
  1: '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F',
  5: '0xa6f73589243a6A7a9023b1Fa0651b1d89c177111',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111)
 */
export const addressManagerConfig = {
  address: addressManagerAddress,
  abi: addressManagerABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// AssetReceiver
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 */
export const assetReceiverABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'user', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnerUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'ReceivedETH',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewERC20',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'id', internalType: 'uint256', type: 'uint256', indexed: false },
    ],
    name: 'WithdrewERC721',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewETH',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_gas', internalType: 'uint256', type: 'uint256' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'CALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_gas', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'DELEGATECALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'setOwner',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC721', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_id', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address payable', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_to', internalType: 'address payable', type: 'address' }],
    name: 'withdrawETH',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 */
export const assetReceiverAddress = {
  1: '0x15DdA60616Ffca20371ED1659dBB78E888f65556',
  10: '0x15DdA60616Ffca20371ED1659dBB78E888f65556',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 */
export const assetReceiverConfig = {
  address: assetReceiverAddress,
  abi: assetReceiverABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// AttestationStation
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 */
export const attestationStationABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'creator',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'about',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'key', internalType: 'bytes32', type: 'bytes32', indexed: true },
      { name: 'val', internalType: 'bytes', type: 'bytes', indexed: false },
    ],
    name: 'AttestationCreated',
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_attestations',
        internalType: 'struct AttestationStation.AttestationData[]',
        type: 'tuple[]',
        components: [
          { name: 'about', internalType: 'address', type: 'address' },
          { name: 'key', internalType: 'bytes32', type: 'bytes32' },
          { name: 'val', internalType: 'bytes', type: 'bytes' },
        ],
      },
    ],
    name: 'attest',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_about', internalType: 'address', type: 'address' },
      { name: '_key', internalType: 'bytes32', type: 'bytes32' },
      { name: '_val', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'attest',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'bytes32', type: 'bytes32' },
    ],
    name: 'attestations',
    outputs: [{ name: '', internalType: 'bytes', type: 'bytes' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 */
export const attestationStationAddress = {
  10: '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77',
  420: '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 */
export const attestationStationConfig = {
  address: attestationStationAddress,
  abi: attestationStationABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// BaseFeeVault
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000019)
 */
export const baseFeeVaultABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_recipient', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'value',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'from',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'Withdrawal',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_WITHDRAWAL_AMOUNT',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RECIPIENT',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'totalProcessed',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'withdraw',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000019)
 */
export const baseFeeVaultAddress = {
  420: '0x4200000000000000000000000000000000000019',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000019)
 */
export const baseFeeVaultConfig = {
  address: baseFeeVaultAddress,
  abi: baseFeeVaultABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// CheckBalanceHigh
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5d7103853f12109A7d27F118e54BbC654ad847E9)
 */
export const checkBalanceHighABI = [
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'params',
        internalType: 'struct CheckBalanceHigh.Params',
        type: 'tuple',
        components: [
          { name: 'target', internalType: 'address', type: 'address' },
          { name: 'threshold', internalType: 'uint256', type: 'uint256' },
        ],
        indexed: false,
      },
    ],
    name: '_EventToExposeStructInABI__Params',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_params', internalType: 'bytes', type: 'bytes' }],
    name: 'check',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5d7103853f12109A7d27F118e54BbC654ad847E9)
 */
export const checkBalanceHighAddress = {
  1: '0x7eC64a8a591bFf829ff6C8be76074D540ACb813F',
  5: '0x7eC64a8a591bFf829ff6C8be76074D540ACb813F',
  420: '0x5d7103853f12109A7d27F118e54BbC654ad847E9',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5d7103853f12109A7d27F118e54BbC654ad847E9)
 */
export const checkBalanceHighConfig = {
  address: checkBalanceHighAddress,
  abi: checkBalanceHighABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// CheckBalanceLow
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x7Ce13D154FAEE5C8B3E6b19d4Add16f21d884474)
 */
export const checkBalanceLowABI = [
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'params',
        internalType: 'struct CheckBalanceLow.Params',
        type: 'tuple',
        components: [
          { name: 'target', internalType: 'address', type: 'address' },
          { name: 'threshold', internalType: 'uint256', type: 'uint256' },
        ],
        indexed: false,
      },
    ],
    name: '_EventToExposeStructInABI__Params',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_params', internalType: 'bytes', type: 'bytes' }],
    name: 'check',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x7Ce13D154FAEE5C8B3E6b19d4Add16f21d884474)
 */
export const checkBalanceLowAddress = {
  1: '0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640',
  5: '0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640',
  420: '0x7Ce13D154FAEE5C8B3E6b19d4Add16f21d884474',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x7Ce13D154FAEE5C8B3E6b19d4Add16f21d884474)
 */
export const checkBalanceLowConfig = {
  address: checkBalanceLowAddress,
  abi: checkBalanceLowABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// CheckGelatoLow
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xF9c8a4Cb4021f57F9f6d69799cA9BefF64524862)
 */
export const checkGelatoLowABI = [
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'params',
        internalType: 'struct CheckGelatoLow.Params',
        type: 'tuple',
        components: [
          { name: 'treasury', internalType: 'address', type: 'address' },
          { name: 'threshold', internalType: 'uint256', type: 'uint256' },
          { name: 'recipient', internalType: 'address', type: 'address' },
        ],
        indexed: false,
      },
    ],
    name: '_EventToExposeStructInABI__Params',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_params', internalType: 'bytes', type: 'bytes' }],
    name: 'check',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xF9c8a4Cb4021f57F9f6d69799cA9BefF64524862)
 */
export const checkGelatoLowAddress = {
  1: '0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa',
  5: '0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa',
  420: '0xF9c8a4Cb4021f57F9f6d69799cA9BefF64524862',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xF9c8a4Cb4021f57F9f6d69799cA9BefF64524862)
 */
export const checkGelatoLowConfig = {
  address: checkGelatoLowAddress,
  abi: checkGelatoLowABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// CheckTrue
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x47443D0C184e022F19BD1578F5bca6B8a9F58E32)
 */
export const checkTrueABI = [
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes', type: 'bytes' }],
    name: 'check',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x47443D0C184e022F19BD1578F5bca6B8a9F58E32)
 */
export const checkTrueAddress = {
  1: '0x5c741a38cb11424711231777D71689C458eE835D',
  5: '0x5c741a38cb11424711231777D71689C458eE835D',
  420: '0x47443D0C184e022F19BD1578F5bca6B8a9F58E32',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x47443D0C184e022F19BD1578F5bca6B8a9F58E32)
 */
export const checkTrueConfig = {
  address: checkTrueAddress,
  abi: checkTrueABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Drippie
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export const drippieABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'reentrant', internalType: 'bool', type: 'bool' },
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
        indexed: false,
      },
    ],
    name: 'DripCreated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'executor',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
      {
        name: 'timestamp',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'DripExecuted',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
        indexed: false,
      },
    ],
    name: 'DripStatusUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'user', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnerUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'ReceivedETH',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewERC20',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'id', internalType: 'uint256', type: 'uint256', indexed: false },
    ],
    name: 'WithdrewERC721',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewETH',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'CALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'DELEGATECALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      {
        name: '_config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'reentrant', internalType: 'bool', type: 'bool' },
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'create',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'drip',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'string', type: 'string' }],
    name: 'drips',
    outputs: [
      {
        name: 'status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
      },
      {
        name: 'config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'reentrant', internalType: 'bool', type: 'bool' },
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
      { name: 'last', internalType: 'uint256', type: 'uint256' },
      { name: 'count', internalType: 'uint256', type: 'uint256' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'executable',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'setOwner',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      {
        name: '_status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
      },
    ],
    name: 'status',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC721', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_id', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address payable', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_to', internalType: 'address payable', type: 'address' }],
    name: 'withdrawETH',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export const drippieAddress = {
  1: '0x44b3A2a040057eBafC601A78647e805fd58B1f50',
} as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export const drippieConfig = {
  address: drippieAddress,
  abi: drippieABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Drippie_goerli
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export const drippieGoerliABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'reentrant', internalType: 'bool', type: 'bool' },
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
        indexed: false,
      },
    ],
    name: 'DripCreated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'executor',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
      {
        name: 'timestamp',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'DripExecuted',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
        indexed: false,
      },
    ],
    name: 'DripStatusUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'user', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnerUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'ReceivedETH',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewERC20',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'id', internalType: 'uint256', type: 'uint256', indexed: false },
    ],
    name: 'WithdrewERC721',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewETH',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'CALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'DELEGATECALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      {
        name: '_config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'reentrant', internalType: 'bool', type: 'bool' },
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'create',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'drip',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'string', type: 'string' }],
    name: 'drips',
    outputs: [
      {
        name: 'status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
      },
      {
        name: 'config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'reentrant', internalType: 'bool', type: 'bool' },
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
      { name: 'last', internalType: 'uint256', type: 'uint256' },
      { name: 'count', internalType: 'uint256', type: 'uint256' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'executable',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'setOwner',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      {
        name: '_status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
      },
    ],
    name: 'status',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC721', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_id', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address payable', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_to', internalType: 'address payable', type: 'address' }],
    name: 'withdrawETH',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export const drippieGoerliAddress = {
  5: '0x44b3A2a040057eBafC601A78647e805fd58B1f50',
} as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export const drippieGoerliConfig = {
  address: drippieGoerliAddress,
  abi: drippieGoerliABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Drippie_optimism-goerli
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x8D8d533C16D23847EB04EEB0925be8900Dd3af86)
 */
export const drippieOptimismGoerliABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
        indexed: false,
      },
    ],
    name: 'DripCreated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'executor',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
      {
        name: 'timestamp',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'DripExecuted',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nameref',
        internalType: 'string',
        type: 'string',
        indexed: true,
      },
      { name: 'name', internalType: 'string', type: 'string', indexed: false },
      {
        name: 'status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
        indexed: false,
      },
    ],
    name: 'DripStatusUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'user', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnerUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'ReceivedETH',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewERC20',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'id', internalType: 'uint256', type: 'uint256', indexed: false },
    ],
    name: 'WithdrewERC721',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewETH',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_gas', internalType: 'uint256', type: 'uint256' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'CALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_gas', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'DELEGATECALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      {
        name: '_config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'create',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'drip',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'string', type: 'string' }],
    name: 'drips',
    outputs: [
      {
        name: 'status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
      },
      {
        name: 'config',
        internalType: 'struct Drippie.DripConfig',
        type: 'tuple',
        components: [
          { name: 'interval', internalType: 'uint256', type: 'uint256' },
          {
            name: 'dripcheck',
            internalType: 'contract IDripCheck',
            type: 'address',
          },
          { name: 'checkparams', internalType: 'bytes', type: 'bytes' },
          {
            name: 'actions',
            internalType: 'struct Drippie.DripAction[]',
            type: 'tuple[]',
            components: [
              {
                name: 'target',
                internalType: 'address payable',
                type: 'address',
              },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
      { name: 'last', internalType: 'uint256', type: 'uint256' },
      { name: 'count', internalType: 'uint256', type: 'uint256' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'executable',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'setOwner',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      {
        name: '_status',
        internalType: 'enum Drippie.DripStatus',
        type: 'uint8',
      },
    ],
    name: 'status',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC721', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_id', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address payable', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_to', internalType: 'address payable', type: 'address' }],
    name: 'withdrawETH',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x8D8d533C16D23847EB04EEB0925be8900Dd3af86)
 */
export const drippieOptimismGoerliAddress = {
  420: '0x8D8d533C16D23847EB04EEB0925be8900Dd3af86',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x8D8d533C16D23847EB04EEB0925be8900Dd3af86)
 */
export const drippieOptimismGoerliConfig = {
  address: drippieOptimismGoerliAddress,
  abi: drippieOptimismGoerliABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// EAS
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5A633F1cc84B03F7588486CF2F386c102061E6e1)
 */
export const easABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  { type: 'error', inputs: [], name: 'AccessDenied' },
  { type: 'error', inputs: [], name: 'AlreadyRevoked' },
  { type: 'error', inputs: [], name: 'AlreadyRevokedOffchain' },
  { type: 'error', inputs: [], name: 'AlreadyTimestamped' },
  { type: 'error', inputs: [], name: 'InsufficientValue' },
  { type: 'error', inputs: [], name: 'InvalidAttestation' },
  { type: 'error', inputs: [], name: 'InvalidAttestations' },
  { type: 'error', inputs: [], name: 'InvalidExpirationTime' },
  { type: 'error', inputs: [], name: 'InvalidLength' },
  { type: 'error', inputs: [], name: 'InvalidOffset' },
  { type: 'error', inputs: [], name: 'InvalidRegistry' },
  { type: 'error', inputs: [], name: 'InvalidRevocation' },
  { type: 'error', inputs: [], name: 'InvalidRevocations' },
  { type: 'error', inputs: [], name: 'InvalidSchema' },
  { type: 'error', inputs: [], name: 'InvalidSignature' },
  { type: 'error', inputs: [], name: 'InvalidVerifier' },
  { type: 'error', inputs: [], name: 'Irrevocable' },
  { type: 'error', inputs: [], name: 'NotFound' },
  { type: 'error', inputs: [], name: 'NotPayable' },
  { type: 'error', inputs: [], name: 'WrongSchema' },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'attester',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'uid', internalType: 'bytes32', type: 'bytes32', indexed: false },
      {
        name: 'schema',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
    ],
    name: 'Attested',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'attester',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'uid', internalType: 'bytes32', type: 'bytes32', indexed: false },
      {
        name: 'schema',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
    ],
    name: 'Revoked',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'revoker',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'data', internalType: 'bytes32', type: 'bytes32', indexed: true },
      {
        name: 'timestamp',
        internalType: 'uint64',
        type: 'uint64',
        indexed: true,
      },
    ],
    name: 'RevokedOffchain',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'data', internalType: 'bytes32', type: 'bytes32', indexed: true },
      {
        name: 'timestamp',
        internalType: 'uint64',
        type: 'uint64',
        indexed: true,
      },
    ],
    name: 'Timestamped',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'request',
        internalType: 'struct AttestationRequest',
        type: 'tuple',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct AttestationRequestData',
            type: 'tuple',
            components: [
              { name: 'recipient', internalType: 'address', type: 'address' },
              {
                name: 'expirationTime',
                internalType: 'uint64',
                type: 'uint64',
              },
              { name: 'revocable', internalType: 'bool', type: 'bool' },
              { name: 'refUID', internalType: 'bytes32', type: 'bytes32' },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'attest',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'delegatedRequest',
        internalType: 'struct DelegatedAttestationRequest',
        type: 'tuple',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct AttestationRequestData',
            type: 'tuple',
            components: [
              { name: 'recipient', internalType: 'address', type: 'address' },
              {
                name: 'expirationTime',
                internalType: 'uint64',
                type: 'uint64',
              },
              { name: 'revocable', internalType: 'bool', type: 'bool' },
              { name: 'refUID', internalType: 'bytes32', type: 'bytes32' },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
          {
            name: 'signature',
            internalType: 'struct EIP712Signature',
            type: 'tuple',
            components: [
              { name: 'v', internalType: 'uint8', type: 'uint8' },
              { name: 'r', internalType: 'bytes32', type: 'bytes32' },
              { name: 's', internalType: 'bytes32', type: 'bytes32' },
            ],
          },
          { name: 'attester', internalType: 'address', type: 'address' },
        ],
      },
    ],
    name: 'attestByDelegation',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [],
    name: 'getAttestTypeHash',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'uid', internalType: 'bytes32', type: 'bytes32' }],
    name: 'getAttestation',
    outputs: [
      {
        name: '',
        internalType: 'struct Attestation',
        type: 'tuple',
        components: [
          { name: 'uid', internalType: 'bytes32', type: 'bytes32' },
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          { name: 'time', internalType: 'uint64', type: 'uint64' },
          { name: 'expirationTime', internalType: 'uint64', type: 'uint64' },
          { name: 'revocationTime', internalType: 'uint64', type: 'uint64' },
          { name: 'refUID', internalType: 'bytes32', type: 'bytes32' },
          { name: 'recipient', internalType: 'address', type: 'address' },
          { name: 'attester', internalType: 'address', type: 'address' },
          { name: 'revocable', internalType: 'bool', type: 'bool' },
          { name: 'data', internalType: 'bytes', type: 'bytes' },
        ],
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'getDomainSeparator',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'getName',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'account', internalType: 'address', type: 'address' }],
    name: 'getNonce',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: 'revoker', internalType: 'address', type: 'address' },
      { name: 'data', internalType: 'bytes32', type: 'bytes32' },
    ],
    name: 'getRevokeOffchain',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [],
    name: 'getRevokeTypeHash',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [],
    name: 'getSchemaRegistry',
    outputs: [
      { name: '', internalType: 'contract ISchemaRegistry', type: 'address' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'data', internalType: 'bytes32', type: 'bytes32' }],
    name: 'getTimestamp',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'uid', internalType: 'bytes32', type: 'bytes32' }],
    name: 'isAttestationValid',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'multiRequests',
        internalType: 'struct MultiAttestationRequest[]',
        type: 'tuple[]',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct AttestationRequestData[]',
            type: 'tuple[]',
            components: [
              { name: 'recipient', internalType: 'address', type: 'address' },
              {
                name: 'expirationTime',
                internalType: 'uint64',
                type: 'uint64',
              },
              { name: 'revocable', internalType: 'bool', type: 'bool' },
              { name: 'refUID', internalType: 'bytes32', type: 'bytes32' },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'multiAttest',
    outputs: [{ name: '', internalType: 'bytes32[]', type: 'bytes32[]' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'multiDelegatedRequests',
        internalType: 'struct MultiDelegatedAttestationRequest[]',
        type: 'tuple[]',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct AttestationRequestData[]',
            type: 'tuple[]',
            components: [
              { name: 'recipient', internalType: 'address', type: 'address' },
              {
                name: 'expirationTime',
                internalType: 'uint64',
                type: 'uint64',
              },
              { name: 'revocable', internalType: 'bool', type: 'bool' },
              { name: 'refUID', internalType: 'bytes32', type: 'bytes32' },
              { name: 'data', internalType: 'bytes', type: 'bytes' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
          {
            name: 'signatures',
            internalType: 'struct EIP712Signature[]',
            type: 'tuple[]',
            components: [
              { name: 'v', internalType: 'uint8', type: 'uint8' },
              { name: 'r', internalType: 'bytes32', type: 'bytes32' },
              { name: 's', internalType: 'bytes32', type: 'bytes32' },
            ],
          },
          { name: 'attester', internalType: 'address', type: 'address' },
        ],
      },
    ],
    name: 'multiAttestByDelegation',
    outputs: [{ name: '', internalType: 'bytes32[]', type: 'bytes32[]' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'multiRequests',
        internalType: 'struct MultiRevocationRequest[]',
        type: 'tuple[]',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct RevocationRequestData[]',
            type: 'tuple[]',
            components: [
              { name: 'uid', internalType: 'bytes32', type: 'bytes32' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'multiRevoke',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'multiDelegatedRequests',
        internalType: 'struct MultiDelegatedRevocationRequest[]',
        type: 'tuple[]',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct RevocationRequestData[]',
            type: 'tuple[]',
            components: [
              { name: 'uid', internalType: 'bytes32', type: 'bytes32' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
          {
            name: 'signatures',
            internalType: 'struct EIP712Signature[]',
            type: 'tuple[]',
            components: [
              { name: 'v', internalType: 'uint8', type: 'uint8' },
              { name: 'r', internalType: 'bytes32', type: 'bytes32' },
              { name: 's', internalType: 'bytes32', type: 'bytes32' },
            ],
          },
          { name: 'revoker', internalType: 'address', type: 'address' },
        ],
      },
    ],
    name: 'multiRevokeByDelegation',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'data', internalType: 'bytes32[]', type: 'bytes32[]' }],
    name: 'multiRevokeOffchain',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'data', internalType: 'bytes32[]', type: 'bytes32[]' }],
    name: 'multiTimestamp',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'request',
        internalType: 'struct RevocationRequest',
        type: 'tuple',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct RevocationRequestData',
            type: 'tuple',
            components: [
              { name: 'uid', internalType: 'bytes32', type: 'bytes32' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
        ],
      },
    ],
    name: 'revoke',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      {
        name: 'delegatedRequest',
        internalType: 'struct DelegatedRevocationRequest',
        type: 'tuple',
        components: [
          { name: 'schema', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'data',
            internalType: 'struct RevocationRequestData',
            type: 'tuple',
            components: [
              { name: 'uid', internalType: 'bytes32', type: 'bytes32' },
              { name: 'value', internalType: 'uint256', type: 'uint256' },
            ],
          },
          {
            name: 'signature',
            internalType: 'struct EIP712Signature',
            type: 'tuple',
            components: [
              { name: 'v', internalType: 'uint8', type: 'uint8' },
              { name: 'r', internalType: 'bytes32', type: 'bytes32' },
              { name: 's', internalType: 'bytes32', type: 'bytes32' },
            ],
          },
          { name: 'revoker', internalType: 'address', type: 'address' },
        ],
      },
    ],
    name: 'revokeByDelegation',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'data', internalType: 'bytes32', type: 'bytes32' }],
    name: 'revokeOffchain',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'data', internalType: 'bytes32', type: 'bytes32' }],
    name: 'timestamp',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5A633F1cc84B03F7588486CF2F386c102061E6e1)
 */
export const easAddress = {
  10: '0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088',
  420: '0x5A633F1cc84B03F7588486CF2F386c102061E6e1',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5A633F1cc84B03F7588486CF2F386c102061E6e1)
 */
export const easConfig = { address: easAddress, abi: easABI } as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// GasPriceOracle
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000000F)
 */
export const gasPriceOracleABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'baseFee',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [],
    name: 'decimals',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'gasPrice',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_data', internalType: 'bytes', type: 'bytes' }],
    name: 'getL1Fee',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_data', internalType: 'bytes', type: 'bytes' }],
    name: 'getL1GasUsed',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1BaseFee',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'overhead',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'scalar',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000000F)
 */
export const gasPriceOracleAddress = {
  420: '0x420000000000000000000000000000000000000F',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000000F)
 */
export const gasPriceOracleConfig = {
  address: gasPriceOracleAddress,
  abi: gasPriceOracleABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L1Block
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000015)
 */
export const l1BlockABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'DEPOSITOR_ACCOUNT',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'basefee',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'batcherHash',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'hash',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1FeeOverhead',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1FeeScalar',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'number',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'sequenceNumber',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_number', internalType: 'uint64', type: 'uint64' },
      { name: '_timestamp', internalType: 'uint64', type: 'uint64' },
      { name: '_basefee', internalType: 'uint256', type: 'uint256' },
      { name: '_hash', internalType: 'bytes32', type: 'bytes32' },
      { name: '_sequenceNumber', internalType: 'uint64', type: 'uint64' },
      { name: '_batcherHash', internalType: 'bytes32', type: 'bytes32' },
      { name: '_l1FeeOverhead', internalType: 'uint256', type: 'uint256' },
      { name: '_l1FeeScalar', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'setL1BlockValues',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'timestamp',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000015)
 */
export const l1BlockAddress = {
  420: '0x4200000000000000000000000000000000000015',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000015)
 */
export const l1BlockConfig = {
  address: l1BlockAddress,
  abi: l1BlockABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L1CrossDomainMessenger
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294)
 */
export const l1CrossDomainMessengerABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'msgHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
    ],
    name: 'FailedRelayedMessage',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'msgHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
    ],
    name: 'RelayedMessage',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'target',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'sender',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
      { name: 'message', internalType: 'bytes', type: 'bytes', indexed: false },
      {
        name: 'messageNonce',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'gasLimit',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'SentMessage',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'sender',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'value',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'SentMessageExtension1',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MESSAGE_VERSION',
    outputs: [{ name: '', internalType: 'uint16', type: 'uint16' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_GAS_CALLDATA_OVERHEAD',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OTHER_MESSENGER',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_CALL_OVERHEAD',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_CONSTANT_OVERHEAD',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_GAS_CHECK_BUFFER',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_RESERVED_GAS',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [
      { name: '_message', internalType: 'bytes', type: 'bytes' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
    ],
    name: 'baseGas',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'failedMessages',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_portal',
        internalType: 'contract OptimismPortal',
        type: 'address',
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messageNonce',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'portal',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_nonce', internalType: 'uint256', type: 'uint256' },
      { name: '_sender', internalType: 'address', type: 'address' },
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint256', type: 'uint256' },
      { name: '_message', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'relayMessage',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_message', internalType: 'bytes', type: 'bytes' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
    ],
    name: 'sendMessage',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'successfulMessages',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'xDomainMessageSender',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294)
 */
export const l1CrossDomainMessengerAddress = {
  1: '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1',
  5: '0x5086d1eEF304eb5284A0f6720f79403b4e9bE294',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294)
 */
export const l1CrossDomainMessengerConfig = {
  address: l1CrossDomainMessengerAddress,
  abi: l1CrossDomainMessengerABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L1ERC721Bridge
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5a7749f83b81B301cAb5f48EB8516B986DAef23D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9)
 */
export const l1Erc721BridgeABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'tokenId',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC721BridgeFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'tokenId',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC721BridgeInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_tokenId', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_tokenId', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC721To',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'deposits',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_tokenId', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeBridgeERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_messenger',
        internalType: 'contract CrossDomainMessenger',
        type: 'address',
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messenger',
    outputs: [
      {
        name: '',
        internalType: 'contract CrossDomainMessenger',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'otherBridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5a7749f83b81B301cAb5f48EB8516B986DAef23D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9)
 */
export const l1Erc721BridgeAddress = {
  1: '0x5a7749f83b81B301cAb5f48EB8516B986DAef23D',
  5: '0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5a7749f83b81B301cAb5f48EB8516B986DAef23D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9)
 */
export const l1Erc721BridgeConfig = {
  address: l1Erc721BridgeAddress,
  abi: l1Erc721BridgeABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L1FeeVault
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000001a)
 */
export const l1FeeVaultABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_recipient', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'value',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'from',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'Withdrawal',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_WITHDRAWAL_AMOUNT',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RECIPIENT',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'totalProcessed',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'withdraw',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000001a)
 */
export const l1FeeVaultAddress = {
  420: '0x420000000000000000000000000000000000001A',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000001a)
 */
export const l1FeeVaultConfig = {
  address: l1FeeVaultAddress,
  abi: l1FeeVaultABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L1StandardBridge
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8)
 */
export const l1StandardBridgeABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC20BridgeFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC20BridgeInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'l1Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'l2Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC20DepositInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'l1Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'l2Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC20WithdrawalFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ETHBridgeFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ETHBridgeInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ETHDepositInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ETHWithdrawalFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC20To',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeETH',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeETHTo',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_l1Token', internalType: 'address', type: 'address' },
      { name: '_l2Token', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'depositERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_l1Token', internalType: 'address', type: 'address' },
      { name: '_l2Token', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'depositERC20To',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'depositETH',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'depositETHTo',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'address', type: 'address' },
    ],
    name: 'deposits',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeBridgeERC20',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeBridgeETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_l1Token', internalType: 'address', type: 'address' },
      { name: '_l2Token', internalType: 'address', type: 'address' },
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeERC20Withdrawal',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeETHWithdrawal',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_messenger',
        internalType: 'contract CrossDomainMessenger',
        type: 'address',
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2TokenBridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messenger',
    outputs: [
      {
        name: '',
        internalType: 'contract CrossDomainMessenger',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'otherBridge',
    outputs: [
      { name: '', internalType: 'contract StandardBridge', type: 'address' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8)
 */
export const l1StandardBridgeAddress = {
  1: '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1',
  5: '0x636Af16bf2f682dD3109e60102b8E1A089FedAa8',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8)
 */
export const l1StandardBridgeConfig = {
  address: l1StandardBridgeAddress,
  abi: l1StandardBridgeABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L2CrossDomainMessenger
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000007)
 */
export const l2CrossDomainMessengerABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      {
        name: '_l1CrossDomainMessenger',
        internalType: 'address',
        type: 'address',
      },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'msgHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
    ],
    name: 'FailedRelayedMessage',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'msgHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
    ],
    name: 'RelayedMessage',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'target',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'sender',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
      { name: 'message', internalType: 'bytes', type: 'bytes', indexed: false },
      {
        name: 'messageNonce',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'gasLimit',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'SentMessage',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'sender',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'value',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'SentMessageExtension1',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MESSAGE_VERSION',
    outputs: [{ name: '', internalType: 'uint16', type: 'uint16' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_GAS_CALLDATA_OVERHEAD',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OTHER_MESSENGER',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_CALL_OVERHEAD',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_CONSTANT_OVERHEAD',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_GAS_CHECK_BUFFER',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RELAY_RESERVED_GAS',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [
      { name: '_message', internalType: 'bytes', type: 'bytes' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
    ],
    name: 'baseGas',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'failedMessages',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1CrossDomainMessenger',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messageNonce',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_nonce', internalType: 'uint256', type: 'uint256' },
      { name: '_sender', internalType: 'address', type: 'address' },
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint256', type: 'uint256' },
      { name: '_message', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'relayMessage',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_message', internalType: 'bytes', type: 'bytes' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
    ],
    name: 'sendMessage',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'successfulMessages',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'xDomainMessageSender',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000007)
 */
export const l2CrossDomainMessengerAddress = {
  420: '0x4200000000000000000000000000000000000007',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000007)
 */
export const l2CrossDomainMessengerConfig = {
  address: l2CrossDomainMessengerAddress,
  abi: l2CrossDomainMessengerABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L2ERC721Bridge
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000014)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000014)
 */
export const l2Erc721BridgeABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_messenger', internalType: 'address', type: 'address' },
      { name: '_otherBridge', internalType: 'address', type: 'address' },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'tokenId',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC721BridgeFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'tokenId',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC721BridgeInitiated',
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_tokenId', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_tokenId', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC721To',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_tokenId', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeBridgeERC721',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messenger',
    outputs: [
      {
        name: '',
        internalType: 'contract CrossDomainMessenger',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'otherBridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000014)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000014)
 */
export const l2Erc721BridgeAddress = {
  10: '0x4200000000000000000000000000000000000014',
  420: '0x4200000000000000000000000000000000000014',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000014)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000014)
 */
export const l2Erc721BridgeConfig = {
  address: l2Erc721BridgeAddress,
  abi: l2Erc721BridgeABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L2OutputOracle
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdfe97868233d1aa22e815a266982f2cf17685a27)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)
 */
export const l2OutputOracleABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_submissionInterval', internalType: 'uint256', type: 'uint256' },
      { name: '_l2BlockTime', internalType: 'uint256', type: 'uint256' },
      {
        name: '_finalizationPeriodSeconds',
        internalType: 'uint256',
        type: 'uint256',
      },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'outputRoot',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
      {
        name: 'l2OutputIndex',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'l2BlockNumber',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'l1Timestamp',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'OutputProposed',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'prevNextOutputIndex',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'newNextOutputIndex',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
    ],
    name: 'OutputsDeleted',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'challenger',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '_l2BlockNumber', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'computeL2Timestamp',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_l2OutputIndex', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'deleteL2Outputs',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'finalizationPeriodSeconds',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '_l2OutputIndex', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'getL2Output',
    outputs: [
      {
        name: '',
        internalType: 'struct Types.OutputProposal',
        type: 'tuple',
        components: [
          { name: 'outputRoot', internalType: 'bytes32', type: 'bytes32' },
          { name: 'timestamp', internalType: 'uint128', type: 'uint128' },
          { name: 'l2BlockNumber', internalType: 'uint128', type: 'uint128' },
        ],
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '_l2BlockNumber', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'getL2OutputAfter',
    outputs: [
      {
        name: '',
        internalType: 'struct Types.OutputProposal',
        type: 'tuple',
        components: [
          { name: 'outputRoot', internalType: 'bytes32', type: 'bytes32' },
          { name: 'timestamp', internalType: 'uint128', type: 'uint128' },
          { name: 'l2BlockNumber', internalType: 'uint128', type: 'uint128' },
        ],
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '_l2BlockNumber', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'getL2OutputIndexAfter',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_startingBlockNumber',
        internalType: 'uint256',
        type: 'uint256',
      },
      { name: '_startingTimestamp', internalType: 'uint256', type: 'uint256' },
      { name: '_proposer', internalType: 'address', type: 'address' },
      { name: '_challenger', internalType: 'address', type: 'address' },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2BlockTime',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'latestBlockNumber',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'latestOutputIndex',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'nextBlockNumber',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'nextOutputIndex',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_outputRoot', internalType: 'bytes32', type: 'bytes32' },
      { name: '_l2BlockNumber', internalType: 'uint256', type: 'uint256' },
      { name: '_l1BlockHash', internalType: 'bytes32', type: 'bytes32' },
      { name: '_l1BlockNumber', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'proposeL2Output',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'proposer',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'startingBlockNumber',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'startingTimestamp',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'submissionInterval',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdfe97868233d1aa22e815a266982f2cf17685a27)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)
 */
export const l2OutputOracleAddress = {
  1: '0xdfe97868233d1aa22e815a266982f2cf17685a27',
  5: '0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdfe97868233d1aa22e815a266982f2cf17685a27)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)
 */
export const l2OutputOracleConfig = {
  address: l2OutputOracleAddress,
  abi: l2OutputOracleABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L2StandardBridge
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000010)
 */
export const l2StandardBridgeABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      {
        name: '_otherBridge',
        internalType: 'address payable',
        type: 'address',
      },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'l1Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'l2Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'DepositFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC20BridgeFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ERC20BridgeInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ETHBridgeFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'ETHBridgeInitiated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'l1Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'l2Token',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'extraData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'WithdrawalInitiated',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OTHER_BRIDGE',
    outputs: [
      { name: '', internalType: 'contract StandardBridge', type: 'address' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeERC20To',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeETH',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'bridgeETHTo',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'address', type: 'address' },
    ],
    name: 'deposits',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_localToken', internalType: 'address', type: 'address' },
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeBridgeERC20',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeBridgeETH',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_l1Token', internalType: 'address', type: 'address' },
      { name: '_l2Token', internalType: 'address', type: 'address' },
      { name: '_from', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'finalizeDeposit',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1TokenBridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messenger',
    outputs: [
      {
        name: '',
        internalType: 'contract CrossDomainMessenger',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_l2Token', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'withdraw',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_l2Token', internalType: 'address', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
      { name: '_minGasLimit', internalType: 'uint32', type: 'uint32' },
      { name: '_extraData', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'withdrawTo',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000010)
 */
export const l2StandardBridgeAddress = {
  420: '0x4200000000000000000000000000000000000010',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000010)
 */
export const l2StandardBridgeConfig = {
  address: l2StandardBridgeAddress,
  abi: l2StandardBridgeABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// L2ToL1MessagePasser
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000016)
 */
export const l2ToL1MessagePasserABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'nonce',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'sender',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'target',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'value',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      {
        name: 'gasLimit',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      { name: 'data', internalType: 'bytes', type: 'bytes', indexed: false },
      {
        name: 'withdrawalHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: false,
      },
    ],
    name: 'MessagePassed',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
    ],
    name: 'WithdrawerBalanceBurnt',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MESSAGE_VERSION',
    outputs: [{ name: '', internalType: 'uint16', type: 'uint16' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'burn',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_gasLimit', internalType: 'uint256', type: 'uint256' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'initiateWithdrawal',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'messageNonce',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'sentMessages',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000016)
 */
export const l2ToL1MessagePasserAddress = {
  420: '0x4200000000000000000000000000000000000016',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000016)
 */
export const l2ToL1MessagePasserConfig = {
  address: l2ToL1MessagePasserAddress,
  abi: l2ToL1MessagePasserABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// MintManager
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x5C4e7Ba1E219E47948e6e3F55019A647bA501005)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)
 */
export const mintManagerABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_upgrader', internalType: 'address', type: 'address' },
      { name: '_governanceToken', internalType: 'address', type: 'address' },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'DENOMINATOR',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MINT_CAP',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MINT_PERIOD',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'governanceToken',
    outputs: [
      { name: '', internalType: 'contract GovernanceToken', type: 'address' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_account', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'mint',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'mintPermittedAfter',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_newMintManager', internalType: 'address', type: 'address' },
    ],
    name: 'upgrade',
    outputs: [],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x5C4e7Ba1E219E47948e6e3F55019A647bA501005)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)
 */
export const mintManagerAddress = {
  10: '0x5C4e7Ba1E219E47948e6e3F55019A647bA501005',
  420: '0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x5C4e7Ba1E219E47948e6e3F55019A647bA501005)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)
 */
export const mintManagerConfig = {
  address: mintManagerAddress,
  abi: mintManagerABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimismMintableERC20Factory
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'deployer',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'OptimismMintableERC20Created',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'StandardL2TokenCreated',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'bridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createOptimismMintableERC20',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
      { name: '_decimals', internalType: 'uint8', type: 'uint8' },
    ],
    name: 'createOptimismMintableERC20WithDecimals',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createStandardL2Token',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_bridge', internalType: 'address', type: 'address' }],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryAddress = {
  1: '0x4200000000000000000000000000000000000012',
} as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryConfig = {
  address: optimismMintableErc20FactoryAddress,
  abi: optimismMintableErc20FactoryABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimismMintableERC20Factory_goerli
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryGoerliABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'deployer',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'OptimismMintableERC20Created',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'StandardL2TokenCreated',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'bridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createOptimismMintableERC20',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
      { name: '_decimals', internalType: 'uint8', type: 'uint8' },
    ],
    name: 'createOptimismMintableERC20WithDecimals',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createStandardL2Token',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_bridge', internalType: 'address', type: 'address' }],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryGoerliAddress = {
  5: '0x4200000000000000000000000000000000000012',
} as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryGoerliConfig = {
  address: optimismMintableErc20FactoryGoerliAddress,
  abi: optimismMintableErc20FactoryGoerliABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimismMintableERC20Factory_optimism-goerli
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryOptimismGoerliABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_bridge', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'deployer',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'OptimismMintableERC20Created',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'StandardL2TokenCreated',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'BRIDGE',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createOptimismMintableERC20',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createStandardL2Token',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryOptimismGoerliAddress = {
  420: '0x4200000000000000000000000000000000000012',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export const optimismMintableErc20FactoryOptimismGoerliConfig = {
  address: optimismMintableErc20FactoryOptimismGoerliAddress,
  abi: optimismMintableErc20FactoryOptimismGoerliABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimismMintableERC721Factory
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000017)
 */
export const optimismMintableErc721FactoryABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_bridge', internalType: 'address', type: 'address' },
      { name: '_remoteChainId', internalType: 'uint256', type: 'uint256' },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'deployer',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'OptimismMintableERC721Created',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'bridge',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createOptimismMintableERC721',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'address', type: 'address' }],
    name: 'isOptimismMintableERC721',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'remoteChainId',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000017)
 */
export const optimismMintableErc721FactoryAddress = {
  10: '0x4200000000000000000000000000000000000017',
} as const

/**
 * [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000017)
 */
export const optimismMintableErc721FactoryConfig = {
  address: optimismMintableErc721FactoryAddress,
  abi: optimismMintableErc721FactoryABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimismMintableERC721Factory_optimism-goerli
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000017)
 */
export const optimismMintableErc721FactoryOptimismGoerliABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_bridge', internalType: 'address', type: 'address' },
      { name: '_remoteChainId', internalType: 'uint256', type: 'uint256' },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'localToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'remoteToken',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'deployer',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'OptimismMintableERC721Created',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'BRIDGE',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'REMOTE_CHAIN_ID',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_remoteToken', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'createOptimismMintableERC721',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'address', type: 'address' }],
    name: 'isOptimismMintableERC721',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000017)
 */
export const optimismMintableErc721FactoryOptimismGoerliAddress = {
  420: '0x4200000000000000000000000000000000000017',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000017)
 */
export const optimismMintableErc721FactoryOptimismGoerliConfig = {
  address: optimismMintableErc721FactoryOptimismGoerliAddress,
  abi: optimismMintableErc721FactoryOptimismGoerliABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimismPortal
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xbEb5Fc579115071764c7423A4f12eDde41f106Ed)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383)
 */
export const optimismPortalABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'account',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'Paused',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'version',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'opaqueData',
        internalType: 'bytes',
        type: 'bytes',
        indexed: false,
      },
    ],
    name: 'TransactionDeposited',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'account',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'Unpaused',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawalHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
      { name: 'success', internalType: 'bool', type: 'bool', indexed: false },
    ],
    name: 'WithdrawalFinalized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawalHash',
        internalType: 'bytes32',
        type: 'bytes32',
        indexed: true,
      },
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
    ],
    name: 'WithdrawalProven',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
      { name: '_gasLimit', internalType: 'uint64', type: 'uint64' },
      { name: '_isCreation', internalType: 'bool', type: 'bool' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'depositTransaction',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [],
    name: 'donateETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_tx',
        internalType: 'struct Types.WithdrawalTransaction',
        type: 'tuple',
        components: [
          { name: 'nonce', internalType: 'uint256', type: 'uint256' },
          { name: 'sender', internalType: 'address', type: 'address' },
          { name: 'target', internalType: 'address', type: 'address' },
          { name: 'value', internalType: 'uint256', type: 'uint256' },
          { name: 'gasLimit', internalType: 'uint256', type: 'uint256' },
          { name: 'data', internalType: 'bytes', type: 'bytes' },
        ],
      },
    ],
    name: 'finalizeWithdrawalTransaction',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'finalizedWithdrawals',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'guardian',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_l2Oracle',
        internalType: 'contract L2OutputOracle',
        type: 'address',
      },
      { name: '_guardian', internalType: 'address', type: 'address' },
      {
        name: '_systemConfig',
        internalType: 'contract SystemConfig',
        type: 'address',
      },
      { name: '_paused', internalType: 'bool', type: 'bool' },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '_l2OutputIndex', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'isOutputFinalized',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2Oracle',
    outputs: [
      { name: '', internalType: 'contract L2OutputOracle', type: 'address' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2Sender',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [{ name: '_byteCount', internalType: 'uint64', type: 'uint64' }],
    name: 'minimumGasLimit',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'params',
    outputs: [
      { name: 'prevBaseFee', internalType: 'uint128', type: 'uint128' },
      { name: 'prevBoughtGas', internalType: 'uint64', type: 'uint64' },
      { name: 'prevBlockNum', internalType: 'uint64', type: 'uint64' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'pause',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'paused',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_tx',
        internalType: 'struct Types.WithdrawalTransaction',
        type: 'tuple',
        components: [
          { name: 'nonce', internalType: 'uint256', type: 'uint256' },
          { name: 'sender', internalType: 'address', type: 'address' },
          { name: 'target', internalType: 'address', type: 'address' },
          { name: 'value', internalType: 'uint256', type: 'uint256' },
          { name: 'gasLimit', internalType: 'uint256', type: 'uint256' },
          { name: 'data', internalType: 'bytes', type: 'bytes' },
        ],
      },
      { name: '_l2OutputIndex', internalType: 'uint256', type: 'uint256' },
      {
        name: '_outputRootProof',
        internalType: 'struct Types.OutputRootProof',
        type: 'tuple',
        components: [
          { name: 'version', internalType: 'bytes32', type: 'bytes32' },
          { name: 'stateRoot', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'messagePasserStorageRoot',
            internalType: 'bytes32',
            type: 'bytes32',
          },
          { name: 'latestBlockhash', internalType: 'bytes32', type: 'bytes32' },
        ],
      },
      { name: '_withdrawalProof', internalType: 'bytes[]', type: 'bytes[]' },
    ],
    name: 'proveWithdrawalTransaction',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'provenWithdrawals',
    outputs: [
      { name: 'outputRoot', internalType: 'bytes32', type: 'bytes32' },
      { name: 'timestamp', internalType: 'uint128', type: 'uint128' },
      { name: 'l2OutputIndex', internalType: 'uint128', type: 'uint128' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'systemConfig',
    outputs: [
      { name: '', internalType: 'contract SystemConfig', type: 'address' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'unpause',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xbEb5Fc579115071764c7423A4f12eDde41f106Ed)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383)
 */
export const optimismPortalAddress = {
  1: '0xbEb5Fc579115071764c7423A4f12eDde41f106Ed',
  5: '0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xbEb5Fc579115071764c7423A4f12eDde41f106Ed)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383)
 */
export const optimismPortalConfig = {
  address: optimismPortalAddress,
  abi: optimismPortalABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Optimist
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 */
export const optimistABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
      { name: '_baseURIAttestor', internalType: 'address', type: 'address' },
      {
        name: '_attestationStation',
        internalType: 'contract AttestationStation',
        type: 'address',
      },
      {
        name: '_optimistAllowlist',
        internalType: 'contract OptimistAllowlist',
        type: 'address',
      },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'owner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'approved',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'tokenId',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
    ],
    name: 'Approval',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'owner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'operator',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'approved', internalType: 'bool', type: 'bool', indexed: false },
    ],
    name: 'ApprovalForAll',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      { name: 'to', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'tokenId',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
    ],
    name: 'Transfer',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'ATTESTATION_STATION',
    outputs: [
      {
        name: '',
        internalType: 'contract AttestationStation',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'BASE_URI_ATTESTATION_KEY',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'BASE_URI_ATTESTOR',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OPTIMIST_ALLOWLIST',
    outputs: [
      { name: '', internalType: 'contract OptimistAllowlist', type: 'address' },
    ],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'approve',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'owner', internalType: 'address', type: 'address' }],
    name: 'balanceOf',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'baseURI',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'tokenId', internalType: 'uint256', type: 'uint256' }],
    name: 'burn',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'tokenId', internalType: 'uint256', type: 'uint256' }],
    name: 'getApproved',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_symbol', internalType: 'string', type: 'string' },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: 'owner', internalType: 'address', type: 'address' },
      { name: 'operator', internalType: 'address', type: 'address' },
    ],
    name: 'isApprovedForAll',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_recipient', internalType: 'address', type: 'address' }],
    name: 'isOnAllowList',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_recipient', internalType: 'address', type: 'address' }],
    name: 'mint',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'name',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'tokenId', internalType: 'uint256', type: 'uint256' }],
    name: 'ownerOf',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: 'from', internalType: 'address', type: 'address' },
      { name: 'to', internalType: 'address', type: 'address' },
      { name: 'tokenId', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'safeTransferFrom',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: 'from', internalType: 'address', type: 'address' },
      { name: 'to', internalType: 'address', type: 'address' },
      { name: 'tokenId', internalType: 'uint256', type: 'uint256' },
      { name: 'data', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'safeTransferFrom',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'bool', type: 'bool' },
    ],
    name: 'setApprovalForAll',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'interfaceId', internalType: 'bytes4', type: 'bytes4' }],
    name: 'supportsInterface',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'symbol',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'pure',
    type: 'function',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
    name: 'tokenIdOfAddress',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_tokenId', internalType: 'uint256', type: 'uint256' }],
    name: 'tokenURI',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: 'from', internalType: 'address', type: 'address' },
      { name: 'to', internalType: 'address', type: 'address' },
      { name: 'tokenId', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'transferFrom',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 */
export const optimistAddress = {
  10: '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5',
  420: '0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 */
export const optimistConfig = {
  address: optimistAddress,
  abi: optimistABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimistAllowlist
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 */
export const optimistAllowlistABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      {
        name: '_attestationStation',
        internalType: 'contract AttestationStation',
        type: 'address',
      },
      { name: '_allowlistAttestor', internalType: 'address', type: 'address' },
      {
        name: '_coinbaseQuestAttestor',
        internalType: 'address',
        type: 'address',
      },
      { name: '_optimistInviter', internalType: 'address', type: 'address' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'ALLOWLIST_ATTESTOR',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'ATTESTATION_STATION',
    outputs: [
      {
        name: '',
        internalType: 'contract AttestationStation',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'COINBASE_QUEST_ATTESTOR',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'COINBASE_QUEST_ELIGIBLE_ATTESTATION_KEY',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OPTIMIST_CAN_MINT_ATTESTATION_KEY',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OPTIMIST_INVITER',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_claimer', internalType: 'address', type: 'address' }],
    name: 'isAllowedToMint',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 */
export const optimistAllowlistAddress = {
  10: '0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180',
  420: '0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 */
export const optimistAllowlistConfig = {
  address: optimistAllowlistAddress,
  abi: optimistAllowlistABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// OptimistInviter
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 */
export const optimistInviterABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      { name: '_inviteGranter', internalType: 'address', type: 'address' },
      {
        name: '_attestationStation',
        internalType: 'contract AttestationStation',
        type: 'address',
      },
    ],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'issuer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'claimer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'InviteClaimed',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'ATTESTATION_STATION',
    outputs: [
      {
        name: '',
        internalType: 'contract AttestationStation',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'CAN_INVITE_ATTESTATION_KEY',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'CLAIMABLE_INVITE_TYPEHASH',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'EIP712_VERSION',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'INVITE_GRANTER',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_COMMITMENT_PERIOD',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_claimer', internalType: 'address', type: 'address' },
      {
        name: '_claimableInvite',
        internalType: 'struct OptimistInviter.ClaimableInvite',
        type: 'tuple',
        components: [
          { name: 'issuer', internalType: 'address', type: 'address' },
          { name: 'nonce', internalType: 'bytes32', type: 'bytes32' },
        ],
      },
      { name: '_signature', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'claimInvite',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_commitment', internalType: 'bytes32', type: 'bytes32' }],
    name: 'commitInvite',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
    name: 'commitmentTimestamps',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_name', internalType: 'string', type: 'string' }],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'address', type: 'address' }],
    name: 'inviteCounts',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_accounts', internalType: 'address[]', type: 'address[]' },
      { name: '_inviteCount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'setInviteCounts',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '', internalType: 'address', type: 'address' },
      { name: '', internalType: 'bytes32', type: 'bytes32' },
    ],
    name: 'usedNonces',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 */
export const optimistInviterAddress = {
  10: '0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929',
  420: '0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 */
export const optimistInviterConfig = {
  address: optimistInviterAddress,
  abi: optimistInviterABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// PortalSender
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x0A893d9576b9cFD9EF78595963dc973238E78210)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xe7FACd39531ee3C313330E93B4d7a8B8A3c84Aa4)
 */
export const portalSenderABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [
      {
        name: '_portal',
        internalType: 'contract OptimismPortal',
        type: 'address',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'PORTAL',
    outputs: [
      { name: '', internalType: 'contract OptimismPortal', type: 'address' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'donate',
    outputs: [],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x0A893d9576b9cFD9EF78595963dc973238E78210)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xe7FACd39531ee3C313330E93B4d7a8B8A3c84Aa4)
 */
export const portalSenderAddress = {
  1: '0x0A893d9576b9cFD9EF78595963dc973238E78210',
  5: '0xe7FACd39531ee3C313330E93B4d7a8B8A3c84Aa4',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x0A893d9576b9cFD9EF78595963dc973238E78210)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xe7FACd39531ee3C313330E93B4d7a8B8A3c84Aa4)
 */
export const portalSenderConfig = {
  address: portalSenderAddress,
  abi: portalSenderABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ProtocolVersions
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x0C24F5098774aA366827D667494e9F889f7cFc08)
 */
export const protocolVersionsABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'version',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'updateType',
        internalType: 'enum ProtocolVersions.UpdateType',
        type: 'uint8',
        indexed: true,
      },
      { name: 'data', internalType: 'bytes', type: 'bytes', indexed: false },
    ],
    name: 'ConfigUpdate',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RECOMMENDED_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'REQUIRED_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_owner', internalType: 'address', type: 'address' },
      { name: '_required', internalType: 'ProtocolVersion', type: 'uint256' },
      {
        name: '_recommended',
        internalType: 'ProtocolVersion',
        type: 'uint256',
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'recommended',
    outputs: [
      { name: 'out_', internalType: 'ProtocolVersion', type: 'uint256' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'required',
    outputs: [
      { name: 'out_', internalType: 'ProtocolVersion', type: 'uint256' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_recommended',
        internalType: 'ProtocolVersion',
        type: 'uint256',
      },
    ],
    name: 'setRecommended',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_required', internalType: 'ProtocolVersion', type: 'uint256' },
    ],
    name: 'setRequired',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x0C24F5098774aA366827D667494e9F889f7cFc08)
 */
export const protocolVersionsAddress = {
  5: '0x0C24F5098774aA366827D667494e9F889f7cFc08',
} as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x0C24F5098774aA366827D667494e9F889f7cFc08)
 */
export const protocolVersionsConfig = {
  address: protocolVersionsAddress,
  abi: protocolVersionsABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ProxyAdmin
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000018)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000018)
 */
export const proxyAdminABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'addressManager',
    outputs: [
      { name: '', internalType: 'contract AddressManager', type: 'address' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_proxy', internalType: 'address payable', type: 'address' },
      { name: '_newAdmin', internalType: 'address', type: 'address' },
    ],
    name: 'changeProxyAdmin',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [
      { name: '_proxy', internalType: 'address payable', type: 'address' },
    ],
    name: 'getProxyAdmin',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '_proxy', internalType: 'address', type: 'address' }],
    name: 'getProxyImplementation',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'address', type: 'address' }],
    name: 'implementationName',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'isUpgrading',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: '', internalType: 'address', type: 'address' }],
    name: 'proxyType',
    outputs: [
      { name: '', internalType: 'enum ProxyAdmin.ProxyType', type: 'uint8' },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_name', internalType: 'string', type: 'string' },
      { name: '_address', internalType: 'address', type: 'address' },
    ],
    name: 'setAddress',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_address',
        internalType: 'contract AddressManager',
        type: 'address',
      },
    ],
    name: 'setAddressManager',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_address', internalType: 'address', type: 'address' },
      { name: '_name', internalType: 'string', type: 'string' },
    ],
    name: 'setImplementationName',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_address', internalType: 'address', type: 'address' },
      {
        name: '_type',
        internalType: 'enum ProxyAdmin.ProxyType',
        type: 'uint8',
      },
    ],
    name: 'setProxyType',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_upgrading', internalType: 'bool', type: 'bool' }],
    name: 'setUpgrading',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_proxy', internalType: 'address payable', type: 'address' },
      { name: '_implementation', internalType: 'address', type: 'address' },
    ],
    name: 'upgrade',
    outputs: [],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_proxy', internalType: 'address payable', type: 'address' },
      { name: '_implementation', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
    ],
    name: 'upgradeAndCall',
    outputs: [],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000018)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000018)
 */
export const proxyAdminAddress = {
  1: '0x4200000000000000000000000000000000000018',
  5: '0x4200000000000000000000000000000000000018',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000018)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000018)
 */
export const proxyAdminConfig = {
  address: proxyAdminAddress,
  abi: proxyAdminABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SchemaRegistry
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)
 */
export const schemaRegistryABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  { type: 'error', inputs: [], name: 'AlreadyExists' },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'uid', internalType: 'bytes32', type: 'bytes32', indexed: true },
      {
        name: 'registerer',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'Registered',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [{ name: 'uid', internalType: 'bytes32', type: 'bytes32' }],
    name: 'getSchema',
    outputs: [
      {
        name: '',
        internalType: 'struct SchemaRecord',
        type: 'tuple',
        components: [
          { name: 'uid', internalType: 'bytes32', type: 'bytes32' },
          {
            name: 'resolver',
            internalType: 'contract ISchemaResolver',
            type: 'address',
          },
          { name: 'revocable', internalType: 'bool', type: 'bool' },
          { name: 'schema', internalType: 'string', type: 'string' },
        ],
      },
    ],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: 'schema', internalType: 'string', type: 'string' },
      {
        name: 'resolver',
        internalType: 'contract ISchemaResolver',
        type: 'address',
      },
      { name: 'revocable', internalType: 'bool', type: 'bool' },
    ],
    name: 'register',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)
 */
export const schemaRegistryAddress = {
  10: '0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13',
  420: '0x2545fa928d5d278cA75Fd47306e4a89096ff6403',
} as const

/**
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)
 */
export const schemaRegistryConfig = {
  address: schemaRegistryAddress,
  abi: schemaRegistryABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SequencerFeeVault
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000011)
 */
export const sequencerFeeVaultABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_recipient', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'value',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
      { name: 'to', internalType: 'address', type: 'address', indexed: false },
      {
        name: 'from',
        internalType: 'address',
        type: 'address',
        indexed: false,
      },
    ],
    name: 'Withdrawal',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'MIN_WITHDRAWAL_AMOUNT',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'RECIPIENT',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1FeeWallet',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'totalProcessed',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'withdraw',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000011)
 */
export const sequencerFeeVaultAddress = {
  420: '0x4200000000000000000000000000000000000011',
} as const

/**
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000011)
 */
export const sequencerFeeVaultConfig = {
  address: sequencerFeeVaultAddress,
  abi: sequencerFeeVaultABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SystemConfig
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60)
 */
export const systemConfigABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'version',
        internalType: 'uint256',
        type: 'uint256',
        indexed: true,
      },
      {
        name: 'updateType',
        internalType: 'enum SystemConfig.UpdateType',
        type: 'uint8',
        indexed: true,
      },
      { name: 'data', internalType: 'bytes', type: 'bytes', indexed: false },
    ],
    name: 'ConfigUpdate',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'BATCH_INBOX_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'L1_CROSS_DOMAIN_MESSENGER_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'L1_ERC_721_BRIDGE_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'L1_STANDARD_BRIDGE_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'L2_OUTPUT_ORACLE_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'OPTIMISM_PORTAL_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'UNSAFE_BLOCK_SIGNER_SLOT',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'batchInbox',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'batcherHash',
    outputs: [{ name: '', internalType: 'bytes32', type: 'bytes32' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'gasLimit',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_owner', internalType: 'address', type: 'address' },
      { name: '_overhead', internalType: 'uint256', type: 'uint256' },
      { name: '_scalar', internalType: 'uint256', type: 'uint256' },
      { name: '_batcherHash', internalType: 'bytes32', type: 'bytes32' },
      { name: '_gasLimit', internalType: 'uint64', type: 'uint64' },
      { name: '_unsafeBlockSigner', internalType: 'address', type: 'address' },
      {
        name: '_config',
        internalType: 'struct ResourceMetering.ResourceConfig',
        type: 'tuple',
        components: [
          { name: 'maxResourceLimit', internalType: 'uint32', type: 'uint32' },
          {
            name: 'elasticityMultiplier',
            internalType: 'uint8',
            type: 'uint8',
          },
          {
            name: 'baseFeeMaxChangeDenominator',
            internalType: 'uint8',
            type: 'uint8',
          },
          { name: 'minimumBaseFee', internalType: 'uint32', type: 'uint32' },
          { name: 'systemTxMaxGas', internalType: 'uint32', type: 'uint32' },
          { name: 'maximumBaseFee', internalType: 'uint128', type: 'uint128' },
        ],
      },
      { name: '_startBlock', internalType: 'uint256', type: 'uint256' },
      { name: '_batchInbox', internalType: 'address', type: 'address' },
      {
        name: '_addresses',
        internalType: 'struct SystemConfig.Addresses',
        type: 'tuple',
        components: [
          {
            name: 'l1CrossDomainMessenger',
            internalType: 'address',
            type: 'address',
          },
          { name: 'l1ERC721Bridge', internalType: 'address', type: 'address' },
          {
            name: 'l1StandardBridge',
            internalType: 'address',
            type: 'address',
          },
          { name: 'l2OutputOracle', internalType: 'address', type: 'address' },
          { name: 'optimismPortal', internalType: 'address', type: 'address' },
          {
            name: 'optimismMintableERC20Factory',
            internalType: 'address',
            type: 'address',
          },
        ],
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1CrossDomainMessenger',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1ERC721Bridge',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l1StandardBridge',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2OutputOracle',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'minimumGasLimit',
    outputs: [{ name: '', internalType: 'uint64', type: 'uint64' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'optimismMintableERC20Factory',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'optimismPortal',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'overhead',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'resourceConfig',
    outputs: [
      {
        name: '',
        internalType: 'struct ResourceMetering.ResourceConfig',
        type: 'tuple',
        components: [
          { name: 'maxResourceLimit', internalType: 'uint32', type: 'uint32' },
          {
            name: 'elasticityMultiplier',
            internalType: 'uint8',
            type: 'uint8',
          },
          {
            name: 'baseFeeMaxChangeDenominator',
            internalType: 'uint8',
            type: 'uint8',
          },
          { name: 'minimumBaseFee', internalType: 'uint32', type: 'uint32' },
          { name: 'systemTxMaxGas', internalType: 'uint32', type: 'uint32' },
          { name: 'maximumBaseFee', internalType: 'uint128', type: 'uint128' },
        ],
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'scalar',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_batcherHash', internalType: 'bytes32', type: 'bytes32' },
    ],
    name: 'setBatcherHash',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_overhead', internalType: 'uint256', type: 'uint256' },
      { name: '_scalar', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'setGasConfig',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_gasLimit', internalType: 'uint64', type: 'uint64' }],
    name: 'setGasLimit',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_config',
        internalType: 'struct ResourceMetering.ResourceConfig',
        type: 'tuple',
        components: [
          { name: 'maxResourceLimit', internalType: 'uint32', type: 'uint32' },
          {
            name: 'elasticityMultiplier',
            internalType: 'uint8',
            type: 'uint8',
          },
          {
            name: 'baseFeeMaxChangeDenominator',
            internalType: 'uint8',
            type: 'uint8',
          },
          { name: 'minimumBaseFee', internalType: 'uint32', type: 'uint32' },
          { name: 'systemTxMaxGas', internalType: 'uint32', type: 'uint32' },
          { name: 'maximumBaseFee', internalType: 'uint128', type: 'uint128' },
        ],
      },
    ],
    name: 'setResourceConfig',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_unsafeBlockSigner', internalType: 'address', type: 'address' },
    ],
    name: 'setUnsafeBlockSigner',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'startBlock',
    outputs: [{ name: '', internalType: 'uint256', type: 'uint256' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'unsafeBlockSigner',
    outputs: [{ name: 'addr_', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'version',
    outputs: [{ name: '', internalType: 'string', type: 'string' }],
  },
] as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60)
 */
export const systemConfigAddress = {
  1: '0x229047fed2591dbec1eF1118d64F7aF3dB9EB290',
  5: '0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60',
} as const

/**
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60)
 */
export const systemConfigConfig = {
  address: systemConfigAddress,
  abi: systemConfigABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SystemDictator
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB)
 */
export const systemDictatorABI = [
  { stateMutability: 'nonpayable', type: 'constructor', inputs: [] },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'EXIT_1_NO_RETURN_STEP',
    outputs: [{ name: '', internalType: 'uint8', type: 'uint8' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'PROXY_TRANSFER_STEP',
    outputs: [{ name: '', internalType: 'uint8', type: 'uint8' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'config',
    outputs: [
      {
        name: 'globalConfig',
        internalType: 'struct SystemDictator.GlobalConfig',
        type: 'tuple',
        components: [
          {
            name: 'addressManager',
            internalType: 'contract AddressManager',
            type: 'address',
          },
          {
            name: 'proxyAdmin',
            internalType: 'contract ProxyAdmin',
            type: 'address',
          },
          { name: 'controller', internalType: 'address', type: 'address' },
          { name: 'finalOwner', internalType: 'address', type: 'address' },
        ],
      },
      {
        name: 'proxyAddressConfig',
        internalType: 'struct SystemDictator.ProxyAddressConfig',
        type: 'tuple',
        components: [
          {
            name: 'l2OutputOracleProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'optimismPortalProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'l1CrossDomainMessengerProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'l1StandardBridgeProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'optimismMintableERC20FactoryProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'l1ERC721BridgeProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'systemConfigProxy',
            internalType: 'address',
            type: 'address',
          },
        ],
      },
      {
        name: 'implementationAddressConfig',
        internalType: 'struct SystemDictator.ImplementationAddressConfig',
        type: 'tuple',
        components: [
          {
            name: 'l2OutputOracleImpl',
            internalType: 'contract L2OutputOracle',
            type: 'address',
          },
          {
            name: 'optimismPortalImpl',
            internalType: 'contract OptimismPortal',
            type: 'address',
          },
          {
            name: 'l1CrossDomainMessengerImpl',
            internalType: 'contract L1CrossDomainMessenger',
            type: 'address',
          },
          {
            name: 'l1StandardBridgeImpl',
            internalType: 'contract L1StandardBridge',
            type: 'address',
          },
          {
            name: 'optimismMintableERC20FactoryImpl',
            internalType: 'contract OptimismMintableERC20Factory',
            type: 'address',
          },
          {
            name: 'l1ERC721BridgeImpl',
            internalType: 'contract L1ERC721Bridge',
            type: 'address',
          },
          {
            name: 'portalSenderImpl',
            internalType: 'contract PortalSender',
            type: 'address',
          },
          {
            name: 'systemConfigImpl',
            internalType: 'contract SystemConfig',
            type: 'address',
          },
        ],
      },
      {
        name: 'systemConfigConfig',
        internalType: 'struct SystemDictator.SystemConfigConfig',
        type: 'tuple',
        components: [
          { name: 'owner', internalType: 'address', type: 'address' },
          { name: 'overhead', internalType: 'uint256', type: 'uint256' },
          { name: 'scalar', internalType: 'uint256', type: 'uint256' },
          { name: 'batcherHash', internalType: 'bytes32', type: 'bytes32' },
          { name: 'gasLimit', internalType: 'uint64', type: 'uint64' },
          {
            name: 'unsafeBlockSigner',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'resourceConfig',
            internalType: 'struct ResourceMetering.ResourceConfig',
            type: 'tuple',
            components: [
              {
                name: 'maxResourceLimit',
                internalType: 'uint32',
                type: 'uint32',
              },
              {
                name: 'elasticityMultiplier',
                internalType: 'uint8',
                type: 'uint8',
              },
              {
                name: 'baseFeeMaxChangeDenominator',
                internalType: 'uint8',
                type: 'uint8',
              },
              {
                name: 'minimumBaseFee',
                internalType: 'uint32',
                type: 'uint32',
              },
              {
                name: 'systemTxMaxGas',
                internalType: 'uint32',
                type: 'uint32',
              },
              {
                name: 'maximumBaseFee',
                internalType: 'uint128',
                type: 'uint128',
              },
            ],
          },
        ],
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'currentStep',
    outputs: [{ name: '', internalType: 'uint8', type: 'uint8' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'dynamicConfigSet',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'exit1',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'exited',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'finalize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'finalized',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_config',
        internalType: 'struct SystemDictator.DeployConfig',
        type: 'tuple',
        components: [
          {
            name: 'globalConfig',
            internalType: 'struct SystemDictator.GlobalConfig',
            type: 'tuple',
            components: [
              {
                name: 'addressManager',
                internalType: 'contract AddressManager',
                type: 'address',
              },
              {
                name: 'proxyAdmin',
                internalType: 'contract ProxyAdmin',
                type: 'address',
              },
              { name: 'controller', internalType: 'address', type: 'address' },
              { name: 'finalOwner', internalType: 'address', type: 'address' },
            ],
          },
          {
            name: 'proxyAddressConfig',
            internalType: 'struct SystemDictator.ProxyAddressConfig',
            type: 'tuple',
            components: [
              {
                name: 'l2OutputOracleProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'optimismPortalProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'l1CrossDomainMessengerProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'l1StandardBridgeProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'optimismMintableERC20FactoryProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'l1ERC721BridgeProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'systemConfigProxy',
                internalType: 'address',
                type: 'address',
              },
            ],
          },
          {
            name: 'implementationAddressConfig',
            internalType: 'struct SystemDictator.ImplementationAddressConfig',
            type: 'tuple',
            components: [
              {
                name: 'l2OutputOracleImpl',
                internalType: 'contract L2OutputOracle',
                type: 'address',
              },
              {
                name: 'optimismPortalImpl',
                internalType: 'contract OptimismPortal',
                type: 'address',
              },
              {
                name: 'l1CrossDomainMessengerImpl',
                internalType: 'contract L1CrossDomainMessenger',
                type: 'address',
              },
              {
                name: 'l1StandardBridgeImpl',
                internalType: 'contract L1StandardBridge',
                type: 'address',
              },
              {
                name: 'optimismMintableERC20FactoryImpl',
                internalType: 'contract OptimismMintableERC20Factory',
                type: 'address',
              },
              {
                name: 'l1ERC721BridgeImpl',
                internalType: 'contract L1ERC721Bridge',
                type: 'address',
              },
              {
                name: 'portalSenderImpl',
                internalType: 'contract PortalSender',
                type: 'address',
              },
              {
                name: 'systemConfigImpl',
                internalType: 'contract SystemConfig',
                type: 'address',
              },
            ],
          },
          {
            name: 'systemConfigConfig',
            internalType: 'struct SystemDictator.SystemConfigConfig',
            type: 'tuple',
            components: [
              { name: 'owner', internalType: 'address', type: 'address' },
              { name: 'overhead', internalType: 'uint256', type: 'uint256' },
              { name: 'scalar', internalType: 'uint256', type: 'uint256' },
              { name: 'batcherHash', internalType: 'bytes32', type: 'bytes32' },
              { name: 'gasLimit', internalType: 'uint64', type: 'uint64' },
              {
                name: 'unsafeBlockSigner',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'resourceConfig',
                internalType: 'struct ResourceMetering.ResourceConfig',
                type: 'tuple',
                components: [
                  {
                    name: 'maxResourceLimit',
                    internalType: 'uint32',
                    type: 'uint32',
                  },
                  {
                    name: 'elasticityMultiplier',
                    internalType: 'uint8',
                    type: 'uint8',
                  },
                  {
                    name: 'baseFeeMaxChangeDenominator',
                    internalType: 'uint8',
                    type: 'uint8',
                  },
                  {
                    name: 'minimumBaseFee',
                    internalType: 'uint32',
                    type: 'uint32',
                  },
                  {
                    name: 'systemTxMaxGas',
                    internalType: 'uint32',
                    type: 'uint32',
                  },
                  {
                    name: 'maximumBaseFee',
                    internalType: 'uint128',
                    type: 'uint128',
                  },
                ],
              },
            ],
          },
        ],
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2OutputOracleDynamicConfig',
    outputs: [
      {
        name: 'l2OutputOracleStartingBlockNumber',
        internalType: 'uint256',
        type: 'uint256',
      },
      {
        name: 'l2OutputOracleStartingTimestamp',
        internalType: 'uint256',
        type: 'uint256',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'oldL1CrossDomainMessenger',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'optimismPortalDynamicConfig',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'phase1',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'phase2',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step1',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step2',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step3',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step4',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step5',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_l2OutputOracleDynamicConfig',
        internalType: 'struct SystemDictator.L2OutputOracleDynamicConfig',
        type: 'tuple',
        components: [
          {
            name: 'l2OutputOracleStartingBlockNumber',
            internalType: 'uint256',
            type: 'uint256',
          },
          {
            name: 'l2OutputOracleStartingTimestamp',
            internalType: 'uint256',
            type: 'uint256',
          },
        ],
      },
      {
        name: '_optimismPortalDynamicConfig',
        internalType: 'bool',
        type: 'bool',
      },
    ],
    name: 'updateDynamicConfig',
    outputs: [],
  },
] as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB)
 */
export const systemDictatorAddress = {
  1: '0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB',
} as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB)
 */
export const systemDictatorConfig = {
  address: systemDictatorAddress,
  abi: systemDictatorABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SystemDictator_goerli
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971)
 */
export const systemDictatorGoerliABI = [
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'version', internalType: 'uint8', type: 'uint8', indexed: false },
    ],
    name: 'Initialized',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'previousOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnershipTransferred',
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'EXIT_1_NO_RETURN_STEP',
    outputs: [{ name: '', internalType: 'uint8', type: 'uint8' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'PROXY_TRANSFER_STEP',
    outputs: [{ name: '', internalType: 'uint8', type: 'uint8' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'config',
    outputs: [
      {
        name: 'globalConfig',
        internalType: 'struct SystemDictator.GlobalConfig',
        type: 'tuple',
        components: [
          {
            name: 'addressManager',
            internalType: 'contract AddressManager',
            type: 'address',
          },
          {
            name: 'proxyAdmin',
            internalType: 'contract ProxyAdmin',
            type: 'address',
          },
          { name: 'controller', internalType: 'address', type: 'address' },
          { name: 'finalOwner', internalType: 'address', type: 'address' },
        ],
      },
      {
        name: 'proxyAddressConfig',
        internalType: 'struct SystemDictator.ProxyAddressConfig',
        type: 'tuple',
        components: [
          {
            name: 'l2OutputOracleProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'optimismPortalProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'l1CrossDomainMessengerProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'l1StandardBridgeProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'optimismMintableERC20FactoryProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'l1ERC721BridgeProxy',
            internalType: 'address',
            type: 'address',
          },
          {
            name: 'systemConfigProxy',
            internalType: 'address',
            type: 'address',
          },
        ],
      },
      {
        name: 'implementationAddressConfig',
        internalType: 'struct SystemDictator.ImplementationAddressConfig',
        type: 'tuple',
        components: [
          {
            name: 'l2OutputOracleImpl',
            internalType: 'contract L2OutputOracle',
            type: 'address',
          },
          {
            name: 'optimismPortalImpl',
            internalType: 'contract OptimismPortal',
            type: 'address',
          },
          {
            name: 'l1CrossDomainMessengerImpl',
            internalType: 'contract L1CrossDomainMessenger',
            type: 'address',
          },
          {
            name: 'l1StandardBridgeImpl',
            internalType: 'contract L1StandardBridge',
            type: 'address',
          },
          {
            name: 'optimismMintableERC20FactoryImpl',
            internalType: 'contract OptimismMintableERC20Factory',
            type: 'address',
          },
          {
            name: 'l1ERC721BridgeImpl',
            internalType: 'contract L1ERC721Bridge',
            type: 'address',
          },
          {
            name: 'portalSenderImpl',
            internalType: 'contract PortalSender',
            type: 'address',
          },
          {
            name: 'systemConfigImpl',
            internalType: 'contract SystemConfig',
            type: 'address',
          },
        ],
      },
      {
        name: 'systemConfigConfig',
        internalType: 'struct SystemDictator.SystemConfigConfig',
        type: 'tuple',
        components: [
          { name: 'owner', internalType: 'address', type: 'address' },
          { name: 'overhead', internalType: 'uint256', type: 'uint256' },
          { name: 'scalar', internalType: 'uint256', type: 'uint256' },
          { name: 'batcherHash', internalType: 'bytes32', type: 'bytes32' },
          { name: 'gasLimit', internalType: 'uint64', type: 'uint64' },
          {
            name: 'unsafeBlockSigner',
            internalType: 'address',
            type: 'address',
          },
        ],
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'currentStep',
    outputs: [{ name: '', internalType: 'uint8', type: 'uint8' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'dynamicConfigSet',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'exit1',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'finalize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'finalized',
    outputs: [{ name: '', internalType: 'bool', type: 'bool' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_config',
        internalType: 'struct SystemDictator.DeployConfig',
        type: 'tuple',
        components: [
          {
            name: 'globalConfig',
            internalType: 'struct SystemDictator.GlobalConfig',
            type: 'tuple',
            components: [
              {
                name: 'addressManager',
                internalType: 'contract AddressManager',
                type: 'address',
              },
              {
                name: 'proxyAdmin',
                internalType: 'contract ProxyAdmin',
                type: 'address',
              },
              { name: 'controller', internalType: 'address', type: 'address' },
              { name: 'finalOwner', internalType: 'address', type: 'address' },
            ],
          },
          {
            name: 'proxyAddressConfig',
            internalType: 'struct SystemDictator.ProxyAddressConfig',
            type: 'tuple',
            components: [
              {
                name: 'l2OutputOracleProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'optimismPortalProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'l1CrossDomainMessengerProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'l1StandardBridgeProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'optimismMintableERC20FactoryProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'l1ERC721BridgeProxy',
                internalType: 'address',
                type: 'address',
              },
              {
                name: 'systemConfigProxy',
                internalType: 'address',
                type: 'address',
              },
            ],
          },
          {
            name: 'implementationAddressConfig',
            internalType: 'struct SystemDictator.ImplementationAddressConfig',
            type: 'tuple',
            components: [
              {
                name: 'l2OutputOracleImpl',
                internalType: 'contract L2OutputOracle',
                type: 'address',
              },
              {
                name: 'optimismPortalImpl',
                internalType: 'contract OptimismPortal',
                type: 'address',
              },
              {
                name: 'l1CrossDomainMessengerImpl',
                internalType: 'contract L1CrossDomainMessenger',
                type: 'address',
              },
              {
                name: 'l1StandardBridgeImpl',
                internalType: 'contract L1StandardBridge',
                type: 'address',
              },
              {
                name: 'optimismMintableERC20FactoryImpl',
                internalType: 'contract OptimismMintableERC20Factory',
                type: 'address',
              },
              {
                name: 'l1ERC721BridgeImpl',
                internalType: 'contract L1ERC721Bridge',
                type: 'address',
              },
              {
                name: 'portalSenderImpl',
                internalType: 'contract PortalSender',
                type: 'address',
              },
              {
                name: 'systemConfigImpl',
                internalType: 'contract SystemConfig',
                type: 'address',
              },
            ],
          },
          {
            name: 'systemConfigConfig',
            internalType: 'struct SystemDictator.SystemConfigConfig',
            type: 'tuple',
            components: [
              { name: 'owner', internalType: 'address', type: 'address' },
              { name: 'overhead', internalType: 'uint256', type: 'uint256' },
              { name: 'scalar', internalType: 'uint256', type: 'uint256' },
              { name: 'batcherHash', internalType: 'bytes32', type: 'bytes32' },
              { name: 'gasLimit', internalType: 'uint64', type: 'uint64' },
              {
                name: 'unsafeBlockSigner',
                internalType: 'address',
                type: 'address',
              },
            ],
          },
        ],
      },
    ],
    name: 'initialize',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'l2OutputOracleDynamicConfig',
    outputs: [
      {
        name: 'l2OutputOracleStartingBlockNumber',
        internalType: 'uint256',
        type: 'uint256',
      },
      {
        name: 'l2OutputOracleStartingTimestamp',
        internalType: 'uint256',
        type: 'uint256',
      },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'oldL1CrossDomainMessenger',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'renounceOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step1',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step2',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step3',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step4',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step5',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'step6',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'transferOwnership',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      {
        name: '_l2OutputOracleDynamicConfig',
        internalType: 'struct SystemDictator.L2OutputOracleDynamicConfig',
        type: 'tuple',
        components: [
          {
            name: 'l2OutputOracleStartingBlockNumber',
            internalType: 'uint256',
            type: 'uint256',
          },
          {
            name: 'l2OutputOracleStartingTimestamp',
            internalType: 'uint256',
            type: 'uint256',
          },
        ],
      },
    ],
    name: 'updateL2OutputOracleDynamicConfig',
    outputs: [],
  },
] as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971)
 */
export const systemDictatorGoerliAddress = {
  5: '0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971',
} as const

/**
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971)
 */
export const systemDictatorGoerliConfig = {
  address: systemDictatorGoerliAddress,
  abi: systemDictatorGoerliABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// TeleportrWithdrawer
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x78A25524D90E3D0596558fb43789bD800a5c3007)
 */
export const teleportrWithdrawerABI = [
  {
    stateMutability: 'nonpayable',
    type: 'constructor',
    inputs: [{ name: '_owner', internalType: 'address', type: 'address' }],
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'user', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'newOwner',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
    ],
    name: 'OwnerUpdated',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      { name: 'from', internalType: 'address', type: 'address', indexed: true },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'ReceivedETH',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewERC20',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'asset',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      { name: 'id', internalType: 'uint256', type: 'uint256', indexed: false },
    ],
    name: 'WithdrewERC721',
  },
  {
    type: 'event',
    anonymous: false,
    inputs: [
      {
        name: 'withdrawer',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'recipient',
        internalType: 'address',
        type: 'address',
        indexed: true,
      },
      {
        name: 'amount',
        internalType: 'uint256',
        type: 'uint256',
        indexed: false,
      },
    ],
    name: 'WithdrewETH',
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_gas', internalType: 'uint256', type: 'uint256' },
      { name: '_value', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'CALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'payable',
    type: 'function',
    inputs: [
      { name: '_target', internalType: 'address', type: 'address' },
      { name: '_data', internalType: 'bytes', type: 'bytes' },
      { name: '_gas', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'DELEGATECALL',
    outputs: [
      { name: '', internalType: 'bool', type: 'bool' },
      { name: '', internalType: 'bytes', type: 'bytes' },
    ],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'data',
    outputs: [{ name: '', internalType: 'bytes', type: 'bytes' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'owner',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'recipient',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_data', internalType: 'bytes', type: 'bytes' }],
    name: 'setData',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: 'newOwner', internalType: 'address', type: 'address' }],
    name: 'setOwner',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_recipient', internalType: 'address', type: 'address' }],
    name: 'setRecipient',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_teleportr', internalType: 'address', type: 'address' }],
    name: 'setTeleportr',
    outputs: [],
  },
  {
    stateMutability: 'view',
    type: 'function',
    inputs: [],
    name: 'teleportr',
    outputs: [{ name: '', internalType: 'address', type: 'address' }],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC20', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
    ],
    name: 'withdrawERC20',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_asset', internalType: 'contract ERC721', type: 'address' },
      { name: '_to', internalType: 'address', type: 'address' },
      { name: '_id', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawERC721',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [
      { name: '_to', internalType: 'address payable', type: 'address' },
      { name: '_amount', internalType: 'uint256', type: 'uint256' },
    ],
    name: 'withdrawETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [{ name: '_to', internalType: 'address payable', type: 'address' }],
    name: 'withdrawETH',
    outputs: [],
  },
  {
    stateMutability: 'nonpayable',
    type: 'function',
    inputs: [],
    name: 'withdrawFromTeleportr',
    outputs: [],
  },
  { stateMutability: 'payable', type: 'receive' },
] as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x78A25524D90E3D0596558fb43789bD800a5c3007)
 */
export const teleportrWithdrawerAddress = {
  1: '0x78A25524D90E3D0596558fb43789bD800a5c3007',
} as const

/**
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x78A25524D90E3D0596558fb43789bD800a5c3007)
 */
export const teleportrWithdrawerConfig = {
  address: teleportrWithdrawerAddress,
  abi: teleportrWithdrawerABI,
} as const

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// eslintIgnore
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// React
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link addressManagerABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111)
 */
export function useAddressManagerRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof addressManagerABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof addressManagerABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof addressManagerAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: addressManagerABI,
    address:
      addressManagerAddress[chainId as keyof typeof addressManagerAddress],
    ...config,
  } as UseContractReadConfig<typeof addressManagerABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link addressManagerABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111)
 */
export function useAddressManagerWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof addressManagerAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof addressManagerABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof addressManagerABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof addressManagerABI, TFunctionName, TMode>({
    abi: addressManagerABI,
    address:
      addressManagerAddress[chainId as keyof typeof addressManagerAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link addressManagerABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdE1FCfB0851916CA5101820A69b13a4E276bd81F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xa6f73589243a6A7a9023b1Fa0651b1d89c177111)
 */
export function useAddressManagerEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof addressManagerABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof addressManagerAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: addressManagerABI,
    address:
      addressManagerAddress[chainId as keyof typeof addressManagerAddress],
    ...config,
  } as UseContractEventConfig<typeof addressManagerABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link assetReceiverABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 */
export function useAssetReceiverRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof assetReceiverABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof assetReceiverABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof assetReceiverAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: assetReceiverABI,
    address: assetReceiverAddress[chainId as keyof typeof assetReceiverAddress],
    ...config,
  } as UseContractReadConfig<typeof assetReceiverABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link assetReceiverABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 */
export function useAssetReceiverWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof assetReceiverAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof assetReceiverABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof assetReceiverABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof assetReceiverABI, TFunctionName, TMode>({
    abi: assetReceiverABI,
    address: assetReceiverAddress[chainId as keyof typeof assetReceiverAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link assetReceiverABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x15DdA60616Ffca20371ED1659dBB78E888f65556)
 */
export function useAssetReceiverEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof assetReceiverABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof assetReceiverAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: assetReceiverABI,
    address: assetReceiverAddress[chainId as keyof typeof assetReceiverAddress],
    ...config,
  } as UseContractEventConfig<typeof assetReceiverABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link attestationStationABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 */
export function useAttestationStationRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof attestationStationABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof attestationStationABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof attestationStationAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: attestationStationABI,
    address:
      attestationStationAddress[
        chainId as keyof typeof attestationStationAddress
      ],
    ...config,
  } as UseContractReadConfig<typeof attestationStationABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link attestationStationABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 */
export function useAttestationStationWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof attestationStationAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof attestationStationABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof attestationStationABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof attestationStationABI, TFunctionName, TMode>({
    abi: attestationStationABI,
    address:
      attestationStationAddress[
        chainId as keyof typeof attestationStationAddress
      ],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link attestationStationABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77)
 */
export function useAttestationStationEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof attestationStationABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof attestationStationAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: attestationStationABI,
    address:
      attestationStationAddress[
        chainId as keyof typeof attestationStationAddress
      ],
    ...config,
  } as UseContractEventConfig<typeof attestationStationABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link baseFeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000019)
 */
export function useBaseFeeVaultRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof baseFeeVaultABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof baseFeeVaultABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof baseFeeVaultAddress } = {} as any
) {
  return useContractRead({
    abi: baseFeeVaultABI,
    address: baseFeeVaultAddress[420],
    ...config,
  } as UseContractReadConfig<typeof baseFeeVaultABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link baseFeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000019)
 */
export function useBaseFeeVaultWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof baseFeeVaultAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof baseFeeVaultABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof baseFeeVaultABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof baseFeeVaultABI, TFunctionName, TMode>({
    abi: baseFeeVaultABI,
    address: baseFeeVaultAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link baseFeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000019)
 */
export function useBaseFeeVaultEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof baseFeeVaultABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof baseFeeVaultAddress } = {} as any
) {
  return useContractEvent({
    abi: baseFeeVaultABI,
    address: baseFeeVaultAddress[420],
    ...config,
  } as UseContractEventConfig<typeof baseFeeVaultABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link checkBalanceHighABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5d7103853f12109A7d27F118e54BbC654ad847E9)
 */
export function useCheckBalanceHighRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof checkBalanceHighABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof checkBalanceHighABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkBalanceHighAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: checkBalanceHighABI,
    address:
      checkBalanceHighAddress[chainId as keyof typeof checkBalanceHighAddress],
    ...config,
  } as UseContractReadConfig<typeof checkBalanceHighABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link checkBalanceHighABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x7eC64a8a591bFf829ff6C8be76074D540ACb813F)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5d7103853f12109A7d27F118e54BbC654ad847E9)
 */
export function useCheckBalanceHighEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof checkBalanceHighABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkBalanceHighAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: checkBalanceHighABI,
    address:
      checkBalanceHighAddress[chainId as keyof typeof checkBalanceHighAddress],
    ...config,
  } as UseContractEventConfig<typeof checkBalanceHighABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link checkBalanceLowABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x7Ce13D154FAEE5C8B3E6b19d4Add16f21d884474)
 */
export function useCheckBalanceLowRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof checkBalanceLowABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof checkBalanceLowABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkBalanceLowAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: checkBalanceLowABI,
    address:
      checkBalanceLowAddress[chainId as keyof typeof checkBalanceLowAddress],
    ...config,
  } as UseContractReadConfig<typeof checkBalanceLowABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link checkBalanceLowABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x381a4eFC2A2C914eA1889722bB4B44Fa6BD5b640)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x7Ce13D154FAEE5C8B3E6b19d4Add16f21d884474)
 */
export function useCheckBalanceLowEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof checkBalanceLowABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkBalanceLowAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: checkBalanceLowABI,
    address:
      checkBalanceLowAddress[chainId as keyof typeof checkBalanceLowAddress],
    ...config,
  } as UseContractEventConfig<typeof checkBalanceLowABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link checkGelatoLowABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xF9c8a4Cb4021f57F9f6d69799cA9BefF64524862)
 */
export function useCheckGelatoLowRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof checkGelatoLowABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof checkGelatoLowABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkGelatoLowAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: checkGelatoLowABI,
    address:
      checkGelatoLowAddress[chainId as keyof typeof checkGelatoLowAddress],
    ...config,
  } as UseContractReadConfig<typeof checkGelatoLowABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link checkGelatoLowABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4f7CFc43f6D262a085F3b946cAC69E7a8E39BBAa)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0xF9c8a4Cb4021f57F9f6d69799cA9BefF64524862)
 */
export function useCheckGelatoLowEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof checkGelatoLowABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkGelatoLowAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: checkGelatoLowABI,
    address:
      checkGelatoLowAddress[chainId as keyof typeof checkGelatoLowAddress],
    ...config,
  } as UseContractEventConfig<typeof checkGelatoLowABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link checkTrueABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5c741a38cb11424711231777D71689C458eE835D)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x47443D0C184e022F19BD1578F5bca6B8a9F58E32)
 */
export function useCheckTrueRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof checkTrueABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof checkTrueABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof checkTrueAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: checkTrueABI,
    address: checkTrueAddress[chainId as keyof typeof checkTrueAddress],
    ...config,
  } as UseContractReadConfig<typeof checkTrueABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link drippieABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export function useDrippieRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof drippieABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof drippieABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof drippieAddress } = {} as any
) {
  return useContractRead({
    abi: drippieABI,
    address: drippieAddress[1],
    ...config,
  } as UseContractReadConfig<typeof drippieABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link drippieABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export function useDrippieWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof drippieAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<typeof drippieABI, string>['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof drippieABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof drippieABI, TFunctionName, TMode>({
    abi: drippieABI,
    address: drippieAddress[1],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link drippieABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export function useDrippieEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof drippieABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof drippieAddress } = {} as any
) {
  return useContractEvent({
    abi: drippieABI,
    address: drippieAddress[1],
    ...config,
  } as UseContractEventConfig<typeof drippieABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link drippieGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export function useDrippieGoerliRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof drippieGoerliABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof drippieGoerliABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof drippieGoerliAddress } = {} as any
) {
  return useContractRead({
    abi: drippieGoerliABI,
    address: drippieGoerliAddress[5],
    ...config,
  } as UseContractReadConfig<typeof drippieGoerliABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link drippieGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export function useDrippieGoerliWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof drippieGoerliAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof drippieGoerliABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof drippieGoerliABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof drippieGoerliABI, TFunctionName, TMode>({
    abi: drippieGoerliABI,
    address: drippieGoerliAddress[5],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link drippieGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x44b3A2a040057eBafC601A78647e805fd58B1f50)
 */
export function useDrippieGoerliEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof drippieGoerliABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof drippieGoerliAddress } = {} as any
) {
  return useContractEvent({
    abi: drippieGoerliABI,
    address: drippieGoerliAddress[5],
    ...config,
  } as UseContractEventConfig<typeof drippieGoerliABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link drippieOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x8D8d533C16D23847EB04EEB0925be8900Dd3af86)
 */
export function useDrippieOptimismGoerliRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof drippieOptimismGoerliABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof drippieOptimismGoerliABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof drippieOptimismGoerliAddress } = {} as any
) {
  return useContractRead({
    abi: drippieOptimismGoerliABI,
    address: drippieOptimismGoerliAddress[420],
    ...config,
  } as UseContractReadConfig<typeof drippieOptimismGoerliABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link drippieOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x8D8d533C16D23847EB04EEB0925be8900Dd3af86)
 */
export function useDrippieOptimismGoerliWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof drippieOptimismGoerliAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof drippieOptimismGoerliABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof drippieOptimismGoerliABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof drippieOptimismGoerliABI,
    TFunctionName,
    TMode
  >({
    abi: drippieOptimismGoerliABI,
    address: drippieOptimismGoerliAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link drippieOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x8D8d533C16D23847EB04EEB0925be8900Dd3af86)
 */
export function useDrippieOptimismGoerliEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof drippieOptimismGoerliABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof drippieOptimismGoerliAddress } = {} as any
) {
  return useContractEvent({
    abi: drippieOptimismGoerliABI,
    address: drippieOptimismGoerliAddress[420],
    ...config,
  } as UseContractEventConfig<typeof drippieOptimismGoerliABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link easABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5A633F1cc84B03F7588486CF2F386c102061E6e1)
 */
export function useEasRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof easABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof easABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof easAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: easABI,
    address: easAddress[chainId as keyof typeof easAddress],
    ...config,
  } as UseContractReadConfig<typeof easABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link easABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5A633F1cc84B03F7588486CF2F386c102061E6e1)
 */
export function useEasWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof easAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<typeof easABI, string>['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof easABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof easABI, TFunctionName, TMode>({
    abi: easABI,
    address: easAddress[chainId as keyof typeof easAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link easABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4E0275Ea5a89e7a3c1B58411379D1a0eDdc5b088)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x5A633F1cc84B03F7588486CF2F386c102061E6e1)
 */
export function useEasEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof easABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof easAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: easABI,
    address: easAddress[chainId as keyof typeof easAddress],
    ...config,
  } as UseContractEventConfig<typeof easABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link gasPriceOracleABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000000F)
 */
export function useGasPriceOracleRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof gasPriceOracleABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof gasPriceOracleABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof gasPriceOracleAddress } = {} as any
) {
  return useContractRead({
    abi: gasPriceOracleABI,
    address: gasPriceOracleAddress[420],
    ...config,
  } as UseContractReadConfig<typeof gasPriceOracleABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l1BlockABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000015)
 */
export function useL1BlockRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l1BlockABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof l1BlockABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1BlockAddress } = {} as any
) {
  return useContractRead({
    abi: l1BlockABI,
    address: l1BlockAddress[420],
    ...config,
  } as UseContractReadConfig<typeof l1BlockABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l1BlockABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000015)
 */
export function useL1BlockWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l1BlockAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<typeof l1BlockABI, string>['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof l1BlockABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof l1BlockABI, TFunctionName, TMode>({
    abi: l1BlockABI,
    address: l1BlockAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l1CrossDomainMessengerABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294)
 */
export function useL1CrossDomainMessengerRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof l1CrossDomainMessengerABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof l1CrossDomainMessengerABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1CrossDomainMessengerAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: l1CrossDomainMessengerABI,
    address:
      l1CrossDomainMessengerAddress[
        chainId as keyof typeof l1CrossDomainMessengerAddress
      ],
    ...config,
  } as UseContractReadConfig<typeof l1CrossDomainMessengerABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l1CrossDomainMessengerABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294)
 */
export function useL1CrossDomainMessengerWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l1CrossDomainMessengerAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l1CrossDomainMessengerABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof l1CrossDomainMessengerABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<
    typeof l1CrossDomainMessengerABI,
    TFunctionName,
    TMode
  >({
    abi: l1CrossDomainMessengerABI,
    address:
      l1CrossDomainMessengerAddress[
        chainId as keyof typeof l1CrossDomainMessengerAddress
      ],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l1CrossDomainMessengerABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5086d1eEF304eb5284A0f6720f79403b4e9bE294)
 */
export function useL1CrossDomainMessengerEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l1CrossDomainMessengerABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1CrossDomainMessengerAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: l1CrossDomainMessengerABI,
    address:
      l1CrossDomainMessengerAddress[
        chainId as keyof typeof l1CrossDomainMessengerAddress
      ],
    ...config,
  } as UseContractEventConfig<typeof l1CrossDomainMessengerABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l1Erc721BridgeABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5a7749f83b81B301cAb5f48EB8516B986DAef23D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9)
 */
export function useL1Erc721BridgeRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l1Erc721BridgeABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof l1Erc721BridgeABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1Erc721BridgeAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: l1Erc721BridgeABI,
    address:
      l1Erc721BridgeAddress[chainId as keyof typeof l1Erc721BridgeAddress],
    ...config,
  } as UseContractReadConfig<typeof l1Erc721BridgeABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l1Erc721BridgeABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5a7749f83b81B301cAb5f48EB8516B986DAef23D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9)
 */
export function useL1Erc721BridgeWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l1Erc721BridgeAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l1Erc721BridgeABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof l1Erc721BridgeABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof l1Erc721BridgeABI, TFunctionName, TMode>({
    abi: l1Erc721BridgeABI,
    address:
      l1Erc721BridgeAddress[chainId as keyof typeof l1Erc721BridgeAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l1Erc721BridgeABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x5a7749f83b81B301cAb5f48EB8516B986DAef23D)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9)
 */
export function useL1Erc721BridgeEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l1Erc721BridgeABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1Erc721BridgeAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: l1Erc721BridgeABI,
    address:
      l1Erc721BridgeAddress[chainId as keyof typeof l1Erc721BridgeAddress],
    ...config,
  } as UseContractEventConfig<typeof l1Erc721BridgeABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l1FeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000001a)
 */
export function useL1FeeVaultRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l1FeeVaultABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof l1FeeVaultABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1FeeVaultAddress } = {} as any
) {
  return useContractRead({
    abi: l1FeeVaultABI,
    address: l1FeeVaultAddress[420],
    ...config,
  } as UseContractReadConfig<typeof l1FeeVaultABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l1FeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000001a)
 */
export function useL1FeeVaultWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l1FeeVaultAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l1FeeVaultABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof l1FeeVaultABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof l1FeeVaultABI, TFunctionName, TMode>({
    abi: l1FeeVaultABI,
    address: l1FeeVaultAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l1FeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x420000000000000000000000000000000000001a)
 */
export function useL1FeeVaultEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l1FeeVaultABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1FeeVaultAddress } = {} as any
) {
  return useContractEvent({
    abi: l1FeeVaultABI,
    address: l1FeeVaultAddress[420],
    ...config,
  } as UseContractEventConfig<typeof l1FeeVaultABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l1StandardBridgeABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8)
 */
export function useL1StandardBridgeRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l1StandardBridgeABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof l1StandardBridgeABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1StandardBridgeAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: l1StandardBridgeABI,
    address:
      l1StandardBridgeAddress[chainId as keyof typeof l1StandardBridgeAddress],
    ...config,
  } as UseContractReadConfig<typeof l1StandardBridgeABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l1StandardBridgeABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8)
 */
export function useL1StandardBridgeWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l1StandardBridgeAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l1StandardBridgeABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof l1StandardBridgeABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof l1StandardBridgeABI, TFunctionName, TMode>({
    abi: l1StandardBridgeABI,
    address:
      l1StandardBridgeAddress[chainId as keyof typeof l1StandardBridgeAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l1StandardBridgeABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x636Af16bf2f682dD3109e60102b8E1A089FedAa8)
 */
export function useL1StandardBridgeEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l1StandardBridgeABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l1StandardBridgeAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: l1StandardBridgeABI,
    address:
      l1StandardBridgeAddress[chainId as keyof typeof l1StandardBridgeAddress],
    ...config,
  } as UseContractEventConfig<typeof l1StandardBridgeABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l2CrossDomainMessengerABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000007)
 */
export function useL2CrossDomainMessengerRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof l2CrossDomainMessengerABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof l2CrossDomainMessengerABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2CrossDomainMessengerAddress } = {} as any
) {
  return useContractRead({
    abi: l2CrossDomainMessengerABI,
    address: l2CrossDomainMessengerAddress[420],
    ...config,
  } as UseContractReadConfig<typeof l2CrossDomainMessengerABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l2CrossDomainMessengerABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000007)
 */
export function useL2CrossDomainMessengerWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l2CrossDomainMessengerAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l2CrossDomainMessengerABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof l2CrossDomainMessengerABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof l2CrossDomainMessengerABI,
    TFunctionName,
    TMode
  >({
    abi: l2CrossDomainMessengerABI,
    address: l2CrossDomainMessengerAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l2CrossDomainMessengerABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000007)
 */
export function useL2CrossDomainMessengerEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l2CrossDomainMessengerABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2CrossDomainMessengerAddress } = {} as any
) {
  return useContractEvent({
    abi: l2CrossDomainMessengerABI,
    address: l2CrossDomainMessengerAddress[420],
    ...config,
  } as UseContractEventConfig<typeof l2CrossDomainMessengerABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l2Erc721BridgeABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000014)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000014)
 */
export function useL2Erc721BridgeRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l2Erc721BridgeABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof l2Erc721BridgeABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2Erc721BridgeAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: l2Erc721BridgeABI,
    address:
      l2Erc721BridgeAddress[chainId as keyof typeof l2Erc721BridgeAddress],
    ...config,
  } as UseContractReadConfig<typeof l2Erc721BridgeABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l2Erc721BridgeABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000014)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000014)
 */
export function useL2Erc721BridgeWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l2Erc721BridgeAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l2Erc721BridgeABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof l2Erc721BridgeABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof l2Erc721BridgeABI, TFunctionName, TMode>({
    abi: l2Erc721BridgeABI,
    address:
      l2Erc721BridgeAddress[chainId as keyof typeof l2Erc721BridgeAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l2Erc721BridgeABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000014)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000014)
 */
export function useL2Erc721BridgeEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l2Erc721BridgeABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2Erc721BridgeAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: l2Erc721BridgeABI,
    address:
      l2Erc721BridgeAddress[chainId as keyof typeof l2Erc721BridgeAddress],
    ...config,
  } as UseContractEventConfig<typeof l2Erc721BridgeABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l2OutputOracleABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdfe97868233d1aa22e815a266982f2cf17685a27)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)
 */
export function useL2OutputOracleRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l2OutputOracleABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof l2OutputOracleABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2OutputOracleAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: l2OutputOracleABI,
    address:
      l2OutputOracleAddress[chainId as keyof typeof l2OutputOracleAddress],
    ...config,
  } as UseContractReadConfig<typeof l2OutputOracleABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l2OutputOracleABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdfe97868233d1aa22e815a266982f2cf17685a27)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)
 */
export function useL2OutputOracleWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l2OutputOracleAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l2OutputOracleABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof l2OutputOracleABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof l2OutputOracleABI, TFunctionName, TMode>({
    abi: l2OutputOracleABI,
    address:
      l2OutputOracleAddress[chainId as keyof typeof l2OutputOracleAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l2OutputOracleABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xdfe97868233d1aa22e815a266982f2cf17685a27)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0)
 */
export function useL2OutputOracleEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l2OutputOracleABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2OutputOracleAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: l2OutputOracleABI,
    address:
      l2OutputOracleAddress[chainId as keyof typeof l2OutputOracleAddress],
    ...config,
  } as UseContractEventConfig<typeof l2OutputOracleABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l2StandardBridgeABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000010)
 */
export function useL2StandardBridgeRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l2StandardBridgeABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof l2StandardBridgeABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2StandardBridgeAddress } = {} as any
) {
  return useContractRead({
    abi: l2StandardBridgeABI,
    address: l2StandardBridgeAddress[420],
    ...config,
  } as UseContractReadConfig<typeof l2StandardBridgeABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l2StandardBridgeABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000010)
 */
export function useL2StandardBridgeWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l2StandardBridgeAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l2StandardBridgeABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof l2StandardBridgeABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof l2StandardBridgeABI, TFunctionName, TMode>({
    abi: l2StandardBridgeABI,
    address: l2StandardBridgeAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l2StandardBridgeABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000010)
 */
export function useL2StandardBridgeEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l2StandardBridgeABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2StandardBridgeAddress } = {} as any
) {
  return useContractEvent({
    abi: l2StandardBridgeABI,
    address: l2StandardBridgeAddress[420],
    ...config,
  } as UseContractEventConfig<typeof l2StandardBridgeABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link l2ToL1MessagePasserABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000016)
 */
export function useL2ToL1MessagePasserRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof l2ToL1MessagePasserABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof l2ToL1MessagePasserABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2ToL1MessagePasserAddress } = {} as any
) {
  return useContractRead({
    abi: l2ToL1MessagePasserABI,
    address: l2ToL1MessagePasserAddress[420],
    ...config,
  } as UseContractReadConfig<typeof l2ToL1MessagePasserABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link l2ToL1MessagePasserABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000016)
 */
export function useL2ToL1MessagePasserWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof l2ToL1MessagePasserAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof l2ToL1MessagePasserABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof l2ToL1MessagePasserABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof l2ToL1MessagePasserABI, TFunctionName, TMode>({
    abi: l2ToL1MessagePasserABI,
    address: l2ToL1MessagePasserAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link l2ToL1MessagePasserABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000016)
 */
export function useL2ToL1MessagePasserEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof l2ToL1MessagePasserABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof l2ToL1MessagePasserAddress } = {} as any
) {
  return useContractEvent({
    abi: l2ToL1MessagePasserABI,
    address: l2ToL1MessagePasserAddress[420],
    ...config,
  } as UseContractEventConfig<typeof l2ToL1MessagePasserABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link mintManagerABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x5C4e7Ba1E219E47948e6e3F55019A647bA501005)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)
 */
export function useMintManagerRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof mintManagerABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof mintManagerABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof mintManagerAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: mintManagerABI,
    address: mintManagerAddress[chainId as keyof typeof mintManagerAddress],
    ...config,
  } as UseContractReadConfig<typeof mintManagerABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link mintManagerABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x5C4e7Ba1E219E47948e6e3F55019A647bA501005)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)
 */
export function useMintManagerWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof mintManagerAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof mintManagerABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof mintManagerABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof mintManagerABI, TFunctionName, TMode>({
    abi: mintManagerABI,
    address: mintManagerAddress[chainId as keyof typeof mintManagerAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link mintManagerABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x5C4e7Ba1E219E47948e6e3F55019A647bA501005)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x038a8825A3C3B0c08d52Cc76E5E361953Cf6Dc76)
 */
export function useMintManagerEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof mintManagerABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof mintManagerAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: mintManagerABI,
    address: mintManagerAddress[chainId as keyof typeof mintManagerAddress],
    ...config,
  } as UseContractEventConfig<typeof mintManagerABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimismMintableErc20FactoryABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof optimismMintableErc20FactoryABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimismMintableErc20FactoryABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimismMintableErc20FactoryAddress } = {} as any
) {
  return useContractRead({
    abi: optimismMintableErc20FactoryABI,
    address: optimismMintableErc20FactoryAddress[1],
    ...config,
  } as UseContractReadConfig<typeof optimismMintableErc20FactoryABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimismMintableErc20FactoryABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimismMintableErc20FactoryAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimismMintableErc20FactoryABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof optimismMintableErc20FactoryABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof optimismMintableErc20FactoryABI,
    TFunctionName,
    TMode
  >({
    abi: optimismMintableErc20FactoryABI,
    address: optimismMintableErc20FactoryAddress[1],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimismMintableErc20FactoryABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof optimismMintableErc20FactoryABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimismMintableErc20FactoryAddress } = {} as any
) {
  return useContractEvent({
    abi: optimismMintableErc20FactoryABI,
    address: optimismMintableErc20FactoryAddress[1],
    ...config,
  } as UseContractEventConfig<typeof optimismMintableErc20FactoryABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimismMintableErc20FactoryGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryGoerliRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof optimismMintableErc20FactoryGoerliABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimismMintableErc20FactoryGoerliABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc20FactoryGoerliAddress
  } = {} as any
) {
  return useContractRead({
    abi: optimismMintableErc20FactoryGoerliABI,
    address: optimismMintableErc20FactoryGoerliAddress[5],
    ...config,
  } as UseContractReadConfig<typeof optimismMintableErc20FactoryGoerliABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimismMintableErc20FactoryGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryGoerliWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimismMintableErc20FactoryGoerliAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimismMintableErc20FactoryGoerliABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof optimismMintableErc20FactoryGoerliABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof optimismMintableErc20FactoryGoerliABI,
    TFunctionName,
    TMode
  >({
    abi: optimismMintableErc20FactoryGoerliABI,
    address: optimismMintableErc20FactoryGoerliAddress[5],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimismMintableErc20FactoryGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryGoerliEvent<
  TEventName extends string
>(
  config: Omit<
    UseContractEventConfig<
      typeof optimismMintableErc20FactoryGoerliABI,
      TEventName
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc20FactoryGoerliAddress
  } = {} as any
) {
  return useContractEvent({
    abi: optimismMintableErc20FactoryGoerliABI,
    address: optimismMintableErc20FactoryGoerliAddress[5],
    ...config,
  } as UseContractEventConfig<typeof optimismMintableErc20FactoryGoerliABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimismMintableErc20FactoryOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryOptimismGoerliRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof optimismMintableErc20FactoryOptimismGoerliABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimismMintableErc20FactoryOptimismGoerliABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc20FactoryOptimismGoerliAddress
  } = {} as any
) {
  return useContractRead({
    abi: optimismMintableErc20FactoryOptimismGoerliABI,
    address: optimismMintableErc20FactoryOptimismGoerliAddress[420],
    ...config,
  } as UseContractReadConfig<typeof optimismMintableErc20FactoryOptimismGoerliABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimismMintableErc20FactoryOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryOptimismGoerliWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimismMintableErc20FactoryOptimismGoerliAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimismMintableErc20FactoryOptimismGoerliABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof optimismMintableErc20FactoryOptimismGoerliABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof optimismMintableErc20FactoryOptimismGoerliABI,
    TFunctionName,
    TMode
  >({
    abi: optimismMintableErc20FactoryOptimismGoerliABI,
    address: optimismMintableErc20FactoryOptimismGoerliAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimismMintableErc20FactoryOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000012)
 */
export function useOptimismMintableErc20FactoryOptimismGoerliEvent<
  TEventName extends string
>(
  config: Omit<
    UseContractEventConfig<
      typeof optimismMintableErc20FactoryOptimismGoerliABI,
      TEventName
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc20FactoryOptimismGoerliAddress
  } = {} as any
) {
  return useContractEvent({
    abi: optimismMintableErc20FactoryOptimismGoerliABI,
    address: optimismMintableErc20FactoryOptimismGoerliAddress[420],
    ...config,
  } as UseContractEventConfig<typeof optimismMintableErc20FactoryOptimismGoerliABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimismMintableErc721FactoryABI}__.
 *
 * [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000017)
 */
export function useOptimismMintableErc721FactoryRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof optimismMintableErc721FactoryABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimismMintableErc721FactoryABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc721FactoryAddress
  } = {} as any
) {
  return useContractRead({
    abi: optimismMintableErc721FactoryABI,
    address: optimismMintableErc721FactoryAddress[10],
    ...config,
  } as UseContractReadConfig<typeof optimismMintableErc721FactoryABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimismMintableErc721FactoryABI}__.
 *
 * [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000017)
 */
export function useOptimismMintableErc721FactoryWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimismMintableErc721FactoryAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimismMintableErc721FactoryABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof optimismMintableErc721FactoryABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof optimismMintableErc721FactoryABI,
    TFunctionName,
    TMode
  >({
    abi: optimismMintableErc721FactoryABI,
    address: optimismMintableErc721FactoryAddress[10],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimismMintableErc721FactoryABI}__.
 *
 * [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x4200000000000000000000000000000000000017)
 */
export function useOptimismMintableErc721FactoryEvent<
  TEventName extends string
>(
  config: Omit<
    UseContractEventConfig<typeof optimismMintableErc721FactoryABI, TEventName>,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc721FactoryAddress
  } = {} as any
) {
  return useContractEvent({
    abi: optimismMintableErc721FactoryABI,
    address: optimismMintableErc721FactoryAddress[10],
    ...config,
  } as UseContractEventConfig<typeof optimismMintableErc721FactoryABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimismMintableErc721FactoryOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000017)
 */
export function useOptimismMintableErc721FactoryOptimismGoerliRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof optimismMintableErc721FactoryOptimismGoerliABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimismMintableErc721FactoryOptimismGoerliABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc721FactoryOptimismGoerliAddress
  } = {} as any
) {
  return useContractRead({
    abi: optimismMintableErc721FactoryOptimismGoerliABI,
    address: optimismMintableErc721FactoryOptimismGoerliAddress[420],
    ...config,
  } as UseContractReadConfig<typeof optimismMintableErc721FactoryOptimismGoerliABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimismMintableErc721FactoryOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000017)
 */
export function useOptimismMintableErc721FactoryOptimismGoerliWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimismMintableErc721FactoryOptimismGoerliAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimismMintableErc721FactoryOptimismGoerliABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof optimismMintableErc721FactoryOptimismGoerliABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<
    typeof optimismMintableErc721FactoryOptimismGoerliABI,
    TFunctionName,
    TMode
  >({
    abi: optimismMintableErc721FactoryOptimismGoerliABI,
    address: optimismMintableErc721FactoryOptimismGoerliAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimismMintableErc721FactoryOptimismGoerliABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000017)
 */
export function useOptimismMintableErc721FactoryOptimismGoerliEvent<
  TEventName extends string
>(
  config: Omit<
    UseContractEventConfig<
      typeof optimismMintableErc721FactoryOptimismGoerliABI,
      TEventName
    >,
    'abi' | 'address'
  > & {
    chainId?: keyof typeof optimismMintableErc721FactoryOptimismGoerliAddress
  } = {} as any
) {
  return useContractEvent({
    abi: optimismMintableErc721FactoryOptimismGoerliABI,
    address: optimismMintableErc721FactoryOptimismGoerliAddress[420],
    ...config,
  } as UseContractEventConfig<typeof optimismMintableErc721FactoryOptimismGoerliABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimismPortalABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xbEb5Fc579115071764c7423A4f12eDde41f106Ed)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383)
 */
export function useOptimismPortalRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof optimismPortalABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof optimismPortalABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimismPortalAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: optimismPortalABI,
    address:
      optimismPortalAddress[chainId as keyof typeof optimismPortalAddress],
    ...config,
  } as UseContractReadConfig<typeof optimismPortalABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimismPortalABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xbEb5Fc579115071764c7423A4f12eDde41f106Ed)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383)
 */
export function useOptimismPortalWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimismPortalAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimismPortalABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof optimismPortalABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof optimismPortalABI, TFunctionName, TMode>({
    abi: optimismPortalABI,
    address:
      optimismPortalAddress[chainId as keyof typeof optimismPortalAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimismPortalABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xbEb5Fc579115071764c7423A4f12eDde41f106Ed)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383)
 */
export function useOptimismPortalEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof optimismPortalABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimismPortalAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: optimismPortalABI,
    address:
      optimismPortalAddress[chainId as keyof typeof optimismPortalAddress],
    ...config,
  } as UseContractEventConfig<typeof optimismPortalABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimistABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 */
export function useOptimistRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof optimistABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof optimistABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimistAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: optimistABI,
    address: optimistAddress[chainId as keyof typeof optimistAddress],
    ...config,
  } as UseContractReadConfig<typeof optimistABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimistABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 */
export function useOptimistWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimistAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimistABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof optimistABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof optimistABI, TFunctionName, TMode>({
    abi: optimistABI,
    address: optimistAddress[chainId as keyof typeof optimistAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimistABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5)
 */
export function useOptimistEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof optimistABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimistAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: optimistABI,
    address: optimistAddress[chainId as keyof typeof optimistAddress],
    ...config,
  } as UseContractEventConfig<typeof optimistABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimistAllowlistABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x482b1945D58f2E9Db0CEbe13c7fcFc6876b41180)
 */
export function useOptimistAllowlistRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof optimistAllowlistABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimistAllowlistABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimistAllowlistAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: optimistAllowlistABI,
    address:
      optimistAllowlistAddress[
        chainId as keyof typeof optimistAllowlistAddress
      ],
    ...config,
  } as UseContractReadConfig<typeof optimistAllowlistABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link optimistInviterABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 */
export function useOptimistInviterRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof optimistInviterABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof optimistInviterABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimistInviterAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: optimistInviterABI,
    address:
      optimistInviterAddress[chainId as keyof typeof optimistInviterAddress],
    ...config,
  } as UseContractReadConfig<typeof optimistInviterABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link optimistInviterABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 */
export function useOptimistInviterWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof optimistInviterAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof optimistInviterABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof optimistInviterABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof optimistInviterABI, TFunctionName, TMode>({
    abi: optimistInviterABI,
    address:
      optimistInviterAddress[chainId as keyof typeof optimistInviterAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link optimistInviterABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x073031A1E1b8F5458Ed41Ce56331F5fd7e1de929)
 */
export function useOptimistInviterEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof optimistInviterABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof optimistInviterAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: optimistInviterABI,
    address:
      optimistInviterAddress[chainId as keyof typeof optimistInviterAddress],
    ...config,
  } as UseContractEventConfig<typeof optimistInviterABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link portalSenderABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x0A893d9576b9cFD9EF78595963dc973238E78210)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xe7FACd39531ee3C313330E93B4d7a8B8A3c84Aa4)
 */
export function usePortalSenderRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof portalSenderABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof portalSenderABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof portalSenderAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: portalSenderABI,
    address: portalSenderAddress[chainId as keyof typeof portalSenderAddress],
    ...config,
  } as UseContractReadConfig<typeof portalSenderABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link portalSenderABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x0A893d9576b9cFD9EF78595963dc973238E78210)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xe7FACd39531ee3C313330E93B4d7a8B8A3c84Aa4)
 */
export function usePortalSenderWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof portalSenderAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof portalSenderABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof portalSenderABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof portalSenderABI, TFunctionName, TMode>({
    abi: portalSenderABI,
    address: portalSenderAddress[chainId as keyof typeof portalSenderAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link protocolVersionsABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x0C24F5098774aA366827D667494e9F889f7cFc08)
 */
export function useProtocolVersionsRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof protocolVersionsABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof protocolVersionsABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof protocolVersionsAddress } = {} as any
) {
  return useContractRead({
    abi: protocolVersionsABI,
    address: protocolVersionsAddress[5],
    ...config,
  } as UseContractReadConfig<typeof protocolVersionsABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link protocolVersionsABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x0C24F5098774aA366827D667494e9F889f7cFc08)
 */
export function useProtocolVersionsWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof protocolVersionsAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof protocolVersionsABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof protocolVersionsABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof protocolVersionsABI, TFunctionName, TMode>({
    abi: protocolVersionsABI,
    address: protocolVersionsAddress[5],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link protocolVersionsABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x0C24F5098774aA366827D667494e9F889f7cFc08)
 */
export function useProtocolVersionsEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof protocolVersionsABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof protocolVersionsAddress } = {} as any
) {
  return useContractEvent({
    abi: protocolVersionsABI,
    address: protocolVersionsAddress[5],
    ...config,
  } as UseContractEventConfig<typeof protocolVersionsABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link proxyAdminABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000018)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000018)
 */
export function useProxyAdminRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof proxyAdminABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof proxyAdminABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof proxyAdminAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: proxyAdminABI,
    address: proxyAdminAddress[chainId as keyof typeof proxyAdminAddress],
    ...config,
  } as UseContractReadConfig<typeof proxyAdminABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link proxyAdminABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000018)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000018)
 */
export function useProxyAdminWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof proxyAdminAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof proxyAdminABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof proxyAdminABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof proxyAdminABI, TFunctionName, TMode>({
    abi: proxyAdminABI,
    address: proxyAdminAddress[chainId as keyof typeof proxyAdminAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link proxyAdminABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x4200000000000000000000000000000000000018)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x4200000000000000000000000000000000000018)
 */
export function useProxyAdminEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof proxyAdminABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof proxyAdminAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: proxyAdminABI,
    address: proxyAdminAddress[chainId as keyof typeof proxyAdminAddress],
    ...config,
  } as UseContractEventConfig<typeof proxyAdminABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link schemaRegistryABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)
 */
export function useSchemaRegistryRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof schemaRegistryABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof schemaRegistryABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof schemaRegistryAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: schemaRegistryABI,
    address:
      schemaRegistryAddress[chainId as keyof typeof schemaRegistryAddress],
    ...config,
  } as UseContractReadConfig<typeof schemaRegistryABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link schemaRegistryABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)
 */
export function useSchemaRegistryWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof schemaRegistryAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof schemaRegistryABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof schemaRegistryABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof schemaRegistryABI, TFunctionName, TMode>({
    abi: schemaRegistryABI,
    address:
      schemaRegistryAddress[chainId as keyof typeof schemaRegistryAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link schemaRegistryABI}__.
 *
 * - [__View Contract on Op Mainnet Optimism Explorer__](https://explorer.optimism.io/address/0x6232208d66bAc2305b46b4Cb6BCB3857B298DF13)
 * - [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x2545fa928d5d278cA75Fd47306e4a89096ff6403)
 */
export function useSchemaRegistryEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof schemaRegistryABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof schemaRegistryAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: schemaRegistryABI,
    address:
      schemaRegistryAddress[chainId as keyof typeof schemaRegistryAddress],
    ...config,
  } as UseContractEventConfig<typeof schemaRegistryABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link sequencerFeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000011)
 */
export function useSequencerFeeVaultRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof sequencerFeeVaultABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof sequencerFeeVaultABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof sequencerFeeVaultAddress } = {} as any
) {
  return useContractRead({
    abi: sequencerFeeVaultABI,
    address: sequencerFeeVaultAddress[420],
    ...config,
  } as UseContractReadConfig<typeof sequencerFeeVaultABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link sequencerFeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000011)
 */
export function useSequencerFeeVaultWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof sequencerFeeVaultAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof sequencerFeeVaultABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof sequencerFeeVaultABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof sequencerFeeVaultABI, TFunctionName, TMode>({
    abi: sequencerFeeVaultABI,
    address: sequencerFeeVaultAddress[420],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link sequencerFeeVaultABI}__.
 *
 * [__View Contract on Optimism Goerli Etherscan__](https://goerli-optimism.etherscan.io/address/0x4200000000000000000000000000000000000011)
 */
export function useSequencerFeeVaultEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof sequencerFeeVaultABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof sequencerFeeVaultAddress } = {} as any
) {
  return useContractEvent({
    abi: sequencerFeeVaultABI,
    address: sequencerFeeVaultAddress[420],
    ...config,
  } as UseContractEventConfig<typeof sequencerFeeVaultABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link systemConfigABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60)
 */
export function useSystemConfigRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof systemConfigABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof systemConfigABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof systemConfigAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractRead({
    abi: systemConfigABI,
    address: systemConfigAddress[chainId as keyof typeof systemConfigAddress],
    ...config,
  } as UseContractReadConfig<typeof systemConfigABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link systemConfigABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60)
 */
export function useSystemConfigWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof systemConfigAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof systemConfigABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof systemConfigABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractWrite<typeof systemConfigABI, TFunctionName, TMode>({
    abi: systemConfigABI,
    address: systemConfigAddress[chainId as keyof typeof systemConfigAddress],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link systemConfigABI}__.
 *
 * - [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x229047fed2591dbec1eF1118d64F7aF3dB9EB290)
 * - [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60)
 */
export function useSystemConfigEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof systemConfigABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof systemConfigAddress } = {} as any
) {
  const { chain } = useNetwork()
  const defaultChainId = useChainId()
  const chainId = config.chainId ?? chain?.id ?? defaultChainId
  return useContractEvent({
    abi: systemConfigABI,
    address: systemConfigAddress[chainId as keyof typeof systemConfigAddress],
    ...config,
  } as UseContractEventConfig<typeof systemConfigABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link systemDictatorABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB)
 */
export function useSystemDictatorRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof systemDictatorABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<typeof systemDictatorABI, TFunctionName, TSelectData>,
    'abi' | 'address'
  > & { chainId?: keyof typeof systemDictatorAddress } = {} as any
) {
  return useContractRead({
    abi: systemDictatorABI,
    address: systemDictatorAddress[1],
    ...config,
  } as UseContractReadConfig<typeof systemDictatorABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link systemDictatorABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB)
 */
export function useSystemDictatorWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof systemDictatorAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof systemDictatorABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<typeof systemDictatorABI, TFunctionName, TMode> & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof systemDictatorABI, TFunctionName, TMode>({
    abi: systemDictatorABI,
    address: systemDictatorAddress[1],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link systemDictatorABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0xB4453CEb33d2e67FA244A24acf2E50CEF31F53cB)
 */
export function useSystemDictatorEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof systemDictatorABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof systemDictatorAddress } = {} as any
) {
  return useContractEvent({
    abi: systemDictatorABI,
    address: systemDictatorAddress[1],
    ...config,
  } as UseContractEventConfig<typeof systemDictatorABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link systemDictatorGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971)
 */
export function useSystemDictatorGoerliRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<
    typeof systemDictatorGoerliABI,
    TFunctionName
  >
>(
  config: Omit<
    UseContractReadConfig<
      typeof systemDictatorGoerliABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof systemDictatorGoerliAddress } = {} as any
) {
  return useContractRead({
    abi: systemDictatorGoerliABI,
    address: systemDictatorGoerliAddress[5],
    ...config,
  } as UseContractReadConfig<typeof systemDictatorGoerliABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link systemDictatorGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971)
 */
export function useSystemDictatorGoerliWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof systemDictatorGoerliAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof systemDictatorGoerliABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof systemDictatorGoerliABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof systemDictatorGoerliABI, TFunctionName, TMode>(
    {
      abi: systemDictatorGoerliABI,
      address: systemDictatorGoerliAddress[5],
      ...config,
    } as any
  )
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link systemDictatorGoerliABI}__.
 *
 * [__View Contract on Goerli Etherscan__](https://goerli.etherscan.io/address/0x1f0613A44c9a8ECE7B3A2e0CdBdF0F5B47A50971)
 */
export function useSystemDictatorGoerliEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof systemDictatorGoerliABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof systemDictatorGoerliAddress } = {} as any
) {
  return useContractEvent({
    abi: systemDictatorGoerliABI,
    address: systemDictatorGoerliAddress[5],
    ...config,
  } as UseContractEventConfig<typeof systemDictatorGoerliABI, TEventName>)
}

/**
 * Wraps __{@link useContractRead}__ with `abi` set to __{@link teleportrWithdrawerABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x78A25524D90E3D0596558fb43789bD800a5c3007)
 */
export function useTeleportrWithdrawerRead<
  TFunctionName extends string,
  TSelectData = ReadContractResult<typeof teleportrWithdrawerABI, TFunctionName>
>(
  config: Omit<
    UseContractReadConfig<
      typeof teleportrWithdrawerABI,
      TFunctionName,
      TSelectData
    >,
    'abi' | 'address'
  > & { chainId?: keyof typeof teleportrWithdrawerAddress } = {} as any
) {
  return useContractRead({
    abi: teleportrWithdrawerABI,
    address: teleportrWithdrawerAddress[1],
    ...config,
  } as UseContractReadConfig<typeof teleportrWithdrawerABI, TFunctionName, TSelectData>)
}

/**
 * Wraps __{@link useContractWrite}__ with `abi` set to __{@link teleportrWithdrawerABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x78A25524D90E3D0596558fb43789bD800a5c3007)
 */
export function useTeleportrWithdrawerWrite<
  TFunctionName extends string,
  TMode extends WriteContractMode = undefined,
  TChainId extends number = keyof typeof teleportrWithdrawerAddress
>(
  config: TMode extends 'prepared'
    ? UseContractWriteConfig<
        PrepareWriteContractResult<
          typeof teleportrWithdrawerABI,
          string
        >['request']['abi'],
        TFunctionName,
        TMode
      > & { address?: Address; chainId?: TChainId }
    : UseContractWriteConfig<
        typeof teleportrWithdrawerABI,
        TFunctionName,
        TMode
      > & {
        abi?: never
        address?: never
        chainId?: TChainId
      } = {} as any
) {
  return useContractWrite<typeof teleportrWithdrawerABI, TFunctionName, TMode>({
    abi: teleportrWithdrawerABI,
    address: teleportrWithdrawerAddress[1],
    ...config,
  } as any)
}

/**
 * Wraps __{@link useContractEvent}__ with `abi` set to __{@link teleportrWithdrawerABI}__.
 *
 * [__View Contract on Ethereum Etherscan__](https://etherscan.io/address/0x78A25524D90E3D0596558fb43789bD800a5c3007)
 */
export function useTeleportrWithdrawerEvent<TEventName extends string>(
  config: Omit<
    UseContractEventConfig<typeof teleportrWithdrawerABI, TEventName>,
    'abi' | 'address'
  > & { chainId?: keyof typeof teleportrWithdrawerAddress } = {} as any
) {
  return useContractEvent({
    abi: teleportrWithdrawerABI,
    address: teleportrWithdrawerAddress[1],
    ...config,
  } as UseContractEventConfig<typeof teleportrWithdrawerABI, TEventName>)
}
