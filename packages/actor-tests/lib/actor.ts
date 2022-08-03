import { performance } from 'perf_hooks'

import { Mutex } from 'async-mutex'
import { sleep } from '@eth-optimism/core-utils'

import {
  sanitizeForMetrics,
  benchDurationsSummary,
  successfulBenchRunsTotal,
  failedActorRunsTotal,
  successfulActorRunsTotal,
  failedBenchRunsTotal,
} from './metrics'
import { ActorLogger, WorkerLogger } from './logger'

// eslint-disable-next-line @typescript-eslint/no-empty-function
const asyncNoop = async () => {}

export type AsyncCB = () => Promise<void>

export interface Bencher {
  bench: (name: string, cb: () => Promise<any>) => Promise<any>
}

export type RunCB<C> = (
  b: Bencher,
  ctx: C,
  logger: WorkerLogger
) => Promise<void>

export interface RunOpts {
  runs: number | null
  runFor: number | null
  concurrency: number
  thinkTime: number
}

class Latch {
  private n: number

  private p: Promise<void>

  private resolver: () => void

  constructor(n: number) {
    this.n = n
    this.p = new Promise((resolve) => {
      this.resolver = resolve
    })
  }

  countDown() {
    this.n--
    if (this.n === 0) {
      this.resolver()
    }
  }

  wait() {
    return this.p
  }
}

export class Runner {
  private readonly workerId: number

  private readonly actor: Actor

  private readonly mtx: Mutex

  private readonly readyLatch: Latch

  private readonly stepper: Bencher

  private readonly logger: WorkerLogger

  constructor(workerId: number, actor: Actor, mtx: Mutex, readyLatch: Latch) {
    this.workerId = workerId
    this.actor = actor
    this.mtx = mtx
    this.readyLatch = readyLatch
    this.stepper = {
      bench: this.bench,
    }
    this.logger = new WorkerLogger(this.actor.name, workerId)
  }

  bench = async (name: string, cb: () => Promise<void>) => {
    const metricLabels = {
      actor_name: sanitizeForMetrics(this.actor.name),
      bench_name: sanitizeForMetrics(name),
    }
    const start = performance.now()
    let res
    try {
      res = await cb()
    } catch (e) {
      failedBenchRunsTotal.inc({
        ...metricLabels,
        worker_id: this.workerId,
      })
      throw e
    }
    benchDurationsSummary.observe(metricLabels, performance.now() - start)
    successfulBenchRunsTotal.inc({
      ...metricLabels,
      worker_id: this.workerId,
    })
    return res
  }

  async run(opts: RunOpts) {
    const actor = this.actor

    this.logger.log('Setting up.')
    let ctx
    try {
      ctx = await this.mtx.runExclusive(this.actor.setupRun)
    } finally {
      this.readyLatch.countDown()
    }
    this.logger.log('Waiting for other workers to finish setup.')
    await this.readyLatch.wait()

    this.logger.log('Executing.')
    const benchStart = performance.now()
    let lastDurPrint = benchStart
    let i = 0
    const metricLabels = {
      actor_name: sanitizeForMetrics(this.actor.name),
    }

    while (true) {
      const now = performance.now()

      if (
        (opts.runs && i === opts.runs) ||
        (opts.runFor && now - benchStart >= opts.runFor)
      ) {
        this.logger.log(`Worker exited.`)
        break
      }

      try {
        await this.actor.run(this.stepper, ctx, this.logger)
      } catch (e) {
        console.error('Error in actor run:')
        console.error(`Benchmark name: ${actor.name}`)
        console.error(`Worker ID:      ${this.workerId}`)
        console.error(`Run index:      ${i}`)
        console.error('Stack trace:')
        console.error(e)
        failedActorRunsTotal.inc(metricLabels)
        await sleep(1000)
        continue
      }

      successfulActorRunsTotal.inc(metricLabels)

      i++

      if (
        (opts.runs && (i % 10 === 0 || i === opts.runs)) ||
        now - lastDurPrint > 10000
      ) {
        this.logger.log(`Completed run ${i} of ${opts.runs}.`)
      }

      if (opts.runFor && now - lastDurPrint > 10000) {
        const runningFor = Math.floor(now - benchStart)
        this.logger.log(`Running for ${runningFor} of ${opts.runFor} ms.`)
        lastDurPrint = now
      }

      if (opts.thinkTime > 0) {
        await sleep(opts.thinkTime)
      }
    }

    await this.mtx.runExclusive(() => this.actor.tearDownRun(ctx))
  }
}

export class Actor {
  public readonly name: string

  private _setupEnv: AsyncCB = asyncNoop

  private _tearDownEnv: AsyncCB = asyncNoop

  private _setupRun: <C>() => Promise<C> = asyncNoop as any

  private _tearDownRun: <C>(ctx: C) => Promise<void> = asyncNoop as any

  // eslint-disable-next-line @typescript-eslint/no-empty-function
  private _run: <C>(b: Bencher, ctx: C, logger: WorkerLogger) => Promise<void> =
    asyncNoop

  private logger: ActorLogger

  constructor(name: string) {
    this.name = name
    this.logger = new ActorLogger(this.name)
  }

  get setupEnv(): AsyncCB {
    return this._setupEnv
  }

  set setupEnv(value: AsyncCB) {
    this._setupEnv = value
  }

  get tearDownEnv(): AsyncCB {
    return this._tearDownEnv
  }

  set tearDownEnv(value: AsyncCB) {
    this._tearDownEnv = value
  }

  get setupRun(): <C>() => Promise<C> {
    return this._setupRun
  }

  set setupRun(value: () => Promise<any>) {
    this._setupRun = value
  }

  get tearDownRun(): <C>(ctx: C) => Promise<void> {
    return this._tearDownRun
  }

  set tearDownRun(value: (ctx: any) => Promise<any>) {
    this._tearDownRun = value
  }

  get run(): RunCB<any> {
    return this._run
  }

  set run(cb: RunCB<any>) {
    this._run = cb
  }

  async exec(opts: RunOpts) {
    this.logger.log('Setting up.')

    try {
      await this.setupEnv()
    } catch (e) {
      console.error(`Error in setupEnv hook for actor "${this.name}":`)
      console.error(e)
      return
    }

    this.logger.log('Starting.')

    const parallelRuns = []
    const mtx = new Mutex()
    const latch = new Latch(opts.concurrency)
    for (let i = 0; i < opts.concurrency; i++) {
      const runner = new Runner(i, this, mtx, latch)
      parallelRuns.push(runner.run(opts))
    }
    await Promise.all(parallelRuns)

    this.logger.log('Tearing down.')

    try {
      await this.tearDownEnv()
    } catch (e) {
      console.error(`Error in after hook for benchmark "${this.name}":`)
      console.error(e)
      return
    }

    this.logger.log('Teardown complete.')
  }
}

export class Runtime {
  private actors: Actor[] = []

  addActor(actor: Actor) {
    this.actors.push(actor)
  }

  async run(opts: Partial<RunOpts>) {
    opts = {
      runs: 1,
      concurrency: 1,
      runFor: null,
      thinkTime: 0,
      ...(opts || {}),
    }
    if (opts.runFor) {
      opts.runs = null
    }

    for (const actor of this.actors) {
      await actor.exec(opts as RunOpts)
    }
  }
}
