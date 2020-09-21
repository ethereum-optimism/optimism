import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, BigNumber, ContractFactory } from 'ethers'
import { cloneDeep } from 'lodash'

/* Internal Imports */
import {
  TestDefinition,
  TestStep,
  isTestStep_SSTORE,
  isTestStep_SLOAD,
  isTestStep_CALL,
  isTestStep_CREATE,
  isTestStep_CREATE2,
  isTestStep_Context,
  ParsedTestStep,
  isRevertFlagError,
  TestParameter,
  isTestStep_evm,
  isTestStep_EXTCODESIZE,
  isTestStep_EXTCODEHASH,
  isTestStep_EXTCODECOPY,
  isTestStep_REVERT,
} from './test.types'
import { encodeRevertData } from '../codec'
import { getModifiableStorageFactory } from '../storage/contract-storage'
import { GAS_LIMIT, NON_NULL_BYTES32 } from '../constants'

export class ExecutionManagerTestRunner {
  private snapshot: string
  private contracts: {
    OVM_SafetyChecker: Contract
    OVM_StateManager: Contract
    OVM_ExecutionManager: Contract
    Helper_TestRunner: Contract
    Factory__Helper_TestRunner_CREATE: ContractFactory
  } = {
    OVM_SafetyChecker: undefined,
    OVM_StateManager: undefined,
    OVM_ExecutionManager: undefined,
    Helper_TestRunner: undefined,
    Factory__Helper_TestRunner_CREATE: undefined,
  }

  public run(test: TestDefinition) {
    test.preState = test.preState || {}
    test.postState = test.postState || {}

    describe(`OVM_ExecutionManager Test: ${test.name}`, () => {
      test.subTests?.map((subTest) => {
        this.run({
          ...subTest,
          preState: {
            ...test.preState,
            ...subTest.preState,
          },
          postState: {
            ...test.postState,
            ...subTest.postState,
          },
        })
      })

      test.parameters?.map((parameter) => {
        beforeEach(async () => {
          await this.initContracts()
        })

        let replacedTest: TestDefinition
        let replacedParameter: TestParameter
        beforeEach(async () => {
          replacedTest = this.setPlaceholderStrings(test)
          replacedParameter = this.setPlaceholderStrings(parameter)
        })

        beforeEach(async () => {
          await this.contracts.OVM_StateManager.__setContractStorage({
            accounts: {
              [this.contracts.Helper_TestRunner.address]: {
                nonce: 0,
                codeHash: NON_NULL_BYTES32,
                ethAddress: this.contracts.Helper_TestRunner.address,
              },
            },
          })
        })

        beforeEach(async () => {
          await this.contracts.OVM_ExecutionManager.__setContractStorage(
            replacedTest.preState.ExecutionManager
          )
          await this.contracts.OVM_StateManager.__setContractStorage(
            replacedTest.preState.StateManager
          )
        })

        afterEach(async () => {
          await this.contracts.OVM_ExecutionManager.__checkContractStorage(
            replacedTest.postState.ExecutionManager
          )
          await this.contracts.OVM_StateManager.__checkContractStorage(
            replacedTest.postState.StateManager
          )
        })

        const itfn = parameter.focus ? it.only : it
        itfn(`should execute: ${parameter.name}`, async () => {
          try {
            for (const step of replacedParameter.steps) {
              await this.runTestStep(step)
            }
          } catch (err) {
            if (parameter.expectInvalidStateAccess) {
              expect(err.toString()).to.contain('VM Exception while processing transaction: revert')
            } else {
              throw err
            }
          }
        })
      })
    })
  }

  private async initContracts() {
    if (this.snapshot) {
      await ethers.provider.send('evm_revert', [this.snapshot])
      return
    }

    this.contracts.OVM_SafetyChecker = await (
      await ethers.getContractFactory('OVM_SafetyChecker')
    ).deploy()
    this.contracts.OVM_ExecutionManager = await (
      await getModifiableStorageFactory('OVM_ExecutionManager')
    ).deploy(this.contracts.OVM_SafetyChecker.address)
    this.contracts.OVM_StateManager = await (
      await getModifiableStorageFactory('OVM_StateManager')
    ).deploy(this.contracts.OVM_ExecutionManager.address)
    this.contracts.Helper_TestRunner = await (
      await ethers.getContractFactory('Helper_TestRunner')
    ).deploy()
    this.contracts.Factory__Helper_TestRunner_CREATE = await ethers.getContractFactory(
      'Helper_TestRunner_CREATE'
    )

    this.snapshot = await ethers.provider.send('evm_snapshot', [])
  }

