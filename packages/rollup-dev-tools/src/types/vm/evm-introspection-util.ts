import {
  ExecutionComparison,
  ExecutionResult,
  ExecutionResultComparison,
  StepContext,
} from './types'

/**
 * Interface defining an EVM utility allowing introspection of EVM
 * state during bytecode processing.
 */
export interface EvmIntrospectionUtil {
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
   * @param stepIndex The index at which context will be captured.
   * @returns The StepContext at the step in question.
   */
  getStepContextBeforeStep(
    bytecode: Buffer,
    stepIndex: number
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
   * @param firstStepIndex The index in the second EVM bytecode at which execution will be compared.
   * @param secondBytecode The second EVM bytecode to run.
   * @param secondStepIndex The index in the second EVM bytecode at which execution will be compared.
   * @returns The ExecutionComparison.
   */
  getExecutionComparisonBeforeStep(
    firstBytecode: Buffer,
    firstStepIndex: number,
    secondBytecode: Buffer,
    secondStepIndex: number
  ): Promise<ExecutionComparison>
}
