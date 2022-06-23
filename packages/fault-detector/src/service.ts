import { BaseServiceV2, Gauge, validators } from '@eth-optimism/common-ts'
import { getChainId, sleep, toRpcHexString } from '@eth-optimism/core-utils'
import { CrossChainMessenger } from '@eth-optimism/sdk'
import { Provider } from '@ethersproject/abstract-provider'
import { Contract, ethers } from 'ethers'
import dateformat from 'dateformat'

import {
  findFirstUnfinalizedStateBatchIndex,
  findEventForStateBatch,
} from './helpers'

type Options = {
  l1RpcProvider: Provider
  l2RpcProvider: Provider
  startBatchIndex: number
}

type Metrics = {
  highestCheckedBatchIndex: Gauge
  highestKnownBatchIndex: Gauge
  isCurrentlyMismatched: Gauge
  inUnexpectedErrorState: Gauge
}

type State = {
  scc: Contract
  messenger: CrossChainMessenger
  highestCheckedBatchIndex: number
}

export class FaultDetector extends BaseServiceV2<Options, Metrics, State> {
  constructor(options?: Partial<Options>) {
    super({
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      version: require('../package.json').version,
      name: 'fault-detector',
      loop: true,
      loopIntervalMs: 1000,
      options,
      optionsSpec: {
        l1RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L1',
          secret: true,
        },
        l2RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L2',
          secret: true,
        },
        startBatchIndex: {
          validator: validators.num,
          default: -1,
          desc: 'Batch index to start checking from',
        },
      },
      metricsSpec: {
        highestCheckedBatchIndex: {
          type: Gauge,
          desc: 'Highest good batch index',
        },
        highestKnownBatchIndex: {
          type: Gauge,
          desc: 'Highest known batch index',
        },
        isCurrentlyMismatched: {
          type: Gauge,
          desc: '0 if state is ok, 1 if state is mismatched',
        },
        inUnexpectedErrorState: {
          type: Gauge,
          desc: '0 if service is ok, 1 service is in unexpected error state',
        },
      },
    })
  }

  async init(): Promise<void> {
    this.state.messenger = new CrossChainMessenger({
      l1SignerOrProvider: this.options.l1RpcProvider,
      l2SignerOrProvider: this.options.l2RpcProvider,
      l1ChainId: await getChainId(this.options.l1RpcProvider),
      l2ChainId: await getChainId(this.options.l2RpcProvider),
    })

    // We use this a lot, a bit cleaner to pull out to the top level of the state object.
    this.state.scc = this.state.messenger.contracts.l1.StateCommitmentChain

    // Figure out where to start syncing from.
    if (this.options.startBatchIndex === -1) {
      this.logger.info(`finding appropriate starting height`)
      this.state.highestCheckedBatchIndex =
        await findFirstUnfinalizedStateBatchIndex(this.state.scc)
    } else {
      this.state.highestCheckedBatchIndex = this.options.startBatchIndex
    }

    this.logger.info(`starting height`, {
      startBatchIndex: this.state.highestCheckedBatchIndex,
    })
  }

  async main(): Promise<void> {
    const latestBatchIndex = await this.state.scc.getTotalBatches()
    if (this.state.highestCheckedBatchIndex >= latestBatchIndex.toNumber()) {
      await sleep(15000)
      return
    }

    this.metrics.highestKnownBatchIndex.set(latestBatchIndex.toNumber())

    this.logger.info(`checking batch`, {
      batchIndex: this.state.highestCheckedBatchIndex,
      latestIndex: latestBatchIndex.toNumber(),
    })

    let event: ethers.Event
    try {
      event = await findEventForStateBatch(
        this.state.scc,
        this.state.highestCheckedBatchIndex
      )
    } catch (err) {
      this.logger.error(`got unexpected error while searching for batch`, {
        batchIndex: this.state.highestCheckedBatchIndex,
        error: err,
      })
    }

    const batchTransaction = await event.getTransaction()
    const [stateRoots] = this.state.scc.interface.decodeFunctionData(
      'appendStateBatch',
      batchTransaction.data
    )

    const batchStart = event.args._prevTotalElements.toNumber() + 1
    const batchSize = event.args._batchSize.toNumber()
    const batchEnd = batchStart + batchSize

    const latestBlock = await this.options.l2RpcProvider.getBlockNumber()
    if (latestBlock < batchEnd) {
      this.logger.info(`node is behind, waiting for sync`, {
        batchEnd,
        latestBlock,
      })
      return
    }

    // `getBlockRange` has a limit of 1000 blocks, so we have to break this request out into
    // multiple requests of maximum 1000 blocks in the case that batchSize > 1000.
    let blocks: any[] = []
    for (let i = 0; i < batchSize; i += 1000) {
      const provider = this.options
        .l2RpcProvider as ethers.providers.JsonRpcProvider
      blocks = blocks.concat(
        await provider.send('eth_getBlockRange', [
          toRpcHexString(batchStart + i),
          toRpcHexString(batchStart + i + Math.min(batchSize - i, 1000) - 1),
          false,
        ])
      )
    }

    for (const [i, stateRoot] of stateRoots.entries()) {
      if (blocks[i].stateRoot !== stateRoot) {
        this.metrics.isCurrentlyMismatched.set(1)
        const fpw = await this.state.scc.FRAUD_PROOF_WINDOW()
        this.logger.error(`state root mismatch`, {
          blockNumber: blocks[i].number,
          expectedStateRoot: blocks[i].stateRoot,
          actualStateRoot: stateRoot,
          finalizationTime: dateformat(
            new Date(
              (ethers.BigNumber.from(blocks[i].timestamp).toNumber() +
                fpw.toNumber()) *
                1000
            ),
            'mmmm dS, yyyy, h:MM:ss TT'
          ),
        })
        return
      }
    }

    this.state.highestCheckedBatchIndex++
    this.metrics.highestCheckedBatchIndex.set(
      this.state.highestCheckedBatchIndex
    )

    // If we got through the above without throwing an error, we should be fine to reset.
    this.metrics.isCurrentlyMismatched.set(0)
    this.metrics.inUnexpectedErrorState.set(0)
  }
}

if (require.main === module) {
  const service = new FaultDetector()
  service.run()
}
