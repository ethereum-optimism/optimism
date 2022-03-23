import { Server } from 'net'

import Config from 'bcfg'
import * as dotenv from 'dotenv'
import { Command, Option } from 'commander'
import { ValidatorSpec, Spec, cleanEnv } from 'envalid'
import { sleep } from '@eth-optimism/core-utils'
import snakeCase from 'lodash/snakeCase'
import express from 'express'
import prometheus, { Registry } from 'prom-client'

import { Logger } from '../common/logger'
import { Metric } from './metrics'

export type Options = {
  [key: string]: any
}

export type OptionsSpec<TOptions extends Options> = {
  [P in keyof Required<TOptions>]: {
    validator: (spec?: Spec<TOptions[P]>) => ValidatorSpec<TOptions[P]>
    desc: string
    default?: TOptions[P]
  }
}

export type MetricsV2 = {
  [key: string]: Metric
}

export type MetricsSpec<TMetrics extends MetricsV2> = {
  [P in keyof Required<TMetrics>]: {
    type: new (configuration: any) => TMetrics[P]
    desc: string
    labels?: string[]
  }
}

/**
 * BaseServiceV2 is an advanced but simple base class for long-running TypeScript services.
 */
export abstract class BaseServiceV2<
  TOptions extends Options,
  TMetrics extends MetricsV2,
  TServiceState
