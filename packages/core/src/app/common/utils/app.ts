import { Process } from './process'

export class BaseApp {
  private processes: Record<string, Process<any>> = {}

  public register(name: string, process: Process<any>): void {
    if (name in this.processes) {
      throw new Error(`Process already registered: ${name}`)
    }

    this.processes[name] = process
  }

  public async start(): Promise<void> {
    await Promise.all(
      Object.values(this.processes).map((process) => {
        return process.start()
      })
    )
  }

  public async stop(): Promise<void> {
    await Promise.all(
      Object.values(this.processes).map((process) => {
        return process.stop()
      })
    )
  }
}
