/* External Imports */
import { bufToHexString, getLogger, keccak256, Logger } from '@pigi/core-utils'
import { Opcode } from '@pigi/rollup-core'

import * as AsyncLock from 'async-lock'

import BN = require('bn.js')
import VM from 'ethereumjs-vm'
import { ExecResult } from 'ethereumjs-vm/dist/evm/evm'
import { ERROR, VmError } from 'ethereumjs-vm/dist/exceptions'
import {
  EvmError,
  EvmErrors,
  EvmIntrospectionUtil,
  ExecutionResultComparison,
  StepContext,
  ExecutionComparison,
  ExecutionResult,
} from '../../types/vm'

const log: Logger = getLogger('evm-util')

type StepCallback = (data, continueFn) => Promise<void>
type StepContextCallback = (context: StepContext) => Promise<void>

const BIG_ENOUGH_GAS_LIMIT: any = new BN('ffffffff', 'hex')
const KEY = 'EvmIntrospectionUtilImpl_LOCK'

export class EvmIntrospectionUtilImpl implements EvmIntrospectionUtil {
  private readonly vm: VM
  private readonly lock: AsyncLock

  constructor() {
    this.lock = new AsyncLock()
  }

  public async getExecutionResult(bytecode: Buffer): Promise<ExecutionResult> {
    const res: ExecResult = await this.runLocked(bytecode)
    const error: EvmError = EvmIntrospectionUtilImpl.getEvmErrorFromVmError(
      res.exceptionError
    )

    const toReturn: ExecutionResult = {
      result: res.returnValue,
    }
    if (!!error) {
      toReturn.error = error
    }
    return toReturn
  }
  public async getStepContextBeforeStep(
    bytecode: Buffer,
    stepIndex: number
  ): Promise<StepContext> {
    let context: StepContext
    const callback: StepCallback = EvmIntrospectionUtilImpl.stepCallbackFactory(
      async (stepContext: StepContext) => {
        if (stepContext.pc === stepIndex && !context) {
          context = stepContext
        }
      }
    )

    await this.runLocked(bytecode, callback)

    return context
  }

  public async getExecutionResultComparison(
    firstBytecode: Buffer,
    secondBytecode: Buffer
  ): Promise<ExecutionResultComparison> {
    const [firstResult, secondResult]: [
      ExecutionResult,
      ExecutionResult
    ] = await Promise.all([
      this.getExecutionResult(firstBytecode),
      this.getExecutionResult(secondBytecode),
    ])

    const resultsDiffer: boolean = !EvmIntrospectionUtilImpl.areExecutionResultsEqual(
      firstResult,
      secondResult
    )
    return {
      resultsDiffer,
      firstResult,
      secondResult,
    }
  }

  public async getExecutionComparisonBeforeStep(
    firstBytecode: Buffer,
    firstStepIndex: number,
    secondBytecode: Buffer,
    secondStepIndex: number
  ): Promise<ExecutionComparison> {
    const [firstContext, secondContext]: [
      StepContext,
      StepContext
    ] = await Promise.all([
      this.getStepContextBeforeStep(firstBytecode, firstStepIndex),
      this.getStepContextBeforeStep(secondBytecode, secondStepIndex),
    ])

    return {
      executionDiffers: !EvmIntrospectionUtilImpl.areStepContextsEqual(
        firstContext,
        secondContext
      ),
      firstContext,
      secondContext,
    }
  }

  private async runLocked(
    bytecode: Buffer,
    stepCallback: StepCallback = EvmIntrospectionUtilImpl.stepCallbackFactory()
  ): Promise<ExecResult> {
    const vm: VM = new VM()
    vm.on('step', stepCallback)

    const bytecodeHash: string = keccak256(bytecode.toString('hex'))
    const hashBuffer: Buffer = Buffer.from(bytecodeHash, 'hex')

    return this.lock.acquire(KEY, async () => {
      const res: ExecResult = await vm.runCode({
        code: bytecode,
        gasLimit: BIG_ENOUGH_GAS_LIMIT,
        address: hashBuffer,
      })
      log.debug(`\nFinished executing ${bytecodeHash}\n`)
      return res
    })
  }

