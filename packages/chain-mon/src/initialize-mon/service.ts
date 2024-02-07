import {
  BaseServiceV2,
  StandardOptions,
  Gauge,
  Counter,
  validators,
  waitForProvider,
} from '@eth-optimism/common-ts'
import { getChainId, compareAddrs } from '@eth-optimism/core-utils'
import { Provider, TransactionResponse } from '@ethersproject/abstract-provider'
import mainnetConfig from '@eth-optimism/contracts-bedrock/deploy-config/mainnet.json'
import goerliConfig from '@eth-optimism/contracts-bedrock/deploy-config/goerli.json'

import { version } from '../../package.json'

const networks = {
  1: {
    name: 'mainnet',
    l1StartingBlockTag: mainnetConfig.l1StartingBlockTag,
    contracts: [
      {
        label: 'SystemConfig',
        address: '0x9ba6e03d8b90de867373db8cf1a58d2f7f006b3a',
      },
    ],
  },
  10: {
    name: 'goerli',
    l1StartingBlockTag: goerliConfig.l1StartingBlockTag,
    contracts: [
    ],
  },
}

type InitializeMonOptions = {
  rpc: Provider
  startBlockNumber: number
}

type InitializeMonMetrics = {
  initializedCalls: Counter
  unexpectedRpcErrors: Counter
}

type InitializeMonState = {
  chainId: number
  highestUncheckedBlockNumber: number
}

export class InitializeMonService extends BaseServiceV2<
  InitializeMonOptions,
  InitializeMonMetrics,
  InitializeMonState
> {
  constructor(options?: Partial<InitializeMonOptions & StandardOptions>) {
    super({
      version,
      name: 'initialize-mon',
      loop: true,
      options: {
        loopIntervalMs: 1000,
        ...options,
      },
      optionsSpec: {
        rpc: {
          validator: validators.provider,
          desc: 'Provider for network to monitor balances on',
        },
        startBlockNumber: {
          validator: validators.num,
          default: -1,
          desc: 'L1 block number to start checking from',
          public: true,
        },
      },
      metricsSpec: {
        initializedCalls: {
          type: Gauge,
          desc: 'Successful transactions to tracked contracts emitting initialized event',
          labels: ['label', 'address'],
        },
        unexpectedRpcErrors: {
          type: Counter,
          desc: 'Number of unexpected RPC errors',
          labels: ['section', 'name'],
        },
      },
    })
  }

  protected async init(): Promise<void> {
    // Connect to L1.
    await waitForProvider(this.options.rpc, {
      logger: this.logger,
      name: 'L1',
    })

    this.state.chainId = await getChainId(this.options.rpc)

    const l1StartingBlockTag = networks[this.state.chainId].l1StartingBlockTag

    if (this.options.startBlockNumber === -1) {
      const block = await this.options.rpc.getBlock(l1StartingBlockTag)
      this.state.highestUncheckedBlockNumber = block.number
    } else {
      this.state.highestUncheckedBlockNumber = this.options.startBlockNumber
    }
  }

  protected async main(): Promise<void> {
    if (
      (await this.options.rpc.getBlockNumber()) <
      this.state.highestUncheckedBlockNumber
    ) {
      this.logger.info('Waiting for new blocks')
      return
    }

    const network = networks[this.state.chainId]
    const contracts = network.contracts

    const block = await this.options.rpc.getBlock(
      this.state.highestUncheckedBlockNumber
    )
    this.logger.info('Checking block', {
      number: block.number,
    })

    const transactions: TransactionResponse[] = []
    for (const txHash of block.transactions) {
      const t = await this.options.rpc.getTransaction(txHash)
      transactions.push(t)
    }

    for (const transaction of transactions) {
      for (const contract of contracts) {
        const to = transaction.to != null ? transaction.to : transaction["creates"]
        if (compareAddrs(contract.address, to)) {
          try {
            const transactionReceipt = await transaction.wait()
            for (const log of transactionReceipt.logs) {
              // keccak256("Initialized(suint8)") = 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498
              if (log.topics.includes('0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498')) {
                this.metrics.initializedCalls.inc({
                  label: contract.label,
                  address: contract.address,
                })
                this.logger.info('initialized event', {
                  label: contract.label,
                  address: contract.address,
                })
              }
            }
          } catch (err) {
            // If error is due to transaction failing, ignore transaction
            if (err.message.length >= 18 && err.message.slice(0, 18) === 'transaction failed') {
              break
            }
            // Otherwise, we have an unexpected RPC error
            this.logger.info(`got unexpected RPC error`, {
              section: 'creations',
              name: 'NULL',
              err,
            })

            this.metrics.unexpectedRpcErrors.inc({
              section: 'creations',
              name: 'NULL',
            })

            return
          }
        }
      }
    }
    this.logger.info('Checked block', {
      number: this.state.highestUncheckedBlockNumber,
    })
    this.state.highestUncheckedBlockNumber++
  }
}

if (require.main === module) {
  const service = new InitializeMonService()
  service.run()
}
