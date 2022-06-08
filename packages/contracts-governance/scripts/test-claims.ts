import fs from 'fs'

import { task } from 'hardhat/config'

import { MerkleDistributorInfo } from '../src/parse-balance-map'

task('test-claims')
  .addParam('inFile', 'Input claims file')
  .addParam('distributorAddress', 'Address of the distributor')
  .setAction(async (args, hre) => {
    const distrib = (
      await hre.ethers.getContractAt(
        'MerkleDistributor',
        args.distributorAddress
      )
    ).connect(hre.ethers.provider)
    console.log('Reading claims...')
    const json = JSON.parse(
      fs.readFileSync(args.inFile, { encoding: 'utf8' })
    ) as MerkleDistributorInfo

    console.log('Smoke testing 100 random claims.')
    const addresses = Object.keys(json.claims)
    for (let i = 0; i < 100; i++) {
      const index = Math.floor(addresses.length * Math.random())
      const addr = addresses[index]
      const claim = json.claims[addr]
      process.stdout.write(`Attempting claim for ${addr} [${i + 1}/100]... `)
      await distrib.callStatic.claim(
        claim.index,
        addr,
        claim.amount,
        claim.proof
      )
      process.stdout.write('OK\n')
    }
    console.log('Smoke test passed.')
  })
