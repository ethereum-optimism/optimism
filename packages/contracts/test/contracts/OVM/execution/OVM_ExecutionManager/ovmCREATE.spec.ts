/* Internal Imports */
import {
  runExecutionManagerTest,
  TestDefinition,
  GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
  ZERO_ADDRESS,
  VERIFIED_EMPTY_CONTRACT_HASH,
  DUMMY_BYTECODE_BYTELEN,
  DUMMY_BYTECODE_HASH,
} from '../../../../helpers'

const CREATED_CONTRACT_1 = '0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb'
const NESTED_CREATED_CONTRACT = '0xcb964b3f4162a0d4f5c997b40e19da5a546bc36f'

const test_ovmCREATE: TestDefinition = {
  name: 'Basic tests for ovmCREATE',
  preState: {
    ExecutionManager: {
      ovmStateManager: '$OVM_STATE_MANAGER',
      ovmSafetyChecker: '$OVM_SAFETY_CHECKER',
      messageRecord: {
        nuisanceGasLeft: GAS_LIMIT,
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
      },
    },
  },
  parameters: [
    {
      name:
        'Should correctly expose code-related opcodes for a created address once deployed',
      parameters: [
        {
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: [
                GAS_LIMIT / 2,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      DUMMY_BYTECODE,
                      // expect creation to succeed?
                      true,
                      [],
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [CREATED_CONTRACT_1],
                  },
                  {
                    functionName: 'ovmEXTCODESIZE',
                    functionParams: [CREATED_CONTRACT_1],
                    expectedReturnStatus: true,
                    expectedReturnValues: [DUMMY_BYTECODE_BYTELEN],
                  },
                  {
                    functionName: 'ovmEXTCODEHASH',
                    functionParams: [CREATED_CONTRACT_1],
                    expectedReturnStatus: true,
                    expectedReturnValues: [DUMMY_BYTECODE_HASH],
                  },
                  {
                    functionName: 'ovmEXTCODECOPY',
                    functionParams: [
                      CREATED_CONTRACT_1,
                      0,
                      DUMMY_BYTECODE_BYTELEN,
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [DUMMY_BYTECODE_HASH],
                  },
                ],
              ],
              expectedReturnStatus: true,
              expectedReturnValues: [],
            },
          ],
        },
      ],
    },
    {
      name:
        'Should return 0 address correctly expose empty code-related opcodes if deployment fails',
      parameters: [
        {
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: [
                GAS_LIMIT / 2,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      DUMMY_BYTECODE,
                      // expect creation to succeed?
                      false,
                      [
                        {
                          functionName: 'ovmREVERT',
                          functionParams: ['0x1234'],
                          expectedReturnStatus: undefined, // TODO: use this wherever not checked
                          expectedReturnValues: undefined,
                        },
                      ],
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [ZERO_ADDRESS],
                  },
                  {
                    functionName: 'ovmEXTCODESIZE',
                    functionParams: [CREATED_CONTRACT_1],
                    expectedReturnStatus: true,
                    expectedReturnValues: [0],
                  },
                  {
                    functionName: 'ovmEXTCODEHASH',
                    functionParams: [CREATED_CONTRACT_1],
                    expectedReturnStatus: true,
                    expectedReturnValues: [NULL_BYTES32],
                  },
                  {
                    functionName: 'ovmEXTCODECOPY',
                    functionParams: [CREATED_CONTRACT_1, 0, 256],
                    expectedReturnStatus: true,
                    expectedReturnValues: ['0x' + '00'.repeat(256)],
                  },
                ],
              ],
              expectedReturnStatus: true,
              expectedReturnValues: [],
            },
          ],
        },
      ],
    },
    {
      name: 'Basic relevant context opcodes should be accessible in initcode',
      parameters: [
        {
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: [
                GAS_LIMIT,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      // code to deploy:
                      DUMMY_BYTECODE,
                      // expect creation to succeed?
                      true,
                      // steps for initcode:
                      [
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
                          expectedReturnValues: [CREATED_CONTRACT_1],
                        },
                        {
                          functionName: 'ovmSLOAD',
                          functionParams: [NON_NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [NULL_BYTES32],
                        },
                      ],
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [CREATED_CONTRACT_1],
                  },
                ],
              ],
              expectedReturnStatus: true,
              expectedReturnValues: [],
            },
          ],
        },
      ],
    },
    {
      name:
        'Internal storage manipulation during initcode should be correctly persisted, and all accessible',
      parameters: [
        {
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: [
                GAS_LIMIT,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      // code to deploy:
                      '$OVM_CALL_HELPER_CODE',
                      // expect creation to succeed?
                      true,
                      // steps for initcode:
                      [
                        {
                          functionName: 'ovmSSTORE',
                          functionParams: [NON_NULL_BYTES32, NON_NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [],
                        },
                        {
                          functionName: 'ovmSLOAD',
                          functionParams: [NON_NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [NON_NULL_BYTES32],
                        },
                      ],
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [CREATED_CONTRACT_1],
                  },
                  {
                    functionName: 'ovmCALL',
                    functionParams: [
                      GAS_LIMIT,
                      CREATED_CONTRACT_1,
                      [
                        {
                          functionName: 'ovmSLOAD',
                          functionParams: [NON_NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [NON_NULL_BYTES32],
                        },
                        {
                          functionName: 'ovmSLOAD',
                          functionParams: [NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [NULL_BYTES32],
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
    },
    {
      name:
        'External storage manipulation during initcode subcalls should correctly be persisted',
      parameters: [
        {
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: [
                GAS_LIMIT,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      // code to deploy:
                      '$OVM_CALL_HELPER_CODE',
                      // expect creation to succeed?
                      true,
                      // steps for initcode:
                      [
                        {
                          functionName: 'ovmCALL',
                          functionParams: [
                            GAS_LIMIT,
                            '$DUMMY_OVM_ADDRESS_2',
                            [
                              {
                                functionName: 'ovmSSTORE',
                                functionParams: [
                                  NULL_BYTES32,
                                  NON_NULL_BYTES32,
                                ],
                                expectedReturnStatus: true,
                                expectedReturnValues: [],
                              },
                              {
                                functionName: 'ovmSLOAD',
                                functionParams: [NULL_BYTES32],
                                expectedReturnStatus: true,
                                expectedReturnValues: [NON_NULL_BYTES32],
                              },
                            ],
                          ],
                          expectedReturnStatus: true,
                          expectedReturnValues: [],
                        },
                      ],
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [CREATED_CONTRACT_1],
                  },
                  {
                    functionName: 'ovmCALL',
                    functionParams: [
                      GAS_LIMIT,
                      '$DUMMY_OVM_ADDRESS_2',
                      [
                        {
                          functionName: 'ovmSLOAD',
                          functionParams: [NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [NON_NULL_BYTES32],
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
    },
    {
      name:
        'External storage manipulation during initcode subcalls should correctly NOT be persisted if ovmREVERTed',
      preState: {
        StateManager: {
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
          },
          verifiedContractStorage: {
            $DUMMY_OVM_ADDRESS_2: {
              [NULL_BYTES32]: true,
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
                GAS_LIMIT / 2,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      // code to deploy:
                      '$OVM_CALL_HELPER_CODE',
                      // expect ovmCREATE to successfully deploy?
                      false,
                      // steps for initcode:
                      [
                        {
                          functionName: 'ovmCALL',
                          functionParams: [
                            GAS_LIMIT,
                            '$DUMMY_OVM_ADDRESS_2',
                            [
                              {
                                functionName: 'ovmSSTORE',
                                functionParams: [
                                  NULL_BYTES32,
                                  NON_NULL_BYTES32,
                                ],
                                expectedReturnStatus: true,
                                expectedReturnValues: [],
                              },
                            ],
                          ],
                          expectedReturnStatus: true,
                          expectedReturnValues: [],
                        },
                        {
                          functionName: 'ovmREVERT',
                          functionParams: ['0xdeadbeef'],
                          expectedReturnStatus: true,
                          expectedReturnValues: [], // technically will return 1 single byte but impossible to assert
                        },
                      ],
                    ],
                    expectedReturnStatus: true,
                    expectedReturnValues: [ZERO_ADDRESS],
                  },
                  {
                    functionName: 'ovmCALL',
                    functionParams: [
                      GAS_LIMIT,
                      '$DUMMY_OVM_ADDRESS_2',
                      [
                        {
                          functionName: 'ovmSLOAD',
                          functionParams: [NULL_BYTES32],
                          expectedReturnStatus: true,
                          expectedReturnValues: [NULL_BYTES32],
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
    },
    {
      name:
        'Should correctly revert on invalid state access in initcode made by a call',
      preState: {
        StateManager: {
          accounts: {
            $DUMMY_OVM_ADDRESS_1: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [CREATED_CONTRACT_1]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
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
                GAS_LIMIT / 2,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      // code to deploy:
                      '$OVM_CALL_HELPER_CODE',
                      // expect ovmCREATE to successfully deploy?
                      false,
                      // steps for initcode:
                      [
                        {
                          functionName: 'ovmCALL',
                          functionParams: [
                            GAS_LIMIT,
                            '$DUMMY_OVM_ADDRESS_3', // invalid state access, not in prestate.SM.accounts
                            [],
                          ],
                          expectedReturnStatus: undefined,
                          expectedReturnValues: undefined,
                        },
                      ],
                    ],
                    expectedReturnStatus: false,
                    expectedReturnValues: [
                      REVERT_FLAGS.INVALID_STATE_ACCESS,
                      '0x',
                      476756501,
                      0,
                    ],
                  },
                ],
              ],
              // note: this would be false in practice, but our code contracts are unsafe, so they do not enforce propagation of ISA flag.
              expectedReturnStatus: true,
              expectedReturnValues: [],
            },
          ],
        },
      ],
    },
    {
      name: 'Invalid state access on nested CREATE should be surfaced',
      preState: {
        StateManager: {
          accounts: {
            $DUMMY_OVM_ADDRESS_1: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [CREATED_CONTRACT_1]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
            },
            [NESTED_CREATED_CONTRACT]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
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
                GAS_LIMIT / 2,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: [
                      // code to deploy:
                      '$OVM_CALL_HELPER_CODE',
                      // expect ovmCREATE to successfully deploy?
                      false,
                      // steps for initcode:
                      [
                        {
                          functionName: 'ovmCREATE',
                          functionParams: [
                            // code to deploy:
                            '$OVM_CALL_HELPER_CODE',
                            // expect ovmCREATE to successfully deploy?
                            false,
                            // steps for initcode:
                            [
                              {
                                functionName: 'ovmCALL',
                                functionParams: [
                                  GAS_LIMIT,
                                  '$DUMMY_OVM_ADDRESS_3', // invalid state access, not in prestate.SM.accounts
                                  [],
                                ],
                                expectedReturnStatus: undefined,
                                expectedReturnValues: undefined,
                              },
                            ],
                          ],
                          expectedReturnStatus: undefined,
                          expectedReturnValues: undefined,
                        },
                      ],
                    ],
                    expectedReturnStatus: false,
                    expectedReturnValues: [
                      REVERT_FLAGS.INVALID_STATE_ACCESS,
                      '0x',
                      476709610,
                      0,
                    ],
                  },
                ],
              ],
              // note: this would be false in practice, but our code contracts are unsafe, so they do not enforce propagation of ISA flag.
              expectedReturnStatus: true,
              expectedReturnValues: [],
            },
          ],
        },
      ],
    },
    {
      name: 'CREATE should fail and return 0 address if out of gas',
      focus: true,
      preState: {
        StateManager: {
          accounts: {
            $DUMMY_OVM_ADDRESS_1: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [CREATED_CONTRACT_1]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
            },
            [NESTED_CREATED_CONTRACT]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
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
                GAS_LIMIT / 2,
                '$DUMMY_OVM_ADDRESS_1',
                [
                  {
                    functionName: 'ovmCREATEToInvalid',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [ZERO_ADDRESS],
                  },
                ],
              ],
              // note: this would be false in practice, but our code contracts are unsafe, so they do not enforce propagation of ISA flag.
              expectedReturnStatus: true,
              expectedReturnValues: [],
            },
          ],
        },
      ],
    },
  ],
}

runExecutionManagerTest(test_ovmCREATE)
