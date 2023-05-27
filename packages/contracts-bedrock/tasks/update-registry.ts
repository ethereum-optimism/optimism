import fs from 'fs'
import path from 'path'

import { task } from 'hardhat/config'

/**
 * Updates the ChainRegistry contract by claiming new deployments
 * and registering new contract addresses for existing deployments
 */

task('update-registry', 'Update OP Stack Chain Registry')
  .addParam('registry', 'The address of the OpStackChainRegistry contract')
  .addParam(
    'deployment',
    'The path to the folder containing the deployment files'
  )
  .setAction(async (args, hre) => {
    const registry = await hre.ethers.getContractAt(
      'ChainRegistry',
      args.registry
    )

    const deploymentFolder = path.resolve(
      __dirname,
      `../deployments/${args.deployment}`
    )

    const deploymentFiles = fs
      .readdirSync(deploymentFolder)
      .filter((file) => file.endsWith('.json'))

    for (const file of deploymentFiles) {
      const filePath = path.join(deploymentFolder, file)
      const artifact = JSON.parse(fs.readFileSync(filePath, 'utf8'))

      const deploymentName = path.basename(path.dirname(filePath))
      const entryName = file.replace('.json', '')
      const entryAddress = artifact.address

      const deploymentAdmin = await hre.ethers.provider.getSigner().getAddress()

      // Check if the deployment has already been claimed.
      const existingAdmin = await registry.deployments(deploymentName)
      if (existingAdmin !== deploymentAdmin) {
        await registry.claimDeployment(deploymentName, deploymentAdmin)

        console.log(
          `Claimed deployment ${deploymentName} for ${deploymentAdmin}`
        )
      }

      // Check if the entry has already been registered.
      const existingAddress = await registry.registry(deploymentName, entryName)
      if (existingAddress !== entryAddress) {
        const entries = [
          {
            entryName,
            entryAddress,
          },
        ]

        await registry.register(deploymentName, entries)

        console.log(`Registered ${entryName} at ${entryAddress}`)
      }
    }

    console.log(
      `Updated registry at ${args.registry} with deployments in ${args.deploymentFolder}`
    )
  })
