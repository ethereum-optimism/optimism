import hre from 'hardhat'
import * as dotenv from 'dotenv'
import { ContractFactory } from 'ethers'

import { getDeploymentBundle } from './core'

const main = async (hre: any) => {
  dotenv.config()

  // 1. Create a ChugSplashDeployer
  // const deployer = await createDeploymentManager(hre, await owner.getAddress())
  const factory: ContractFactory = await hre.ethers.getContractFactory(
    'ChugSplashDeployer'
  )
  const deployer = factory.attach('0x420000000000000000000000000000000000000A')

  // 2. Generate the bundle of actions (SET_CODE or SET_STORAGE)
  const bundle = await getDeploymentBundle(
    hre,
    './chugsplash-deploy/deploy-l2.json',
    deployer.address
  )

  // 3. Approve the bundle of actions.
  await deployer.approveTransactionBundle(bundle.hash, bundle.actions.length)

  // 4. Execute the bundle of actions.
  for (const action of bundle.actions) {
    console.log(`Executing chugsplash action`)
    console.log(`Target: ${action.target}`)
    console.log(`Type: ${action.type === 0 ? 'SET_CODE' : 'SET_STORAGE'}`)
    // const result = await deployer.executeAction(
    //   {
    //     actionType: action.type,
    //     target: action.target,
    //     data: action.data,
    //   },
    //   {
    //     actionIndex: 0,
    //     siblings: []
    //   }
    // )
    // await result.wait()
  }

  // 5. Verify the correctness of the deployment?
}

// misc improvements:
// want to minimize the need to perform unnecessary actions
// want to be able to perform multiple actions at the same time

main(hre)