> {
  /**
   * Whether or not the service will loop.
   */
  protected loop: boolean

  /**
   * Waiting period in ms between loops, if the service will loop.
   */
  protected loopIntervalMs: number

  /**
   * Whether or not the service is currently running.
   */
  protected running: boolean

  /**
   * Whether or not the service has run to completion.
   */
  protected done: boolean

  /**
   * Logger class for this service.
   */
  protected logger: Logger

  /**
   * Service state, persisted between loops.
   */
  protected state: TServiceState

  /**
   * Service options.
   */
  protected readonly options: TOptions

  /**
   * Metrics.
   */
  protected readonly metrics: TMetrics

  /**
   * Registry for prometheus metrics.
   */
  protected readonly metricsRegistry: Registry

  /**
   * Metrics server.
   */
  protected metricsServer: Server

  /**
   * Port for the metrics server.
   */
  protected readonly metricsServerPort: number

  /**
   * Hostname for the metrics server.
   */
  protected readonly metricsServerHostname: string

  /**
   * @param params Options for the construction of the service.
   * @param params.name Name for the service. This name will determine the prefix used for logging,
   * metrics, and loading environment variables.
   * @param params.optionsSpec Settings for input options. You must specify at least a
   * description for each option.
   * @param params.metricsSpec Settings that define which metrics are collected. All metrics that
   * you plan to collect must be defined within this object.
   * @param params.options Options to pass to the service.
   * @param params.loops Whether or not the service should loop. Defaults to true.
   * @param params.loopIntervalMs Loop interval in milliseconds. Defaults to zero.
   * @param params.metricsServerPort Port for the metrics server. Defaults to 7300.
   * @param params.metricsServerHostname Hostname for the metrics server. Defaults to 0.0.0.0.
   */
  constructor(params: {
    name: string
    optionsSpec: OptionsSpec<TOptions>
    metricsSpec: MetricsSpec<TMetrics>
    options?: Partial<TOptions>
    loop?: boolean
    loopIntervalMs?: number
    metricsServerPort?: number
    metricsServerHostname?: string
  }) {
    this.loop = params.loop !== undefined ? params.loop : true
    this.loopIntervalMs =
      params.loopIntervalMs !== undefined ? params.loopIntervalMs : 0
    this.state = {} as TServiceState

    /**
     * Special snake_case function which accounts for the common strings "L1" and "L2" which would
     * normally be split into "L_1" and "L_2" by the snake_case function.
     *
     * @param str String to convert to snake_case.
     * @returns snake_case string.
     */
    const opSnakeCase = (str: string) => {
      const reg = /l_1|l_2/g
      const repl = str.includes('l1') ? 'l1' : 'l2'
      return snakeCase(str).replace(reg, repl)
    }

    // Use commander as a way to communicate info about the service. We don't actually *use*
    // commander for anything besides the ability to run `ts-node ./service.ts --help`.
    const program = new Command()
    for (const [optionName, optionSpec] of Object.entries(params.optionsSpec)) {
      program.addOption(
        new Option(`--${optionName.toLowerCase()}`, `${optionSpec.desc}`).env(
          `${opSnakeCase(
            params.name.replace(/-/g, '_')
          ).toUpperCase()}__${opSnakeCase(optionName).toUpperCase()}`
        )
      )
    }

    const longestMetricNameLength = Object.keys(params.metricsSpec).reduce(
      (acc, key) => {
        const nameLength = snakeCase(key).length
        if (nameLength > acc) {
          return nameLength
        } else {
          return acc
        }
      },
      0
    )

    program.addHelpText(
      'after',
      `\nMetrics:\n${Object.entries(params.metricsSpec)
        .map(([metricName, metricSpec]) => {
          const parsedName = opSnakeCase(metricName)
          return `  ${parsedName}${' '.repeat(
            longestMetricNameLength - parsedName.length + 2
          )}${metricSpec.desc} (type: ${metricSpec.type.name})`
        })
        .join('\n')}
      `
    )

    // Load all configuration values from the environment and argv.
    program.parse()
    dotenv.config()
    const config = new Config(params.name)
    config.load({
      env: true,
      argv: true,
    })

    // Clean configuration values using the options spec.
    // Since BCFG turns everything into lower case, we're required to turn all of the input option
    // names into lower case for the validation step. We'll turn the names back into their original
    // names when we're done.
    const cleaned = cleanEnv<TOptions>(
      { ...config.env, ...config.args },
      Object.entries(params.optionsSpec || {}).reduce((acc, [key, val]) => {
        acc[key.toLowerCase()] = val.validator({
          desc: val.desc,
          default: val.default,
        })
        return acc
      }, {}) as any,
      Object.entries(params.options || {}).reduce((acc, [key, val]) => {
        acc[key.toLowerCase()] = val
        return acc
      }, {}) as any
    )

    // Turn the lowercased option names back into camelCase.
    this.options = Object.keys(params.optionsSpec || {}).reduce((acc, key) => {
      acc[key] = cleaned[key.toLowerCase()]
      return acc
    }, {}) as TOptions

    // Create the metrics objects.
    this.metrics = Object.keys(params.metricsSpec || {}).reduce((acc, key) => {
      const spec = params.metricsSpec[key]
      acc[key] = new spec.type({
        name: `${opSnakeCase(params.name)}_${opSnakeCase(key)}`,
        help: spec.desc,
        labelNames: spec.labels || [],
      })
      return acc
    }, {}) as TMetrics

    // Create the metrics server.
    this.metricsRegistry = prometheus.register
    this.metricsServerPort = params.metricsServerPort || 7300
    this.metricsServerHostname = params.metricsServerHostname || '0.0.0.0'

    this.logger = new Logger({ name: params.name })

    // Gracefully handle stop signals.
    const stop = async (signal: string) => {
      this.logger.info(`stopping service with signal`, { signal })
      await this.stop()
      process.exit(0)
    }
    process.on('SIGTERM', stop)
    process.on('SIGINT', stop)
  }

  /**
   * Runs the main function. If this service is set up to loop, will repeatedly loop around the
   * main function. Will also catch unhandled errors.
   */
  public async run(): Promise<void> {
    this.done = false

    // Start the metrics server if not yet running.
    if (!this.metricsServer) {
      this.logger.info('starting metrics server')

      await new Promise((resolve) => {
        const app = express()

        app.get('/metrics', async (_, res) => {
          res.status(200).send(await this.metricsRegistry.metrics())
        })

        this.metricsServer = app.listen(
          this.metricsServerPort,
          this.metricsServerHostname,
          () => {
            resolve(null)
          }
        )
      })

      this.logger.info(`metrics started`, {
        port: this.metricsServerPort,
        hostname: this.metricsServerHostname,
        route: '/metrics',
      })
    }

    if (this.init) {
      this.logger.info('initializing service')
      await this.init()
      this.logger.info('service initialized')
    }

    if (this.loop) {
      this.logger.info('starting main loop')
      this.running = true
      while (this.running) {
        try {
          await this.main()
        } catch (err) {
          this.logger.error('caught an unhandled exception', {
            message: err.message,
            stack: err.stack,
            code: err.code,
          })
        }

        // Sleep between loops if we're still running (service not stopped).
        if (this.running) {
          await sleep(this.loopIntervalMs)
        }
      }
    } else {
      this.logger.info('running main function')
      await this.main()
    }

    this.done = true
  }

  /**
   * Tries to gracefully stop the service. Service will continue running until the current loop
   * iteration is finished and will then stop looping.
   */
  public async stop(): Promise<void> {
    this.running = false

    // Wait until the main loop has finished.
    this.logger.info('stopping service, waiting for main loop to finish')
    while (!this.done) {
      await sleep(1000)
    }

    // Shut down the metrics server if it's running.
    if (this.metricsServer) {
      this.logger.info('stopping metrics server')
      await new Promise((resolve) => {
        this.metricsServer.close(() => {
          resolve(null)
        })
      })
      this.logger.info('metrics server stopped')
      this.metricsServer = undefined
    }
  }

  /**
   * Initialization function. Runs once before the main function.
   */
  protected init?(): Promise<void>

  /**
   * Main function. Runs repeatedly when run() is called.
   */
  protected abstract main(): Promise<void>
}
