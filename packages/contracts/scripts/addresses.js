const ethers = require('ethers')
const yargs = require('yargs/yargs')
const { hideBin } = require('yargs/helpers')
const argv = yargs(hideBin(process.argv)).argv

;(async () => {
  console.log(`Listing known addresses for: ${argv.network}`)

  const provider = new ethers.providers.JsonRpcProvider(argv.rpcUrl)

  // Get a reference to the address manager and throw if unable to do so.
  let Lib_AddressManager
  try {
    const def__Lib_AddressManager = require(`../deployments/${argv.network}/Lib_AddressManager.json`)
    Lib_AddressManager = new ethers.Contract(
      def__Lib_AddressManager.address,
      def__Lib_AddressManager.abi,
      provider
    )
  } catch (err) {
    throw new Error(`unable to get a reference to Lib_AddressManager`)
  }

  const events = await Lib_AddressManager.queryFilter(
    Lib_AddressManager.filters.AddressSet()
  )

  const addresses = {}
  for (const event of events) {
    addresses[event.args._name] = event.args._newAddress
  }

  const table = []
  for (const name of Object.keys(addresses)) {
    if (addresses[name] !== ethers.constants.AddressZero) {
      table.push({
        name: name,
        address: addresses[name]
      })
    }
  }

  console.table(table)
})()
