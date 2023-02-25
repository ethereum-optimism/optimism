#!/usr/bin/env node
import { cac } from 'cac'
import type { Address } from '@wagmi/core'

import { readOptionsValidators, ReadOptions } from './commands/read'
import * as logger from './lib/logger'
// @ts-ignore it's mad about me importing something not in tsconfig.includes
import packageJson from '../package.json'
import { WriteOptions, writeOptionsValidators } from './commands/write'

const cli = cac('atst')

cli
  .command('read', 'read an attestation')
  .option('--creator <string>', readOptionsValidators.creator.description!)
  .option('--about <string>', readOptionsValidators.about.description!)
  .option('--key <string>', readOptionsValidators.key.description!)
  .option('--data-type [string]', readOptionsValidators.dataType.description!, {
    default: readOptionsValidators.dataType.parse(undefined),
  })
  .option('--rpc-url [url]', readOptionsValidators.rpcUrl.description!, {
    default: readOptionsValidators.rpcUrl.parse(undefined),
  })
  .option('--contract [address]', readOptionsValidators.contract.description!, {
    default: readOptionsValidators.contract.parse(undefined),
  })
  .example(
    () =>
      `atst read --key "optimist.base-uri" --about 0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5 --creator 0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3`
  )
  .action(async (options: ReadOptions) => {
    const { read } = await import('./commands/read')

    // TODO use the native api to do this instead of parsing the raw args
    // by default options parses addresses as numbers without precision
    // we should use the args parsing library to do this directly
    // but for now I didn't bother to figure out how to do that
    const { rawArgs } = cli
    const about = rawArgs[rawArgs.indexOf('--about') + 1] as Address
    const creator = rawArgs[rawArgs.indexOf('--creator') + 1] as Address
    const contract = rawArgs.includes('--contract')
      ? (rawArgs[rawArgs.indexOf('--contract') + 1] as Address)
      : options.contract

    await read({ ...options, about, creator, contract })
  })

cli
  .command('write', 'write an attestation')
  .option(
    '--private-key <string>',
    writeOptionsValidators.privateKey.description!
  )
  .option('--data-type [string]', readOptionsValidators.dataType.description!, {
    default: writeOptionsValidators.dataType.parse(undefined),
  })
  .option('--about <string>', writeOptionsValidators.about.description!)
  .option('--key <string>', writeOptionsValidators.key.description!)
  .option('--value <string>', writeOptionsValidators.value.description!)
  .option('--rpc-url [url]', writeOptionsValidators.rpcUrl.description!, {
    default: writeOptionsValidators.rpcUrl.parse(undefined),
  })
  .option(
    '--contract [address]',
    writeOptionsValidators.contract.description!,
    {
      default: writeOptionsValidators.contract.parse(undefined),
    }
  )
  .example(
    () =>
      `atst write --key "optimist.base-uri" --about 0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5 --value "my attestation" --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 --rpc-url http://localhost:8545`
  )
  .action(async (options: WriteOptions) => {
    const { write } = await import('./commands/write')

    // TODO use the native api to do this instead of parsing the raw args
    // by default options parses addresses as numbers without precision
    // we should use the args parsing library to do this directly
    // but for now I didn't bother to figure out how to do that
    const { rawArgs } = cli
    const privateKey = rawArgs[rawArgs.indexOf('--private-key') + 1] as Address
    const about = rawArgs[rawArgs.indexOf('--about') + 1] as Address
    const contract = rawArgs.includes('--contract')
      ? (rawArgs[rawArgs.indexOf('--contract') + 1] as Address)
      : options.contract

    await write({ ...options, about, privateKey, contract })
  })

cli.help()
cli.version(packageJson.version)

void (async () => {
  try {
    // Parse CLI args without running command
    cli.parse(process.argv, { run: false })
    if (!cli.matchedCommand && cli.args.length === 0) {
      cli.outputHelp()
    }
    await cli.runMatchedCommand()
  } catch (error) {
    logger.error(`\n${(error as Error).message}`)
    process.exit(1)
  }
})()
