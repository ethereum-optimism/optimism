import { ExecutionManagerTestRunner } from '../../../helpers/test-utils/test-runner'
import { TestDefinition } from '../../../helpers/test-utils/test.types2'
import {
  GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
  VERIFIED_EMPTY_CONTRACT_HASH,
} from '../../../helpers'

const CREATED_CONTRACT_1 = '0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb'
const DUMMY_REVERT_DATA =
  '0xdeadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420'

const test: TestDefinition = {
  name: 'An Example Test',
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
        },
      },
    },
  },
  subTests: [
    {
      name: 'An Example Subtest',
      parameters: [
        {
          name: 'An Example CALL revert test',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: GAS_LIMIT,
                target: '$DUMMY_OVM_ADDRESS_1',
                subSteps: [
                  {
                    functionName: 'ovmCALL',
                    functionParams: {
                      gasLimit: GAS_LIMIT,
                      target: '$DUMMY_OVM_ADDRESS_2',
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
              expectedReturnStatus: true,
            },
          ],
        },
        {
          name: 'An Example CREATE test',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: GAS_LIMIT,
                target: '$DUMMY_OVM_ADDRESS_1',
                subSteps: [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: {
                      subSteps: [
                        {
                          functionName: 'ovmCALLER',
                          expectedReturnStatus: true,
                          expectedReturnValue: '$DUMMY_OVM_ADDRESS_1',
                        },
                      ],
                    },
                    expectedReturnStatus: true,
                    expectedReturnValue: CREATED_CONTRACT_1,
                  },
                ],
              },
              expectedReturnStatus: true,
            },
          ],
        },
        {
          name: 'An Example CALL test',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: GAS_LIMIT,
                target: '$DUMMY_OVM_ADDRESS_1',
                subSteps: [
                  {
                    functionName: 'ovmADDRESS',
                    expectedReturnStatus: true,
                    expectedReturnValue: '$DUMMY_OVM_ADDRESS_1',
                  },
                  {
                    functionName: 'ovmCALL',
                    functionParams: {
                      gasLimit: GAS_LIMIT,
                      target: '$DUMMY_OVM_ADDRESS_2',
                      subSteps: [
                        {
                          functionName: 'ovmADDRESS',
                          expectedReturnStatus: true,
                          expectedReturnValue: '$DUMMY_OVM_ADDRESS_2',
                        },
                        {
                          functionName: 'ovmCALLER',
                          expectedReturnStatus: true,
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
      ],
    },
  ],
}

const runner = new ExecutionManagerTestRunner()
runner.run(test)
