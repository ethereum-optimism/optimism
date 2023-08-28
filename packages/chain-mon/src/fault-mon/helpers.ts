import { Contract } from 'ethers'
import { Logger } from '@eth-optimism/common-ts'
import { BedrockOutputData } from '@eth-optimism/core-utils'

/**
 * Finds the BedrockOutputData that corresponds to a given output index.
 *
 * @param oracle Output oracle contract
 * @param index Output index to search for.
 * @returns BedrockOutputData corresponding to the output index.
 */
export const findOutputForIndex = async (
  oracle: Contract,
  index: number,
  logger?: Logger
): Promise<BedrockOutputData> => {
  try {
    const proposal = await oracle.getL2Output(index)
    return {
      outputRoot: proposal.outputRoot,
      l1Timestamp: proposal.timestamp.toNumber(),
      l2BlockNumber: proposal.l2BlockNumber.toNumber(),
      l2OutputIndex: index,
    }
  } catch (err) {
    logger?.fatal('error when calling L2OuputOracle.getL2Output', {
      errors: err,
    })
    throw new Error(`unable to find output for index ${index}`)
  }
}

/**
 * Finds the first L2 output index that has not yet passed the fault proof window.
 *
 * @param oracle Output oracle contract.
 * @returns Starting L2 output index.
 */
export const findFirstUnfinalizedOutputIndex = async (
  oracle: Contract,
  fpw: number,
  logger?: Logger
): Promise<number> => {
  const latestBlock = await oracle.provider.getBlock('latest')
  const totalOutputs = (await oracle.nextOutputIndex()).toNumber()

  // Perform a binary search to find the next batch that will pass the challenge period.
  let lo = 0
  let hi = totalOutputs
  while (lo !== hi) {
    const mid = Math.floor((lo + hi) / 2)
    const outputData = await findOutputForIndex(oracle, mid, logger)

    if (outputData.l1Timestamp + fpw < latestBlock.timestamp) {
      lo = mid + 1
    } else {
      hi = mid
    }
  }

  // Result will be zero if the chain is less than FPW seconds old. Only returns undefined in the
  // case that no batches have been submitted for an entire challenge period.
  if (lo === totalOutputs) {
    return undefined
  } else {
    return lo
  }
}
