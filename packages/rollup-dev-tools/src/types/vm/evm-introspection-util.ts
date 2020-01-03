import { ExecutionResultComparison } from './types'

/**
 * Interface defining an EVM utility allowing introspection of EVM
 * state during bytecode processing.
 */
export interface EvmIntrospectionUtil {
  /**
   * Gets the ExecutionResultComparison between two different sets of bytecode to run.
   *
   * @param binaryOne The first EVM bytecode to run.
   * @param binaryTwo The second EVM bytecode to run.
   * @returns The ExecutionResultComparison comparing the execution results.
   */
  getExecutionResultComparison(
    binaryOne: Buffer,
    binaryTwo: Buffer
  ): Promise<ExecutionResultComparison>
}
