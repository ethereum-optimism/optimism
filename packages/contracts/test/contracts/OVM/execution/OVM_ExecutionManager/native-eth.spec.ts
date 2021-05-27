/* Internal Imports */
import { remove0x, toHexString } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'
import { predeploys } from '../../../../../src'
import {
  ExecutionManagerTestRunner,
  TestDefinition,
  OVM_TX_GAS_LIMIT,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  VERIFIED_EMPTY_CONTRACT_HASH,
} from '../../../../helpers'

const uniswapERC20BalanceOfStorageLayoutKey = '0000000000000000000000000000000000000000000000000000000000000005'
// TODO: use fancy chugsplash storage getter once possible
const getOvmEthBalanceSlot = (addressOrPlaceholder: string): string => {
  let address: string
  if (addressOrPlaceholder.startsWith('$DUMMY_OVM_ADDRESS_')) {
    address = ExecutionManagerTestRunner.getDummyAddress(addressOrPlaceholder)
  } else {
    address = addressOrPlaceholder
  }
  const balanceOfSlotPreimage = ethers.utils.hexZeroPad(address, 32) + uniswapERC20BalanceOfStorageLayoutKey
  const balanceOfSlot = ethers.utils.keccak256(balanceOfSlotPreimage)
  return balanceOfSlot
}

const INITIAL_BALANCE = 1234
const CALL_VALUE = 69

const test_nativeETH: TestDefinition = {
  name: 'Basic tests for ovmCALL',
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
      contractStorage: {
        [predeploys.OVM_ETH]: {
          [getOvmEthBalanceSlot('$DUMMY_OVM_ADDRESS_1')]: {
            getStorageXOR: true,
            value: toHexString(INITIAL_BALANCE),
          },
          [getOvmEthBalanceSlot('$DUMMY_OVM_ADDRESS_2')]: {
            getStorageXOR: true,
            value: '0x00',
          },
        },
      },
      verifiedContractStorage: {
        [predeploys.OVM_ETH]: {
          [getOvmEthBalanceSlot('$DUMMY_OVM_ADDRESS_1')]: true,
          [getOvmEthBalanceSlot('$DUMMY_OVM_ADDRESS_2')]: true,
        },
      },
    },
  },
  parameters: [
    {
      name: 'ovmCALL(ADDRESS_1) => ovmBALANCE(ADDRESS_1)',
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_1',
                },
                expectedReturnStatus: true,
                expectedReturnValue: INITIAL_BALANCE,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name:
        'ovmCALL(ADDRESS_1) => ovmCALL(ADDRESS_2, value) [successful call]',
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              // expected initial balances:
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_1'
                },
                expectedReturnStatus: true,
                expectedReturnValue: INITIAL_BALANCE
              },
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_2'
                },
                expectedReturnStatus: true,
                expectedReturnValue: 0
              },
              // do the call with some value
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_2',
                  value: CALL_VALUE,
                  subSteps: [
                    // check that the ovmCALLVALUE is updated
                    {
                      functionName: 'ovmCALLVALUE',
                      expectedReturnValue: CALL_VALUE
                    },
                    // check that the balances have been updated
                    {
                      functionName: 'ovmBALANCE',
                      functionParams: {
                        address: '$DUMMY_OVM_ADDRESS_1'
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: INITIAL_BALANCE - CALL_VALUE
                    },
                    {
                      functionName: 'ovmBALANCE',
                      functionParams: {
                        address: '$DUMMY_OVM_ADDRESS_2'
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: CALL_VALUE
                    },
                  ],
                },
                expectedReturnStatus: true,
              },
              // check that the ovmCALLVALUE is reset back to 0
              {
                functionName: 'ovmCALLVALUE',
                expectedReturnValue: 0
              },
              // check that the balances have persisted
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_1'
                },
                expectedReturnStatus: true,
                expectedReturnValue: INITIAL_BALANCE - CALL_VALUE
              },
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_2'
                },
                expectedReturnStatus: true,
                expectedReturnValue: CALL_VALUE
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name:
        'ovmCALL(ADDRESS_1) => ovmCALL(ADDRESS_2, value) [reverting call]',
      focus: true,
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              // expected initial balances:
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_1'
                },
                expectedReturnStatus: true,
                expectedReturnValue: INITIAL_BALANCE
              },
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_2'
                },
                expectedReturnStatus: true,
                expectedReturnValue: 0
              },
              // do the call with some value
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_2',
                  value: CALL_VALUE,
                  subSteps: [
                    // check that the ovmCALLVALUE is updated
                    {
                      functionName: 'ovmCALLVALUE',
                      expectedReturnValue: CALL_VALUE
                    },
                    // check that the balances have been updated
                    {
                      functionName: 'ovmBALANCE',
                      functionParams: {
                        address: '$DUMMY_OVM_ADDRESS_1'
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: INITIAL_BALANCE - CALL_VALUE
                    },
                    {
                      functionName: 'ovmBALANCE',
                      functionParams: {
                        address: '$DUMMY_OVM_ADDRESS_2'
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: CALL_VALUE
                    },
                    // now revert everything
                    {
                      functionName: 'ovmREVERT',
                      expectedReturnStatus: false,
                      expectedReturnValue: {
                        flag: REVERT_FLAGS.INTENTIONAL_REVERT,
                        onlyValidateFlag: true
                      }
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: {
                  ovmSuccess: false,
                  returnData: '0x'
                }
              },
              // check that the ovmCALLVALUE is reset back to 0
              {
                functionName: 'ovmCALLVALUE',
                expectedReturnValue: 0
              },
              // check that the balances have NOT persisted
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_1'
                },
                expectedReturnStatus: true,
                expectedReturnValue: INITIAL_BALANCE
              },
              {
                functionName: 'ovmBALANCE',
                functionParams: {
                  address: '$DUMMY_OVM_ADDRESS_2'
                },
                expectedReturnStatus: true,
                expectedReturnValue: 0
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
  ],
}

const runner = new ExecutionManagerTestRunner()
runner.run(test_nativeETH)
