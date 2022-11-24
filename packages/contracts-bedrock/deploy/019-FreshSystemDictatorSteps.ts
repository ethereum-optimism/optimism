import assert from 'assert'

import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  assertContractVariable,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const ProxyAdmin = await getContractFromArtifact(hre, 'ProxyAdmin', {
    signerOrProvider: deployer,
  })
  const FreshSystemDictator = await getContractFromArtifact(
    hre,
    'FreshSystemDictator',
    {
      signerOrProvider: deployer,
    }
  )
  let finalOwner = hre.deployConfig.finalSystemOwner
  if (finalOwner === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A proxy admin owner address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      finalOwner = deployer
    } else {
      throw new Error(`must specify the finalSystemOwner on live networks`)
    }
  }
  if (hre.deployConfig.l2OutputOracleGenesisL2Output === ethers.constants.HashZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A genesis L2 output was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
    } else {
      throw new Error(`must specify the finalSystemOwner on live networks`)
    }
  }
  if (hre.network.config.live) {
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

    await FreshSystemDictator.updateL2OutputOracleDynamicConfig({
      l2OutputOracleStartingL2Output:
        hre.deployConfig.l2OutputOracleGenesisL2Output,
      l2OutputOracleStartingBlockNumber:
        hre.deployConfig.l2OutputOracleStartingBlockNumber,
      l2OutputOracleStartingTimestamp: deployL2StartingTimestamp,
    })
  } else {
    console.log(`Please update dynamic oracle config...`)
  }
  if ((await ProxyAdmin.owner()) !== FreshSystemDictator.address) {
    console.log(`Transferring proxy admin ownership to the FreshSystemDictator`)
    await ProxyAdmin.transferOwnership(FreshSystemDictator.address)
  } else {
    console.log(`Proxy admin already owned by the FreshSystemDictator`)
  }

  if ((await FreshSystemDictator.currentStep()) === 1) {
    console.log(`Executing step 1`)
    await FreshSystemDictator.step1()

    // Check L2OutputOracle was initialized properly.
    const L2OutputOracle = await getContractFromArtifact(
      hre,
      'L2OutputOracleProxy',
      {
        iface: 'L2OutputOracle',
      }
    )
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

    // Check BatchInbox was initialized properly.
    const BatchInbox = await getContractFromArtifact(
      hre,
      'BatchInboxProxy',
      {
        iface: 'BatchInbox',
      }
    )
    await assertContractVariable(
      BatchInbox,
      'proposer',
      hre.deployConfig.batchSenderAddress
    )
    await assertContractVariable(
      BatchInbox,
      'owner',
      hre.deployConfig.batchInboxAddress
    )

    // Check OptimismPortal was initialized properly.
    const OptimismPortal = await getContractFromArtifact(
      hre,
      'OptimismPortalProxy',
      {
        iface: 'OptimismPortal',
      }
    )
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

    // Check L1CrossDomainMessenger was initialized properly.
    const L1CrossDomainMessenger = await getContractFromArtifact(
      hre,
      'L1CrossDomainMessengerProxy',
      {
        iface: 'L1CrossDomainMessenger',
      }
    )
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
      finalOwner
    )

    // Check L1StandardBridge was initialized properly.
    const L1StandardBridge = await getContractFromArtifact(
      hre,
      'L1StandardBridgeProxy',
      {
        iface: 'L1StandardBridge',
      }
    )
    await assertContractVariable(
      L1StandardBridge,
      'messenger',
      L1CrossDomainMessenger.address
    )

    // Check OptimismMintableERC20Factory was initialized properly.
    const OptimismMintableERC20Factory = await getContractFromArtifact(
      hre,
      'OptimismMintableERC20FactoryProxy',
      {
        iface: 'OptimismMintableERC20Factory',
      }
    )
    await assertContractVariable(
      OptimismMintableERC20Factory,
      'bridge',
      L1StandardBridge.address
    )

    // Check L1ERC721Bridge was initialized properly.
    const L1ERC721Bridge = await getContractFromArtifact(
      hre,
      'L1ERC721BridgeProxy',
      {
        iface: 'L1ERC721Bridge',
      }
    )
    await assertContractVariable(
      L1ERC721Bridge,
      'messenger',
      L1CrossDomainMessenger.address
    )
  } else {
    console.log(`Step 1 executed`)
  }

  if ((await FreshSystemDictator.currentStep()) === 2) {
    console.log(`Executing step 2`)
    await FreshSystemDictator.step2()

    // Check the ProxyAdmin owner was changed properly.
    await assertContractVariable(
      ProxyAdmin,
      'owner',
      finalOwner
    )
  } else {
    console.log(`Step 2 executed`)
  }
}

deployFn.tags = ['FreshSystemDictatorSteps', 'fresh']

export default deployFn