/* Internal Imports */
import { runExecutionManagerTest } from '../../../helpers/test-utils/test-parsing'
import { NON_NULL_BYTES32, GAS_LIMIT } from '../../../helpers'

runExecutionManagerTest(
  {
    name: 'Top level test',
    preState: {
      ExecutionManager: {
        ovmStateManager: "$OVM_STATE_MANAGER",
        messageRecord: {
          nuisanceGasLeft: GAS_LIMIT / 2
        }
      },
      StateManager: {
        accounts: {
          "$DUMMY_OVM_ADDRESS_1": {
            codeHash: NON_NULL_BYTES32,
            ethAddress: "$OVM_CALL_HELPER"
          },
          "$DUMMY_OVM_ADDRESS_2": {
            codeHash: NON_NULL_BYTES32,
            ethAddress: "$OVM_CALL_HELPER"
          }
        }
      }
    },
    parameters: [
      {
        name: 'Do two ovmCALLs to one address',
        postState: {
          ExecutionManager: {
            messageRecord: {
              nuisanceGasLeft: GAS_LIMIT / 2 - (332) * 100 * 1
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
                        "$DUMMY_OVM_ADDRESS_1",
                        [
                          {
                            functionName: 'ovmADDRESS',
                            functionParams: [],
                            returnStatus: true,
                            returnValues: ["$DUMMY_OVM_ADDRESS_1"]
                          },
                          {
                            functionName: 'ovmCALLER',
                            functionParams: [],
                            returnStatus: true,
                            returnValues: ["$DUMMY_OVM_ADDRESS_1"]
                          }
                        ]
                      ],
                      returnStatus: true,
                      returnValues: []
                    },
                  ]
                ],
                returnStatus: true,
                returnValues: []
              }
            ]
          }
        ]
      },
      {
        name: 'Do two ovmCALLs to two different addresses',
        postState: {
          ExecutionManager: {
            messageRecord: {
              nuisanceGasLeft: GAS_LIMIT / 2 - (332) * 100 * 2
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
                            functionName: 'ovmADDRESS',
                            functionParams: [],
                            returnStatus: true,
                            returnValues: ["$DUMMY_OVM_ADDRESS_2"]
                          },
                          {
                            functionName: 'ovmCALLER',
                            functionParams: [],
                            returnStatus: true,
                            returnValues: ["$DUMMY_OVM_ADDRESS_1"]
                          }
                        ]
                      ],
                      returnStatus: true,
                      returnValues: []
                    },
                  ]
                ],
                returnStatus: true,
                returnValues: []
              }
            ]
          }
        ]
      }
    ]
  }
)
