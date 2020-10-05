/* Internal Imports */
import {
  ExecutionManagerTestRunner,
  TestDefinition,
  OVM_TX_GAS_LIMIT,
  NON_NULL_BYTES32,
  ZERO_ADDRESS,
  VERIFIED_EMPTY_CONTRACT_HASH,
} from '../../../../helpers'

const GAS_METADATA_ADDRESS = '0x06a506a506a506a506a506a506a506a506a506a5'

enum GasMetadataKey {
  CURRENT_EPOCH_START_TIMESTAMP,
  CUMULATIVE_SEQUENCER_QUEUE_GAS,
  CUMULATIVE_L1TOL2_QUEUE_GAS,
  PREV_EPOCH_SEQUENCER_QUEUE_GAS,
  PREV_EPOCH_L1TOL2_QUEUE_GAS,
}

const keyToBytes32 = (key: GasMetadataKey): string => {
  return '0x' + `0${key}`.padStart(64, '0')
}

const test_run: TestDefinition = {
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
        [GAS_METADATA_ADDRESS]: {
          [keyToBytes32(GasMetadataKey.CURRENT_EPOCH_START_TIMESTAMP)]: 1,
          [keyToBytes32(GasMetadataKey.CUMULATIVE_SEQUENCER_QUEUE_GAS)]: 0,
          [keyToBytes32(GasMetadataKey.CUMULATIVE_L1TOL2_QUEUE_GAS)]: 0,
          [keyToBytes32(GasMetadataKey.PREV_EPOCH_SEQUENCER_QUEUE_GAS)]: 0,
          [keyToBytes32(GasMetadataKey.PREV_EPOCH_L1TOL2_QUEUE_GAS)]: 0,
        },
      },
      verifiedContractStorage: {
        [GAS_METADATA_ADDRESS]: {
          [keyToBytes32(GasMetadataKey.CURRENT_EPOCH_START_TIMESTAMP)]: true,
          [keyToBytes32(GasMetadataKey.CUMULATIVE_SEQUENCER_QUEUE_GAS)]: true,
          [keyToBytes32(GasMetadataKey.CUMULATIVE_L1TOL2_QUEUE_GAS)]: true,
          [keyToBytes32(GasMetadataKey.PREV_EPOCH_SEQUENCER_QUEUE_GAS)]: true,
          [keyToBytes32(GasMetadataKey.PREV_EPOCH_L1TOL2_QUEUE_GAS)]: true,
        },
      },
    },
  },
  parameters: [
    {
      name: 'run => ovmCALL(ADDRESS_1) => ovmADDRESS',
      steps: [
        {
          functionName: 'run',
          functionParams: {
            timestamp: 0,
            queueOrigin: 0,
            entrypoint: '$OVM_CALL_HELPER',
            origin: ZERO_ADDRESS,
            msgSender: ZERO_ADDRESS,
            gasLimit: OVM_TX_GAS_LIMIT,
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_1',
                  subSteps: [
                    {
                      functionName: 'ovmADDRESS',
                      expectedReturnValue: '$DUMMY_OVM_ADDRESS_1',
                    },
                  ],
                },
                expectedReturnStatus: true,
              },
            ],
          },
        },
      ],
    },
  ],
}

const runner = new ExecutionManagerTestRunner()
runner.run(test_run)
