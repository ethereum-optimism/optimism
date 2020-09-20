import { expect } from '../../setup'

/* External Imports */
import { Contract } from 'ethers'
import { cloneDeep } from 'lodash'
import { ethers } from '@nomiclabs/buidler'

/* Internal Imports */
import { getModifiableStorageFactory } from '../storage/contract-storage'
import { GAS_LIMIT, NON_NULL_BYTES32 } from '../constants'
import { getTestGenerator, getInitcode, getBytecode } from './test-generation'
import { TestParameters, TestDefinition, isTestDefinition } from './test.types'
import { int } from '@nomiclabs/buidler/internal/core/params/argumentTypes'

const getDummyOVMAddress = (kv: string): string => {
  return '0x' + (kv.split('$DUMMY_OVM_ADDRESS_')[1] + '0').repeat(20)
}

const setPlaceholderStrings = (
  test: any,
  ovmExecutionManager: Contract,
  ovmStateManager: Contract,
  ovmSafetyChecker: Contract,
  ovmCallHelper: Contract,
  ovmRevertHelper: Contract
): any => {
  const setPlaceholder = (kv: string): string => {
    if (kv === '$OVM_EXECUTION_MANAGER') {
      return ovmExecutionManager.address
    } else if (kv === '$OVM_STATE_MANAGER') {
      return ovmStateManager.address
    } else if (kv === '$OVM_SAFETY_CHECKER') {
      return ovmSafetyChecker.address
    } else if (kv === '$OVM_CALL_HELPER_CODE') {
      return getBytecode('Helper_CodeContractForCalls')
    } else if (kv === '$OVM_CALL_HELPER') {
      return ovmCallHelper.address
    } else if (kv === '$OVM_REVERT_HELPER') {
      return ovmRevertHelper.address
    } else if (kv.startsWith('$DUMMY_OVM_ADDRESS_')) {
      return getDummyOVMAddress(kv)
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
        ovmSafetyChecker,
        ovmCallHelper,
        ovmRevertHelper
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
        ovmSafetyChecker,
        ovmCallHelper,
        ovmRevertHelper
      )
    }
  } else if (typeof test === 'string') {
    test = setPlaceholder(test)
  }

  return test
}

