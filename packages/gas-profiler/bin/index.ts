#!/usr/bin/env node

import * as fs from 'fs'
import { ArgumentParser, Const } from 'argparse'
import { GasProfiler } from '../src/gas-profiler'

const parser = new ArgumentParser({
  addHelp: true,
  description: 'Smart contract gas profiler',
})

parser.addArgument(['-c', '--contract-json'], {
  help: `Path to the contract's compiled JSON file.`,
  required: true,
  dest: 'contractJsonPath',
})

parser.addArgument(['-s', '--contract-source'], {
  help: `(Optional) Path to the contract's source file.`,
  dest: 'contractSourcePath',
})

parser.addArgument(['-m', '--method'], {
  help: `Contract method to call`,
  required: true,
})

parser.addArgument(['-p', '--params'], {
  help: `(Optional) Contract method parameters`,
  nargs: Const.ONE_OR_MORE,
  defaultValue: [],
})

parser.addArgument(['-t', '--trace'], {
  action: 'storeTrue',
  help: `(Optional) Generates a full gas trace. Source file must be specified.`,
  dest: 'trace',
})

const readJson = (filePath: string): any => {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'))
}

const main = async () => {
  const args = parser.parseArgs()
  const contract = readJson(args.contractJsonPath)

  const profiler = new GasProfiler()
  await profiler.init()

  if (args.trace) {
    const profile = await profiler.profile(contract, args.contractSourcePath, {
      method: args.method,
      params: args.params,
    })
    const pretty = profiler.prettify(profile.trace)
    console.log()
    console.log(pretty)
    console.log('Total gas used: ', profile.gasUsed)
  } else {
    const profile = await profiler.execute(contract, {
      method: args.method,
      params: args.params,
    })
    console.log('Total gas used: ', profile.gasUsed)
  }

  await profiler.kill()
}

main()
