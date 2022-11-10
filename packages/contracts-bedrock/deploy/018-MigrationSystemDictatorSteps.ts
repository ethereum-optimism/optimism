import assert from 'assert'

import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'
import { awaitCondition } from '@eth-optimism/core-utils'

import {
  assertContractVariable,
  getContractsFromArtifacts,
  getDeploymentAddress,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  let isLiveDeployer = false
  if (hre.deployConfig.controller === deployer) {
    isLiveDeployer = true
  }

  // Set up required contract references.
  const [
    MigrationSystemDictator,
    ProxyAdmin,
    AddressManager,
    L1CrossDomainMessenger,
    L1StandardBridgeProxy,
    L1StandardBridgeProxyWithSigner,
    L1StandardBridge,
    L2OutputOracle,
    OptimismPortal,
    OptimismMintableERC20Factory,
    L1ERC721Bridge,
  ] = await getContractsFromArtifacts(hre, [
    {
      name: 'MigrationSystemDictator',
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
      iface: 'L1ERC721Bridge',
      signerOrProvider: deployer,
    },
  ])

  // Transfer ownership of the ProxyAdmin to the MigrationSystemDictator.
  if ((await ProxyAdmin.owner()) !== MigrationSystemDictator.address) {
    console.log(`Setting ProxyAdmin owner to MSD`)
    await ProxyAdmin.transferOwnership(MigrationSystemDictator.address)
  } else {
    console.log(`Proxy admin already owned by MSD`)
  }

  // Transfer ownership of the AddressManager to MigrationSystemDictator.
  if ((await AddressManager.owner()) !== MigrationSystemDictator.address) {
    if (isLiveDeployer) {
      console.log(`Setting AddressManager owner to MSD`)
      await AddressManager.transferOwnership(MigrationSystemDictator.address)
    } else {
      console.log(`Please transfer AddressManager owner to MSD`)
      console.log(`MSD address: ${MigrationSystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(async () => {
      const owner = await AddressManager.owner()
      return owner === MigrationSystemDictator.address
    })
  } else {
    console.log(`AddressManager already owned by the MigrationSystemDictator`)
  }

  // Transfer ownership of the L1CrossDomainMessenger to MigrationSystemDictator.
  if (
    (await L1CrossDomainMessenger.owner()) !== MigrationSystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1CrossDomainMessenger owner to MSD`)
      await L1CrossDomainMessenger.transferOwnership(
        MigrationSystemDictator.address
      )
    } else {
      console.log(`Please transfer L1CrossDomainMessenger owner to MSD`)
      console.log(`MSD address: ${MigrationSystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(async () => {
      const owner = await L1CrossDomainMessenger.owner()
      return owner === MigrationSystemDictator.address
    })
  } else {
    console.log(`L1CrossDomainMessenger already owned by MSD`)
  }

  // Transfer ownership of the L1StandardBridge (proxy) to MigrationSystemDictator.
  if (
    (await L1StandardBridgeProxy.callStatic.getOwner({
      from: ethers.constants.AddressZero,
    })) !== MigrationSystemDictator.address
  ) {
    if (isLiveDeployer) {
      console.log(`Setting L1StandardBridge owner to MSD`)
      await L1StandardBridgeProxyWithSigner.setOwner(
        MigrationSystemDictator.address
      )
    } else {
      console.log(`Please transfer L1StandardBridge (proxy) owner to MSD`)
      console.log(`MSD address: ${MigrationSystemDictator.address}`)
    }

    // Wait for the ownership transfer to complete.
    await awaitCondition(async () => {
      const owner = await L1StandardBridgeProxy.callStatic.getOwner({
        from: ethers.constants.AddressZero,
      })
      return owner === MigrationSystemDictator.address
    })
  } else {
    console.log(`L1StandardBridge already owned by MSD`)
  }

  const checks = {
    1: async () => {
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
    2: async () => {
      await assertContractVariable(L1CrossDomainMessenger, 'paused', true)
      const deads = [
        'Proxy__OVM_L1CrossDomainMessenger',
        'Proxy__OVM_L1StandardBridge',
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
      ]
      for (const dead of deads) {
        assert(
          (await AddressManager.getAddress(dead)) ===
            ethers.constants.AddressZero
        )
      }
    },
    3: async () => {
      await assertContractVariable(AddressManager, 'owner', ProxyAdmin.address)
      assert(
        (await L1StandardBridgeProxy.callStatic.getOwner({
          from: ethers.constants.AddressZero,
        })) === ProxyAdmin.address
      )
    },
    4: async () => {
      // Check L2OutputOracle was initialized properly.
      await assertContractVariable(
        L2OutputOracle,
        'latestBlockNumber',
        hre.deployConfig.l2OutputOracleStartingBlockNumber
      )
      await assertContractVariable(
        L2OutputOracle,
        'proposer',
        hre.deployConfig.l2OutputOracleProposer
      )
      await assertContractVariable(
        L2OutputOracle,
        'owner',
        hre.deployConfig.l2OutputOracleOwner
      )
      if (
        hre.deployConfig.l2OutputOracleGenesisL2Output !==
        ethers.constants.HashZero
      ) {
        const genesisOutput = await L2OutputOracle.getL2Output(
          hre.deployConfig.l2OutputOracleStartingBlockNumber
        )
        assert(
          genesisOutput.outputRoot ===
            hre.deployConfig.l2OutputOracleGenesisL2Output,
          `L2OutputOracle was not initialized with the correct genesis output root`
        )
      }

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
        (
          await (hre as any).ethers.provider.getBalance(OptimismPortal.address)
        ).gt(0)
      )

      // Check L1CrossDomainMessenger was initialized properly.
      try {
        await L1CrossDomainMessenger.xDomainMessageSender()
        assert(false, `L1CrossDomainMessenger was not initialized properly`)
      } catch (err) {
        assert(
          err.message.includes('xDomainMessageSender is not set'),
          `L1CrossDomainMessenger was not initialized properly`
        )
      }
      await assertContractVariable(
        L1CrossDomainMessenger,
        'owner',
        MigrationSystemDictator.address
      )

      // Check L1StandardBridge was initialized properly.
      await assertContractVariable(
        L1StandardBridge,
        'messenger',
        L1CrossDomainMessenger.address
      )
      assert(
        (
          await (hre as any).ethers.provider.getBalance(
            L1StandardBridge.address
          )
        ).eq(0)
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
    5: async () => {
      await assertContractVariable(L1CrossDomainMessenger, 'paused', false)
    },
    6: async () => {
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
    },
  }

  for (let i = 1; i <= 6; i++) {
    if ((await MigrationSystemDictator.currentStep()) === i) {
      if (isLiveDeployer) {
        console.log(`Executing step ${i}...`)
        await MigrationSystemDictator[`step${i}`]()
      } else {
        console.log(`Please execute step ${i}...`)
      }

      await awaitCondition(async () => {
        const step = await MigrationSystemDictator.currentStep()
        return step === i + 1
      })

      // Run post step checks
      await checks[i]()
    } else {
      console.log(`Step ${i} executed`)
    }
  }
}

deployFn.tags = ['MigrationSystemDictatorSteps', 'migration']

export default deployFn