  private setPlaceholderStrings(obj: any) {
    const getReplacementString = (kv: string): string => {
      if (kv === '$OVM_EXECUTION_MANAGER') {
        return this.contracts.OVM_ExecutionManager.address
      } else if (kv === '$OVM_STATE_MANAGER') {
        return this.contracts.OVM_StateManager.address
      } else if (kv === '$OVM_SAFETY_CHECKER') {
        return this.contracts.OVM_SafetyChecker.address
      } else if (kv === '$OVM_CALL_HELPER') {
        return this.contracts.Helper_TestRunner.address
      } else if (kv.startsWith('$DUMMY_OVM_ADDRESS_')) {
        return '0x' + (kv.split('$DUMMY_OVM_ADDRESS_')[1] + '0').repeat(20)
      } else {
        return kv
      }
    }

    let ret: any = cloneDeep(obj)
    if (Array.isArray(ret)) {
      ret = ret.map((element: any) => {
        return this.setPlaceholderStrings(element)
      })
    } else if (typeof ret === 'object' && ret !== null) {
      for (const key of Object.keys(ret)) {
        const replacedKey = getReplacementString(key)

        if (replacedKey !== key) {
          ret[replacedKey] = ret[key]
          delete ret[key]
        }

        ret[replacedKey] = this.setPlaceholderStrings(ret[replacedKey])
      }
    } else if (typeof ret === 'string') {
      ret = getReplacementString(ret)
    }

    return ret
  }

  private async runTestStep(step: TestStep) {
    await this.contracts.OVM_ExecutionManager.ovmCALL(
      GAS_LIMIT / 2,
      this.contracts.Helper_TestRunner.address,
      this.contracts.Helper_TestRunner.interface.encodeFunctionData(
        'runSingleTestStep',
        [this.parseTestStep(step)]
      )
    )
  }

  private parseTestStep(step: TestStep): ParsedTestStep {
    return {
      functionName: step.functionName,
      functionData: this.encodeFunctionData(step),
      expectedReturnStatus: this.getReturnStatus(step),
      expectedReturnData: this.encodeExpectedReturnData(step),
    }
  }

  private getReturnStatus(step: TestStep): boolean {
    if (isTestStep_evm(step)) {
      return false
    } else if (isTestStep_Context(step)) {
      return true
    } else {
      return step.expectedReturnStatus
    }
  }

  private encodeFunctionData(step: TestStep): string {
    if (isTestStep_evm(step)) {
      if (isRevertFlagError(step.returnData)) {
        return encodeRevertData(
          step.returnData.flag,
          step.returnData.data,
          step.returnData.nuisanceGasLeft,
          step.returnData.ovmGasRefund
        )
      } else {
        return step.returnData || '0x'
      }
    }

    let functionParams: any[] = []
    if (
      isTestStep_SSTORE(step) ||
      isTestStep_SLOAD(step) ||
      isTestStep_EXTCODESIZE(step) ||
      isTestStep_EXTCODEHASH(step) ||
      isTestStep_EXTCODECOPY(step)
    ) {
      functionParams = Object.values(step.functionParams)
    } else if (isTestStep_CALL(step)) {
      functionParams = [
        step.functionParams.gasLimit,
        step.functionParams.target,
        step.functionParams.calldata ||
          this.contracts.Helper_TestRunner.interface.encodeFunctionData(
            'runMultipleTestSteps',
            [
              step.functionParams.subSteps.map((subStep) => {
                return this.parseTestStep(subStep)
              }),
            ]
          ),
      ]
    } else if (isTestStep_CREATE(step)) {
      functionParams = [
        this.contracts.Factory__Helper_TestRunner_CREATE.getDeployTransaction(
          step.functionParams.bytecode || '0x',
          step.functionParams.subSteps?.map((subStep) => {
            return this.parseTestStep(subStep)
          }) || []
        ).data,
      ]
    } else if (isTestStep_REVERT(step)) {
      functionParams = [step.revertData || '0x']
    }

    return this.contracts.OVM_ExecutionManager.interface.encodeFunctionData(
      step.functionName,
      functionParams
    )
  }

  private encodeExpectedReturnData(step: TestStep): string {
    if (isTestStep_evm(step)) {
      return '0x'
    }

    if (isTestStep_REVERT(step)) {
      return step.expectedReturnValue || '0x'
    }

    if (isRevertFlagError(step.expectedReturnValue)) {
      return encodeRevertData(
        step.expectedReturnValue.flag,
        step.expectedReturnValue.data,
        step.expectedReturnValue.nuisanceGasLeft,
        step.expectedReturnValue.ovmGasRefund
      )
    }

    let returnData: any[] = []
    if (isTestStep_CALL(step)) {
      if (step.expectedReturnValue === '0x00') {
        return step.expectedReturnValue
      } else {
        returnData = [
          step.expectedReturnStatus,
          step.expectedReturnValue || '0x',
        ]
      }
    } else if (BigNumber.isBigNumber(step.expectedReturnValue)) {
      returnData = [step.expectedReturnValue.toHexString()]
    } else if (step.expectedReturnValue !== undefined) {
      if (step.expectedReturnValue === '0x00') {
        return step.expectedReturnValue
      } else {
        returnData = [step.expectedReturnValue]
      }
    }

    return this.contracts.OVM_ExecutionManager.interface.encodeFunctionResult(
      step.functionName,
      returnData
    )
  }
}