  private static parseStepContext(data: any): StepContext {
    return {
      pc: data['pc'],
      opcode: Opcode.parseByName(data['opcode']['name']),
      stack: data['stack'],
      stackDepth: data['depth'],
      memory: Buffer.from(data['memory']),
      memoryWordCount: data['memoryWordCount'],
    }
  }

  private static stepCallbackFactory(fn?: StepContextCallback): StepCallback {
    return async (data, continueFn) => {
      // log.debug(`raw step data is: ${JSON.stringify(data)}`)
      try {
        const stepContext: StepContext = EvmIntrospectionUtilImpl.parseStepContext(
          data
        )

        if (!!fn) {
          await fn(stepContext)
        }

        const address: string = EvmIntrospectionUtilImpl.getCodeHashTag(
          data['address']
        )
        log.debug(
          `Code hash [${address}] step data: ${EvmIntrospectionUtilImpl.getStepContextString(
            stepContext
          )}`
        )
      } finally {
        continueFn()
      }
    }
  }

  private static getEvmErrorFromVmError(
    vmError: VmError
  ): EvmError | undefined {
    if (!vmError || !vmError.error) {
      return undefined
    }
    switch (vmError.error) {
      case ERROR.OUT_OF_GAS:
        return EvmErrors.OUT_OF_GAS_ERROR
      case ERROR.STACK_UNDERFLOW:
        return EvmErrors.STACK_UNDERFLOW_ERROR
      case ERROR.STACK_OVERFLOW:
        return EvmErrors.STACK_OVERFLOW_ERROR
      case ERROR.INVALID_JUMP:
        return EvmErrors.INVALID_JUMP_ERROR
      case ERROR.INVALID_OPCODE:
        return EvmErrors.INVALID_OPCODE_ERROR
      case ERROR.OUT_OF_RANGE:
        return EvmErrors.OUT_OF_RANGE_ERROR
      case ERROR.REVERT:
        return EvmErrors.REVERT_ERROR
      case ERROR.STATIC_STATE_CHANGE:
        return EvmErrors.STATIC_STATE_CHANGE_ERROR
      case ERROR.INTERNAL_ERROR:
        return EvmErrors.INTERNAL_ERROR
      case ERROR.CREATE_COLLISION:
        return EvmErrors.CREATE_COLLISION_ERROR
      case ERROR.STOP:
        return EvmErrors.STOP_ERROR
      case ERROR.REFUND_EXHAUSTED:
        return EvmErrors.REFUND_EXHAUSTED_ERROR
      default:
        throw Error(`Unrecognized VmError: ${vmError.error}`)
    }
  }

  private static getCodeHashTag(codeBuffer: Buffer): string {
    return codeBuffer.toString('hex').substr(0, 10)
  }

  private static getStepContextString(stepContext: StepContext): string {
    return `{pc: ${stepContext.pc}, opcode: ${
      stepContext.opcode.name
    }, stackDepth: ${stepContext.stackDepth}, stack: [${stepContext.stack
      .map((x) => bufToHexString(x))
      .join(',')}], memoryWordCount: ${
      stepContext.memoryWordCount
    }, memory: [${bufToHexString(stepContext.memory)}]`
  }

  private static areExecutionResultsEqual(
    first: ExecutionResult,
    second: ExecutionResult
  ): boolean {
    return (
      (!first && !second) ||
      (!!first &&
        !!second &&
        first.result.equals(second.result) &&
        ((!first.error && !second.error) ||
          (!!first.error && !!second.error && first.error === second.error)))
    )
  }

  private static areStepContextsEqual(
    first: StepContext,
    second: StepContext
  ): boolean {
    return (
      (!first && !second) ||
      (!!first &&
        !!second &&
        first.pc === second.pc &&
        first.opcode === second.opcode &&
        first.stackDepth === second.stackDepth &&
        first.memoryWordCount === second.memoryWordCount &&
        first.memory.equals(second.memory) &&
        first.stack.map((b) => b.toString()).join() ===
          second.stack.map((b) => b.toString()).join())
    )
  }
}
