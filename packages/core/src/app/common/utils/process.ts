import { EventEmitter } from 'events'
import uuid = require('uuid')

/**
 * Represents a basic process with start/stop functionality.
 */
export class Process<Subject> {
  public subject: Subject
  public readonly pid = uuid.v4()
  private ready = false
  private statusEmitter = new EventEmitter()
  private onStarted: Promise<void>
  private onStopped: Promise<void>

  constructor() {
    this.reset()
  }

  /**
   * @returns `true` if the process is ready, `false` otherwise.
   */
  public isReady(): boolean {
    return this.ready
  }

  /**
   * Starts the process.
   */
  public async start(): Promise<void> {
    if (this.ready) {
      return
    }

    this.reset()

    await this.onStart()
    this.ready = true
    this.statusEmitter.emit('started')
  }

  /**
   * Stops the process.
   */
  public async stop(): Promise<void> {
    if (!this.ready) {
      return
    }

    this.reset()

    await this.onStop()
    this.ready = false
    this.statusEmitter.emit('stopped')
  }

  /**
   * Waits until the process is started.
   */
  public async waitUntilStarted(): Promise<void> {
    if (!this.ready) {
      return this.onStarted
    }
  }

  /**
   * Waits until the process is stopped.
   */
  public async waitUntilStopped(): Promise<void> {
    if (this.ready) {
      return this.onStopped
    }
  }

  /**
   * Runs when the process is started.
   */
  protected async onStart(): Promise<void> {
    return
  }

  /**
   * Runs when the process is stopped.
   */
  protected async onStop(): Promise<void> {
    return
  }

  /**
   * Asserts that the process is ready and
   * throws otherwise.
   */
  protected assertReady(): void {
    if (!this.isReady()) {
      throw new Error('Process is not ready.')
    }
  }

  /**
   * Initializes lifecycle promises.
   */
  private reset(): void {
    this.statusEmitter.removeAllListeners()
    this.onStarted = new Promise<void>((resolve, _) => {
      this.statusEmitter.on('started', () => {
        resolve()
      })
    })
    this.onStopped = new Promise<void>((resolve, _) => {
      this.statusEmitter.on('stopped', () => {
        resolve()
      })
    })
  }
}
