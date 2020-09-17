import { expect } from '../../setup'

/* External Imports */
import { Contract } from 'ethers'
import { cloneDeep } from 'lodash'

/* Internal Imports */
import { getModifiableStorageFactory } from '../storage/contract-storage'
import { GAS_LIMIT } from '../constants'
import { getTestGenerator } from './test-generation'
import { TestParameters, TestDefinition, isTestDefinition } from './test.types'

const setPlaceholderStrings = (
  test: any,
  ovmExecutionManager: Contract,
  ovmStateManager: Contract,
  ovmCallHelper: Contract
): any => {
  const setPlaceholder = (
    kv: string
  ): string => {
    if (kv === '$OVM_EXECUTION_MANAGER') {
      return ovmExecutionManager.address
    } else if (kv === '$OVM_STATE_MANAGER') {
      return ovmStateManager.address
    } else if (kv === '$OVM_CALL_HELPER') {
      return ovmCallHelper.address
    } else if (kv.startsWith('$DUMMY_OVM_ADDRESS_')) {
      return '0x' + kv.split('$DUMMY_OVM_ADDRESS_')[1].padStart(40, '0')
    } else {
      return kv
    }
  }

  if (Array.isArray(test)) {
    test = test.map((element) => {
      return setPlaceholderStrings(
        element,
        ovmExecutionManager,
        ovmStateManager,
        ovmCallHelper
      )
    })
  } else if (typeof test === 'object' && test !== null) {
    for (const key of Object.keys(test)) {
      const replacedKey = setPlaceholder(key)

      if (replacedKey !== key) {
        test[replacedKey] = test[key]
        delete test[key]
      }

      test[replacedKey] = setPlaceholderStrings(
        test[replacedKey],
        ovmExecutionManager,
        ovmStateManager,
        ovmCallHelper
      )
    }
  } else if (typeof test === 'string') {
    test = setPlaceholder(test)
  }

  return test
}

const fixtureDeployContracts = async (): Promise <{
  OVM_SafetyChecker: Contract,
  OVM_StateManager: Contract,
  OVM_ExecutionManager: Contract,
  OVM_CallHelper: Contract
}> => {
  const Factory__OVM_SafetyChecker = await getModifiableStorageFactory(
    'OVM_SafetyChecker'
  )
  const Factory__OVM_StateManager = await getModifiableStorageFactory(
    'OVM_StateManager'
  )
  const Factory__OVM_ExecutionManager = await getModifiableStorageFactory(
    'OVM_ExecutionManager'
  )
  const Factory__Helper_CodeContractForCalls = await getModifiableStorageFactory(
    'Helper_CodeContractForCalls'
  )

  const OVM_SafetyChecker = await Factory__OVM_SafetyChecker.deploy()
  const OVM_StateManager = await Factory__OVM_StateManager.deploy()
  const OVM_ExecutionManager = await Factory__OVM_ExecutionManager.deploy(OVM_SafetyChecker.address)
  const OVM_CallHelper = await Factory__Helper_CodeContractForCalls.deploy()

  return {
    OVM_SafetyChecker,
    OVM_StateManager,
    OVM_ExecutionManager,
    OVM_CallHelper,
  }
}

export const runExecutionManagerTest = (
  test: TestDefinition
): void => {
  test.preState = test.preState || {}
  test.postState = test.postState || {}

  describe(`Standard test: ${test.name}`, () => {
    test.parameters.map((parameters) => {
      if (isTestDefinition(parameters)) {
        runExecutionManagerTest(
          {
            ...parameters,
            preState: {
              ...test.preState,
              ...parameters.preState
            },
            postState: {
              ...test.postState,
              ...parameters.postState
            }
          }
        )
      } else {
        let OVM_StateManager: Contract
        let OVM_ExecutionManager: Contract
        let OVM_CallHelper: Contract
        beforeEach(async () => {
          const contracts = await fixtureDeployContracts()
          OVM_StateManager = contracts.OVM_StateManager
          OVM_ExecutionManager = contracts.OVM_ExecutionManager
          OVM_CallHelper = contracts.OVM_CallHelper
        })

        let replacedParams: TestParameters
        let replacedTest: TestDefinition
        beforeEach(async () => {
          replacedParams = setPlaceholderStrings(
            cloneDeep(parameters),
            OVM_ExecutionManager,
            OVM_StateManager,
            OVM_CallHelper
          )
          replacedTest = setPlaceholderStrings(
            cloneDeep(test),
            OVM_ExecutionManager,
            OVM_StateManager,
            OVM_CallHelper
          )
        })

        beforeEach(async () => {
          await OVM_ExecutionManager.__setContractStorage(replacedTest.preState.ExecutionManager)
          await OVM_StateManager.__setContractStorage(replacedTest.preState.StateManager)
        })

        afterEach(async () => {
          await OVM_ExecutionManager.__checkContractStorage({
            ...replacedTest.preState.ExecutionManager,
            ...replacedTest.postState.ExecutionManager
          })
          await OVM_StateManager.__checkContractStorage({
            ...replacedTest.preState.StateManager,
            ...replacedTest.postState.StateManager
          })
        })

        parameters.steps.map((step, idx) => {
          it(`should run test: ${test.name} ${idx}`, async () => {
            const testGenerator = getTestGenerator(
              replacedParams.steps[idx],
              OVM_ExecutionManager,
              OVM_CallHelper
            )

            const callResult = await OVM_ExecutionManager.provider.call({
              to: OVM_ExecutionManager.address,
              data: testGenerator.getCalldata(),
              gasLimit: GAS_LIMIT
            })
    
            await OVM_ExecutionManager.signer.sendTransaction({
              to: OVM_ExecutionManager.address,
              data: testGenerator.getCalldata(),
              gasLimit: GAS_LIMIT
            })

            expect(callResult).to.equal(testGenerator.getReturnData())
          })
        })
      }
    })
  })
}
