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

import { version } from '../../package.json'

const networks = {
  1: {
    name: 'mainnet',
    l1StartingBlockTag: mainnetConfig.l1StartingBlockTag,
    accounts: [
      {
        label: 'Proposer',
        wallet: mainnetConfig.l2OutputOracleProposer,
        target: '0xdfe97868233d1aa22e815a266982f2cf17685a27',
      },
      {
        label: 'Batcher',
        wallet: mainnetConfig.batchSenderAddress,
        target: mainnetConfig.batchInboxAddress,
      },
    ],
  },
}

type WalletMonOptions = {
  rpc: Provider
  startBlockNumber: number
}

type WalletMonMetrics = {
  validatedCalls: Counter
  unexpectedCalls: Counter
  unexpectedRpcErrors: Counter
}

type WalletMonState = {
  chainId: number
  highestUncheckedBlockNumber: number
}

export class WalletMonService extends BaseServiceV2<
  WalletMonOptions,
  WalletMonMetrics,
  WalletMonState
> {
  constructor(options?: Partial<WalletMonOptions & StandardOptions>) {
    super({
      version,
      name: 'wallet-mon',
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
        validatedCalls: {
          type: Gauge,
          desc: 'Transactions from the account checked',
          labels: ['wallet', 'target', 'nickname'],
        },
        unexpectedCalls: {
          type: Counter,
          desc: 'Number of unexpected wallets',
          labels: ['wallet', 'target', 'nickname', 'transactionHash'],
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
    const accounts = network.accounts

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
      for (const account of accounts) {
        if (compareAddrs(account.wallet, transaction.from)) {
          if (compareAddrs(account.target, transaction.to)) {
            this.metrics.validatedCalls.inc({
              nickname: account.label,
              wallet: account.address,
              target: account.target,
            })
            this.logger.info('validated call', {
              nickname: account.label,
              wallet: account.address,
              target: account.target,
            })
          } else {
            this.metrics.unexpectedCalls.inc({
              nickname: account.label,
              wallet: account.address,
              target: transaction.to,
              transactionHash: transaction.hash,
            })
            this.logger.error('Unexpected call detected', {
              nickname: account.label,
              address: account.address,
              target: transaction.to,
              transactionHash: transaction.hash,
            })
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
  const service = new WalletMonService()
  service.run()
}
