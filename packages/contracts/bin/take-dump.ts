/* External Imports */
import * as fs from 'fs'
import * as path from 'path'
import * as mkdirp from 'mkdirp'

const ensure = (value, key) => {
  if (typeof value === 'undefined' || value === null || Number.isNaN(value)) {
    throw new Error(`${key} is undefined, null or NaN`)
  }
}

/* Internal Imports */
import { makeL2GenesisFile } from '../src/make-genesis'
;(async () => {
  const outdir = path.resolve(__dirname, '../dist/dumps')
  const outfile = path.join(outdir, 'state-dump.latest.json')
  mkdirp.sync(outdir)

  const env = process.env

  // An account that represents the owner of the whitelist
  const whitelistOwner = env.WHITELIST_OWNER
  // The gas price oracle owner, can update values is GasPriceOracle L2 predeploy
  const gasPriceOracleOwner = env.GAS_PRICE_ORACLE_OWNER
  // The initial overhead value for the GasPriceOracle
  const gasPriceOracleOverhead = parseInt(
    env.GAS_PRICE_ORACLE_OVERHEAD || '2750',
    10
  )
  // The initial scalar value for the GasPriceOracle. The actual
  // scalar is scaled downwards by the number of decimals
  const gasPriceOracleScalar = parseInt(
    env.GAS_PRICE_ORACLE_SCALAR || '1500000',
    10
  )
  // The initial decimals that scale down the scalar in the GasPriceOracle
  const gasPriceOracleDecimals = parseInt(
    env.GAS_PRICE_ORACLE_DECIMALS || '6',
    10
  )
  // The initial L1 base fee in the GasPriceOracle. This determines how
  // expensive the L1 portion of the transaction fee is.
  const gasPriceOracleL1BaseFee = parseInt(
    env.GAS_PRICE_ORACLE_L1_BASE_FEE || '1',
    10
  )
  // The initial L2 gas price set in the GasPriceOracle
  const gasPriceOracleGasPrice = parseInt(
    env.GAS_PRICE_ORACLE_GAS_PRICE || '1',
    10
  )
  // The L2 block gas limit, used in the L2 block headers as well to limit
  // the amount of execution for a single block.
  const l2BlockGasLimit = parseInt(env.L2_BLOCK_GAS_LIMIT, 10)
  // The L2 chain id, added to the chain config
  const l2ChainId = parseInt(env.L2_CHAIN_ID, 10)
  // The block signer address, added to the block extradata for clique consensus
  const blockSignerAddress = env.BLOCK_SIGNER_ADDRESS
  // The L1 standard bridge address for cross domain messaging
  const l1StandardBridgeAddress = env.L1_STANDARD_BRIDGE_ADDRESS
  // The L1 fee wallet address, used to restrict the account that fees on L2 can
  // be withdrawn to on L1
  const l1FeeWalletAddress = env.L1_FEE_WALLET_ADDRESS
  // The L1 cross domain messenger address, used for cross domain messaging
  const l1CrossDomainMessengerAddress = env.L1_CROSS_DOMAIN_MESSENGER_ADDRESS
  // The block height at which the berlin hardfork activates
  const berlinBlock = parseInt(env.BERLIN_BLOCK, 10) || 0

  ensure(whitelistOwner, 'WHITELIST_OWNER')
  ensure(gasPriceOracleOwner, 'GAS_PRICE_ORACLE_OWNER')
  ensure(l2BlockGasLimit, 'L2_BLOCK_GAS_LIMIT')
  ensure(l2ChainId, 'L2_CHAIN_ID')
  ensure(blockSignerAddress, 'BLOCK_SIGNER_ADDRESS')
  ensure(l1StandardBridgeAddress, 'L1_STANDARD_BRIDGE_ADDRESS')
  ensure(l1FeeWalletAddress, 'L1_FEE_WALLET_ADDRESS')
  ensure(l1CrossDomainMessengerAddress, 'L1_CROSS_DOMAIN_MESSENGER_ADDRESS')
  ensure(berlinBlock, 'BERLIN_BLOCK')

  // Basic warning so users know that the whitelist will be disabled if the owner is the zero address.
  if (env.WHITELIST_OWNER === '0x' + '00'.repeat(20)) {
    console.log(
      'WARNING: whitelist owner is address(0), whitelist will be disabled'
    )
  }

  const genesis = await makeL2GenesisFile({
    whitelistOwner,
    gasPriceOracleOwner,
    gasPriceOracleOverhead,
    gasPriceOracleScalar,
    gasPriceOracleL1BaseFee,
    gasPriceOracleGasPrice,
    gasPriceOracleDecimals,
    l2BlockGasLimit,
    l2ChainId,
    blockSignerAddress,
    l1StandardBridgeAddress,
    l1FeeWalletAddress,
    l1CrossDomainMessengerAddress,
    berlinBlock,
  })

  fs.writeFileSync(outfile, JSON.stringify(genesis, null, 4))
})()
