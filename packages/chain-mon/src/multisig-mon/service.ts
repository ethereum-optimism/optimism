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
}

type MultisigMonMetrics = {
  safeNonce: Gauge
  unexpectedRpcErrors: Counter
}

type MultisigMonState = {
  accounts: Array<{ address: string; nickname: string }>
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
          desc: 'JSON array of [{ address, nickname, safe }] to monitor balances and nonces of',
          public: true,
        },
      },
      metricsSpec: {
        safeNonce: {
          type: Gauge,
          desc: 'Safe nonce',
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
  }
}

if (require.main === module) {
  const service = new MultisigMonService()
  service.run()
}
