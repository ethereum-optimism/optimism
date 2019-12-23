/* External Imports */
import { RollupBlock } from '@pigi/rollup-core'

/**
 * Handles all rollup block queueing, submission, and monitoring.
 */
export interface RollupBlockSubmitter {
  submitBlock(rollupBlock: RollupBlock): Promise<void>
}
