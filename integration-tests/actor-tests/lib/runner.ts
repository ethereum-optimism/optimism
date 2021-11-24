import * as path from 'path'
import { defaultRuntime } from './convenience'
import { metricsRegistry } from './metrics'
import { RunOpts } from './actor'

const args = process.argv.slice(2)

if (args.length < 3) {
  console.error(
    'Usage: runner.ts <test-file> <concurrency> <run-for> [think-time?]'
  )
  process.exit(1)
}

const testFile = args[0]

const concurrency = Number(args[1])
if (isNaN(concurrency)) {
  console.error(`Invalid concurrency value "${concurrency}".`)
  process.exit(1)
}

const runFor = Number(args[2])
if (isNaN(runFor)) {
  console.error(`Invalid run-for value "${args[0]}".`)
  process.exit(1)
}

let thinkTime = 0
if (args.length >= 4) {
  thinkTime = Number(args[3])
  if (isNaN(thinkTime)) {
    console.error(`Invalid think-time value "${thinkTime}".`)
    process.exit(1)
  }
}

try {
  require(path.resolve(path.join(process.cwd(), testFile)))
} catch (e) {
  console.error(`Invalid test file ${testFile}:`)
  console.error(e)
  process.exit(1)
}

const opts: Partial<RunOpts> = {
  runFor,
  concurrency,
  thinkTime,
}

defaultRuntime
  .run(opts)
  .then(() => metricsRegistry.metrics())
  .then((metrics) => {
    console.log('')
    console.log(metrics)
  })
  .catch((err) => {
    console.error('Error running:')
    console.error(err)
    process.exit(1)
  })