const fixtureDeployContracts = async (): Promise<{
  OVM_SafetyChecker: Contract
  OVM_StateManager: Contract
  OVM_ExecutionManager: Contract
  OVM_CallHelper: Contract
  OVM_RevertHelper: Contract
  OVM_CreateStorer: Contract
  OVM_InvalidHelper: Contract
}> => {
  const Factory__OVM_SafetyChecker = await ethers.getContractFactory(
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
  const Factory__Helper_CodeContractForReverts = await ethers.getContractFactory(
    'Helper_CodeContractForReverts'
  )
  const Factory__Helper_CreateEMResponsesStorer = await ethers.getContractFactory(
    'Helper_CreateEMResponsesStorer'
  )

  const Helper_CodeContractForInvalid = await ethers.getContractFactory(
    'Helper_CodeContractForInvalid'
  )

  const OVM_SafetyChecker = await Factory__OVM_SafetyChecker.deploy()
  const OVM_ExecutionManager = await Factory__OVM_ExecutionManager.deploy(
    OVM_SafetyChecker.address
  )
  const OVM_StateManager = await Factory__OVM_StateManager.deploy(
    OVM_ExecutionManager.address
  )
  const OVM_CallHelper = await Factory__Helper_CodeContractForCalls.deploy()
  const OVM_RevertHelper = await Factory__Helper_CodeContractForReverts.deploy()
  const OVM_CreateStorer = await Factory__Helper_CreateEMResponsesStorer.deploy()
  const OVM_InvalidHelper = await Helper_CodeContractForInvalid.deploy()

  return {
    OVM_SafetyChecker,
    OVM_StateManager,
    OVM_ExecutionManager,
    OVM_CallHelper,
    OVM_RevertHelper,
    OVM_CreateStorer,
    OVM_InvalidHelper,
  }
}

export const runExecutionManagerTest = (test: TestDefinition): void => {
  test.preState = test.preState || {}
  test.postState = test.postState || {}

  describe(`Standard test: ${test.name}`, () => {
    test.parameters.map((parameters) => {
      if (isTestDefinition(parameters)) {
        runExecutionManagerTest({
          ...parameters,
          preState: {
            ...test.preState,
            ...parameters.preState,
          },
          postState: {
            ...test.postState,
            ...parameters.postState,
          },
        })
      } else {
        let OVM_SafetyChecker: Contract
        let OVM_StateManager: Contract
        let OVM_ExecutionManager: Contract
        let OVM_CallHelper: Contract
        let OVM_RevertHelper: Contract
        let OVM_CreateStorer: Contract
        let OVM_InvalidHelper: Contract
        beforeEach(async () => {
          const contracts = await fixtureDeployContracts()
          OVM_SafetyChecker = contracts.OVM_SafetyChecker
          OVM_StateManager = contracts.OVM_StateManager
          OVM_ExecutionManager = contracts.OVM_ExecutionManager
          OVM_CallHelper = contracts.OVM_CallHelper
          OVM_RevertHelper = contracts.OVM_RevertHelper
          OVM_CreateStorer = contracts.OVM_CreateStorer
          OVM_InvalidHelper = contracts.OVM_InvalidHelper
        })

        let replacedParams: TestParameters
        let replacedTest: TestDefinition
        beforeEach(async () => {
          replacedParams = setPlaceholderStrings(
            cloneDeep(parameters),
            OVM_ExecutionManager,
            OVM_StateManager,
            OVM_SafetyChecker,
            OVM_CallHelper,
            OVM_RevertHelper
          )
          replacedTest = setPlaceholderStrings(
            cloneDeep(test),
            OVM_ExecutionManager,
            OVM_StateManager,
            OVM_SafetyChecker,
            OVM_CallHelper,
            OVM_RevertHelper
          )
        })

        beforeEach(async () => {
          await OVM_ExecutionManager.__setContractStorage(
            replacedTest.preState.ExecutionManager
          )
          await OVM_StateManager.__setContractStorage(
            replacedTest.preState.StateManager
          )
        })

        afterEach(async () => {
          await OVM_ExecutionManager.__checkContractStorage({
            ...replacedTest.postState.ExecutionManager,
          })
          await OVM_StateManager.__checkContractStorage({
            ...replacedTest.postState.StateManager,
          })
        })

        parameters.steps.map((step, idx) => {
          const scopedFunction = !!test.focus ? it.only : it
          scopedFunction(`should run test: ${test.name} ${idx}`, async () => {
            const testGenerator = getTestGenerator(
              replacedParams.steps[idx],
              OVM_ExecutionManager,
              OVM_CallHelper,
              OVM_CreateStorer,
              (await ethers.getContractFactory('Helper_CodeContractForCreates'))
                .interface,
              OVM_RevertHelper,
              OVM_InvalidHelper
            )

            const callResult = await OVM_ExecutionManager.provider.call({
              to: OVM_ExecutionManager.address,
              data: testGenerator.getCalldata(),
              gasLimit: GAS_LIMIT,
            })

            await OVM_ExecutionManager.signer.sendTransaction({
              to: OVM_ExecutionManager.address,
              data: testGenerator.getCalldata(),
              gasLimit: GAS_LIMIT,
            })

            const interpretation = testGenerator.interpretActualReturnData(
              callResult,
              true
            )
            console.log('interpretation of actual results:\n' + interpretation) // in future we can add conditional here but for now always assume succeed

            const interpretationOfExpected = testGenerator.interpretActualReturnData(
              testGenerator.getReturnData(),
              true
            )
            console.log(
              'interpretation of expected: \n' + interpretationOfExpected
            )

            expect(callResult).to.equal(testGenerator.getReturnData()) //, 'got bad response, looks like it did:\n' + testGenerator.interpretActualReturnData(callResult))
          })
        })
      }
    })
  })
}
