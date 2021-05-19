/* External Imports */
import * as fs from 'fs'
import * as path from 'path'
import * as mkdirp from 'mkdirp'

const env = process.env
const CHAIN_ID = env.CHAIN_ID || '420'

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
  }

  const dump = await makeStateDump(config as RollupDeployConfig)
  fs.writeFileSync(outfile, JSON.stringify(dump, null, 4))
})()
