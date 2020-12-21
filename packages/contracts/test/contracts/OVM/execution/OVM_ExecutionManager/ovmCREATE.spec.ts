/* Internal Imports */
import {
  ExecutionManagerTestRunner,
  TestDefinition,
  OVM_TX_GAS_LIMIT,
  NULL_BYTES32,
  NON_NULL_BYTES32,
  REVERT_FLAGS,
  DUMMY_BYTECODE,
  ZERO_ADDRESS,
  VERIFIED_EMPTY_CONTRACT_HASH,
  DUMMY_BYTECODE_BYTELEN,
  DUMMY_BYTECODE_HASH,
  getStorageXOR,
} from '../../../../helpers'

const CREATED_CONTRACT_1 = '0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb'
const CREATED_CONTRACT_2 = '0x2bda4a99d5be88609d23b1e4ab5d1d34fb1c2feb'
const CREATED_CONTRACT_BY_2_1 = '0xe0d8be8101f36ebe6b01abacec884422c39a1f62'
const CREATED_CONTRACT_BY_2_2 = '0x15ac629e1a3866b17179ee4ae86de5cbda744335'
const NESTED_CREATED_CONTRACT = '0xcb964b3f4162a0d4f5c997b40e19da5a546bc36f'
const DUMMY_REVERT_DATA =
  '0xdeadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420deadbeef1e5420'

const NON_WHITELISTED_DEPLOYER = '0x1234123412341234123412341234123412341234'
const NON_WHITELISTED_DEPLOYER_KEY =
  '0x0000000000000000000000001234123412341234123412341234123412341234'
const CREATED_BY_NON_WHITELISTED_DEPLOYER =
  '0x794e4aa3be128b0fc01ba12543b70bf9d77072fc'

const WHITELISTED_DEPLOYER = '0x3456345634563456345634563456345634563456'
const WHITELISTED_DEPLOYER_KEY =
  '0x0000000000000000000000003456345634563456345634563456345634563456'
const CREATED_BY_WHITELISTED_DEPLOYER =
  '0x9f397a91ccb7cc924d1585f1053bc697d30f343f'

