/* Internal Imports */
import {
    ExecutionManagerTestRunner,
    TestDefinition,
    OVM_TX_GAS_LIMIT,
    NON_NULL_BYTES32,
    REVERT_FLAGS,
    VERIFIED_EMPTY_CONTRACT_HASH,
    NUISANCE_GAS_COSTS,
    Helper_TestRunner_BYTELEN,
  } from '../../../../helpers'
  
  const DUMMY_REVERT_DATA =
    '0xdeadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420'

  console.log('cost multiplied:', Helper_TestRunner_BYTELEN * NUISANCE_GAS_COSTS.NUISANCE_GAS_PER_CONTRACT_BYTE, 'bytelenL:',Helper_TestRunner_BYTELEN )
  
  const test_nuisanceGas: TestDefinition = {
    name: 'Basic tests for nuisance gas',
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
          $DUMMY_OVM_ADDRESS_3: {
            codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
            ethAddress: '0x' + '00'.repeat(20),
          },
        },
      },
    },
    subTests: [
        {
            name: 'ovmCALL consumes nuisance gas of CODESIZE * NUISANCE_GAS_PER_CONTRACT_BYTE',
            postState: {
                ExecutionManager: {
                    messageRecord: {
                        nuisanceGasLeft: OVM_TX_GAS_LIMIT - Helper_TestRunner_BYTELEN * NUISANCE_GAS_COSTS.NUISANCE_GAS_PER_CONTRACT_BYTE
                    }
                }
            },
            parameters: [
                {
                    name: 'single ovmCALL',
                    focus: true,
                    steps: [
                        // do a non-nuisance-gas-consuming opcode (test runner auto-wraps in ovmCALL)
                        {
                          functionName: 'ovmADDRESS',
                          expectedReturnValue: "$DUMMY_OVM_ADDRESS_1",
                        },
                    ],
                }
            ]
        },
        {
            name: 'ovmCALL only consumes nuisance gas of CODESIZE * NUISANCE_GAS_PER_CONTRACT_BYTE for same contract called twice',
            postState: {
                ExecutionManager: {
                    messageRecord: {
                        nuisanceGasLeft: OVM_TX_GAS_LIMIT - Helper_TestRunner_BYTELEN * NUISANCE_GAS_COSTS.NUISANCE_GAS_PER_CONTRACT_BYTE
                    }
                }
            },
            parameters: [
                {
                    name: 'nested ovmCALL',
                    focus: true,
                    steps: [
                        {
                          functionName: 'ovmCALL',
                          functionParams: {
                              gasLimit: OVM_TX_GAS_LIMIT,
                              target: "$DUMMY_OVM_ADDRESS_1",
                              subSteps: []
                          },
                          expectedReturnStatus: true
                        },
                    ],
                }
            ]
        },
        {
            name: 'ovmCALL only consumes nuisance gas of CODESIZE * NUISANCE_GAS_PER_CONTRACT_BYTE twice for two separate ovmCALLS',
            postState: {
                ExecutionManager: {
                    messageRecord: {
                        nuisanceGasLeft: OVM_TX_GAS_LIMIT - 2 * ( Helper_TestRunner_BYTELEN * NUISANCE_GAS_COSTS.NUISANCE_GAS_PER_CONTRACT_BYTE )
                    }
                }
            },
            parameters: [
                {
                    name: 'nested ovmCALL',
                    // focus: true,
                    steps: [
                        {
                          functionName: 'ovmCALL',
                          functionParams: {
                              gasLimit: OVM_TX_GAS_LIMIT,
                              target: "$DUMMY_OVM_ADDRESS_2",
                              subSteps: []
                          },
                          expectedReturnStatus: true
                        },
                    ],
                }
            ]
        }
    ],
    parameters: [
      {
        name: 'ovmCALL(ADDRESS_1) => ovmSSTORE',
        steps: [
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
      {
        name:
          'ovmCALL(ADDRESS_1) => ovmSSTORE + ovmSLOAD, ovmCALL(ADDRESS_1) => ovmSLOAD',
        steps: [
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
          'ovmCALL(ADDRESS_1) => ovmCALL(ADDRESS_2) => ovmADDRESS + ovmCALLER',
        steps: [
          {
            functionName: 'ovmCALL',
            functionParams: {
              gasLimit: OVM_TX_GAS_LIMIT,
              target: '$DUMMY_OVM_ADDRESS_1',
              subSteps: [
                {
                  functionName: 'ovmCALL',
                  functionParams: {
                    gasLimit: OVM_TX_GAS_LIMIT,
                    target: '$DUMMY_OVM_ADDRESS_2',
                    subSteps: [
                      {
                        functionName: 'ovmADDRESS',
                        expectedReturnValue: '$DUMMY_OVM_ADDRESS_2',
                      },
                      {
                        functionName: 'ovmCALLER',
                        expectedReturnValue: '$DUMMY_OVM_ADDRESS_1',
                      },
                    ],
                  },
                  expectedReturnStatus: true,
                },
              ],
            },
            expectedReturnStatus: true,
          },
        ],
      },
      {
        name: 'ovmCALL(ADDRESS_1) => ovmCALL(ADDRESS_3)',
        steps: [
          {
            functionName: 'ovmCALL',
            functionParams: {
              gasLimit: OVM_TX_GAS_LIMIT,
              target: '$DUMMY_OVM_ADDRESS_1',
              subSteps: [
                {
                  functionName: 'ovmCALL',
                  functionParams: {
                    gasLimit: OVM_TX_GAS_LIMIT,
                    target: '$DUMMY_OVM_ADDRESS_3',
                    calldata: '0x',
                  },
                  expectedReturnStatus: true,
                },
              ],
            },
            expectedReturnStatus: true,
            expectedReturnValue: '0x',
          },
        ],
      },
      {
        name: 'ovmCALL(ADDRESS_1) => INTENTIONAL_REVERT',
        steps: [
          {
            functionName: 'ovmCALL',
            functionParams: {
              gasLimit: OVM_TX_GAS_LIMIT,
              target: '$DUMMY_OVM_ADDRESS_1',
              subSteps: [
                {
                  functionName: 'evmREVERT',
                  returnData: {
                    flag: REVERT_FLAGS.INTENTIONAL_REVERT,
                    data: DUMMY_REVERT_DATA,
                  },
                },
              ],
            },
            expectedReturnStatus: false,
            expectedReturnValue: DUMMY_REVERT_DATA,
          },
        ],
      },
      {
        name: 'ovmCALL(ADDRESS_1) => EXCEEDS_NUISANCE_GAS',
        steps: [
          {
            functionName: 'ovmCALL',
            functionParams: {
              gasLimit: OVM_TX_GAS_LIMIT,
              target: '$DUMMY_OVM_ADDRESS_1',
              subSteps: [
                {
                  functionName: 'evmREVERT',
                  returnData: {
                    flag: REVERT_FLAGS.EXCEEDS_NUISANCE_GAS,
                  },
                },
              ],
            },
            expectedReturnStatus: false,
            expectedReturnValue: '0x',
          },
        ],
      },
    ],
  }
  
  const runner = new ExecutionManagerTestRunner()
  runner.run(test_nuisanceGas)
  