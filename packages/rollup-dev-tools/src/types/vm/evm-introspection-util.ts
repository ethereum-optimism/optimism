/* External Imports */
import { Address } from '@eth-optimism/rollup-core'

/* Internal Imports */
import {
  ExecutionComparison,
  ExecutionResult,
  ExecutionResultComparison,
  StepContext,
  CallContext,
} from './types'

/**
 * Interface defining an EVM utility allowing introspection of EVM
 * state during bytecode processing.
 */
export interface EvmIntrospectionUtil {
  /**
   * Deploys a contract with the provided bytecode and returns its resulting address.
   *
   * @param initcode The initcode of the contract to deploy.
   * @parameter abiEncodedParameters The ABI-encoded constructor args.
   * @returns The ExecutionResult containing the deployed contract address or the deployment error.
   */
  deployContract(
    initcode: Buffer,
    abiEncodedParameters?: Buffer
  ): Promise<ExecutionResult>

  /**
   * Deploys a contract with the provided bytecode and to the specified address.
   *
   * @param bytecode The bytecode of the contract to deploy.
   * @param address The address to deploy the bytecode to
   */
  deployBytecodeToAddress(
    deployedBytecode: Buffer,
    address: Buffer
  ): Promise<void>

  /**
   * Gets the deployed bytecode for a given contract address which has been deployed
   *
   * @param address The address of the contract to get bytecode of.
   * @returns The deployed bytecode.
   */
  getContractDeployedBytecode(address: Buffer): Promise<Buffer>

  /**
   * Calls the provided method of the provided contract, passing in the
   * provided parameters.
   *
   * @param address The address of the contract to call.
   * @param method The method to call as a string.
   * @param paramTypes The ordered array of parameter types
   * @param abiEncodedParams The ABI-encoded parameters for the call.
   * @returns The ExecutionResult of the call
   */
  callContract(
    address: Address,
    method: string,
    paramTypes?: string[],
    abiEncodedParams?: Buffer
  ): Promise<ExecutionResult>

  /**
   * Gets the call details and call context right before the execution of the
   * provided bytecode at its first CALL.
   *
   * @param bytecode The bytecode leading to a CALL to execute.
   * @returns The CallContext at the first CALL in the bytecode.
   */
  getCallContext(bytecode: Buffer): Promise<CallContext>

  /**
   * Gets the result from executing the provided bytecode.
   *
   * @param bytecode The bytecode to execute.
   * @returns The ExecutionResult
   */
  getExecutionResult(bytecode: Buffer): Promise<ExecutionResult>

  /**
   * Gets the execution context right before the execution of the
   * provided bytecode at the provided index.
   *
   * @param bytecode The bytecode to execute.
   * @param bytecodeIndex The index at which context will be captured.
   * @returns The StepContext at the step in question.
   */
  getStepContextBeforeStep(
    bytecode: Buffer,
    bytecodeIndex: number
  ): Promise<StepContext>

  /**
   * Gets the ExecutionResultComparison between two different sets of bytecode to run.
   *
   * @param firstBytecode The first EVM bytecode to run.
   * @param secondBytecode The second EVM bytecode to run.
   * @returns The ExecutionResultComparison comparing the execution results.
   */
  getExecutionResultComparison(
    firstBytecode: Buffer,
    secondBytecode: Buffer
  ): Promise<ExecutionResultComparison>

  /**
   * Gets the ExecutionComparison, comparing the execution of the two provided bytecodes
   * before the two provided indexes in the bytecode.
   *
   * @param firstBytecode The first EVM bytecode to run.
   * @param firstBytecodeIndex The index in the second EVM bytecode at which execution will be compared.
   * @param secondBytecode The second EVM bytecode to run.
   * @param secondBytecodeIndex The index in the second EVM bytecode at which execution will be compared.
   * @returns The ExecutionComparison.
   */
  getExecutionComparisonBeforeStep(
    firstBytecode: Buffer,
    firstBytecodeIndex: number,
    secondBytecode: Buffer,
    secondBytecodeIndex: number
  ): Promise<ExecutionComparison>
}
