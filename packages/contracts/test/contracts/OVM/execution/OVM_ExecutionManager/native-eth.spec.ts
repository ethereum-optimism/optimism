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
        },
      },
      verifiedContractStorage: {
        [predeploys.OVM_ETH]: {
          [getOvmEthBalanceSlot('$DUMMY_OVM_ADDRESS_1')]: true,
        },
      },
    },
  },
  parameters: [
    {
      name: 'ovmCALL(ADDRESS_1) => ovmBALANCE(ADDRESS_1)',
      focus: true,
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
  ],
}

const runner = new ExecutionManagerTestRunner()
runner.run(test_nativeETH)
