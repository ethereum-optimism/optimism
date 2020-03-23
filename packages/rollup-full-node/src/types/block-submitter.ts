/* External Imports */
import {L2ToL1Message, RollupBlock} from '@eth-optimism/rollup-core'

/**
 * Handles all rollup block queueing, submission, and monitoring.
 */
export interface RollupBlockSubmitter {
  submitBlock(rollupBlock: RollupBlock): Promise<void>
}

/**
 * Temporary until block submission works properly.
 */
export interface L2ToL1MessageSubmitter {
  submitMessage(l2ToL1Message: L2ToL1Message): Promise<void>
}