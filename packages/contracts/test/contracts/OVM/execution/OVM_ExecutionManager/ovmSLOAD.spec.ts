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

const test_ovmSLOAD: TestDefinition = {
  name: "External storage manipulation during initcode subcalls should correctly NOT be persisted if ovmREVERTed",
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
      },
      verifiedContractStorage: {
        "$DUMMY_OVM_ADDRESS_1": {
          [NON_NULL_BYTES32]: true
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
            GAS_LIMIT,
            "$DUMMY_OVM_ADDRESS_1",
            [
              {
                functionName: 'ovmSLOAD',
                functionParams: [NON_NULL_BYTES32],
                expectedReturnStatus: true,
                expectedReturnValues: [NULL_BYTES32]
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

runExecutionManagerTest(test_ovmSLOAD)
