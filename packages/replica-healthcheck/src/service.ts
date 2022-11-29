import { Provider, Block } from '@ethersproject/abstract-provider'
import {
  BaseServiceV2,
  Counter,
  Gauge,
  validators,
} from '@eth-optimism/common-ts'
import { sleep } from '@eth-optimism/core-utils'

import { version } from '../package.json'

type HealthcheckOptions = {
  referenceRpcProvider: Provider
  targetRpcProvider: Provider
  onDivergenceWaitMs?: number
}

type HealthcheckMetrics = {
  lastMatchingStateRootHeight: Gauge
  isCurrentlyDiverged: Gauge
  referenceHeight: Gauge
  targetHeight: Gauge
  heightDifference: Gauge
  targetConnectionFailures: Counter
  referenceConnectionFailures: Counter
}

type HealthcheckState = {}

export class HealthcheckService extends BaseServiceV2<
  HealthcheckOptions,
  HealthcheckMetrics,
  HealthcheckState
> {
  constructor(options?: Partial<HealthcheckOptions>) {
    super({
      version,
      name: 'healthcheck',
      loopIntervalMs: 5000,
      options,
      optionsSpec: {
        referenceRpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L1',
          secret: true,
        },
        targetRpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L2',
          secret: true,
        },
        onDivergenceWaitMs: {
          validator: validators.num,
          desc: 'Waiting time in ms per loop when divergence is detected',
          default: 60_000,
        },
      },
      metricsSpec: {
        lastMatchingStateRootHeight: {
          type: Gauge,
          desc: 'Highest matching state root between target and reference',
        },
        isCurrentlyDiverged: {
          type: Gauge,
          desc: 'Whether or not the two nodes are currently diverged',
        },
        referenceHeight: {
          type: Gauge,
          desc: 'Block height of the reference client',
        },
        targetHeight: {
          type: Gauge,
          desc: 'Block height of the target client',
        },
        heightDifference: {
          type: Gauge,
          desc: 'Difference in block heights between the two clients',
        },
        targetConnectionFailures: {
          type: Counter,
          desc: 'Number of connection failures to the target client',
        },
        referenceConnectionFailures: {
          type: Counter,
          desc: 'Number of connection failures to the reference client',
        },
      },
    })
  }

  async main() {
    // Get the latest block from the target client and check for connection failures.
    let targetLatest: Block
    try {
      targetLatest = await this.options.targetRpcProvider.getBlock('latest')
    } catch (err) {
      if (err.message.includes('could not detect network')) {
        this.logger.error('target client not connected')
        this.metrics.targetConnectionFailures.inc()
        return
      } else {
        throw err
      }
    }

    // Get the latest block from the reference client and check for connection failures.
    let referenceLatest: Block
    try {
      referenceLatest = await this.options.referenceRpcProvider.getBlock(
        'latest'
      )
    } catch (err) {
      if (err.message.includes('could not detect network')) {
        this.logger.error('reference client not connected')
        this.metrics.referenceConnectionFailures.inc()
        return
      } else {
        throw err
      }
    }

    // Later logic will depend on the height difference.
    const heightDiff = Math.abs(referenceLatest.number - targetLatest.number)
    const minBlock = Math.min(targetLatest.number, referenceLatest.number)

    // Update these metrics first so they'll refresh no matter what.
    this.metrics.targetHeight.set(targetLatest.number)
    this.metrics.referenceHeight.set(referenceLatest.number)
    this.metrics.heightDifference.set(heightDiff)

    this.logger.info(`latest block heights`, {
      targetHeight: targetLatest.number,
      referenceHeight: referenceLatest.number,
      heightDifference: heightDiff,
      minBlockNumber: minBlock,
    })

    const reference = await this.options.referenceRpcProvider.getBlock(minBlock)
    if (!reference) {
      // This is ok, but we should log it and restart the loop.
      this.logger.info(`reference block was not found`, {
        blockNumber: reference.number,
      })
      return
    }

    const target = await this.options.targetRpcProvider.getBlock(minBlock)
    if (!target) {
      // This is ok, but we should log it and restart the loop.
      this.logger.info(`target block was not found`, {
        blockNumber: target.number,
      })
      return
    }

    // We used to use state roots here, but block hashes are even more reliable because they will
    // catch discrepancies in blocks that may not impact the state. For example, if clients have
    // blocks with two different timestamps, the state root will only diverge if the timestamp is
    // actually used during the transaction(s) within the block.
    if (reference.hash !== target.hash) {
      this.logger.error(`reference client has different hash for block`, {
        blockNumber: target.number,
        referenceHash: reference.hash,
        targetHash: target.hash,
      })

      // The main loop polls for "latest" so aren't checking every block. We need to use a binary
      // search to find the first block where a mismatch occurred.
      this.logger.info(`beginning binary search to find first mismatched block`)

      let start = 0
      let end = target.number
      while (start !== end) {
        const mid = Math.floor((start + end) / 2)
        this.logger.info(`checking block`, { blockNumber: mid })
        const blockA = await this.options.referenceRpcProvider.getBlock(mid)
        const blockB = await this.options.targetRpcProvider.getBlock(mid)

        if (blockA.hash === blockB.hash) {
          start = mid + 1
        } else {
          end = mid
        }
      }

      this.logger.info(`found first mismatched block`, { blockNumber: end })
      this.metrics.lastMatchingStateRootHeight.set(end)
      this.metrics.isCurrentlyDiverged.set(1)

      // Old version of the service would exit here, but we want to keep looping just in case the
      // the system recovers later. This is better than exiting because it means we don't have to
      // restart the entire service. Running these checks once per minute will not trigger too many
      // requests, so this should be fine.
      await sleep(this.options.onDivergenceWaitMs)
      return
    }

    this.logger.info(`blocks are matching`, {
      blockNumber: target.number,
    })

    // Update latest matching state root height and reset the diverged metric in case it was set.
    this.metrics.lastMatchingStateRootHeight.set(target.number)
    this.metrics.isCurrentlyDiverged.set(0)
  }
}

if (require.main === module) {
  const service = new HealthcheckService()
  service.run()
}
