import fs from 'fs'

import { task } from 'hardhat/config'

import { parseBalanceMap } from '../src/parse-balance-map'

task('generate-merkle-root')
  .addParam(
    'inFile',
    'Input JSON file location containing a map of account addresses to string balances'
  )
  .addParam('outFile', 'Output JSON file location for the Merkle data.')
  .setAction(async (args, hre) => {
    console.log('Reading balances map...')
    const json = JSON.parse(fs.readFileSync(args.inFile, { encoding: 'utf8' }))

    if (typeof json !== 'object') {
      throw new Error('Invalid JSON')
    }

    console.log('Parsing balances map...')
    const data = parseBalanceMap(json)
    console.log('Writing claims...')
    fs.writeFileSync(args.outFile, JSON.stringify(data, null, ' '))
    console.log(`Merkle root: ${data.merkleRoot}`)
    console.log(`Token total: ${hre.ethers.utils.formatEther(data.tokenTotal)}`)
    console.log(`Num claims:  ${Object.keys(data.claims).length}`)
  })
