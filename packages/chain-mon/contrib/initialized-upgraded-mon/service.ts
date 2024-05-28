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
import sepoliaConfig from '@eth-optimism/contracts-bedrock/deploy-config/sepolia.json'

import { version } from '../../package.json'

const networks = {
  1: {
    name: 'mainnet',
    l1StartingBlockTag: mainnetConfig.l1StartingBlockTag,
  },
  10: {
    name: 'op-mainnet',
    l1StartingBlockTag: null,
  },
  11155111: {
    name: 'sepolia',
    l1StartingBlockTag: sepoliaConfig.l1StartingBlockTag,
  },
  11155420: {
    name: 'op-sepolia',
    l1StartingBlockTag: null,
  },
  420: {
    name: 'op-goerli',
    l1StartingBlockTag: null,
  },
}

// keccak256("Initialized(uint8)") = 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498
const topic_initialized =
  '0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498'

// keccak256("Upgraded(address)") = 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b
const topic_upgraded =
  '0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b'

type InitializedUpgradedMonOptions = {
  rpc: Provider
  startBlockNumber: number
  contracts: string
}

type InitializedUpgradedMonMetrics = {
  initializedCalls: Counter
  upgradedCalls: Counter
  unexpectedRpcErrors: Counter
}

type InitializedUpgradedMonState = {
  chainId: number
  highestUncheckedBlockNumber: number
  contracts: Array<{ label: string; address: string }>
}

export class InitializedUpgradedMonService extends BaseServiceV2<
  InitializedUpgradedMonOptions,
  InitializedUpgradedMonMetrics,
  InitializedUpgradedMonState
> {
  constructor(
    options?: Partial<InitializedUpgradedMonOptions & StandardOptions>
  ) {
    super({
      version,
      name: 'initialized-upgraded-mon',
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
        contracts: {
          validator: validators.str,
          desc: 'JSON array of [{ label, address }] to monitor contracts for',
          public: true,
        },
      },
      metricsSpec: {
        initializedCalls: {
          type: Gauge,
          desc: 'Successful transactions to tracked contracts emitting initialized event',
          labels: ['label', 'address'],
        },
        upgradedCalls: {
          type: Gauge,
          desc: 'Successful transactions to tracked contracts emitting upgraded event',
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
      const block_number =
        l1StartingBlockTag != null
          ? (await this.options.rpc.getBlock(l1StartingBlockTag)).number
          : 0
      this.state.highestUncheckedBlockNumber = block_number
    } else {
      this.state.highestUncheckedBlockNumber = this.options.startBlockNumber
    }

    try {
      this.state.contracts = JSON.parse(this.options.contracts)
    } catch (e) {
      throw new Error(
        'unable to start service because provided options is not valid json'
      )
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
      for (const contract of this.state.contracts) {
        const to =
          transaction.to != null ? transaction.to : transaction['creates']
        if (compareAddrs(contract.address, to)) {
          try {
            const transactionReceipt = await transaction.wait()
            for (const log of transactionReceipt.logs) {
              if (log.topics.includes(topic_initialized)) {
                this.metrics.initializedCalls.inc({
                  label: contract.label,
                  address: contract.address,
                })
                this.logger.info('initialized event', {
                  label: contract.label,
                  address: contract.address,
                })
              } else if (log.topics.includes(topic_upgraded)) {
                this.metrics.upgradedCalls.inc({
                  label: contract.label,
                  address: contract.address,
                })
                this.logger.info('upgraded event', {
                  label: contract.label,
                  address: contract.address,
                })
              }
            }
          } catch (err) {
            // If error is due to transaction failing, ignore transaction
            if (
              err.message.length >= 18 &&
              err.message.slice(0, 18) === 'transaction failed'
            ) {
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
  const service = new InitializedUpgradedMonService()
  service.run()
}
