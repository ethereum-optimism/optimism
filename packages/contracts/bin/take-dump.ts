/* External Imports */
import * as fs from 'fs'
import * as path from 'path'
import * as mkdirp from 'mkdirp'

const env = process.env
const CHAIN_ID = env.CHAIN_ID || '420'

// Defaults to 0xFF....FF = *no upgrades*
const L2_CHUG_SPLASH_DEPLOYER_OWNER = env.L2_CHUG_SPLASH_DEPLOYER_OWNER || '0x' + 'FF'.repeat(20)
const GAS_PRICE_ORACLE_OWNER = env.GAS_PRICE_ORACLE_OWNER || '0x' + 'FF'.repeat(20)

/* Internal Imports */
import { makeStateDump } from '../src/state-dump/make-dump'
import { RollupDeployConfig } from '../src/contract-deployment'
;(async () => {
  const outdir = path.resolve(__dirname, '../dist/dumps')
  const outfile = path.join(outdir, 'state-dump.latest.json')
  mkdirp.sync(outdir)

  const config = {
    ovmGlobalContext: {
      ovmCHAINID: parseInt(CHAIN_ID, 10),
    },
    l2ChugSplashDeployerOwner: L2_CHUG_SPLASH_DEPLOYER_OWNER,
    gasPriceOracleOwner: GAS_PRICE_ORACLE_OWNER
  }

  const dump = await makeStateDump(config as RollupDeployConfig)
  fs.writeFileSync(outfile, JSON.stringify(dump, null, 4))
})()
