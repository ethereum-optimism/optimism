export interface Runnable {
  readonly started: boolean
  start(): Promise<void>
  stop(): Promise<void>
}

export class BaseRunnable {
  private _started = false

  get started(): boolean {
    return this._started
  }

  protected onStart(): Promise<void> {
    return
  }

  protected onStop(): Promise<void> {
    return
  }

  public async start(): Promise<void> {
    if (this._started) {
      return
    }

    await this.onStart()
    this._started = true
  }

  public async stop(): Promise<void> {
    if (!this._started) {
      return
    }

    await this.onStop()
    this._started = false
  }
}
