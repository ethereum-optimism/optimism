import hre from 'hardhat'
import * as dotenv from 'dotenv'

import { createDeploymentManager, getDeploymentBundle } from './core'

const main = async (hre: any) => {
  dotenv.config()

  const [owner] = await hre.ethers.getSigners()

  // 1. Create a ChugSplashDeployer
  const deployer = await createDeploymentManager(hre, await owner.getAddress())

  // 2. Generate the bundle of actions (SET_CODE or SET_STORAGE)
  const bundle = await getDeploymentBundle(
    hre,
    './deployments/old-deploy.json',
    deployer.address
  )

  // 3. Approve the bundle of actions.
  await deployer.approveTransactionBundle(bundle.hash, bundle.actions.length)

  // 4. Execute the bundle of actions.
  for (const action of bundle.actions) {
    console.log(`Executing chugsplash action`)
    console.log(`Target: ${action.target}`)
    console.log(`Type: ${action.type === 0 ? 'SET_CODE' : 'SET_STORAGE'}`)
    await deployer.executeAction(
      action.type,
      action.target,
      action.data,
      8_000_000 // TODO: how to handle gas?
    )
  }

  // 5. Verify the correctness of the deployment?
}

// misc improvements:
// want to minimize the need to perform unnecessary actions
// want to be able to perform multiple actions at the same time

main(hre)
