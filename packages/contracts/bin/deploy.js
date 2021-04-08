#!/usr/bin/env node

const path = require('path')
const { spawn } = require('child_process')
const dirtree = require('directory-tree')

const main = async () => {
  const task = spawn(path.join(__dirname, 'deploy.ts'))

  await new Promise((resolve) => {
    task.on('exit', () => {
      resolve()
    })
  })

  // Stuff below this line is currently required for CI to work properly. We probably want to
  // update our CI so this is no longer necessary. But I'm adding it for backwards compat so we can
  // get the hardhat-deploy stuff merged. Woot.
  const nicknames = {
    'Lib_AddressManager': 'AddressManager',
    'mockOVM_BondManager': 'OVM_BondManager'
  }

  const contracts = dirtree(
    path.resolve(__dirname, `../deployments/custom`)
  ).children.filter((child) => {
    return child.extension === '.json'
  }).reduce((contracts, child) => {
    const contractName = child.name.replace('.json', '')
    const artifact = require(path.resolve(__dirname, `../deployments/custom/${child.name}`))
    contracts[nicknames[contractName] || contractName] = artifact.address
    return contracts
  }, {})

  // We *must* console.log here because CI will pipe the output of this script into an
  // addresses.json file. Also something we should probably remove.
  console.log(JSON.stringify(contracts, null, 2))
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.log(
      JSON.stringify({ error: error.message, stack: error.stack }, null, 2)
    )
    process.exit(1)
  })
