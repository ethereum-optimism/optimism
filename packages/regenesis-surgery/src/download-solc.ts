/**
 * Optimism PBC
 */

/* eslint @typescript-eslint/no-var-requires: "off" */

const axios = require('axios')
import { access } from 'fs/promises'
import path from 'path'
import * as fs from 'fs'

import {
  LOCAL_SOLC_DIR,
  EMSCRIPTEN_BUILD_LIST,
  EMSCRIPTEN_BUILD_PATH,
} from './constants'

const OVM_BUILD_PATH = (version: string) => {
  return `https://raw.githubusercontent.com/ethereum-optimism/solc-bin/9455107699d2f7ad9b09e1005c7c07f4b5dd6857/bin/soljson-${version}.js`
}

export const downloadSolc = async (version: string, ovm?: boolean) => {
  // TODO: why is this one missing?
  if (version === 'v0.5.16-alpha.7') {
    return
  }
  console.error(`Downloading ${version} ${ovm ? 'ovm' : 'solidity'}`)

  let target
  let file
  if (!ovm) {
    const res = await axios.get(EMSCRIPTEN_BUILD_LIST)
    const list = await res.data.builds
    for (const entry of list) {
      const longVersion = `v${entry.longVersion}`
      if (version === longVersion) {
        target = entry
      }
    }
    if (!target) {
      throw new Error(`Cannot find compiler version ${version}`)
    }
    file = path.join(LOCAL_SOLC_DIR, target.path)
  } else {
    file = `${path.join(LOCAL_SOLC_DIR, version)}.js`
  }

  try {
    await access(file, fs.constants.F_OK)
    console.error(`${version} already downloaded`)
  } catch (e) {
    let bin
    if (ovm) {
      bin = await axios.get(OVM_BUILD_PATH(version))
    } else {
      bin = await axios.get(`${EMSCRIPTEN_BUILD_PATH}/${target.path}`)
    }
    fs.writeFileSync(file, bin.data)
  }
}
