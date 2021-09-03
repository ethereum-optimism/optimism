/* External Imports */
import * as fs from 'fs'
import * as path from 'path'
import * as mkdirp from 'mkdirp'

/* Internal Imports */
import { makeStateDump } from '../src/make-dump'
;(async () => {
  const outdir = path.resolve(__dirname, '../dist/dumps')
  const outfile = path.join(outdir, 'state-dump.latest.json')
  mkdirp.sync(outdir)

  const dump = await makeStateDump({
    whitelistConfig: {
      owner: process.env.WHITELIST_OWNER,
      allowArbitraryContractDeployment: true,
    },
    gasPriceOracleConfig: {
      owner: process.env.GAS_PRICE_ORACLE_OWNER,
      initialGasPrice: 0,
    },
    l1StandardBridgeAddress: process.env.L1_STANDARD_BRIDGE_ADDRESS,
    l1FeeWalletAddress: process.env.L1_FEE_WALLET_ADDRESS,
    l1CrossDomainMessengerAddress:
      process.env.L1_CROSS_DOMAIN_MESSENGER_ADDRESS,
  })

  fs.writeFileSync(outfile, JSON.stringify(dump, null, 4))
})()
