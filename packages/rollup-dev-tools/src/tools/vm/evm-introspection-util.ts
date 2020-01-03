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
  EvmIntrospectionUtil,
  ExecutionResultComparison,
  OUT_OF_GAS_ERROR,
  STOP_ERROR,
  REFUND_EXHAUSTED_ERROR,
  CREATE_COLLISION_ERROR,
  INTERNAL_ERROR,
  STATIC_STATE_CHANGE_ERROR,
  REVERT_ERROR,
  OUT_OF_RANGE_ERROR,
  INVALID_OPCODE_ERROR,
  INVALID_JUMP_ERROR,
  STACK_OVERFLOW_ERROR,
  STACK_UNDERFLOW_ERROR,
  StepContext,
} from '../../types/vm'

const log: Logger = getLogger('evm-util')

const BIG_ENOUGH_GAS_LIMIT: any = new BN('ffffffff', 'hex')

export class EvmIntrospectionUtilImpl implements EvmIntrospectionUtil {
  private readonly vm: VM
  private readonly lock: AsyncLock
  private currentlyRunning: string = 'none'

  constructor() {
    this.vm = new VM()
    this.lock = new AsyncLock()
    this.vm.on('step', this.onStep)
  }

  public async getExecutionResultComparison(
    binaryOne: Buffer,
    binaryTwo: Buffer
  ): Promise<ExecutionResultComparison> {
    const binaryOneHash: string = keccak256(binaryOne.toString('hex'))
    const binaryOneHashBuffer: Buffer = Buffer.from(binaryOneHash, 'hex')
    const res1: ExecResult = await this.vm.runCode({
      code: binaryOne,
      gasLimit: BIG_ENOUGH_GAS_LIMIT,
      address: binaryOneHashBuffer,
    })
    log.debug(`\nFinished executing ${binaryOneHash}\n`)

    const binaryTwoHash: string = keccak256(binaryTwo.toString('hex'))
    const binaryTwoHashBuffer: Buffer = Buffer.from(binaryTwoHash, 'hex')
    const res2: ExecResult = await this.vm.runCode({
      code: binaryTwo,
      gasLimit: BIG_ENOUGH_GAS_LIMIT,
      address: binaryTwoHashBuffer,
    })
    log.debug(`\nFinished executing ${binaryTwoHash}\n`)

    const firstError: EvmError = this.getEvmErrorFromVmError(
      res1.exceptionError
    )
    const secondError: EvmError = this.getEvmErrorFromVmError(
      res2.exceptionError
    )

    const toReturn: ExecutionResultComparison = {
      resultsDiffer: !(
        res1.returnValue.equals(res2.returnValue) && firstError === secondError
      ),
      firstResult: res1.returnValue,
      secondResult: res2.returnValue,
    }
    if (!!firstError) {
      toReturn.firstError = firstError
    }
    if (!!secondError) {
      toReturn.secondError = secondError
    }

    return toReturn
  }

  private async onStep(data, continueFn: () => void): Promise<void> {
    try {
      const stepContext: StepContext = {
        pc: data['pc'],
        opcode: Opcode.parseByName(data['opcode']['name']),
        stack: data['stack'],
        stackDepth: data['depth'],
        memory: Buffer.of(data['memory']),
        memoryWordCount: data['memoryWordCount'],
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

  private getEvmErrorFromVmError(vmError: VmError): EvmError | undefined {
    if (!vmError || !vmError.error) {
      return undefined
    }
    switch (vmError.error) {
      case ERROR.OUT_OF_GAS:
        return OUT_OF_GAS_ERROR
      case ERROR.STACK_UNDERFLOW:
        return STACK_UNDERFLOW_ERROR
      case ERROR.STACK_OVERFLOW:
        return STACK_OVERFLOW_ERROR
      case ERROR.INVALID_JUMP:
        return INVALID_JUMP_ERROR
      case ERROR.INVALID_OPCODE:
        return INVALID_OPCODE_ERROR
      case ERROR.OUT_OF_RANGE:
        return OUT_OF_RANGE_ERROR
      case ERROR.REVERT:
        return REVERT_ERROR
      case ERROR.STATIC_STATE_CHANGE:
        return STATIC_STATE_CHANGE_ERROR
      case ERROR.INTERNAL_ERROR:
        return INTERNAL_ERROR
      case ERROR.CREATE_COLLISION:
        return CREATE_COLLISION_ERROR
      case ERROR.STOP:
        return STOP_ERROR
      case ERROR.REFUND_EXHAUSTED:
        return REFUND_EXHAUSTED_ERROR
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
}
