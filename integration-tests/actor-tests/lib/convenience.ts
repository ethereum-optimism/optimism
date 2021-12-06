import { AsyncCB, Actor, RunCB, Runtime } from './actor'

export const defaultRuntime = new Runtime()

let currBenchmark: Actor | null = null

export const actor = (name: string, cb: () => void) => {
  if (currBenchmark) {
    throw new Error('Cannot call actor within actor.')
  }

  currBenchmark = new Actor(name)
  cb()
  defaultRuntime.addActor(currBenchmark)
  currBenchmark = null
}

export const setupActor = (cb: AsyncCB) => {
  if (!currBenchmark) {
    throw new Error('Cannot call setupEnv outside of actor.')
  }
  currBenchmark.setupEnv = cb
}

export const setupRun = (cb: () => Promise<any>) => {
  if (!currBenchmark) {
    throw new Error('Cannot call setupRun outside of actor.')
  }
  currBenchmark.setupRun = cb
}

export const tearDownRun = (cb: AsyncCB) => {
  if (!currBenchmark) {
    throw new Error('Cannot call tearDownRun outside of actor.')
  }
  currBenchmark.tearDownRun = cb
}

export const run = (cb: RunCB<any>) => {
  if (!currBenchmark) {
    throw new Error('Cannot call run outside of actor.')
  }
  currBenchmark.run = cb
}

export const retryOnGasTooLow = async (cb: () => Promise<any>) => {
  while (true) {
    try {
      return await cb()
    } catch (e) {
      if (e.toString().includes('gas price too low')) {
        await new Promise((resolve) => setTimeout(resolve, 100))
        continue
      }

      throw e
    }
  }
}
