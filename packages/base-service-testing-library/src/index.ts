import supertestRequest from 'supertest'
import { BaseServiceV2, MetricsV2 } from '@eth-optimism/common-ts'

interface Options {
  throwOnConsoleErrors?: boolean
  throwOnConsoleWarns?: boolean
}

export class WrappedBaseClass<
  TOptions,
  TMetrics extends MetricsV2,
  TServerState
> {
  constructor(
    public readonly service: BaseServiceV2<TOptions, TMetrics, TServerState>,
    public readonly options: Options = {}
  ) {
    if (options.throwOnConsoleErrors) {
      const oldError = (service as any).logger.error.bind(
        (service as any).logger
      )
      ;(service as any).logger.error = (...args) => {
        oldError(...args)
        throw new Error(
          'There was an error and base-service-testing-library throwOnConsoleError is set to true'
        )
      }
    }
    if (options.throwOnConsoleWarns) {
      const oldWarn = (service as any).logger.warn.bind((service as any).logger)
      ;(service as any).logger.warn = (...args) => {
        oldWarn(...args)
        throw new Error(
          'There was an warning and base-service-testing-library throwOnConsoleWarn is set to true'
        )
      }
    }
  }

  public readonly testApi = () => {
    if ((this.service as any).done) {
      throw new Error(
        'testApi called when server is not running.  Did you forget to  start server in beforeEach?'
      )
    }
    return supertestRequest((this.service as any).server)
  }

  public readonly run = this.service.run.bind(this.service)

  public readonly stop = this.service.stop.bind(this.service)

  get done() {
    return (this.service as any).done
  }

  get healthy() {
    return (this.service as any).healthy
  }

  get state() {
    return (this.service as any).state
  }

  public readonly setState = (newState: TServerState) => {
    ;(this.service as any).state = newState
  }

  // TODO add async methods to control the looping after that pr is in
}
