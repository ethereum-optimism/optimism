import assert from 'assert'

import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { awaitCondition } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'

import {
  assertContractVariable,
  getContractsFromArtifacts,
  getDeploymentAddress,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // Set up required contract references.
  const [
    SystemDictator,
    ProxyAdmin,
    AddressManager,
    L1CrossDomainMessenger,
    L1StandardBridgeProxy,
    L1StandardBridgeProxyWithSigner,
    L1StandardBridge,
    L2OutputOracle,
    OptimismPortal,
    OptimismMintableERC20Factory,
    L1ERC721BridgeProxy,
    L1ERC721BridgeProxyWithSigner,
    L1ERC721Bridge,
  ] = await getContractsFromArtifacts(hre, [
    {
      name: 'SystemDictatorProxy',
      iface: 'SystemDictator',
      signerOrProvider: deployer,
    },
    {
      name: 'ProxyAdmin',
      signerOrProvider: deployer,
    },
    {
      name: 'Lib_AddressManager',
      signerOrProvider: deployer,
    },
    {
      name: 'Proxy__OVM_L1CrossDomainMessenger',
      iface: 'L1CrossDomainMessenger',
      signerOrProvider: deployer,
    },
    {
      name: 'Proxy__OVM_L1StandardBridge',
    },
    {
      name: 'Proxy__OVM_L1StandardBridge',
      signerOrProvider: deployer,
    },
    {
      name: 'Proxy__OVM_L1StandardBridge',
      iface: 'L1StandardBridge',
      signerOrProvider: deployer,
    },
    {
      name: 'L2OutputOracleProxy',
      iface: 'L2OutputOracle',
      signerOrProvider: deployer,
    },
    {
      name: 'OptimismPortalProxy',
      iface: 'OptimismPortal',
      signerOrProvider: deployer,
    },
    {
      name: 'OptimismMintableERC20FactoryProxy',
      iface: 'OptimismMintableERC20Factory',
      signerOrProvider: deployer,
    },
    {
      name: 'L1ERC721BridgeProxy',
    },
    {
      name: 'L1ERC721BridgeProxy',
      signerOrProvider: deployer,
    },
    {
      name: 'L1ERC721BridgeProxy',
      iface: 'L1ERC721Bridge',
      signerOrProvider: deployer,
    },
  ])

  // If we have the key for the controller then we don't need to wait for external txns.
  const isLiveDeployer =
    deployer.toLowerCase() === hre.deployConfig.controller.toLowerCase()

  // Transfer ownership of the ProxyAdmin to the SystemDictator.
  if ((await ProxyAdmin.owner()) !== SystemDictator.address) {
    console.log(`Setting ProxyAdmin owner to MSD`)
    await ProxyAdmin.transferOwnership(SystemDictator.address)
  } else {
    console.log(`Proxy admin already owned by MSD`)
  }

  // We don't need to transfer proxy addresses if we're already beyond the proxy transfer step.
  const needsProxyTransfer =
    (await SystemDictator.currentStep()) <=
    (await SystemDictator.PROXY_TRANSFER_STEP())

  // Transfer ownership of the AddressManager to SystemDictator.
  if (
    needsProxyTransfer &&
    (await AddressManager.owner()) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting AddressManager owner to MSD`)
      await AddressManager.transferOwnership(SystemDictator.address)
    } else {
      console.log(`Please transfer AddressManager owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await AddressManager.owner()
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`AddressManager already owned by the SystemDictator`)
  }

  // Transfer ownership of the L1CrossDomainMessenger to SystemDictator.
  if (
    needsProxyTransfer &&
    (await AddressManager.getAddress('OVM_L1CrossDomainMessenger')) !==
      ethers.constants.AddressZero &&
    (await L1CrossDomainMessenger.owner()) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1CrossDomainMessenger owner to MSD`)
      await L1CrossDomainMessenger.transferOwnership(SystemDictator.address)
    } else {
      console.log(`Please transfer L1CrossDomainMessenger owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await L1CrossDomainMessenger.owner()
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`L1CrossDomainMessenger already owned by MSD`)
  }

  // Transfer ownership of the L1StandardBridge (proxy) to SystemDictator.
  if (
    needsProxyTransfer &&
    (await L1StandardBridgeProxy.callStatic.getOwner({
      from: ethers.constants.AddressZero,
    })) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1StandardBridge owner to MSD`)
      await L1StandardBridgeProxyWithSigner.setOwner(SystemDictator.address)
    } else {
      console.log(`Please transfer L1StandardBridge (proxy) owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await L1StandardBridgeProxy.callStatic.getOwner({
          from: ethers.constants.AddressZero,
        })
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`L1StandardBridge already owned by MSD`)
  }

  // Transfer ownership of the L1ERC721Bridge (proxy) to SystemDictator.
  if (
    needsProxyTransfer &&
    (await L1ERC721BridgeProxy.callStatic.admin({
      from: ethers.constants.AddressZero,
    })) !== SystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1ERC721Bridge owner to MSD`)
      await L1ERC721BridgeProxyWithSigner.changeAdmin(SystemDictator.address)
    } else {
      console.log(`Please transfer L1ERC721Bridge (proxy) owner to MSD`)
      console.log(`MSD address: ${SystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(
      async () => {
        const owner = await L1ERC721BridgeProxy.callStatic.admin({
          from: ethers.constants.AddressZero,
        })
        return owner === SystemDictator.address
      },
      30000,
      1000
    )
  } else {
    console.log(`L1ERC721Bridge already owned by MSD`)
  }

  /**
   * Mini helper for checking if the current step is a target step.
   *
   * @param step Target step.
   * @returns True if the current step is the target step.
   */
  const isStep = async (step: number): Promise<boolean> => {
    return (await SystemDictator.currentStep()) === step
  }

  /**
   * Mini helper for executing a given step.
   *
   * @param opts Options for executing the step.
   * @param opts.step Step to execute.
   * @param opts.message Message to print before executing the step.
   * @param opts.checks Checks to perform after executing the step.
   */
  const doStep = async (opts: {
    step: number
    message: string
    checks: () => Promise<void>
  }): Promise<void> => {
    if (!(await isStep(opts.step))) {
      console.log(`Step already completed: ${opts.step}`)
      return
    }

    // Extra message to help the user understand what's going on.
    console.log(opts.message)

    // Either automatically or manually execute the step.
    if (isLiveDeployer) {
      console.log(`Executing step ${opts.step}...`)
      await SystemDictator[`step${opts.step}`]()
    } else {
      console.log(`Please execute step ${opts.step}...`)
    }

    // Wait for the step to complete.
    await awaitCondition(
      async () => {
        return (await SystemDictator.currentStep()) === opts.step + 1
      },
      30000,
      1000
    )

    // Perform post-step checks.
    await opts.checks()
  }

  // Step 1 is a freebie, it doesn't impact the system.
  await doStep({
    step: 1,
    message: `
      Step 1 will configure the ProxyAdmin contract, you can safely execute this step at any time
      without impacting the functionality of the rest of the system.
    `,
    checks: async () => {
      await assertContractVariable(
        ProxyAdmin,
        'addressManager',
        AddressManager.address
      )
      assert(
        (await ProxyAdmin.implementationName(
          getDeploymentAddress(hre, 'Proxy__OVM_L1CrossDomainMessenger')
        )) === 'OVM_L1CrossDomainMessenger'
      )
      assert(
        (await ProxyAdmin.proxyType(
          getDeploymentAddress(hre, 'Proxy__OVM_L1CrossDomainMessenger')
        )) === 2
      )
      assert(
        (await ProxyAdmin.proxyType(
          getDeploymentAddress(hre, 'Proxy__OVM_L1StandardBridge')
        )) === 1
      )
    },
  })

  // Step 2 shuts down the system.
  await doStep({
    step: 2,
    message: `
      Step 2 will stop deposits and withdrawals via the L1CrossDomainMessenger and will stop the
      DTL from syncing new deposits via the CTC, effectively shutting down the legacy system. Once
      this step has been executed, you should immediately begin the L2 migration process. If you
      need to restart the system, run exit1() followed by finalize().
    `,
    checks: async () => {
      assert(
        (await AddressManager.getAddress('OVM_L1CrossDomainMessenger')) ===
          ethers.constants.AddressZero
      )
    },
  })

  // Step 3 clears out some state from the AddressManager.
  await doStep({
    step: 3,
    message: `
      Step 3 will clear out some legacy state from the AddressManager. Once you execute this step,
      you WILL NOT BE ABLE TO RESTART THE SYSTEM using exit1(). You should confirm that the L2
      system is entirely operational before executing this step.
    `,
    checks: async () => {
      const deads = [
        'OVM_CanonicalTransactionChain',
        'OVM_L2CrossDomainMessenger',
        'OVM_DecompressionPrecompileAddress',
        'OVM_Sequencer',
        'OVM_Proposer',
        'OVM_ChainStorageContainer-CTC-batches',
        'OVM_ChainStorageContainer-CTC-queue',
        'OVM_CanonicalTransactionChain',
        'OVM_StateCommitmentChain',
        'OVM_BondManager',
        'OVM_ExecutionManager',
        'OVM_FraudVerifier',
        'OVM_StateManagerFactory',
        'OVM_StateTransitionerFactory',
        'OVM_SafetyChecker',
        'OVM_L1MultiMessageRelayer',
        'BondManager',
      ]
      for (const dead of deads) {
        assert(
          (await AddressManager.getAddress(dead)) ===
            ethers.constants.AddressZero
        )
      }
    },
  })

  // Step 4 transfers ownership of the AddressManager and L1StandardBridge to the ProxyAdmin.
  await doStep({
    step: 4,
    message: `
      Step 4 will transfer ownership of the AddressManager and L1StandardBridge to the ProxyAdmin.
    `,
    checks: async () => {
      await assertContractVariable(AddressManager, 'owner', ProxyAdmin.address)

      assert(
        (await L1StandardBridgeProxy.callStatic.getOwner({
          from: ethers.constants.AddressZero,
        })) === ProxyAdmin.address
      )
    },
  })

  // Make sure the dynamic system configuration has been set.
  if ((await isStep(5)) && !(await SystemDictator.dynamicConfigSet())) {
    console.log(`
      You must now set the dynamic L2OutputOracle configuration by calling the function
      updateL2OutputOracleDynamicConfig. You will need to provide the
      l2OutputOracleStartingBlockNumber and the l2OutputOracleStartingTimestamp which can both be
      found by querying the last finalized block in the L2 node.
    `)

    if (isLiveDeployer) {
      console.log(`Updating dynamic oracle config...`)

      // Use default starting time if not provided
      let deployL2StartingTimestamp =
        hre.deployConfig.l2OutputOracleStartingTimestamp
      if (deployL2StartingTimestamp < 0) {
        const l1StartingBlock = await hre.ethers.provider.getBlock(
          hre.deployConfig.l1StartingBlockTag
        )
        if (l1StartingBlock === null) {
          throw new Error(
            `Cannot fetch block tag ${hre.deployConfig.l1StartingBlockTag}`
          )
        }
        deployL2StartingTimestamp = l1StartingBlock.timestamp
      }

      await SystemDictator.updateL2OutputOracleDynamicConfig({
        l2OutputOracleStartingBlockNumber:
          hre.deployConfig.l2OutputOracleStartingBlockNumber,
        l2OutputOracleStartingTimestamp: deployL2StartingTimestamp,
      })
    } else {
      console.log(`Please update dynamic oracle config...`)
    }

    await awaitCondition(
      async () => {
        return SystemDictator.dynamicConfigSet()
      },
      30000,
      1000
    )
  }

  // Step 5 initializes all contracts and pauses the new L1CrossDomainMessenger.
  await doStep({
    step: 5,
    message: `
      Step 5 will initialize all Bedrock contracts but will leave the new L1CrossDomainMessenger
      paused. After this step is executed, users will be able to deposit and withdraw assets via
      the OptimismPortal but not via the L1CrossDomainMessenger. The Proposer will also be able to
      submit L2 outputs to the L2OutputOracle.
    `,
    checks: async () => {
      // Check L2OutputOracle was initialized properly.
      await assertContractVariable(
        L2OutputOracle,
        'latestBlockNumber',
        hre.deployConfig.l2OutputOracleStartingBlockNumber
      )

      // Check OptimismPortal was initialized properly.
      await assertContractVariable(
        OptimismPortal,
        'l2Sender',
        '0x000000000000000000000000000000000000dEaD'
      )
      const resourceParams = await OptimismPortal.params()
      assert(
        resourceParams.prevBaseFee.eq(await OptimismPortal.INITIAL_BASE_FEE()),
        `OptimismPortal was not initialized with the correct initial base fee`
      )
      assert(
        resourceParams.prevBoughtGas.eq(0),
        `OptimismPortal was not initialized with the correct initial bought gas`
      )
      assert(
        !resourceParams.prevBlockNum.eq(0),
        `OptimismPortal was not initialized with the correct initial block number`
      )
      assert(
        (await hre.ethers.provider.getBalance(L1StandardBridge.address)).eq(0)
      )

      // Check L1CrossDomainMessenger was initialized properly.
      await assertContractVariable(L1CrossDomainMessenger, 'paused', true)
      try {
        await L1CrossDomainMessenger.xDomainMessageSender()
        assert(false, `L1CrossDomainMessenger was not initialized properly`)
      } catch (err) {
        // Expected.
      }
      await assertContractVariable(
        L1CrossDomainMessenger,
        'owner',
        SystemDictator.address
      )

      // Check L1StandardBridge was initialized properly.
      await assertContractVariable(
        L1StandardBridge,
        'messenger',
        L1CrossDomainMessenger.address
      )
      assert(
        (await hre.ethers.provider.getBalance(L1StandardBridge.address)).eq(0)
      )

      // Check OptimismMintableERC20Factory was initialized properly.
      await assertContractVariable(
        OptimismMintableERC20Factory,
        'bridge',
        L1StandardBridge.address
      )

      // Check L1ERC721Bridge was initialized properly.
      await assertContractVariable(
        L1ERC721Bridge,
        'messenger',
        L1CrossDomainMessenger.address
      )
    },
  })

  // Step 6 unpauses the new L1CrossDomainMessenger.
  await doStep({
    step: 6,
    message: `
      Step 6 will unpause the new L1CrossDomainMessenger. After this step is executed, users will
      be able to deposit and withdraw assets via the L1CrossDomainMessenger and the system will be
      fully operational.
    `,
    checks: async () => {
      await assertContractVariable(L1CrossDomainMessenger, 'paused', false)
    },
  })

  // At the end we finalize the upgrade.
  if (await isStep(7)) {
    console.log(`
      You must now finalize the upgrade by calling finalize() on the SystemDictator. This will
      transfer ownership of the ProxyAdmin and the L1CrossDomainMessenger to the final system owner
      as specified in the deployment configuration.
    `)

    if (isLiveDeployer) {
      console.log(`Finalizing deployment...`)
      await SystemDictator.finalize()
    } else {
      console.log(`Please finalize deployment...`)
    }

    await awaitCondition(
      async () => {
        return SystemDictator.finalized()
      },
      30000,
      1000
    )

    await assertContractVariable(
      L1CrossDomainMessenger,
      'owner',
      hre.deployConfig.finalSystemOwner
    )
    await assertContractVariable(
      ProxyAdmin,
      'owner',
      hre.deployConfig.finalSystemOwner
    )
  }
}

deployFn.tags = ['SystemDictatorSteps']

export default deployFn
