import { expect } from '../setup'
import {
  ChugSplashActionBundle,
  executeActionsFromConfig,
  makeActionBundleFromConfig,
} from '../../src'

/* Imports: External */
import hre from 'hardhat'
import { Contract, Signer, ContractFactory } from 'ethers'

// relative path to deploy.ts
const CONFIG_PATH = '../../test/chugsplash/example-configs/deploy-l2.json'
const INVALID_CONFIG_PATH =
  '../../test/chugsplash/example-configs/invalid-config.json'

describe('ChugSplash deploy script', () => {
  let signer: Signer
  let Factory__L2ChugSplashDeployer: ContractFactory

  before(async () => {
    ;[signer] = await hre.ethers.getSigners()
    Factory__L2ChugSplashDeployer = await hre.ethers.getContractFactory(
      'L2ChugSplashDeployer'
    )
  })

  describe('executeActionsFromConfig', () => {
    let L2ChugSplashDeployer: Contract
    let currActionBundle: ChugSplashActionBundle
    beforeEach(async () => {
      L2ChugSplashDeployer = await Factory__L2ChugSplashDeployer.connect(
        signer
      ).deploy(await signer.getAddress())
      currActionBundle = await makeActionBundleFromConfig(
        hre,
        require(CONFIG_PATH)
      )
    })

    it('should correctly send executeAction transactions', async () => {
      await L2ChugSplashDeployer.connect(signer).approveTransactionBundle(
        currActionBundle.root,
        currActionBundle.actions.length
      )

      const receipts = await executeActionsFromConfig({
        hre,
        signer,
        chugsplashDeployerAddress: L2ChugSplashDeployer.address,
        upgradeConfigPath: CONFIG_PATH,
      })

      expect(receipts.length).to.eq(currActionBundle.actions.length)
    })

    it('should not execute if the bundle is not approved', async () => {
      const receipts = await executeActionsFromConfig({
        hre,
        signer,
        chugsplashDeployerAddress: L2ChugSplashDeployer.address,
        upgradeConfigPath: CONFIG_PATH,
        timeoutInMs: 2_000,
      })

      expect(receipts).to.be.undefined
    })

    it('should keep retrying until timeout', async () => {
      const receipts = await executeActionsFromConfig({
        hre,
        signer,
        chugsplashDeployerAddress: L2ChugSplashDeployer.address,
        upgradeConfigPath: CONFIG_PATH,
      })

      await L2ChugSplashDeployer.connect(signer).approveTransactionBundle(
        currActionBundle.root,
        currActionBundle.actions.length
      )

      // setTimeout(async () => {await L2ChugSplashDeployer.connect(signer).approveTransactionBundle(
      //   currActionBundle.root,
      //   currActionBundle.actions.length
      // )}, 2_000)

      expect(receipts.length).to.eq(currActionBundle.actions.length)
    })

    it('should throw error with invalid config', async () => {
      expect(async () => {
        await executeActionsFromConfig({
          hre,
          signer,
          chugsplashDeployerAddress: L2ChugSplashDeployer.address,
          upgradeConfigPath: INVALID_CONFIG_PATH,
        })
      }).to.throw
    })
  })
})