const test_ovmCREATE: TestDefinition = {
  name: 'Basic tests for ovmCREATE',
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
        [CREATED_CONTRACT_1]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
        [CREATED_CONTRACT_2]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
        [CREATED_CONTRACT_BY_2_1]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
        [CREATED_CONTRACT_BY_2_2]: {
          codeHash: '0x' + '01'.repeat(32),
          ethAddress: '0x' + '00'.repeat(20),
        },
        [NESTED_CREATED_CONTRACT]: {
          codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
          ethAddress: '0x' + '00'.repeat(20),
        },
      },
      contractStorage: {
        $DUMMY_OVM_ADDRESS_2: {
          [NULL_BYTES32]: getStorageXOR(NULL_BYTES32),
        },
      },
      verifiedContractStorage: {
        $DUMMY_OVM_ADDRESS_1: {
          [NON_NULL_BYTES32]: true,
        },
        $DUMMY_OVM_ADDRESS_2: {
          [NULL_BYTES32]: true,
        },
      },
    },
  },
  parameters: [
    {
      name: 'ovmCREATE, ovmEXTCODESIZE(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            bytecode: DUMMY_BYTECODE,
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmEXTCODESIZE',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: DUMMY_BYTECODE_BYTELEN,
        },
      ],
    },
    {
      name: 'ovmCREATE, ovmEXTCODEHASH(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            bytecode: DUMMY_BYTECODE,
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmEXTCODEHASH',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: DUMMY_BYTECODE_HASH,
        },
      ],
    },
    {
      name: 'ovmCREATE, ovmEXTCODECOPY(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            bytecode: DUMMY_BYTECODE,
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmEXTCODECOPY',
          functionParams: {
            address: CREATED_CONTRACT_1,
            offset: 0,
            length: DUMMY_BYTECODE_BYTELEN,
          },
          expectedReturnStatus: true,
          expectedReturnValue: DUMMY_BYTECODE,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT, ovmEXTCODESIZE(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmEXTCODESIZE',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: 0,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT, ovmEXTCODEHASH(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmEXTCODEHASH',
          functionParams: {
            address: CREATED_CONTRACT_1,
          },
          expectedReturnStatus: true,
          expectedReturnValue: NULL_BYTES32,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmREVERT, ovmEXTCODECOPY(CREATED)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmEXTCODECOPY',
          functionParams: {
            address: CREATED_CONTRACT_1,
            offset: 0,
            length: 256,
          },
          expectedReturnStatus: true,
          expectedReturnValue: '0x' + '00'.repeat(256),
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmADDRESS',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmADDRESS',
                expectedReturnValue: CREATED_CONTRACT_1,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
      ],
    },
    {
      name: 'ovmCALL => ovmCREATE => ovmCALLER',
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              {
                functionName: 'ovmCREATE',
                functionParams: {
                  subSteps: [
                    {
                      functionName: 'ovmSLOAD',
                      functionParams: {
                        key: NON_NULL_BYTES32,
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: NULL_BYTES32,
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: CREATED_CONTRACT_2,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmSSTORE + ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmSSTORE',
                functionParams: {
                  key: NON_NULL_BYTES32,
                  value: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
              },
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NON_NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
      ],
    },
    {
      name:
        'ovmCREATE => ovmSSTORE, ovmCALL(CREATED) => ovmSLOAD(EXIST) + ovmSLOAD(NONEXIST)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmSSTORE',
                functionParams: {
                  key: NON_NULL_BYTES32,
                  value: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: CREATED_CONTRACT_1,
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NON_NULL_BYTES32,
              },
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name:
        'ovmCREATE => ovmCALL(ADDRESS_1) => ovmSSTORE, ovmCALL(ADDRESS_1) => ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_1',
                  subSteps: [
                    {
                      functionName: 'ovmSSTORE',
                      functionParams: {
                        key: NON_NULL_BYTES32,
                        value: NON_NULL_BYTES32,
                      },
                      expectedReturnStatus: true,
                    },
                  ],
                },
                expectedReturnStatus: true,
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: CREATED_CONTRACT_1,
        },
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_1',
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NON_NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NON_NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name:
        'ovmCREATE => (ovmCALL(ADDRESS_2) => ovmSSTORE) + ovmREVERT, ovmCALL(ADDRESS_2) => ovmSLOAD',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_2',
                  subSteps: [
                    {
                      functionName: 'ovmSSTORE',
                      functionParams: {
                        key: NULL_BYTES32,
                        value: NON_NULL_BYTES32,
                      },
                      expectedReturnStatus: true,
                    },
                  ],
                },
                expectedReturnStatus: true,
              },
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_2',
            subSteps: [
              {
                functionName: 'ovmSLOAD',
                functionParams: {
                  key: NULL_BYTES32,
                },
                expectedReturnStatus: true,
                expectedReturnValue: NULL_BYTES32,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmCALL(ADDRESS_NONEXIST)',
      expectInvalidStateAccess: true,
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCALL',
                functionParams: {
                  gasLimit: OVM_TX_GAS_LIMIT,
                  target: '$DUMMY_OVM_ADDRESS_3',
                  calldata: '0x',
                },
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: false,
          expectedReturnValue: {
            flag: REVERT_FLAGS.INVALID_STATE_ACCESS,
          },
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmCREATE => ovmCALL(ADDRESS_NONEXIST)',
      steps: [
        {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: '$DUMMY_OVM_ADDRESS_2',
            subSteps: [
              {
                functionName: 'ovmCREATE',
                functionParams: {
                  subSteps: [
                    {
                      functionName: 'ovmCREATE',
                      functionParams: {
                        bytecode: '0x',
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: ZERO_ADDRESS,
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: CREATED_CONTRACT_BY_2_1,
              },
            ],
          },
          expectedReturnStatus: true,
        },
      ],
    },
    {
      name: 'ovmCREATE => ovmCREATE => ovmCALL(ADDRESS_NONEXIST)',
      expectInvalidStateAccess: true,
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCREATE',
                functionParams: {
                  subSteps: [
                    {
                      functionName: 'ovmCALL',
                      functionParams: {
                        gasLimit: OVM_TX_GAS_LIMIT,
                        target: '$DUMMY_OVM_ADDRESS_3',
                        calldata: '0x',
                      },
                      expectedReturnStatus: true,
                      expectedReturnValue: '0x00',
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: false,
          expectedReturnValue: {
            flag: REVERT_FLAGS.INVALID_STATE_ACCESS,
          },
        },
      ],
    },
    {
      name: 'OZ-AUDIT: ovmCREATE => ((ovmCREATE => ovmADDRESS), ovmREVERT)',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'ovmCREATE',
                functionParams: {
                  subSteps: [
                    {
                      functionName: 'ovmADDRESS',
                      expectedReturnValue: NESTED_CREATED_CONTRACT,
                    },
                  ],
                },
                expectedReturnStatus: true,
                expectedReturnValue: NESTED_CREATED_CONTRACT,
              },
              {
                functionName: 'ovmREVERT',
                revertData: DUMMY_REVERT_DATA,
                expectedReturnStatus: true,
                expectedReturnValue: '0x00',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
      ],
    },
    {
      name: 'ovmCREATE => OUT_OF_GAS',
      steps: [
        {
          functionName: 'ovmCREATE',
          functionParams: {
            subSteps: [
              {
                functionName: 'evmINVALID',
              },
            ],
          },
          expectedReturnStatus: true,
          expectedReturnValue: ZERO_ADDRESS,
        },
      ],
    },
  ],
  subTests: [
    {
      name: 'Deployer whitelist tests',
      preState: {
        StateManager: {
          accounts: {
            [NON_WHITELISTED_DEPLOYER]: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [WHITELISTED_DEPLOYER]: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [CREATED_BY_WHITELISTED_DEPLOYER]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
            },
          },
          contractStorage: {
            ['0x4200000000000000000000000000000000000002']: {
              // initialized? true
              '0x0000000000000000000000000000000000000000000000000000000000000010': getStorageXOR(
                '0x' + '00'.repeat(31) + '01'
              ),
              // allowArbitraryDeployment? false
              '0x0000000000000000000000000000000000000000000000000000000000000012': getStorageXOR(
                NULL_BYTES32
              ),
              // non-whitelisted deployer is whitelisted? false
              [NON_WHITELISTED_DEPLOYER_KEY]: getStorageXOR(NULL_BYTES32),
              // whitelisted deployer is whitelisted? true
              [WHITELISTED_DEPLOYER_KEY]: getStorageXOR(
                '0x' + '00'.repeat(31) + '01'
              ),
            },
          },
          verifiedContractStorage: {
            ['0x4200000000000000000000000000000000000002']: {
              '0x0000000000000000000000000000000000000000000000000000000000000010': 1,
              '0x0000000000000000000000000000000000000000000000000000000000000012': 1,
              [NON_WHITELISTED_DEPLOYER_KEY]: 1,
              [WHITELISTED_DEPLOYER_KEY]: 1,
            },
          },
        },
      },
      parameters: [
        {
          name: 'ovmCREATE by WHITELISTED_DEPLOYER',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: OVM_TX_GAS_LIMIT / 2,
                target: WHITELISTED_DEPLOYER,
                subSteps: [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: {
                      bytecode: DUMMY_BYTECODE,
                    },
                    expectedReturnStatus: true,
                    expectedReturnValue: CREATED_BY_WHITELISTED_DEPLOYER,
                  },
                ],
              },
              expectedReturnStatus: true,
            },
          ],
        },
        {
          name: 'ovmCREATE by NON_WHITELISTED_DEPLOYER',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: OVM_TX_GAS_LIMIT / 2,
                target: NON_WHITELISTED_DEPLOYER,
                subSteps: [
                  {
                    functionName: 'ovmCREATE',
                    functionParams: {
                      subSteps: [],
                    },
                    expectedReturnStatus: false,
                    expectedReturnValue: {
                      flag: REVERT_FLAGS.CREATOR_NOT_ALLOWED,
                      onlyValidateFlag: true,
                    },
                  },
                ],
              },
              expectedReturnStatus: true,
              expectedReturnValue: {
                ovmSuccess: false,
                returnData: '0x',
              },
            },
          ],
        },
        {
          name: 'ovmCREATE2 by NON_WHITELISTED_DEPLOYER',
          steps: [
            {
              functionName: 'ovmCALL',
              functionParams: {
                gasLimit: OVM_TX_GAS_LIMIT / 2,
                target: NON_WHITELISTED_DEPLOYER,
                subSteps: [
                  {
                    functionName: 'ovmCREATE2',
                    functionParams: {
                      salt: NULL_BYTES32,
                      bytecode: '0x',
                    },
                    expectedReturnStatus: false,
                    expectedReturnValue: {
                      flag: REVERT_FLAGS.CREATOR_NOT_ALLOWED,
                      onlyValidateFlag: true,
                    },
                  },
                ],
              },
              expectedReturnStatus: true,
              expectedReturnValue: {
                ovmSuccess: false,
                returnData: '0x',
              },
            },
          ],
        },
      ],
    },
    {
      name: 'Deployer whitelist tests',
      preState: {
        StateManager: {
          accounts: {
            [NON_WHITELISTED_DEPLOYER]: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [WHITELISTED_DEPLOYER]: {
              codeHash: NON_NULL_BYTES32,
              ethAddress: '$OVM_CALL_HELPER',
            },
            [CREATED_BY_NON_WHITELISTED_DEPLOYER]: {
              codeHash: VERIFIED_EMPTY_CONTRACT_HASH,
              ethAddress: '0x' + '00'.repeat(20),
            },
          },
          contractStorage: {
            ['0x4200000000000000000000000000000000000002']: {
              // initialized? true
              '0x0000000000000000000000000000000000000000000000000000000000000010': getStorageXOR(
                '0x' + '00'.repeat(31) + '01'
              ),
              // allowArbitraryDeployment? true
              '0x0000000000000000000000000000000000000000000000000000000000000012': getStorageXOR(
                '0x' + '00'.repeat(31) + '01'
              ),
              // non-whitelisted deployer is whitelisted? false
              [NON_WHITELISTED_DEPLOYER_KEY]: getStorageXOR(NULL_BYTES32),
              // whitelisted deployer is whitelisted? true
              [WHITELISTED_DEPLOYER_KEY]: getStorageXOR(
                '0x' + '00'.repeat(31) + '01'
              ),
            },
          },
          verifiedContractStorage: {
            ['0x4200000000000000000000000000000000000002']: {
              '0x0000000000000000000000000000000000000000000000000000000000000010': 1,
              '0x0000000000000000000000000000000000000000000000000000000000000012': 1,
              [NON_WHITELISTED_DEPLOYER_KEY]: 1,
              [WHITELISTED_DEPLOYER_KEY]: 1,
            },
          },
        },
      },
      subTests: [
        {
          name: 'when arbitrary contract deployment is enabled',
          parameters: [
            {
              name: 'ovmCREATE by NON_WHITELISTED_DEPLOYER',
              steps: [
                {
                  functionName: 'ovmCALL',
                  functionParams: {
                    gasLimit: OVM_TX_GAS_LIMIT / 2,
                    target: NON_WHITELISTED_DEPLOYER,
                    subSteps: [
                      {
                        functionName: 'ovmCREATE',
                        functionParams: {
                          subSteps: [],
                        },
                        expectedReturnStatus: true,
                        expectedReturnValue: CREATED_BY_NON_WHITELISTED_DEPLOYER,
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
    },
  ],
}

const runner = new ExecutionManagerTestRunner()
runner.run(test_ovmCREATE)
