import { sanitizeForMetrics } from './metrics'

abstract class Logger {
  log(msg: string) {
    const date = new Date()
    process.stderr.write(`[${date.toISOString()}] ${msg}\n`)
  }
}

export class ActorLogger extends Logger {
  private readonly name: string

  constructor(name: string) {
    super()
    this.name = name
  }

  log(msg: string) {
    super.log(`[actor:${sanitizeForMetrics(this.name)}] ${msg}`)
  }
}

export class WorkerLogger extends Logger {
  private readonly name: string

  private readonly workerId: number

  constructor(name: string, workerId: number) {
    super()
    this.name = name
    this.workerId = workerId
  }

  log(msg: string) {
    super.log(
      `[bench:${sanitizeForMetrics(this.name)}] [wid:${this.workerId}] ${msg}`
    )
  }
}
