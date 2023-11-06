const L2_DEFAULTS = {
  L2STANDARDBRIDGE: '0x4200000000000000000000000000000000000010',
  L2TOL1MESSAGEPASSER: '0x4200000000000000000000000000000000000016',
  L2BOBA: '0x4200000000000000000000000000000000000023',
}

const SEPOLIA = {
  L2OUTPUTORACLE: '0x8b53174E4ab32257682B8F0412458fdCcB14E609',
  OPTIMISMPORTAL: '0x04E1D761dE3e35df654f89df8AFbc430ce5519a7',
  L1STANDARDBRIDGE: '0xfAdea0361bcEf113C7a8543778f4a50bb64E6911',
  L1BOBA: '0x33faF65b3DfcC6A1FccaD4531D9ce518F0FDc896',
  ...L2_DEFAULTS,
}

const GOERLI = { ...L2_DEFAULTS }

const MAINNET = { ...L2_DEFAULTS }

const ABI = {
  OPTIMISMPORTAL: [
    {
      inputs: [
        {
          components: [
            {
              internalType: 'uint256',
              name: 'nonce',
              type: 'uint256',
            },
            {
              internalType: 'address',
              name: 'sender',
              type: 'address',
            },
            {
              internalType: 'address',
              name: 'target',
              type: 'address',
            },
            {
              internalType: 'uint256',
              name: 'value',
              type: 'uint256',
            },
            {
              internalType: 'uint256',
              name: 'gasLimit',
              type: 'uint256',
            },
            {
              internalType: 'bytes',
              name: 'data',
              type: 'bytes',
            },
          ],
          internalType: 'struct Types.WithdrawalTransaction',
          name: '_tx',
          type: 'tuple',
        },
        {
          internalType: 'uint256',
          name: '_l2OutputIndex',
          type: 'uint256',
        },
        {
          components: [
            {
              internalType: 'bytes32',
              name: 'version',
              type: 'bytes32',
            },
            {
              internalType: 'bytes32',
              name: 'stateRoot',
              type: 'bytes32',
            },
            {
              internalType: 'bytes32',
              name: 'messagePasserStorageRoot',
              type: 'bytes32',
            },
            {
              internalType: 'bytes32',
              name: 'latestBlockhash',
              type: 'bytes32',
            },
          ],
          internalType: 'struct Types.OutputRootProof',
          name: '_outputRootProof',
          type: 'tuple',
        },
        {
          internalType: 'bytes[]',
          name: '_withdrawalProof',
          type: 'bytes[]',
        },
      ],
      name: 'proveWithdrawalTransaction',
      outputs: [],
      stateMutability: 'nonpayable',
      type: 'function',
    },
    {
      inputs: [
        {
          components: [
            {
              internalType: 'uint256',
              name: 'nonce',
              type: 'uint256',
            },
            {
              internalType: 'address',
              name: 'sender',
              type: 'address',
            },
            {
              internalType: 'address',
              name: 'target',
              type: 'address',
            },
            {
              internalType: 'uint256',
              name: 'value',
              type: 'uint256',
            },
            {
              internalType: 'uint256',
              name: 'gasLimit',
              type: 'uint256',
            },
            {
              internalType: 'bytes',
              name: 'data',
              type: 'bytes',
            },
          ],
          internalType: 'struct Types.WithdrawalTransaction',
          name: '_tx',
          type: 'tuple',
        },
      ],
      name: 'finalizeWithdrawalTransaction',
      outputs: [],
      stateMutability: 'nonpayable',
      type: 'function',
    },
  ],
}

module.exports = { SEPOLIA, GOERLI, MAINNET, ABI }
