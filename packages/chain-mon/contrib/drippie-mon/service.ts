import {
  BaseServiceV2,
  StandardOptions,
  Gauge,
  Counter,
  validators,
} from '@eth-optimism/common-ts'
import { Provider } from '@ethersproject/abstract-provider'
import { ethers } from 'ethers'
import * as DrippieArtifact from '@eth-optimism/contracts-bedrock/forge-artifacts/Drippie.sol/Drippie.json'

import { version } from '../../package.json'

type DrippieMonOptions = {
  rpc: Provider
  drippieAddress: string
}

type DrippieMonMetrics = {
  isExecutable: Gauge
  executedDripCount: Gauge
  unexpectedRpcErrors: Counter
}

type DrippieMonState = {
  drippie: ethers.Contract
}

export class DrippieMonService extends BaseServiceV2<
  DrippieMonOptions,
  DrippieMonMetrics,
  DrippieMonState
> {
  constructor(options?: Partial<DrippieMonOptions & StandardOptions>) {
    super({
      version,
      name: 'drippie-mon',
      loop: true,
      options: {
        loopIntervalMs: 60_000,
        ...options,
      },
      optionsSpec: {
        rpc: {
          validator: validators.provider,
          desc: 'Provider for network where Drippie is deployed',
        },
        drippieAddress: {
          validator: validators.str,
          desc: 'Address of Drippie contract',
          public: true,
        },
      },
      metricsSpec: {
        isExecutable: {
          type: Gauge,
          desc: 'Whether or not the drip is currently executable',
          labels: ['name'],
        },
        executedDripCount: {
          type: Gauge,
          desc: 'Number of times a drip has been executed',
          labels: ['name'],
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
    this.state.drippie = new ethers.Contract(
      this.options.drippieAddress,
      DrippieArtifact.abi,
      this.options.rpc
    )
  }

  protected async main(): Promise<void> {
    let dripCreatedEvents: ethers.Event[]
    try {
      dripCreatedEvents = await this.state.drippie.queryFilter(
        this.state.drippie.filters.DripCreated()
      )
    } catch (err) {
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

    // Not the most efficient thing in the world. Will end up making one request for every drip
    // created. We don't expect there to be many drips, so this is fine for now. We can also cache
    // and skip any archived drips to cut down on a few requests. Worth keeping an eye on this to
    // see if it's a bottleneck.
    for (const event of dripCreatedEvents) {
      const name = event.args.name

      let drip: any
      try {
        drip = await this.state.drippie.drips(name)
      } catch (err) {
        this.logger.info(`got unexpected RPC error`, {
          section: 'drips',
          name,
          err,
        })

        this.metrics.unexpectedRpcErrors.inc({
          section: 'drips',
          name,
        })

        continue
      }

      this.logger.info(`getting drip executable status`, {
        name,
        count: drip.count.toNumber(),
      })

      this.metrics.executedDripCount.set(
        {
          name,
        },
        drip.count.toNumber()
      )

      let executable: boolean
      try {
        // To avoid making unnecessary RPC requests, filter out any drips that we don't expect to
        // be executable right now. Only active drips (status = 2) and drips that are due to be
        // executed are expected to be executable (but might not be based on the dripcheck).
        if (
          drip.status === 2 &&
          drip.last.toNumber() + drip.config.interval.toNumber() <
            Date.now() / 1000
        ) {
          executable = await this.state.drippie.executable(name)
        } else {
          executable = false
        }
      } catch (err) {
        // All reverts include the string "Drippie:", so we can check for that.
        if (err.message.includes('Drippie:')) {
          // Not executable yet.
          executable = false
        } else {
          this.logger.info(`got unexpected RPC error`, {
            section: 'executable',
            name,
            err,
          })

          this.metrics.unexpectedRpcErrors.inc({
            section: 'executable',
            name,
          })

          continue
        }
      }

      this.logger.info(`got drip executable status`, {
        name,
        executable,
      })

      this.metrics.isExecutable.set(
        {
          name,
        },
        executable ? 1 : 0
      )
    }
  }
}

if (require.main === module) {
  const service = new DrippieMonService()
  service.run()
}
