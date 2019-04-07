import { DebugLogger } from './debug-logger'
import { Process } from './process'

export class BaseApp {
  private logger = new DebugLogger('app')
  private processes: Record<string, Process<any>> = {}

  public register(name: string, process: Process<any>): void {
    if (name in this.processes) {
      throw new Error(`process already registered: ${name}`)
    }

    this.processes[name] = process
  }

  public async start(): Promise<void> {
    await this.execute(async (name: string, process: Process<any>) => {
      this.logger.log(`starting process: ${name}`)
      await process.start()
      this.logger.log(`started process: ${name}`)
    })
  }

  public async stop(): Promise<void> {
    await this.execute(async (name: string, process: Process<any>) => {
      this.logger.log(`stopping process: ${name}`)
      await process.stop()
      this.logger.log(`stopped process: ${name}`)
    })
  }

  private async execute(
    fn: (name: string, process: Process<any>) => Promise<void>
  ): Promise<void> {
    await Promise.all(
      Object.keys(this.processes).map((name) => {
        return new Promise<void>(async (resolve, reject) => {
          try {
            await fn(name, this.processes[name])
          } catch (err) {
            reject(err)
            return
          }
          resolve()
        })
      })
    )
  }
}
