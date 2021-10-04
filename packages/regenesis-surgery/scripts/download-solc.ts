/* eslint @typescript-eslint/no-var-requires: "off" */
import { access, mkdir } from 'fs/promises'
import fetch from 'node-fetch'
import path from 'path'
import fs from 'fs'

import {
  COMPILER_VERSIONS_TO_SOLC,
  EMSCRIPTEN_BUILD_LIST,
  EMSCRIPTEN_BUILD_PATH,
  LOCAL_SOLC_DIR,
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
    const res = await fetch(EMSCRIPTEN_BUILD_LIST)
    const data: any = await res.json()
    const list = data.builds
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
    let res
    if (ovm) {
      res = await fetch(OVM_BUILD_PATH(version))
    } else {
      res = await fetch(`${EMSCRIPTEN_BUILD_PATH}/${target.path}`)
    }
    const bin = await res.text()
    fs.writeFileSync(file, bin)
  }
}

export const downloadAllSolcVersions = async () => {
  try {
    await mkdir(LOCAL_SOLC_DIR)
  } catch (e) {
    // directory already exists
  }

  for (const version of Object.keys(COMPILER_VERSIONS_TO_SOLC)) {
    await downloadSolc(version, true) // using ovm
  }
  for (const version of Object.values(COMPILER_VERSIONS_TO_SOLC)) {
    await downloadSolc(version)
  }
}
