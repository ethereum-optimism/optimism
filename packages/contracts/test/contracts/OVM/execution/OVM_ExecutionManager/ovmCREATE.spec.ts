/* Internal Imports */
import {
  ExecutionManagerTestRunner,
  TestDefinition,
  OVM_TX_GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
  ZERO_ADDRESS,
  VERIFIED_EMPTY_CONTRACT_HASH,
  DUMMY_BYTECODE_BYTELEN,
  DUMMY_BYTECODE_HASH,
  getStorageXOR,
} from '../../../../helpers'

const CREATED_CONTRACT_1 = '0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb'
const CREATED_CONTRACT_2 = '0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb'
const NESTED_CREATED_CONTRACT = '0xcb964b3f4162a0d4f5c997b40e19da5a546bc36f'
const DUMMY_REVERT_DATA =
  '0xdeadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420'

const test_ovmCREATE: TestDefinition = {
  name: 'Basic tests for ovmCREATE',
  preState: {
    ExecutionManager: {
      ovmStateManager: '$OVM_STATE_MANAGER',
      ovmSafetyChecker: '$OVM_SAFETY_CHECKER',
      messageRecord: {
        nuisanceGasLeft: OVM_TX_GAS_LIMIT,
      },
    },
    StateManager: {
      owner: '$OVM_EXECUTION_MANAGER',
      accounts: {
        $DUMMY_OVM_ADDRESS_1: {
          codeHash: NON_NULL_BYTES32,
          ethAddress: '$OVM_CALL_HELPER',
        },
        $DUMMY_OVM_ADDRESS_2: {
          codeHash: NON_NULL_BYTES32,
          ethAddress: '$OVM_CALL_HELPER',
        },
        [CREATED_CONTRACT_1]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
        [CREATED_CONTRACT_2]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
        [NESTED_CREATED_CONTRACT]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
      },
      contractStorage: {
        $DUMMY_OVM_ADDRESS_2: {
          [NULL_BYTES32]: getStorageXOR(NULL_BYTES32),
        },
      },
      verifiedContractStorage: {
        $DUMMY_OVM_ADDRESS_1: {
          [NON_NULL_BYTES32]: true,
        },
        $DUMMY_OVM_ADDRESS_2: {
          [NULL_BYTES32]: true,
        },
      },
    },
  },
  parameters: [
    {
      name: 'ovmCREATE, ovmEXTCODESIZE(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            bytecode: DUMMY_BYTECODE,
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmEXTCODESIZE',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: DUMMY_BYTECODE_BYTELEN,
        },
      ],
    },
    {
      name: 'ovmCREATE, ovmEXTCODEHASH(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            bytecode: DUMMY_BYTECODE,
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmEXTCODEHASH',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: DUMMY_BYTECODE_HASH,
        },
      ],
    },
    {
      name: 'ovmCREATE, ovmEXTCODECOPY(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            bytecode: DUMMY_BYTECODE,
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmEXTCODECOPY',
          functionParams: {
            address: CREATED_CONTRACT_1,
            offset: 0,
            length: DUMMY_BYTECODE_BYTELEN,
          },
          expectedReturnStatus: true,
          expectedReturnValue: DUMMY_BYTECODE,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT, ovmEXTCODESIZE(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmEXTCODESIZE',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: 0,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT, ovmEXTCODEHASH(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmEXTCODEHASH',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: NULL_BYTES32,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT, ovmEXTCODECOPY(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmEXTCODECOPY',
          functionParams: {
            address: CREATED_CONTRACT_1,
            offset: 0,
            length: 256,
          },
          expectedReturnStatus: true,
          expectedReturnValue: '0x' + '00'.repeat(256),
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmADDRESS',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmADDRESS',
                expectedReturnValue: CREATED_CONTRACT_1,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
      ],
    },
    {
      name: 'ovmCALL => ovmCREATE => ovmCALLER',
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              {
                functionName: 'ovmCREATE',
                functionParams: {
                  subSteps: [
                    {
                      functionName: 'ovmSLOAD',
                      functionParams: {
                        key: NON_NULL_BYTES32,
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: NULL_BYTES32,
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: CREATED_CONTRACT_2,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmSSTORE + ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmSSTORE',
                functionParams: {
                  key: NON_NULL_BYTES32,
                  value: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
              },
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NON_NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
      ],
    },
    {
      name:
        'ovmCREATE => ovmSSTORE, ovmCALL(CREATED) => ovmSLOAD(EXIST) + ovmSLOAD(NONEXIST)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmSSTORE',
                functionParams: {
                  key: NON_NULL_BYTES32,
                  value: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: CREATED_CONTRACT_1,
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NON_NULL_BYTES32,
              },
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name:
        'ovmCREATE => ovmCALL(ADDRESS_1) => ovmSSTORE, ovmCALL(ADDRESS_1) => ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_1',
                  subSteps: [
                    {
                      functionName: 'ovmSSTORE',
                      functionParams: {
                        key: NON_NULL_BYTES32,
                        value: NON_NULL_BYTES32,
                      },
                      expectedReturnStatus: true,
                    },
                  ],
                },
                expectedReturnStatus: true,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NON_NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name:
        'ovmCREATE => (ovmCALL(ADDRESS_2) => ovmSSTORE) + ovmREVERT, ovmCALL(ADDRESS_2) => ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_2',
                  subSteps: [
                    {
                      functionName: 'ovmSSTORE',
                      functionParams: {
                        key: NULL_BYTES32,
                        value: NON_NULL_BYTES32,
                      },
                      expectedReturnStatus: true,
                    },
                  ],
                },
                expectedReturnStatus: true,
              },
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_2',
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmCALL(ADDRESS_NONEXIST)',
      expectInvalidStateAccess: true,
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_3',
                  calldata: '0x',
                },
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: false,
          expectedReturnValue: {
            flag: REVERT_FLAGS.INVALID_STATE_ACCESS,
          },
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmCREATE => ovmCALL(ADDRESS_NONEXIST)',
      expectInvalidStateAccess: true,
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCREATE',
                functionParams: {
                  subSteps: [
                    {
                      functionName: 'ovmCALL',
                      functionParams: {
                        gasLimit: OVM_TX_GAS_LIMIT,
                        target: '$DUMMY_OVM_ADDRESS_3',
                        calldata: '0x',
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: '0x00',
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: false,
          expectedReturnValue: {
            flag: REVERT_FLAGS.INVALID_STATE_ACCESS,
          },
        },
      ],
    },
    {
      name: 'ovmCREATE => OUT_OF_GAS',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'evmINVALID',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
      ],
    },
  ],
}

const runner = new ExecutionManagerTestRunner()
runner.run(test_ovmCREATE)
