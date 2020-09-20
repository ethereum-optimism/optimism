/* Internal Imports */
import {
  runExecutionManagerTest,
  TestDefinition,
  GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
  VERIFIED_EMPTY_CONTRACT_HASH
} from '../../../../helpers'


const DUMMY_REVERT_DATA = "0xdeadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420"

const test_ovmCALL: TestDefinition = {
  name: "Basic tests for ovmCALL",
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
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20)
        }
      }
    }
  },
  parameters: [
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            GAS_LIMIT / 2,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALL',
                functionParams: [
                  GAS_LIMIT / 2,
                  "$DUMMY_OVM_ADDRESS_2",
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
            GAS_LIMIT / 2,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALL',
                functionParams: [
                  GAS_LIMIT / 2,
                  "$DUMMY_OVM_ADDRESS_2",
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
            GAS_LIMIT / 2,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALL',
                functionParams: [
                  GAS_LIMIT / 2,
                  "$DUMMY_OVM_ADDRESS_3",
                  []
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [true, "0x"]
              },
            ]
          ],
          expectedReturnStatus: true,
          expectedReturnValues: []
        }
      ]
    }
  ]
}

const test_ovmCALL_revert: TestDefinition = {
  name: "Basic reverts in a code contract called via ovmCALL",
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
          ethAddress: "$OVM_REVERT_HELPER"
        },
        "$DUMMY_OVM_ADDRESS_3": {
          codeHash: NON_NULL_BYTES32,
          ethAddress: "$OVM_REVERT_HELPER"
        },
        "$DUMMY_OVM_ADDRESS_4": {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20)
        }
      }
    }
  },
  parameters: [
    {
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: [
            GAS_LIMIT / 2,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALLToRevert',
                functionParams: [
                  GAS_LIMIT / 2,
                  "$DUMMY_OVM_ADDRESS_2",
                  [
                    REVERT_FLAGS.INTENTIONAL_REVERT,
                    DUMMY_REVERT_DATA,
                    GAS_LIMIT / 2,
                    0
                  ]
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [false, DUMMY_REVERT_DATA]
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
            GAS_LIMIT / 2,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmCALLToRevert',
                functionParams: [
                  GAS_LIMIT / 2,
                  "$DUMMY_OVM_ADDRESS_2",
                  [
                    REVERT_FLAGS.EXCEEDS_NUISANCE_GAS,
                    "0x",
                    0,
                    0
                  ]
                ],
                expectedReturnStatus: true,
                expectedReturnValues: [false, "0x"]
              },
            ]
          ],
          expectedReturnStatus: true,
          expectedReturnValues: []
        },
      ]
    },
  ]
}

runExecutionManagerTest(test_ovmCALL)
runExecutionManagerTest(test_ovmCALL_revert)
