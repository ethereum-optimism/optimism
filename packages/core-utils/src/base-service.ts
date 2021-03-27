/* Imports: Internal */
import { Logger } from './common/logger'

type OptionSettings<TOptions> = {
  [P in keyof TOptions]?: {
    default?: TOptions[P]
    validate?: (val: any) => boolean
  }
}

/**
 * Base for other "Service" objects. Handles your standard initialization process, can dynamically
 * start and stop.
 */
export class BaseService<TServiceOptions> {
  protected name: string
  protected optionSettings: OptionSettings<TServiceOptions>
  protected logger: Logger
  protected initialized: boolean = false
  protected running: boolean = false

  /**
   * @param options Options to pass to the service.
   */
  constructor(protected options: TServiceOptions) {}

  /**
   * Initializes the service.
   */
  public async init(): Promise<void> {
    if (this.initialized) {
      return
    }

    // Apparently I'm going crazy and just now finding out that class variables are undefined
    // within the constructor? Anyway, this means I have to do all of this initialization logic
    // during a separate init function or everything is undefined.
    if (this.logger === undefined) {
      // tslint:disable-next-line
      this.logger = new Logger({name: this.name})
    }

    this._mergeDefaultOptions()
    this._validateOptions()

    this.initialized = true

    try {
      this.logger.info('Service is initializing...')
      await this._init()
      this.logger.info('Service has initialized.')
    } catch (err) {
      this.initialized = false
      throw err
    }
  }

  /**
   * Starts the service.
   */
  public async start(): Promise<void> {
    if (this.running) {
      return
    }

    if (this.logger === undefined) {
      // tslint:disable-next-line
      this.logger = new Logger({name: this.name})
    }

    this.running = true

    this.logger.info('Service is starting...')
    await this.init()
    await this._start()
    this.logger.info('Service has started')
  }

  /**
   * Stops the service.
   */
  public async stop(): Promise<void> {
    if (!this.running) {
      return
    }

    this.logger.info('Service is stopping...')
    await this._stop()
    this.logger.info('Service has stopped')

    this.running = false
  }

  /**
   * Internal init function. Parent should implement.
   */
  protected async _init(): Promise<void> {
    return
  }

  /**
   * Internal start function. Parent should implement.
   */
  protected async _start(): Promise<void> {
    return
  }

  /**
   * Internal stop function. Parent should implement.
   */
  protected async _stop(): Promise<void> {
    return
  }

  /**
   * Combines user provided and default options. Honestly there's no point for this function to
   * live within this class and be all stateful, but I didn't realize that until after I wrote it.
   * So we're gonna have to deal with that for now. Whatever, it's an easy fix if anyone else
   * feels like tackling it.
   */
  private _mergeDefaultOptions(): void {
    if (this.optionSettings === undefined) {
      return
    }

    for (const optionName of Object.keys(this.optionSettings)) {
      const optionDefault = this.optionSettings[optionName].default
      if (optionDefault === undefined) {
        continue
      }

      if (
        this.options[optionName] !== undefined &&
        this.options[optionName] !== null
      ) {
        continue
      }

      // TODO: Maybe make a copy of this default instead of directly assigning?
      this.options[optionName] = optionDefault
    }
  }

  /**
   * Performs option validation against the option settings attached to this class. Another
   * function that really shouldn't be part of this class in particular. Good mini project though!
   */
  private _validateOptions(): void {
    if (this.optionSettings === undefined) {
      return
    }

    for (const optionName of Object.keys(this.optionSettings)) {
      const optionValidationFunction = this.optionSettings[optionName].validate
      if (optionValidationFunction === undefined) {
        continue
      }

      const optionValue = this.options[optionName]

      if (optionValidationFunction(optionValue) === false) {
        throw new Error(
          `Provided input for option "${optionName}" is invalid: ${optionValue}`
        )
      }
    }
  }
}
