/* Imports: Internal */
import { Logger } from '../utils/logger'

export class BaseService<TServiceOptions> {
  protected name: string
  protected logger: Logger
  protected initialized: boolean = false
  protected running: boolean = false

  constructor(protected options: TServiceOptions) {}

  public async init(): Promise<void> {
    if (this.initialized) {
      return
    }

    this.initialized = true
    this.logger = new Logger(this.name)

    try {
      await this._init()
    } catch (err) {
      this.initialized = false
      throw err
    }
  }

  public async start(): Promise<void> {
    await this.init()
    if (this.running) {
      return
    }

    this.running = true
    this._start()
  }

  public async stop(): Promise<void> {
    if (!this.running) {
      return
    }

    await this._stop()
    this.running = false
  }

  protected async _init(): Promise<void> {}

  protected async _start(): Promise<void> {}

  protected async _stop(): Promise<void> {}
}
