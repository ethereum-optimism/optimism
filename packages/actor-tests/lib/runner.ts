import * as path from 'path'

import { Command } from 'commander'

import { defaultRuntime } from './convenience'
import { RunOpts } from './actor'
import { serveMetrics } from './metrics'
import pkg from '../package.json'

const program = new Command()
program.version(pkg.version)
program.name('actor-tests')

program
  .requiredOption('-f, --file <file>', 'test file to run')
  .option('-r, --runs <n>', 'number of runs. cannot be use with -t/--time')
  .option(
    '-t, --time <ms>',
    'how long to run in milliseconds. cannot be used with -r/--runs'
  )
  .option('-c, --concurrency <n>', 'number of concurrent workers to spawn', '1')
  .option('--think-time <n>', 'how long to wait between each run', '0')
  .option(
    '-s, --serve [port]',
    'Serve metrics with optional port number',
    '8545'
  )

program.parse(process.argv)

const options = program.opts()
const testFile = options.file
const runsNum = Number(options.runs)
const timeNum = Number(options.time)
const concNum = Number(options.concurrency)
const thinkNum = Number(options.thinkTime)
const shouldServeMetrics = options.serve !== undefined
const metricsPort = options.serve || 8545

if (isNaN(runsNum) && isNaN(timeNum)) {
  console.error('Must define either a number of runs or how long to run.')
  process.exit(1)
}

if (isNaN(concNum) || concNum <= 0) {
  console.error('Invalid concurrency value.')
  process.exit(1)
}

if (isNaN(thinkNum) || thinkNum < 0) {
  console.error('Invalid think time value.')
  process.exit(1)
}

try {
  require(path.resolve(path.join(process.cwd(), testFile)))
} catch (e) {
  console.error(`Invalid test file ${testFile}:`)
  console.error(e)
  process.exit(1)
}

const opts: Partial<RunOpts> = {
  runFor: timeNum,
  concurrency: concNum,
  thinkTime: thinkNum,
  runs: runsNum,
}

if (shouldServeMetrics) {
  process.stderr.write(`Serving metrics on http://0.0.0.0:${metricsPort}.\n`)
  serveMetrics(metricsPort)
}

defaultRuntime
  .run(opts)
  .then(() => {
    process.stderr.write('Run complete.\n')
    process.exit(0)
  })
  .catch((err) => {
    console.error('Error:')
    console.error(err)
    process.exit(1)
  })
