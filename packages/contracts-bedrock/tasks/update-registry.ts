import fs from 'fs'
import path from 'path'

import { task } from 'hardhat/config'

task('update-registry', 'Update OP Stack Chain Registry')
  .addParam('registry', 'The address of the OpStackChainRegistry contract')
  .addParam(
    'deploymentFolder',
    'The path to the folder containing the deployment files'
  )
  .setAction(async (args, hre) => {
    const OpStackChainRegistry = await hre.ethers.getContractFactory(
      'OpStackChainRegistry'
    )
    const registry = OpStackChainRegistry.attach(args.registry)

    const deploymentFolder = path.resolve(__dirname, args.deploymentFolder)

    const deploymentFiles = fs
      .readdirSync(deploymentFolder)
      .filter((file) => file.endsWith('.json'))

    for (const file of deploymentFiles) {
      const filePath = path.join(deploymentFolder, file)
      const artifact = JSON.parse(fs.readFileSync(filePath, 'utf8'))

      const deploymentName = file.replace('.json', '')
      const entryName = deploymentName
      const entryAddress = artifact.address

      const deploymentAdmin = await hre.ethers.provider.getSigner().getAddress()

      // Check if the deployment has already been claimed.
      const existingAdmin = await registry.deployments(deploymentName)
      if (existingAdmin !== deploymentAdmin) {
        const deployment = {
          deploymentName,
          deploymentAdmin,
        }

        await registry.claimDeployment(deployment)
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
      }
    }

    console.log(
      `Updated registry at ${args.registry} with deployments in ${args.deploymentFolder}`
    )
  })
