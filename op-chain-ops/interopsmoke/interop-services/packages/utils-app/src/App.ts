import { serve } from '@hono/node-server'
import type { OptionValues } from 'commander'
import { Command, Option } from 'commander'
import { Hono } from 'hono'
import type { Logger } from 'pino'
import { pino } from 'pino'
import { Registry } from 'prom-client'

import { requestLoggingMiddleware } from './middleware.js'

export interface Config {
  name: string
  version: string
  description?: string
}

export abstract class App {
  protected readonly program: Command
  protected readonly metricsRegistry: Registry

  // Must be set on run(), available to all hooks
  protected logger!: Logger
  protected options!: OptionValues
  protected adminApi?: Hono

  // A signal available to main() to resolve the
  // main promise during something like an interrupt.
  protected isShuttingDown = false

  private metricsServer: ReturnType<typeof serve> | null = null
  private adminServer: ReturnType<typeof serve> | null = null

  constructor(config: Config) {
    this.program = new Command()
    this.metricsRegistry = new Registry()

    // Setup program (remove shorthands for help & version)
    this.program
      .name(config.name)
      .helpOption('--help', 'show help')
      .version(config.version, '--version', 'show version')
      .description(config.description || '')

    // Common options
    this.program
      .addOption(
        new Option('--log-level <level>', 'Set log level')
          .default('debug')
          .choices(['debug', 'info', 'warn', 'error', 'silent'])
          .env('LOG_LEVEL'),
      )
      .addOption(
        new Option('--admin-enabled', 'Enable admin API server')
          .default(false)
          .env('ADMIN_ENABLED'),
      )
      .addOption(
        new Option('--admin-port <port>', 'Port for admin API')
          .default('9000')
          .env('ADMIN_PORT'),
      )
      .addOption(
        new Option('--metrics-enabled', 'Enable metrics collection')
          .default(false)
          .env('METRICS_ENABLED'),
      )
      .addOption(
        new Option('--metrics-port <port>', 'Port for metrics server')
          .default('7300')
          .env('METRICS_PORT'),
      )

    // Register additional options
    for (const option of this.additionalOptions()) {
      this.program.addOption(option)
    }
  }

  public async run(): Promise<void> {
    try {
      // Parse arguments
      this.program.parse()
      this.options = this.program.opts()

      // Setup Logger & Metrics Registry
      this.logger = pino({
        level: this.options.logLevel,
        transport: {
          target: 'pino-pretty',
          options: {
            colorize: true,
            translateTime: true,
            ignore: 'pid,hostname',
          },
        },
      })

      // Register handlers only after the logger is initialized. Constructing an
      // app to inspect its configuration must not leave process listeners behind.
      this.__setupSignalHandlers()

      this.logger.info('starting application...')

      // Start metrics server if enabled
      if (this.options.metricsEnabled) {
        await this.__startMetricsServer(this.options.metricsPort)
      }

      // Run pre-main hook
      await this.preMain()

      // Start admin api server if enabled
      if (this.options.adminEnabled) {
        await this.__startAdminApiServer(this.options.adminPort)
      }

      // Run main hook
      await this.main()

      // Run shutdown
      await this.__shutdown()

      this.logger.info('application completed successfully')
    } catch (error) {
      this.logger.error({ err: error }, 'application failed')
      process.exitCode = 1
    }
  }

  /** Optional options to add to the program */
  protected additionalOptions(): Option[] {
    return []
  }

  /** Optional hook to run before the main function, after parsing arguments */
  protected async preMain(): Promise<void> {}

  /** Optional hook to signal ready state for admin api */
  protected async ready(): Promise<boolean> {
    return true
  }

  /** Optional hook to run on interruption */
  protected async shutdown(): Promise<void> {}

  /** Optional hook to initialize the admin api */
  protected initializeAdminApi(): Hono {
    const adminApi = new Hono()
    adminApi.use(requestLoggingMiddleware(this.logger))

    adminApi.get('/healthz', (c) => c.text('OK'))
    adminApi.get('/readyz', async (c) => {
      const ready = await this.ready().catch((error) => {
        this.logger.error({ error }, 'failed ready check')
        return false
      })

      return c.body(ready ? 'OK' : 'NOT_READY')
    })
    return adminApi
  }

  private async __startMetricsServer(port: string): Promise<void> {
    const metricsApi = new Hono()
    metricsApi.use(requestLoggingMiddleware(this.logger))
    metricsApi.get('/metrics', async (c) => {
      c.header('Content-Type', this.metricsRegistry.contentType)
      return c.body(await this.metricsRegistry.metrics())
    })

    this.logger.info(`starting metrics server on port :${port}`)
    this.metricsServer = serve({
      fetch: metricsApi.fetch,
      port: Number(port),
    })
  }

  private async __startAdminApiServer(port: string): Promise<void> {
    this.adminApi = this.initializeAdminApi()

    this.logger.info(`starting admin api server on port :${port}`)
    this.adminServer = serve({
      fetch: this.adminApi.fetch,
      port: Number(port),
    })
  }

  private async __stopMetricsServer(): Promise<void> {
    if (this.metricsServer) {
      return new Promise((resolve, reject) => {
        this.logger.info('stopping metrics server...')
        this.metricsServer!.close((error) => {
          if (error) {
            this.logger.error({ error }, 'error closing metrics server')
            reject(error)
          } else {
            resolve()
          }
        })
      })
    }
  }

  private async __stopAdminServer(): Promise<void> {
    if (this.adminServer) {
      return new Promise((resolve, reject) => {
        this.logger.info('stopping admin server...')
        this.adminServer!.close((error) => {
          if (error) {
            this.logger.error({ error }, 'error closing admin api server')
            reject(error)
          } else {
            resolve()
          }
        })
      })
    }
  }

  private __setupSignalHandlers() {
    const signals = ['SIGINT', 'SIGTERM'] as const
    let shutdownCount = 0

    signals.forEach((signal) => {
      process.on(signal, async () => {
        this.logger.info(`received ${signal}, shutting down...`)

        shutdownCount++
        if (this.isShuttingDown && shutdownCount > 5) {
          this.logger.warn('forcing shutdown')
          process.exit(1)
        }

        try {
          await this.__shutdown()
        } catch (error) {
          this.logger.error({ error }, 'error during shutdown')
          process.exitCode = 1
        }
      })
    })
  }

  private async __shutdown(): Promise<void> {
    if (this.isShuttingDown) return
    this.isShuttingDown = true

    // shutdown processes
    await Promise.allSettled([
      this.shutdown(),
      this.__stopMetricsServer(),
      this.__stopAdminServer(),
    ])
  }

  protected abstract main(): Promise<void>
}
