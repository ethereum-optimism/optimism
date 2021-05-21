/* External Imports */
import { ethers } from 'ethers'
// import { merge } from 'lodash'

/* Internal Imports */
import {
  ExecutionManagerTestRunner,
  TestDefinition,
  OVM_TX_GAS_LIMIT,
  NON_NULL_BYTES32,
  getStorageXOR,
  REVERT_FLAGS,
} from '../../../../helpers'
import { predeploys } from '../../../../../src/predeploys'

const UPGRADE_EXECUTOR_ADDRESS = predeploys.L2ChugSplashDeployer
const UPGRADED_ADDRESS = '0x1234123412341234123412341234123412341234'
const UPGRADED_CODE = '0x1234'
const UPGRADED_CODEHASH = ethers.utils.keccak256(UPGRADED_CODE)

const sharedPreState = {
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
      [UPGRADE_EXECUTOR_ADDRESS]: {
        codeHash: NON_NULL_BYTES32,
        ethAddress: '$OVM_CALL_HELPER',
      },
    },
    contractStorage: {
      [UPGRADED_ADDRESS]: {
        [NON_NULL_BYTES32]: getStorageXOR(ethers.constants.HashZero),
      },
    },
  },
}

const verifiedUpgradePreState = {
  StateManager: {
    accounts: {
      [UPGRADED_ADDRESS]: {
        codeHash: NON_NULL_BYTES32,
        ethAddress: '$OVM_CALL_HELPER',
      },
    },
    // verifiedContractStorage: {
    //   [UPGRADED_ADDRESS]: {
    //     [NON_NULL_BYTES32]: true,
    //   },
    // },
  },
}

const test_ovmSETCODEFunctionality: TestDefinition = {
  name: 'Functionality tests for ovmSETCODE',
  preState: sharedPreState,
  subTests: [
    {
      name: 'ovmSETCODE -- success case',
      preState: verifiedUpgradePreState,
      postState: {
        StateManager: {
          accounts: {
            [UPGRADED_ADDRESS]: {
              codeHash: UPGRADED_CODEHASH,
              ethAddress: '$OVM_CALL_HELPER',
            },
          },
        },
      },
      parameters: [
        {
          name: 'success case',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: OVM_TX_GAS_LIMIT,
                target: UPGRADE_EXECUTOR_ADDRESS,
                subSteps: [
                  {
                    functionName: 'ovmSETCODE',
                    functionParams: {
                      address: UPGRADED_ADDRESS,
                      code: UPGRADED_CODE,
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
    {
      name: 'ovmSETCODE -- unauthorized case',
      preState: verifiedUpgradePreState,
      parameters: [
        {
          name: 'unauthorized case',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: OVM_TX_GAS_LIMIT,
                target: '$DUMMY_OVM_ADDRESS_1',
                subSteps: [
                  {
                    functionName: 'ovmSETCODE',
                    functionParams: {
                      address: UPGRADED_ADDRESS,
                      code: UPGRADED_CODE,
                    },
                    expectedReturnStatus: false,
                    expectedReturnValue: {
                      flag: REVERT_FLAGS.CALLER_NOT_ALLOWED,
                      onlyValidateFlag: true,
                    },
                  },
                ],
              },
              expectedReturnStatus: false,
            },
          ],
        },
      ],
    },
  ],
}

const test_ovmSETCODEAccess: TestDefinition = {
  name: 'State access compliance tests for ovmSETCODE',
  preState: sharedPreState,
  subTests: [
    {
      name: 'ovmSETSTORAGE (UNVERIFIED_ACCOUNT)',
      parameters: [
        {
          name: 'ovmSETSTORAGE with a missing account',
          expectInvalidStateAccess: true,
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: OVM_TX_GAS_LIMIT,
                target: UPGRADE_EXECUTOR_ADDRESS,
                subSteps: [
                  {
                    functionName: 'ovmSETCODE',
                    functionParams: {
                      address: UPGRADED_ADDRESS,
                      code: UPGRADED_CODE,
                    },
                    expectedReturnStatus: false,
                    expectedReturnValue: {
                      flag: REVERT_FLAGS.INVALID_STATE_ACCESS,
                      onlyValidateFlag: true,
                    },
                  },
                ],
              },
              expectedReturnStatus: false,
              expectedReturnValue: {
                flag: REVERT_FLAGS.INVALID_STATE_ACCESS,
                onlyValidateFlag: true,
              },
            },
          ],
        },
      ],
    },
  ],
}
const runner = new ExecutionManagerTestRunner()
runner.run(test_ovmSETCODEFunctionality)
runner.run(test_ovmSETCODEAccess)
