import { BaseServiceV2, Gauge, validators } from '@eth-optimism/common-ts'
import { sleep, toRpcHexString } from '@eth-optimism/core-utils'
import { CrossChainMessenger } from '@eth-optimism/sdk'
import { Provider } from '@ethersproject/abstract-provider'
import { ethers } from 'ethers'
import dateformat from 'dateformat'

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
  messenger: CrossChainMessenger
  highestCheckedBatchIndex: number
}

export class FaultDetector extends BaseServiceV2<Options, Metrics, State> {
  constructor(options?: Partial<Options>) {
    super({
      name: 'fault-detector',
      loop: true,
      loopIntervalMs: 1000,
      options,
      optionsSpec: {
        l1RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L1',
        },
        l2RpcProvider: {
          validator: validators.provider,
          desc: 'Provider for interacting with L2',
        },
        startBatchIndex: {
          validator: validators.num,
          default: 0,
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
    const network = await this.options.l1RpcProvider.getNetwork()
    this.state.messenger = new CrossChainMessenger({
      l1SignerOrProvider: this.options.l1RpcProvider,
      l2SignerOrProvider: this.options.l2RpcProvider,
      l1ChainId: network.chainId,
    })

    this.state.highestCheckedBatchIndex = this.options.startBatchIndex
  }

  async main(): Promise<void> {
    const latestBatchIndex =
      await this.state.messenger.contracts.l1.StateCommitmentChain.getTotalBatches()
    if (this.state.highestCheckedBatchIndex >= latestBatchIndex.toNumber()) {
      await sleep(15000)
      return
    }

    this.metrics.highestKnownBatchIndex.set(latestBatchIndex.toNumber())

    this.logger.info(`checking batch`, {
      batchIndex: this.state.highestCheckedBatchIndex,
    })

    const targetEvents =
      await this.state.messenger.contracts.l1.StateCommitmentChain.queryFilter(
        this.state.messenger.contracts.l1.StateCommitmentChain.filters.StateBatchAppended(
          this.state.highestCheckedBatchIndex
        )
      )

    if (targetEvents.length === 0) {
      this.logger.error(`unable to find event for batch`, {
        batchIndex: this.state.highestCheckedBatchIndex,
      })
      this.metrics.inUnexpectedErrorState.set(1)
      return
    }

    if (targetEvents.length > 1) {
      this.logger.error(`found too many events for batch`, {
        batchIndex: this.state.highestCheckedBatchIndex,
      })
      this.metrics.inUnexpectedErrorState.set(1)
      return
    }

    const targetEvent = targetEvents[0]
    const batchTransaction = await targetEvent.getTransaction()
    const [stateRoots] =
      this.state.messenger.contracts.l1.StateCommitmentChain.interface.decodeFunctionData(
        'appendStateBatch',
        batchTransaction.data
      )

    const batchStart = targetEvent.args._prevTotalElements.toNumber() + 1
    const batchSize = targetEvent.args._batchSize.toNumber()

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
        const fpw =
          await this.state.messenger.contracts.l1.StateCommitmentChain.FRAUD_PROOF_WINDOW()
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

    this.metrics.highestCheckedBatchIndex.set(
      this.state.highestCheckedBatchIndex
    )
    this.state.highestCheckedBatchIndex++

    // If we got through the above without throwing an error, we should be fine to reset.
    this.metrics.isCurrentlyMismatched.set(0)
    this.metrics.inUnexpectedErrorState.set(0)
  }
}

if (require.main === module) {
  const service = new FaultDetector()
  service.run()
}
