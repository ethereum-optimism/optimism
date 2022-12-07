import fs from 'fs'

import { Command } from 'commander'
import { ethers } from 'ethers'
import { getContractInterface } from '@eth-optimism/contracts'

import { version } from '../package.json'
import { advancedQueryFilter } from '../src/advanced-query'

const program = new Command()

program
  .name('migration-data-query')
  .description('CLI for querying Bedrock migration data')
  .version(version)

program
  .command('parse-state-dump')
  .description('parses state dump to json')
  .option('--file <file>', 'path to state dump file')
  .action(async (options) => {
    const iface = getContractInterface('OVM_L2ToL1MessagePasser')
    const dump = fs.readFileSync(options.file, 'utf-8')

    const addrs: string[] = []
    const msgs: any[] = []
    for (const line of dump.split('\n')) {
      if (line.startsWith('ETH')) {
        addrs.push(line.split('|')[1].replace('\r', ''))
      } else if (line.startsWith('MSG')) {
        const msg = '0x' + line.split('|')[2].replace('\r', '')
        const parsed = iface.decodeFunctionData('passMessageToL1', msg)
        msgs.push({
          who: line.split('|')[1],
          msg: parsed._message,
        })
      }
    }

    fs.writeFileSync(
      './data/evm-addresses.json',
      JSON.stringify(addrs, null, 2)
    )
    fs.writeFileSync('./data/evm-messages.json', JSON.stringify(msgs, null, 2))
  })

program
  .command('evm-sent-messages')
  .description('queries messages sent after the EVM upgrade')
  .option('--rpc <rpc>', 'rpc url to use')
  .action(async (options) => {
    const provider = new ethers.providers.JsonRpcProvider(options.rpc)

    const xdm = new ethers.Contract(
      '0x4200000000000000000000000000000000000007',
      getContractInterface('L2CrossDomainMessenger'),
      provider
    )

    const sent: any[] = await advancedQueryFilter(xdm, {
      queryFilter: xdm.filters.SentMessage(),
    })

    const messages: any[] = []
    for (const s of sent) {
      messages.push({
        who: '0x4200000000000000000000000000000000000007',
        msg: xdm.interface.encodeFunctionData('relayMessage', [
          s.args.target,
          s.args.sender,
          s.args.message,
          s.args.messageNonce,
        ]),
      })
    }

    fs.writeFileSync(
      './data/evm-messages.json',
      JSON.stringify(messages, null, 2)
    )
  })

program
  .command('sent-slots')
  .description('queries storage slots in the message passer')
  .option('--rpc <rpc>', 'rpc url to use')
  .action(async (options) => {
    const provider = new ethers.providers.JsonRpcProvider(options.rpc)

    let nextKey = '0x'
    let slots: any[] = []
    while (nextKey) {
      const latestBlock = await provider.getBlock('latest')
      const ret = await provider.send('debug_storageRangeAt', [
        latestBlock.hash,
        0,
        '0x4200000000000000000000000000000000000000',
        nextKey,
        10000,
      ])

      slots = slots.concat(
        Object.values(ret.storage).map((s: any) => {
          return s.key
        })
      )

      // Update next key and potentially try again
      nextKey = ret.nextKey
    }

    fs.writeFileSync('./data/slots.json', JSON.stringify(slots, null, 2))
  })

program
  .command('accounting')
  .description('verifies that we have sufficient slot data')
  .action(async () => {
    const parseMessageFile = (
      path: string
    ): Array<{
      message: string
      slot: string
    }> => {
      const messages: any[] = JSON.parse(fs.readFileSync(path, 'utf8'))
      return messages.map((message) => {
        return {
          message,
          slot: ethers.utils.keccak256(
            ethers.utils.hexConcat([
              ethers.utils.keccak256(
                ethers.utils.hexConcat([message.msg, message.who])
              ),
              ethers.constants.HashZero,
            ])
          ),
        }
      })
    }

    const ovmMessages = parseMessageFile('./data/ovm-messages.json')
    const evmMessages = parseMessageFile('./data/evm-messages.json')
    const slotList: string[] = JSON.parse(
      fs.readFileSync('./data/slots.json', 'utf8')
    )

    const unaccounted = slotList.filter((slot) => {
      return (
        !ovmMessages.some((m) => m.slot === slot) &&
        !evmMessages.some((m) => m.slot === slot)
      )
    })

    console.log(`Total slots: ${slotList.length}`)
    console.log(`Unaccounted slots: ${unaccounted.length}`)
  })

program.parse(process.argv)
