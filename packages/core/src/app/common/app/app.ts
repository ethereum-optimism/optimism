import { DebugLogger } from '../utils'
import { Process } from './process'

/**
 * Basic application framework. Makes it easy to
 * start/stop different processes.
 */
export class BaseApp {
  private logger = new DebugLogger('app')
  private processes: Record<string, Process<any>> = {}

  /**
   * Registers a process to the app.
   * @param name Name of the process.
   * @param process Process to register.
   */
  public register(name: string, process: Process<any>): void {
    if (name in this.processes) {
      throw new Error(`process already registered: ${name}`)
    }

    this.processes[name] = process
  }

  /**
   * Queries a specific process by name.
   * @param name Name of the process.
   * @returns the process registered with that name.
   */
  public query(name: string): Process<any> {
    if (!(name in this.processes)) {
      throw new Error(`process does not exist: ${name}`)
    }

    return this.processes[name]
  }

  /**
   * Starts all processes.
   */
  public async start(): Promise<void> {
    await this.execute(async (name: string, process: Process<any>) => {
      this.logger.log(`starting process: ${name}`)
      await process.start()
      this.logger.log(`started process: ${name}`)
    })
  }

  /**
   * Stops all processes.
   */
  public async stop(): Promise<void> {
    await this.execute(async (name: string, process: Process<any>) => {
      this.logger.log(`stopping process: ${name}`)
      await process.stop()
      this.logger.log(`stopped process: ${name}`)
    })
  }

  /**
   * Executes some function in parallel for all processes.
   * @param fn Function to execute.
   */
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
