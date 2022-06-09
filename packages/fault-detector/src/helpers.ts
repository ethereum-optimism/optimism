import { Contract, ethers } from 'ethers'

/**
 * Finds the Event that corresponds to a given state batch by index.
 *
 * @param scc StateCommitmentChain contract.
 * @param index State batch index to search for.
 * @returns Event corresponding to the batch.
 */
export const findEventForStateBatch = async (
  scc: Contract,
  index: number
): Promise<ethers.Event> => {
  const events = await scc.queryFilter(scc.filters.StateBatchAppended(index))

  // Only happens if the batch with the given index does not exist yet.
  if (events.length === 0) {
    throw new Error(`unable to find event for batch`)
  }

  // Should never happen.
  if (events.length > 1) {
    throw new Error(`found too many events for batch`)
  }

  return events[0]
}

/**
 * Finds the first state batch index that has not yet passed the fault proof window.
 *
 * @param scc StateCommitmentChain contract.
 * @returns Starting state root batch index.
 */
export const findFirstUnfinalizedStateBatchIndex = async (
  scc: Contract
): Promise<number> => {
  const fpw = (await scc.FRAUD_PROOF_WINDOW()).toNumber()
  const latestBlock = await scc.provider.getBlock('latest')
  const totalBatches = (await scc.getTotalBatches()).toNumber()

  // Perform a binary search to find the next batch that will pass the challenge period.
  let lo = 0
  let hi = totalBatches
  while (lo !== hi) {
    const mid = Math.floor((lo + hi) / 2)
    const event = await findEventForStateBatch(scc, mid)
    const block = await event.getBlock()

    if (block.timestamp + fpw < latestBlock.timestamp) {
      lo = mid + 1
    } else {
      hi = mid
    }
  }

  // Result will be zero if the chain is less than FPW seconds old. Only returns undefined in the
  // case that no batches have been submitted for an entire challenge period.
  if (lo === totalBatches) {
    return undefined
  } else {
    return lo
  }
}
