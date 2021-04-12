// Helper script for checking if the local / remote bytecode/addresses matches for a deployment
const ethers = require('ethers')
const dirtree = require('directory-tree')
const yargs = require('yargs/yargs')
const { hideBin } = require('yargs/helpers')
const argv = yargs(hideBin(process.argv)).argv

const nicknames = {
  'mockOVM_BondManager': 'OVM_BondManager'
}

;(async () => {
  console.log(`Checking deployment for network: ${argv.network}`)

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

  const contracts = dirtree(`./deployments/${argv.network}`).children.filter((child) => {
    return child.extension === '.json'
  }).map((child) => {
    return child.name.replace('.json', '')
  })

  for (const contract of contracts) {
    const deployment = require(`../deployments/${argv.network}/${contract}.json`)

    if (contract !== 'Lib_AddressManager') {
      const address = await Lib_AddressManager.getAddress(nicknames[contract] || contract)
      if (address !== deployment.address) {
        console.log(`✖ ${contract} (ADDRESS MISMATCH DETECTED)`)
        continue
      }
    }

    // First do some basic checks on the local bytecode and remote bytecode.
    const local = deployment.deployedBytecode
    const remote = await provider.getCode(deployment.address)
    if (ethers.utils.keccak256(local) !== ethers.utils.keccak256(remote)) {
      console.log(`✖ ${contract} (CODE MISMATCH DETECTED)`)
      continue
    }

    console.log(`✓ ${contract}`)
  }
})()
