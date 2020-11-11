import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, BigNumber, ContractFactory } from 'ethers'
import { cloneDeep, merge } from 'lodash'
import { smoddit, smockit, ModifiableContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  TestDefinition,
  ParsedTestStep,
  TestParameter,
  TestStep,
  TestStep_CALL,
  TestStep_Run,
  isRevertFlagError,
  isTestStep_SSTORE,
  isTestStep_SLOAD,
  isTestStep_CALL,
  isTestStep_CREATE,
  isTestStep_CREATE2,
  isTestStep_CREATEEOA,
  isTestStep_Context,
  isTestStep_evm,
  isTestStep_Run,
  isTestStep_EXTCODESIZE,
  isTestStep_EXTCODEHASH,
  isTestStep_EXTCODECOPY,
  isTestStep_REVERT,
  isTestStep_SETNONCE,
} from './test.types'
import { encodeRevertData, REVERT_FLAGS } from '../codec'
import {
  OVM_TX_GAS_LIMIT,
  RUN_OVM_TEST_GAS,
  NON_NULL_BYTES32,
} from '../constants'

export class ExecutionManagerTestRunner {
  private snapshot: string
  private contracts: {
    OVM_SafetyChecker: Contract
    OVM_StateManager: ModifiableContract
    OVM_ExecutionManager: ModifiableContract
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
          preState: merge(
            cloneDeep(test.preState),
            cloneDeep(subTest.preState)
          ),
          postState: merge(
            cloneDeep(test.postState),
            cloneDeep(subTest.postState)
          ),
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

        beforeEach(() => {
          this.contracts.OVM_StateManager.smodify.set({
            accounts: {
              [this.contracts.Helper_TestRunner.address]: {
                nonce: 0,
                codeHash: NON_NULL_BYTES32,
                ethAddress: this.contracts.Helper_TestRunner.address,
              },
            },
          })
        })

        beforeEach(() => {
          this.contracts.OVM_ExecutionManager.smodify.set(
            replacedTest.preState.ExecutionManager
          )
          this.contracts.OVM_StateManager.smodify.set(
            replacedTest.preState.StateManager
          )
        })

        afterEach(async () => {
          expect(
            await this.contracts.OVM_ExecutionManager.smodify.check(
              replacedTest.postState.ExecutionManager
            )
          ).to.equal(true)

          expect(
            await this.contracts.OVM_StateManager.smodify.check(
              replacedTest.postState.StateManager
            )
          ).to.equal(true)
        })

        const itfn = parameter.focus ? it.only : it
        itfn(`should execute: ${parameter.name}`, async () => {
          try {
            for (const step of replacedParameter.steps) {
              await this.runTestStep(step)
            }
          } catch (err) {
            if (parameter.expectInvalidStateAccess) {
              expect(err.toString()).to.contain(
                'VM Exception while processing transaction: revert'
              )
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
      this.snapshot = await ethers.provider.send('evm_snapshot', [])
      return
    }

    const AddressManager = await (
      await ethers.getContractFactory('Lib_AddressManager')
    ).deploy()

    const SafetyChecker = await (
      await ethers.getContractFactory('OVM_SafetyChecker')
    ).deploy()

    const MockSafetyChecker = smockit(SafetyChecker)
    MockSafetyChecker.smocked.isBytecodeSafe.will.return.with(true)

    this.contracts.OVM_SafetyChecker = MockSafetyChecker

    await AddressManager.setAddress(
      'OVM_SafetyChecker',
      this.contracts.OVM_SafetyChecker.address
    )

    this.contracts.OVM_ExecutionManager = await (
      await smoddit('OVM_ExecutionManager')
    ).deploy(
      AddressManager.address,
      {
        minTransactionGasLimit: 0,
        maxTransactionGasLimit: 1_000_000_000,
        maxGasPerQueuePerEpoch: 1_000_000_000_000,
        secondsPerEpoch: 600,
      },
      {
        ovmCHAINID: 420,
      }
    )

    this.contracts.OVM_StateManager = await (
      await smoddit('OVM_StateManager')
    ).deploy(await this.contracts.OVM_ExecutionManager.signer.getAddress())
    await this.contracts.OVM_StateManager.setExecutionManager(
      this.contracts.OVM_ExecutionManager.address
    )

    this.contracts.Helper_TestRunner = await (
      await ethers.getContractFactory('Helper_TestRunner')
    ).deploy()

    this.contracts.Factory__Helper_TestRunner_CREATE = await ethers.getContractFactory(
      'Helper_TestRunner_CREATE'
    )

    this.snapshot = await ethers.provider.send('evm_snapshot', [])
  }

  public static getDummyAddress(placeholder: string): string {
    return '0x' + (placeholder.split('$DUMMY_OVM_ADDRESS_')[1] + '0').repeat(20)
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
        return ExecutionManagerTestRunner.getDummyAddress(kv)
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

  private async runTestStep(step: TestStep | TestStep_Run) {
    if (isTestStep_Run(step)) {
      let calldata: string
      if (step.functionParams.data) {
        calldata = step.functionParams.data
      } else {
        const runStep: TestStep_CALL = {
          functionName: 'ovmCALL',
          functionParams: {
            gasLimit: OVM_TX_GAS_LIMIT,
            target: ExecutionManagerTestRunner.getDummyAddress(
              '$DUMMY_OVM_ADDRESS_1'
            ),
            subSteps: step.functionParams.subSteps,
          },
          expectedReturnStatus: true,
        }

        calldata = this.encodeFunctionData(runStep)
      }

      await this.contracts.OVM_ExecutionManager.run(
        {
          timestamp: step.functionParams.timestamp,
          blockNumber: 0,
          l1QueueOrigin: step.functionParams.queueOrigin,
          l1TxOrigin: step.functionParams.origin,
          entrypoint: step.functionParams.entrypoint,
          gasLimit: step.functionParams.gasLimit,
          data: calldata,
        },
        this.contracts.OVM_StateManager.address,
        { gasLimit: RUN_OVM_TEST_GAS }
      )
    } else {
      await this.contracts.OVM_ExecutionManager.ovmCALL(
        OVM_TX_GAS_LIMIT,
        ExecutionManagerTestRunner.getDummyAddress('$DUMMY_OVM_ADDRESS_1'),
        this.contracts.Helper_TestRunner.interface.encodeFunctionData(
          'runSingleTestStep',
          [this.parseTestStep(step)]
        ),
        { gasLimit: RUN_OVM_TEST_GAS }
      )
    }
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
    } else if (isTestStep_CALL(step)) {
      if (
        isRevertFlagError(step.expectedReturnValue) &&
        (step.expectedReturnValue.flag === REVERT_FLAGS.INVALID_STATE_ACCESS ||
          step.expectedReturnValue.flag === REVERT_FLAGS.STATIC_VIOLATION)
      ) {
        return step.expectedReturnStatus
      } else {
        return true
      }
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
      isTestStep_SETNONCE(step) ||
      isTestStep_EXTCODESIZE(step) ||
      isTestStep_EXTCODEHASH(step) ||
      isTestStep_EXTCODECOPY(step) ||
      isTestStep_CREATEEOA(step)
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

    if (isRevertFlagError(step.expectedReturnValue)) {
      return encodeRevertData(
        step.expectedReturnValue.flag,
        step.expectedReturnValue.data,
        step.expectedReturnValue.nuisanceGasLeft,
        step.expectedReturnValue.ovmGasRefund
      )
    }

    if (isTestStep_REVERT(step)) {
      return step.expectedReturnValue || '0x'
    }

    let returnData: any[] = []
    if (isTestStep_CALL(step)) {
      if (step.expectedReturnValue === '0x00') {
        return step.expectedReturnValue
      } else if (
        typeof step.expectedReturnValue === 'string' ||
        step.expectedReturnValue === undefined
      ) {
        returnData = [
          step.expectedReturnStatus,
          step.expectedReturnValue || '0x',
        ]
      } else {
        returnData = [
          step.expectedReturnValue.ovmSuccess,
          step.expectedReturnValue.returnData,
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
