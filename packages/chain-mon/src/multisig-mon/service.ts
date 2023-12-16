import { exec } from 'child_process'

import {
  BaseServiceV2,
  StandardOptions,
  Gauge,
  Counter,
  validators,
} from '@eth-optimism/common-ts'
import { Provider } from '@ethersproject/abstract-provider'
import { ethers } from 'ethers'

import Safe from '../abi/IGnosisSafe.0.8.19.json'
import { version } from '../../package.json'

type MultisigMonOptions = {
  rpc: Provider
  accounts: string
  onePassServiceToken: string
}

type MultisigMonMetrics = {
  safeNonce: Gauge
  latestPreSignedPauseNonce: Gauge
  unexpectedRpcErrors: Counter
}

type MultisigMonState = {
  accounts: Array<{ address: string; nickname: string; vault: string }>
}

export class MultisigMonService extends BaseServiceV2<
  MultisigMonOptions,
  MultisigMonMetrics,
  MultisigMonState
> {
  constructor(options?: Partial<MultisigMonOptions & StandardOptions>) {
    super({
      version,
      name: 'multisig-mon',
      loop: true,
      options: {
        loopIntervalMs: 60_000,
        ...options,
      },
      optionsSpec: {
        rpc: {
          validator: validators.provider,
          desc: 'Provider for network to monitor balances on',
        },
        accounts: {
          validator: validators.str,
          desc: 'JSON array of [{ address, nickname, vault }] to monitor balances and nonces of',
          public: true,
        },
        onePassServiceToken: {
          validator: validators.str,
          desc: '1Password Service Token',
        },
      },
      metricsSpec: {
        safeNonce: {
          type: Gauge,
          desc: 'Safe nonce',
          labels: ['address', 'nickname'],
        },
        latestPreSignedPauseNonce: {
          type: Gauge,
          desc: 'Latest pre-signed pause nonce',
          labels: ['address', 'nickname'],
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
    this.state.accounts = JSON.parse(this.options.accounts)
  }

  protected async main(): Promise<void> {
    for (const account of this.state.accounts) {
      // get the nonce 1pass
      if (this.options.onePassServiceToken) {
        await this.getOnePassNonce(account)
      }

      // get the nonce from deployed safe
      await this.getSafeNonce(account)
    }
  }

  private async getSafeNonce(account: {
    address: string
    nickname: string
    vault: string
  }) {
    try {
      const safeContract = new ethers.Contract(
        account.address,
        Safe.abi,
        this.options.rpc
      )
      const safeNonce = await safeContract.nonce()
      this.logger.info(`got nonce`, {
        address: account.address,
        nickname: account.nickname,
        nonce: safeNonce.toString(),
      })

      this.metrics.safeNonce.set(
        { address: account.address, nickname: account.nickname },
        parseInt(safeNonce.toString(), 10)
      )
    } catch (err) {
      this.logger.error(`got unexpected RPC error`, {
        section: 'safeNonce',
        name: 'getSafeNonce',
        err,
      })
      this.metrics.unexpectedRpcErrors.inc({
        section: 'safeNonce',
        name: 'getSafeNonce',
      })
    }
  }

  private async getOnePassNonce(account: {
    address: string
    nickname: string
    vault: string
  }) {
    try {
      exec(
        `OP_SERVICE_ACCOUNT_TOKEN=${this.options.onePassServiceToken} op item list --format json --vault="${account.vault}"`,
        (error, stdout, stderr) => {
          if (error) {
            this.logger.error(`got unexpected error from onepass: ${error}`, {
              section: 'onePassNonce',
              name: 'getOnePassNonce',
            })
            return
          }
          if (stderr) {
            this.logger.error(`got unexpected error from onepass`, {
              section: 'onePassNonce',
              name: 'getOnePassNonce',
              stderr,
            })
            return
          }
          const items = JSON.parse(stdout)
          let latestNonce = -1
          this.logger.debug(`items in vault '${account.vault}':`)
          for (const item of items) {
            const title = item['title']
            this.logger.debug(`- ${title}`)
            if (title.startsWith('ready-') && title.endsWith('.json')) {
              const nonce = parseInt(title.substring(6, title.length - 5), 10)
              if (nonce > latestNonce) {
                latestNonce = nonce
              }
            }
          }
          this.metrics.latestPreSignedPauseNonce.set(
            { address: account.address, nickname: account.nickname },
            latestNonce
          )
          this.logger.debug(`latestNonce: ${latestNonce}`)
        }
      )
    } catch (err) {
      this.logger.error(`got unexpected error from onepass`, {
        section: 'onePassNonce',
        name: 'getOnePassNonce',
        err,
      })
      this.metrics.unexpectedRpcErrors.inc({
        section: 'onePassNonce',
        name: 'getOnePassNonce',
      })
    }
  }
}

if (require.main === module) {
  const service = new MultisigMonService()
  service.run()
}
