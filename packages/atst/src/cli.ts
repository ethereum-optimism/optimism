#!/usr/bin/env node
import { cac } from 'cac'
import { Address } from '@wagmi/core'

import { optionsValidators, ReadOptions } from './commands/read'
import * as logger from './logger'
// @ts-ignore it's mad about me importing something not in tsconfig.includes
import packageJson from '../package.json'

const cli = cac('atst')

cli
  .command('read', 'read an attestation')
  .option('--creator <string>', optionsValidators.creator.description)
  .option('--about <string>', optionsValidators.about.description)
  .option('--key <string>', optionsValidators.key.description)
  .option('--data-type [string]', optionsValidators.dataType.description, {
    default: optionsValidators.dataType.parse(undefined),
  })
  .option('--rpc-url [url]', optionsValidators.rpcUrl.description, {
    default: optionsValidators.rpcUrl.parse(undefined),
  })
  .option('--contract [address]', optionsValidators.contract.description, {
    default: optionsValidators.contract.parse(undefined),
  })
  .example(
    // todo make example better
    // TODO make it so you don't have to --specify --every --flag --like --this
    // instead allow for creator about and key to be specified as positional arguments
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
