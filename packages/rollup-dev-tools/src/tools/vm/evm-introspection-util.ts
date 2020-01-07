/* External Imports */
import {
  bufToHexString,
  getLogger,
  keccak256,
  Logger,
  add0x,
  remove0x,
  logError,
  hexStrToBuf,
} from '@pigi/core-utils'
import { Address, Opcode } from '@pigi/rollup-core'

import * as AsyncLock from 'async-lock'
import * as abi from 'ethereumjs-abi'

import BN = require('bn.js')
import VM from 'ethereumjs-vm'
import { ethers } from 'ethers'
import { promisify } from 'util'
import { Transaction } from 'ethereumjs-tx'
import { EVMResult, ExecResult } from 'ethereumjs-vm/dist/evm/evm'
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
const DEFAULT_ACCOUNT_PK: string =
  '0xe331b6d69882b4cb4ea581d88e0b604039a3de5967688d3dcffdd2270c0fd109'

export class EvmIntrospectionUtilImpl implements EvmIntrospectionUtil {
  private nonce: number

  private readonly vm: VM
  private readonly lock: AsyncLock
  private readonly wallet: ethers.Wallet

  public static async create(
    accountPK: string = DEFAULT_ACCOUNT_PK
  ): Promise<EvmIntrospectionUtilImpl> {
    const util: EvmIntrospectionUtilImpl = new EvmIntrospectionUtilImpl(
      accountPK
    )

    await util.init()
    return util
  }

  private constructor(private readonly accountPK: string) {
    this.vm = new VM()
    this.lock = new AsyncLock()
    this.wallet = new ethers.Wallet(add0x(accountPK))
    this.nonce = 0
  }

  private async init(): Promise<void> {
    // Give account 100 ETH
    await promisify(this.vm.stateManager.putAccount.bind(this.vm.stateManager))(
      Buffer.from(remove0x(this.wallet.address), 'hex'),
      { balance: 100e18 }
    )
  }

  public async deployContract(
    bytecode: Buffer,
    abiEncodedParameters?: Buffer
  ): Promise<ExecutionResult> {
    const params: string = !!abiEncodedParameters
      ? abiEncodedParameters.toString('hex')
      : ''
    const data: string = add0x(bytecode.toString('hex') + params)

    const tx: Transaction = new Transaction({
      value: 0,
      gasLimit: BIG_ENOUGH_GAS_LIMIT,
      gasPrice: 1,
      data,
      nonce: this.nonce++,
    })

    tx.sign(Buffer.from(remove0x(this.wallet.privateKey), 'hex'))

    const deployResult: EVMResult = await this.vm.runTx({ tx })

    if (!!deployResult.execResult.exceptionError) {
      const msg: string = `Error deploying contract [${bytecode.toString(
        'hex'
      )}] with params: [${params}]: ${
        deployResult.execResult.exceptionError.errorType
      }`
      log.debug(msg)
      return {
        error: EvmIntrospectionUtilImpl.getEvmErrorFromVmError(
          deployResult.execResult.exceptionError
        ),
        result: deployResult.createdAddress,
      }
    }

    return {
      result: deployResult.createdAddress,
    }
  }

  public async callContract(
    address: Address,
    method: string,
    methodTypes: string[] = [],
    abiEncodedParams: Buffer = Buffer.from('', 'hex')
  ): Promise<ExecutionResult> {
    const data: Buffer = Buffer.concat([
      abi.methodID(method, methodTypes),
      abiEncodedParams,
    ])

    const result: EVMResult = await this.vm.runCall({
      to: hexStrToBuf(address),
      caller: hexStrToBuf(this.wallet.address),
      origin: hexStrToBuf(this.wallet.address),
      data,
    })

    if (result.execResult.exceptionError) {
      const params: string = bufToHexString(abiEncodedParams)
      const msg: string = `Error calling contract [${address}] method [${method}] with params: [${params}]: ${JSON.stringify(
        result.execResult.exceptionError
      )}`
      log.error(msg)
      return {
        error: EvmIntrospectionUtilImpl.getEvmErrorFromVmError(
          result.execResult.exceptionError
        ),
        result: result.execResult.returnValue,
      }
    }

    return {
      result: result.execResult.returnValue,
    }
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
    bytecodeIndex: number
  ): Promise<StepContext> {
    let context: StepContext
    const callback: StepCallback = EvmIntrospectionUtilImpl.stepCallbackFactory(
      async (stepContext: StepContext) => {
        if (stepContext.pc === bytecodeIndex && !context) {
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
    firstBytecodeIndex: number,
    secondBytecode: Buffer,
    secondBytecodeIndex: number
  ): Promise<ExecutionComparison> {
    const [firstContext, secondContext]: [
      StepContext,
      StepContext
    ] = await Promise.all([
      this.getStepContextBeforeStep(firstBytecode, firstBytecodeIndex),
      this.getStepContextBeforeStep(secondBytecode, secondBytecodeIndex),
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
    const bytecodeHash: string = keccak256(bytecode.toString('hex'))
    const hashBuffer: Buffer = Buffer.from(bytecodeHash, 'hex')

    return this.lock.acquire(KEY, async () => {
      this.vm.on('step', stepCallback)

      try {
        const res: ExecResult = await this.vm.runCode({
          code: bytecode,
          gasLimit: BIG_ENOUGH_GAS_LIMIT,
          address: hashBuffer,
        })
        log.debug(`\nFinished executing ${bytecodeHash}\n`)

        return res
      } catch (e) {
        logError(
          log,
          `Error running bytecode ${add0x(bytecode.toString('hex'))}`,
          e
        )
        throw e
      } finally {
        // Always make sure to unsubscribe one-time step callback function
        this.vm.removeListener('step', stepCallback)
      }
    })
  }

  private static parseStepContext(data: any): StepContext {
    const stack: Buffer[] = data['stack'].map((x) => x.toBuffer())
    return {
      pc: data['pc'],
      opcode: Opcode.parseByName(data['opcode']['name']),
      stack,
      stackDepth: stack.length,
      memory: Buffer.from(data['memory']),
      memoryWordCount: data['memoryWordCount'].toNumber(),
    }
  }

  private static stepCallbackFactory(fn?: StepContextCallback): StepCallback {
    return async (data, continueFn) => {
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
        // first.pc === second.pc &&  -- This will probably not line up for different executions
        first.opcode === second.opcode &&
        first.stackDepth === second.stackDepth &&
        first.memoryWordCount === second.memoryWordCount &&
        first.memory.equals(second.memory) &&
        first.stack.map((b) => b.toString('hex')).join() ===
          second.stack.map((b) => b.toString('hex')).join())
    )
  }
}
