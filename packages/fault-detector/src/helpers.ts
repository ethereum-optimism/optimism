import { Contract } from 'ethers'

/**
 * Partial event interface, meant to reduce the size of the event cache to avoid
 * running out of memory.
 */
export interface PartialEvent {
  blockNumber: number
  transactionHash: string
  args: any
}

// Event caching is necessary for the fault detector to work properly with Geth.
const caches: {
  [contractAddress: string]: {
    highestBlock: number
    eventCache: Map<string, PartialEvent>
  }
} = {}

/**
 * Retrieves the cache for a given address.
 *
 * @param address Address to get cache for.
 * @returns Address cache.
 */
const getCache = (
  address: string
): {
  highestBlock: number
  eventCache: Map<string, PartialEvent>
} => {
  if (!caches[address]) {
    caches[address] = {
      highestBlock: 0,
      eventCache: new Map(),
    }
  }

  return caches[address]
}

/**
 * Updates the event cache for the SCC.
 *
 * @param scc The State Commitment Chain contract.
 */
export const updateStateBatchEventCache = async (
  scc: Contract
): Promise<void> => {
  const cache = getCache(scc.address)
  let currentBlock = cache.highestBlock
  const endingBlock = await scc.provider.getBlockNumber()
  let step = endingBlock - currentBlock
  let failures = 0
  while (currentBlock < endingBlock) {
    try {
      const events = await scc.queryFilter(
        scc.filters.StateBatchAppended(),
        currentBlock,
        currentBlock + step
      )
      for (const event of events) {
        cache.eventCache[event.args._batchIndex.toNumber()] = {
          blockNumber: event.blockNumber,
          transactionHash: event.transactionHash,
          args: event.args,
        }
      }

      // Update the current block and increase the step size for the next iteration.
      currentBlock += step
      step = Math.ceil(step * 2)
    } catch {
      // Might happen if we're querying too large an event range.
      step = Math.floor(step / 2)

      // When the step gets down to zero, we're pretty much guaranteed that range size isn't the
      // problem. If we get three failures like this in a row then we should just give up.
      if (step === 0) {
        failures++
      } else {
        failures = 0
      }

      // We've failed 3 times in a row, we're probably stuck.
      if (failures >= 3) {
        throw new Error('failed to update event cache')
      }
    }
  }

  // Update the highest block.
  cache.highestBlock = endingBlock
}

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
): Promise<PartialEvent> => {
  const cache = getCache(scc.address)

  // Try to find the event in cache first.
  if (cache.eventCache[index]) {
    return cache.eventCache[index]
  }

  // Update the event cache if we don't have the event.
  await updateStateBatchEventCache(scc)

  // Event better be in cache now!
  if (cache.eventCache[index] === undefined) {
    throw new Error(`unable to find event for batch ${index}`)
  }

  return cache.eventCache[index]
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
    const block = await scc.provider.getBlock(event.blockNumber)

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
