/* External Imports */
import * as fs from 'fs'
import * as path from 'path'
import * as mkdirp from 'mkdirp'

/* Internal Imports */
import { makeL2GenesisFile } from '../src/make-genesis'
;(async () => {
  const outdir = path.resolve(__dirname, '../dist/dumps')
  const outfile = path.join(outdir, 'state-dump.latest.json')
  mkdirp.sync(outdir)

  // Basic warning so users know that the whitelist will be disabled if the owner is the zero address.
  if (process.env.WHITELIST_OWNER === '0x' + '00'.repeat(20)) {
    console.log(
      'WARNING: whitelist owner is address(0), whitelist will be disabled'
    )
  }

  const genesis = await makeL2GenesisFile({
    whitelistOwner: process.env.WHITELIST_OWNER,
    gasPriceOracleOwner: process.env.GAS_PRICE_ORACLE_OWNER,
    initialGasPrice: 0,
    l2BlockGasLimit: parseInt(process.env.L2_BLOCK_GAS_LIMIT, 10),
    l2ChainId: parseInt(process.env.L2_CHAIN_ID, 10),
    blockSignerAddress: process.env.BLOCK_SIGNER_ADDRESS,
    l1StandardBridgeAddress: process.env.L1_STANDARD_BRIDGE_ADDRESS,
    l1FeeWalletAddress: process.env.L1_FEE_WALLET_ADDRESS,
    l1CrossDomainMessengerAddress:
      process.env.L1_CROSS_DOMAIN_MESSENGER_ADDRESS,
  })

  fs.writeFileSync(outfile, JSON.stringify(genesis, null, 4))
})()
