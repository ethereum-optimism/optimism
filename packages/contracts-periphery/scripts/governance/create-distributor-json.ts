import fs from 'fs'

import { task } from 'hardhat/config'
import { parse } from 'csv-parse'
import { BigNumber } from 'ethers'

task('create-distributor-json')
  .addParam('inFile', 'CSV to read')
  .addParam('outFile', 'JSON to create')
  .addOptionalParam(
    'mnemonic',
    'Mnemonic',
    process.env.DISTRIBUTOR_FALLBACK_MNEMONIC
  )
  .setAction(async (args, hre) => {
    const parser = fs.createReadStream(args.inFile).pipe(parse())
    const records = []
    let total = BigNumber.from(0)
    for await (const record of parser) {
      const name = record[0].trim()
      const amt = record[record.length - 4].trim().replace(/,/gi, '')
      const address = record[record.length - 3].trim()

      records.push({
        name,
        amountHuman: amt,
        amount: hre.ethers.utils.parseEther(amt).toString(),
        address,
        fallbackIndex: -1,
      })
      total = total.add(amt)
    }

    records.sort((a, b) => {
      if (a.name > b.name) {
        return 1
      }

      if (a.name < b.name) {
        return -1
      }

      return 0
    })

    for (let i = 0; i < records.length; i++) {
      const record = records[i]
      if (record.address.slice(0, 2) !== '0x') {
        console.log(
          `Generating fallback address for ${record.name}. Account index: ${i}`
        )
        const wallet = hre.ethers.Wallet.fromMnemonic(
          args.mnemonic,
          `m/44'/60'/0'/0/${i}`
        )
        record.address = wallet.address
        record.fallbackIndex = i
      }
    }

    fs.writeFileSync(args.outFile, JSON.stringify(records, null, ' '))
    console.log(`Total: ${total.toString()}`)
    if (total.eq(1_434_262_041)) {
      console.log('AMOUNTS VERIFIED')
    } else {
      throw new Error('AMOUNTS INVALID')
    }
  })
