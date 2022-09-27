import * as path from 'path'
import * as fs from 'fs'
import { exec } from 'child_process'
import { promisify } from 'util'

import * as mkdirp from 'mkdirp'
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import { remove0x } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'

import { predeploys } from '../src/predeploys'
import { getContractFromArtifact } from '../src/deploy-utils'
import { names } from '../src/address-names'

task('take-dump').setAction(async ({}, hre) => {
  /* eslint-disable @typescript-eslint/no-var-requires */

  // Needs to be imported here or hardhat will throw a fit about hardhat being imported from
  // within the configuration file.
  const {
    computeStorageSlots,
    getStorageLayout,
  } = require('@defi-wonderland/smock/dist/src/utils')

  // Needs to be imported here because the artifacts can only be generated after the contracts have
  // been compiled, but compiling the contracts will import the config file which, as a result,
  // will import this file.
  const { getContractArtifact } = require('../src/contract-artifacts')

  /* eslint-enable @typescript-eslint/no-var-requires */

  // Basic warning so users know that the whitelist will be disabled if the owner is the zero address.
  if (
    hre.deployConfig.ovmWhitelistOwner === undefined ||
    hre.deployConfig.ovmWhitelistOwner === ethers.constants.AddressZero
  ) {
    console.log(
      'WARNING: whitelist owner is undefined or address(0), whitelist will be disabled'
    )
  }

  const variables = {
    OVM_DeployerWhitelist: {
      owner: hre.deployConfig.ovmWhitelistOwner,
    },
    OVM_GasPriceOracle: {
      _owner: hre.deployConfig.ovmGasPriceOracleOwner,
      gasPrice: hre.deployConfig.gasPriceOracleL2GasPrice,
      l1BaseFee: hre.deployConfig.gasPriceOracleL1BaseFee,
      overhead: hre.deployConfig.gasPriceOracleOverhead,
      scalar: hre.deployConfig.gasPriceOracleScalar,
      decimals: hre.deployConfig.gasPriceOracleDecimals,
    },
    L2StandardBridge: {
      l1TokenBridge: (
        await getContractFromArtifact(
          hre,
          names.managed.contracts.Proxy__OVM_L1StandardBridge
        )
      ).address,
      messenger: predeploys.L2CrossDomainMessenger,
    },
    OVM_SequencerFeeVault: {
      l1FeeWallet: hre.deployConfig.ovmFeeWalletAddress,
    },
    OVM_ETH: {
      l2Bridge: predeploys.L2StandardBridge,
      l1Token: ethers.constants.AddressZero,
      _name: 'Ether',
      _symbol: 'ETH',
    },
    L2CrossDomainMessenger: {
      // We default the xDomainMsgSender to this value to save gas.
      // See usage of this default in the L2CrossDomainMessenger contract.
      xDomainMsgSender: '0x000000000000000000000000000000000000dEaD',
      l1CrossDomainMessenger: (
        await getContractFromArtifact(
          hre,
          names.managed.contracts.Proxy__OVM_L1CrossDomainMessenger
        )
      ).address,
      // Set the messageNonce to a high value to avoid overwriting old sent messages.
      messageNonce: 100000,
    },
    WETH9: {
      name: 'Wrapped Ether',
      symbol: 'WETH',
      decimals: 18,
    },
  }

  const dump = {}
  for (const predeployName of Object.keys(predeploys)) {
    const predeployAddress = predeploys[predeployName]
    dump[predeployAddress] = {
      balance: '00',
      storage: {},
    }

    if (predeployName === 'OVM_L1BlockNumber') {
      // OVM_L1BlockNumber is a special case where we just inject a specific bytecode string.
      // We do this because it uses the custom L1BLOCKNUMBER opcode (0x4B) which cannot be
      // directly used in Solidity (yet). This bytecode string simply executes the 0x4B opcode
      // and returns the address given by that opcode.
      dump[predeployAddress].code = '0x4B60005260206000F3'
    } else {
      const artifact = getContractArtifact(predeployName)
      dump[predeployAddress].code = artifact.deployedBytecode
    }

    // Compute and set the required storage slots for each contract that needs it.
    if (predeployName in variables) {
      const storageLayout = await getStorageLayout(predeployName)
      const slots = computeStorageSlots(storageLayout, variables[predeployName])
      for (const slot of slots) {
        dump[predeployAddress].storage[slot.key] = slot.val
      }
    }
  }

  // Grab the commit hash so we can stick it in the genesis file.
  let commit: string
  try {
    const { stdout } = await promisify(exec)('git rev-parse HEAD')
    commit = stdout.replace('\n', '')
  } catch {
    console.log('unable to get commit hash, using empty hash instead')
    commit = '0000000000000000000000000000000000000000'
  }

  const genesis = {
    commit,
    config: {
      chainId: hre.deployConfig.l2ChainId,
      homesteadBlock: 0,
      eip150Block: 0,
      eip155Block: 0,
      eip158Block: 0,
      byzantiumBlock: 0,
      constantinopleBlock: 0,
      petersburgBlock: 0,
      istanbulBlock: 0,
      muirGlacierBlock: 0,
      berlinBlock: hre.deployConfig.hfBerlinBlock,
      clique: {
        period: 0,
        epoch: 30000,
      },
    },
    difficulty: '1',
    gasLimit: hre.deployConfig.l2BlockGasLimit.toString(10),
    extradata:
      '0x' +
      '00'.repeat(32) +
      remove0x(hre.deployConfig.ovmBlockSignerAddress) +
      '00'.repeat(65),
    alloc: dump,
  }

  // Make sure the output location exists
  const outdir = path.resolve(__dirname, '../genesis')
  const outfile = path.join(outdir, `${hre.network.name}.json`)
  mkdirp.sync(outdir)

  // Write the genesis file
  fs.writeFileSync(outfile, JSON.stringify(genesis, null, 4))
})
