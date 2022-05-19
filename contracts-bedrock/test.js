const ethers = require('ethers')
const fs = require('fs')

const outfile = 'results.csv'
const url = process.env.ETH_RPC_URL || 'http://localhost:8545'

;(async () => {
  const provider = new ethers.providers.StaticJsonRpcProvider(url)
  const tip = await provider.getBlock()
  const start = tip.number - 500

  console.log(`starting at ${start}`)
  console.log(`ending at ${tip.number}`)

  const list = [
    ['height', 'basefee']
  ]

  for (let i = start; i <= tip.number; i += 10) {
    const promises = []
    for (let j = 0; j < 10; j++) {
      if (j <= tip.number) {
        if (i+j % 500 === 0) {
          console.log(`fetching block ${i+j}`)
        }
        promises.push(provider.getBlock(i+j))
      }
    }
    const blocks = await Promise.all(promises)
    for (const block of blocks) {
      if (block !== null) {
        list.push([block.number.toString(), block.baseFeePerGas.toString()])
      }
    }
  }
  let str = ''
  for (const [height, basefee] of list) {
    str += `${height},${basefee}\n`
  }
  fs.writeFileSync(outfile, str)
})().catch(err => {
  console.log(err)
})
