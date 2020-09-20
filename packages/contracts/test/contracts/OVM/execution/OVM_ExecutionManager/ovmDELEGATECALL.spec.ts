/* Internal Imports */
import {
  runExecutionManagerTest,
  TestDefinition,
  GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE
} from '../../../../helpers'

const CREATED_CONTRACT_1 = "0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb"
const CREATED_CONTRACT_2 = "0xe0d8be8101f36ebe6b01abacec884422c39a1f62"

const test_ovmDELEGATECALL: TestDefinition = {
  name: "Basic tests for ovmDELEGATECALL",
  preState: {
    ExecutionManager: {
      ovmStateManager: "$OVM_STATE_MANAGER",
      ovmSafetyChecker: "$OVM_SAFETY_CHECKER",
      messageRecord: {
        nuisanceGasLeft: GAS_LIMIT
      }
    },
    StateManager: {
      owner: "$OVM_EXECUTION_MANAGER",
      accounts: {
        "$DUMMY_OVM_ADDRESS_1": {
          codeHash: NON_NULL_BYTES32,
          ethAddress: "$OVM_CALL_HELPER"
        },
        "$DUMMY_OVM_ADDRESS_2": {
          codeHash: NON_NULL_BYTES32,
          ethAddress: "$OVM_CALL_HELPER"
        },
        "$DUMMY_OVM_ADDRESS_3": {
          codeHash: NON_NULL_BYTES32,
          ethAddress: "$OVM_CALL_HELPER"
        },
        "$DUMMY_OVM_ADDRESS_4": {
          codeHash: NON_NULL_BYTES32,
          ethAddress: "$OVM_CALL_HELPER"
        },
      }
    }
  },
  parameters: [
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            GAS_LIMIT,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALL',
                functionParams: [
                  GAS_LIMIT,
                  "$DUMMY_OVM_ADDRESS_2",
                  [
                    {
                      functionName: 'ovmDELEGATECALL',
                      functionParams: [
                        GAS_LIMIT,
                        "$DUMMY_OVM_ADDRESS_3",
                        [
                          {
                            functionName: 'ovmCALLER',
                            functionParams: [],
                            expectedReturnStatus: true,
                            expectedReturnValues: ["$DUMMY_OVM_ADDRESS_1"]
                          },
                          {
                            functionName: 'ovmADDRESS',
                            functionParams: [],
                            expectedReturnStatus: true,
                            expectedReturnValues: ["$DUMMY_OVM_ADDRESS_2"]
                          },
                          {
                            functionName: 'ovmSSTORE',
                            functionParams: [ NON_NULL_BYTES32, NON_NULL_BYTES32 ],
                            expectedReturnStatus: true,
                            expectedReturnValues: []
                          },
                          {
                            functionName: 'ovmSLOAD',
                            functionParams: [ NON_NULL_BYTES32 ],
                            expectedReturnStatus: true,
                            expectedReturnValues: [NON_NULL_BYTES32]
                          }
                        ]
                      ],
                      expectedReturnStatus: true,
                      expectedReturnValues: ["$DUMMY_OVM_ADDRESS_1"]
                    },
                  ]
                ],
                expectedReturnStatus: true,
                expectedReturnValues: []
              },
              {
                functionName: 'ovmCALL',
                functionParams: [
                  GAS_LIMIT / 2,
                  "$DUMMY_OVM_ADDRESS_2",
                  [
                    {
                      functionName: 'ovmSLOAD',
                      functionParams: [ NON_NULL_BYTES32 ],
                      expectedReturnStatus: true,
                      expectedReturnValues: [NON_NULL_BYTES32]
                    }
                  ]
                ],
                expectedReturnStatus: true,
                expectedReturnValues: []
              },
            ]
          ],
          expectedReturnStatus: true,
          expectedReturnValues: []
        }
      ]
    },
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            GAS_LIMIT,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmDELEGATECALL',
                functionParams: [
                  GAS_LIMIT,
                  "$DUMMY_OVM_ADDRESS_2",
                  [
                    {
                      functionName: 'ovmCREATE',
                      functionParams: [
                        DUMMY_BYTECODE,
                        true,
                        []
                      ],
                      expectedReturnStatus: true,
                      expectedReturnValues: [CREATED_CONTRACT_1]
                    },
                  ]
                ],
                expectedReturnStatus: true,
                expectedReturnValues: []
              },
            ]
          ],
          expectedReturnStatus: true,
          expectedReturnValues: []
        }
      ]
    },
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            GAS_LIMIT,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALL',
                functionParams: [
                  GAS_LIMIT,
                  "$DUMMY_OVM_ADDRESS_2",
                  [
                    {
                      functionName: 'ovmDELEGATECALL',
                      functionParams: [
                        GAS_LIMIT,
                        "$DUMMY_OVM_ADDRESS_3",
                        [
                          {
                            functionName: "ovmDELEGATECALL",
                            functionParams: [
                              GAS_LIMIT,
                              "$DUMMY_OVM_ADDRESS_4",
                              [
                                {
                                  functionName: "ovmCALLER",
                                  functionParams: [],
                                  expectedReturnStatus: true,
                                  expectedReturnValues: [ "$DUMMY_OVM_ADDRESS_1" ]
                                },
                                {
                                  functionName: "ovmADDRESS",
                                  functionParams: [],
                                  expectedReturnStatus: true,
                                  expectedReturnValues: [ "$DUMMY_OVM_ADDRESS_2" ]
                                },
                                {
                                  functionName: 'ovmCREATE',
                                  functionParams: [
                                    DUMMY_BYTECODE,
                                    true,
                                    []
                                  ],
                                  expectedReturnStatus: true,
                                  expectedReturnValues: [CREATED_CONTRACT_2]
                                }
                             ]
                            ],
                            expectedReturnStatus: true,
                            expectedReturnValues: []
                          }
                        ]
                      ],
                      expectedReturnStatus: true,
                      expectedReturnValues: []
                    },
                  ]
                ],
                expectedReturnStatus: true,
                expectedReturnValues: []
              },
              {
                functionName: "ovmADDRESS",
                functionParams: [],
                expectedReturnStatus: true,
                expectedReturnValues: [ "$DUMMY_OVM_ADDRESS_1" ]
              }
            ]
          ],
          expectedReturnStatus: true,
          expectedReturnValues: []
        }
      ]
    }
  ]
}

runExecutionManagerTest(test_ovmDELEGATECALL)
