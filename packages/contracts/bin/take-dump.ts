/* External Imports */
import * as fs from 'fs'
import * as path from 'path'
import * as mkdirp from 'mkdirp'

/* Internal Imports */
import { makeStateDump } from '../src'
;(async () => {
  const outdir = path.resolve(__dirname, '../build/dumps')
  const outfile = path.join(outdir, 'state-dump.latest.json')
  mkdirp.sync(outdir)

  const dump = await makeStateDump()
  fs.writeFileSync(outfile, JSON.stringify(dump, null, 4))
})()
