/* Imports: External */
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

import { predeploys } from '../src/predeploys'
import { getContractDefinition } from '../src/contract-defs'

task('set-l2-gasprice')
  .addOptionalParam(
    'l2GasPrice',
    'Gas Price to set on L2',
    undefined,
    types.int
  )
  .addOptionalParam('transactionGasPrice', 'tx.gasPrice', undefined, types.int)
  .addOptionalParam(
    'overhead',
    'amortized additional gas used by each batch that users must pay for',
    undefined,
    types.int
  )
  .addOptionalParam(
    'scalar',
    'amount to scale up the gas to charge',
    undefined,
    types.int
  )
  .addOptionalParam(
    'contractsRpcUrl',
    'Sequencer HTTP Endpoint',
    process.env.CONTRACTS_RPC_URL,
    types.string
  )
  .addOptionalParam(
    'contractsDeployerKey',
    'Private Key',
    process.env.CONTRACTS_DEPLOYER_KEY,
    types.string
  )
  .setAction(async (args) => {
    const provider = new ethers.providers.JsonRpcProvider(args.contractsRpcUrl)
    const signer = new ethers.Wallet(args.contractsDeployerKey).connect(
      provider
    )

    const GasPriceOracleArtifact = getContractDefinition('OVM_GasPriceOracle')

    const GasPriceOracle = new ethers.Contract(
      predeploys.OVM_GasPriceOracle,
      GasPriceOracleArtifact.abi,
      signer
    )

    const addr = await signer.getAddress()
    console.log(`Using signer ${addr}`)
    const owner = await GasPriceOracle.callStatic.owner()
    if (owner !== addr) {
      throw new Error(`Incorrect key. Owner ${owner}, Signer ${addr}`)
    }

    // List the current values
    const gasPrice = await GasPriceOracle.callStatic.gasPrice()
    const scalar = await GasPriceOracle.callStatic.scalar()
    const overhead = await GasPriceOracle.callStatic.overhead()

    console.log('Current values:')
    console.log(`Gas Price: ${gasPrice.toString()}`)
    console.log(`Scalar: ${scalar.toString()}`)
    console.log(`Overhead: ${overhead.toString()}`)

    if (args.l2GasPrice !== undefined) {
      console.log(`Setting gas price to ${args.l2GasPrice}`)
      const tx = await GasPriceOracle.connect(signer).setGasPrice(
        args.l2GasPrice,
        { gasPrice: args.transactionGasPrice }
      )

      const receipt = await tx.wait()
      console.log(`Success - ${receipt.transactionHash}`)
    }

    if (args.scalar !== undefined) {
      console.log(`Setting scalar to ${args.scalar}`)
      const tx = await GasPriceOracle.connect(signer).setScalar(args.scalar, {
        gasPrice: args.transactionGasPrice,
      })

      const receipt = await tx.wait()
      console.log(`Success - ${receipt.transactionHash}`)
    }

    if (args.overhead !== undefined) {
      console.log(`Setting overhead to ${args.overhead}`)
      const tx = await GasPriceOracle.connect(signer).setOverhead(
        args.overhead,
        { gasPrice: args.transactionGasPrice }
      )

      const receipt = await tx.wait()
      console.log(`Success - ${receipt.transactionHash}`)
    }
  })
