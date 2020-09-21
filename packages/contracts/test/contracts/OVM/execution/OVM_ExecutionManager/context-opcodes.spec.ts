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

  const globalContext = {
    ovmCHAINID: 420
  }

  const transactionContext = {
    ovmTIMESTAMP: 12341234,
    ovmGASLIMIT: 45674567,
    ovmTXGASLIMIT: 78907890,
    ovmL1QUEUEORIGIN: 1,
    ovmL1TXORIGIN: '0x1234123412341234123412341234123412341234'
  }

  const messageContext = {
      ovmCALLER: '0x6789678967896789678967896789678967896789',
      ovmADDRESS: '0x4567456745674567456745674567456745674567'
  }
  
  const test_ovmContextOpcodes: TestDefinition = {
    name: 'unit tests for basic getter opcodes',
    preState: {
      ExecutionManager: {
        globalContext,
        transactionContext,
        messageContext
      },
    },
    parameters: [
      {
        name: 'gets ovmCALLER',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmCALLER',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [messageContext.ovmCALLER]
                }
            ],
          },
        ],
      },
      {
        name: 'gets ovmADDRESS',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmADDRESS',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [messageContext.ovmADDRESS]
                }
            ],
          },
        ],
      },
      {
        name: 'gets ovmTIMESTAMP',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmTIMESTAMP',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [transactionContext.ovmTIMESTAMP]
                }
            ],
          },
        ],
      },
      {
        name: 'gets ovmGASLIMIT',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmGASLIMIT',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [transactionContext.ovmGASLIMIT]
                }
            ],
          },
        ],
      },
      {
        name: 'gets ovmQUEUEORIGIN',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmL1QUEUEORIGIN',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [transactionContext.ovmL1QUEUEORIGIN]
                }
            ],
          },
        ],
      },
      {
        name: 'gets ovmL1TXORIGIN',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmL1TXORIGIN',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [transactionContext.ovmL1TXORIGIN]
                }
            ],
          },
        ],
      },
      {
        name: 'gets ovmCHAINID',
        parameters: [
          {
            steps: [
                {
                    functionName: 'ovmCHAINID',
                    functionParams: [],
                    expectedReturnStatus: true,
                    expectedReturnValues: [globalContext.ovmCHAINID]
                }
            ],
          },
        ],
      },
    ],
  }
  
  runExecutionManagerTest(test_ovmContextOpcodes)
  