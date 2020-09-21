/* Internal Imports */
import {
  runExecutionManagerTest,
  TestDefinition,
  GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
} from '../../../../helpers'

const test_ovmREVERT: TestDefinition = {
  name: 'basic ovmREVERT unit tests',
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
      },
    },
  },
  parameters: [
    {
      name: 'ovmREVERT inside ovmCALL should cause EM to revert',
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
                    functionName: 'ovmREVERT',
                    functionParams: ['0xdeadbeef'],
                    expectedReturnStatus: false,
                    expectedReturnValues: [
                      REVERT_FLAGS.INTENTIONAL_REVERT,
                      '0xdeadbeef',
                      GAS_LIMIT / 2,
                      0,
                    ],
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
    // TODO: fix this.  only way to do it is manually set up and call ovmREVERT directly inside a context which mirrors that during creation.
    // {
    //   name: "ovmREVERT inside ovmCREATE ?",
    //   parameters: [
    //     {
    //       steps: [
    //         {
    //           functionName: "ovmCALL",
    //           functionParams: [
    //             GAS_LIMIT / 2,
    //             "$DUMMY_OVM_ADDRESS_1",
    //             [
    //               {
    //                 functionName: "ovmCREATE",
    //                 functionParams: [
    //                   USELESS_BYTECODE,
    //                   false, // "create will be successful?"
    //                   [
    //                     {
    //                       functionName: "ovmREVERT",
    //                       functionParams: [ "0xdeadbeef" ],
    //                       expectedReturnStatus: false,
    //                       expectedReturnValues: [ "0x00" ] // no return values for reversion in constructor
    //                     },
    //                     // TODO: check internally flagged storage here
    //                   ]
    //                 ],
    //                 expectedReturnStatus: true,
    //                 expectedReturnValues: [ CREATED_CONTRACT_1 ]
    //               }
    //             ],
    //           ],
    //           expectedReturnStatus: true,
    //           expectedReturnValues: []
    //         }
    //       ]
    //     }
    //   ]
    // }
  ],
}

runExecutionManagerTest(test_ovmREVERT)
