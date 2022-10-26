import { ethers } from 'ethers'

/**
 * Helper function for querying all events for a given contract/filter. Improves on the standard
 * event querying functionality by decreasing the block range by half when a query errors out. If
 * the query succeeds, event range will return back to the default size, and so on. Also allows
 * more advanced filtering during the querying process to avoid OOM issues.
 *
 * @param contract Contract to query events for.
 * @param options Options for the query.
 * @returns Array of events.
 */
export const advancedQueryFilter = async (
  contract: ethers.Contract,
  options: {
    queryFilter: ethers.EventFilter
    filter?: (event: ethers.Event) => boolean
    startBlock?: number
    endBlock?: number
  }
): Promise<ethers.Event[]> => {
  const defaultStep = 500000
  const end = options.endBlock ?? (await contract.provider.getBlockNumber())
  let step = defaultStep
  let i = options.startBlock ?? 0
  let events: ethers.Event[] = []

  while (i < end) {
    try {
      const allEvents = await contract.queryFilter(
        options.queryFilter,
        i,
        i + step
      )
      const matching = options.filter
        ? allEvents.filter(options.filter)
        : allEvents
      events = events.concat(matching)
      i += step
      step = step * 2
    } catch (err) {
      step = Math.floor(step / 2)
      if (step < 1) {
        throw err
      }
    }
  }

  return events
}
