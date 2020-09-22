/* Internal Imports */
import {
  runExecutionManagerTest,
  TestDefinition,
  OVM_TX_GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
} from '../../../../helpers'

const test_ovmSTATICCALL: TestDefinition = {
  name: 'Basic checks on staticcall',
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
      },
      verifiedContractStorage: {
        $DUMMY_OVM_ADDRESS_2: {
          [NON_NULL_BYTES32]: true,
        },
      },
    },
  },
  parameters: [
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            OVM_TX_GAS_LIMIT,
            '$DUMMY_OVM_ADDRESS_1',
            [
              {
                functionName: 'ovmSTATICCALL',
                functionParams: [
                  OVM_TX_GAS_LIMIT / 2,
                  '$DUMMY_OVM_ADDRESS_2',
                  [
                    {
                      functionName: 'ovmSSTORE',
                      functionParams: [NULL_BYTES32, NULL_BYTES32],
                      expectedReturnStatus: false,
                      expectedReturnValues: [
                        REVERT_FLAGS.STATIC_VIOLATION,
                        '0x',
                        OVM_TX_GAS_LIMIT / 2,
                        0,
                      ],
                    },
                    {
                      functionName: 'ovmCREATE',
                      functionParams: [DUMMY_BYTECODE, false, []],
                      expectedReturnStatus: false,
                      expectedReturnValues: [
                        REVERT_FLAGS.STATIC_VIOLATION,
                        '0x',
                        OVM_TX_GAS_LIMIT / 2,
                        0,
                      ],
                    },
                    {
                      functionName: 'ovmSLOAD',
                      functionParams: [NON_NULL_BYTES32],
                      expectedReturnStatus: true,
                      expectedReturnValues: [NULL_BYTES32],
                    },
                    {
                      functionName: 'ovmCALLER',
                      functionParams: [],
                      expectedReturnStatus: true,
                      expectedReturnValues: ['$DUMMY_OVM_ADDRESS_1'],
                    },
                    {
                      functionName: 'ovmADDRESS',
                      functionParams: [],
                      expectedReturnStatus: true,
                      expectedReturnValues: ['$DUMMY_OVM_ADDRESS_2'],
                    },
                  ],
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [],
              },
            ],
          ],
          expectedReturnStatus: true,
          expectedReturnValues: [],
        },
      ],
    },
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            OVM_TX_GAS_LIMIT,
            '$DUMMY_OVM_ADDRESS_1',
            [
              {
                functionName: 'ovmSTATICCALL',
                functionParams: [
                  OVM_TX_GAS_LIMIT,
                  '$DUMMY_OVM_ADDRESS_2',
                  [
                    {
                      functionName: 'ovmCALL',
                      functionParams: [
                        OVM_TX_GAS_LIMIT,
                        '$DUMMY_OVM_ADDRESS_2',
                        [
                          {
                            functionName: 'ovmSLOAD',
                            functionParams: [NON_NULL_BYTES32],
                            expectedReturnStatus: true,
                            expectedReturnValues: [NULL_BYTES32],
                          },
                          {
                            functionName: 'ovmSSTORE',
                            functionParams: [NULL_BYTES32, NULL_BYTES32],
                            expectedReturnStatus: false,
                            expectedReturnValues: [
                              REVERT_FLAGS.STATIC_VIOLATION,
                              '0x',
                              867484476,
                              2906,
                            ],
                          },
                          {
                            functionName: 'ovmCREATE',
                            functionParams: [DUMMY_BYTECODE, false, []],
                            expectedReturnStatus: false,
                            expectedReturnValues: [
                              REVERT_FLAGS.STATIC_VIOLATION,
                              '0x',
                              867484476,
                              2906,
                            ],
                          },
                        ],
                      ],
                      expectedReturnStatus: true,
                      expectedReturnValues: [],
                    },
                  ],
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [],
              },
            ],
          ],
          expectedReturnStatus: true,
          expectedReturnValues: [],
        },
      ],
    },
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            OVM_TX_GAS_LIMIT,
            '$DUMMY_OVM_ADDRESS_1',
            [
              {
                functionName: 'ovmSTATICCALL',
                functionParams: [
                  OVM_TX_GAS_LIMIT / 2,
                  '$DUMMY_OVM_ADDRESS_2',
                  [
                    {
                      functionName: 'ovmSTATICCALL',
                      functionParams: [OVM_TX_GAS_LIMIT, '$DUMMY_OVM_ADDRESS_2', []],
                      expectedReturnStatus: true,
                      expectedReturnValues: [],
                    },
                    {
                      functionName: 'ovmSSTORE',
                      functionParams: [NULL_BYTES32, NULL_BYTES32],
                      expectedReturnStatus: false,
                      expectedReturnValues: [
                        REVERT_FLAGS.STATIC_VIOLATION,
                        '0x',
                        OVM_TX_GAS_LIMIT / 2,
                        33806,
                      ],
                    },
                  ],
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [],
              },
            ],
          ],
          expectedReturnStatus: true,
          expectedReturnValues: [],
        },
      ],
    },
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            OVM_TX_GAS_LIMIT,
            '$DUMMY_OVM_ADDRESS_1',
            [
              {
                functionName: 'ovmSTATICCALLToRevert',
                functionParams: [
                  OVM_TX_GAS_LIMIT / 2,
                  '$DUMMY_OVM_ADDRESS_2',
                  [REVERT_FLAGS.STATIC_VIOLATION, '0x', OVM_TX_GAS_LIMIT / 2, 0],
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [false, '0x'],
              },
            ],
          ],
          expectedReturnStatus: true,
          expectedReturnValues: [],
        },
      ],
    },
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            OVM_TX_GAS_LIMIT,
            '$DUMMY_OVM_ADDRESS_1',
            [
              {
                functionName: 'ovmSTATICCALL',
                functionParams: [
                  OVM_TX_GAS_LIMIT,
                  '$DUMMY_OVM_ADDRESS_1',
                  [
                    {
                      functionName: 'ovmSTATICCALLToRevert',
                      functionParams: [
                        OVM_TX_GAS_LIMIT / 2,
                        '$DUMMY_OVM_ADDRESS_2',
                        [REVERT_FLAGS.STATIC_VIOLATION, '0x', OVM_TX_GAS_LIMIT / 2, 0],
                      ],
                      expectedReturnStatus: true,
                      expectedReturnValues: [false, '0x'],
                    },
                  ],
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [],
              },
            ],
          ],
          expectedReturnStatus: true,
          expectedReturnValues: [],
        },
      ],
    },
  ],
}

runExecutionManagerTest(test_ovmSTATICCALL)
