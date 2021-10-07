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

/**
 * Downloads a specific solc version.
 *
 * @param version Solc version to download.
 * @param ovm If true, downloads from the OVM repository.
 */
export const downloadSolc = async (version: string, ovm?: boolean) => {
  // TODO: why is this one missing?
  if (version === 'v0.5.16-alpha.7') {
    return
  }

  console.error(`Downloading ${version} ${ovm ? 'ovm' : 'solidity'}`)

  // File is the location where we'll put the downloaded compiler.
  let file: string
  // Remote is the URL we'll query if the file doesn't already exist.
  let remote: string

  // Exact file/remote will depend on if downloading OVM or EVM compiler.
  if (ovm) {
    file = `${path.join(LOCAL_SOLC_DIR, version)}.js`
    remote = OVM_BUILD_PATH(version)
  } else {
    const res = await fetch(EMSCRIPTEN_BUILD_LIST)
    const data: any = await res.json()
    const list = data.builds

    // Make sure the target version actually exists
    let target: any
    for (const entry of list) {
      const longVersion = `v${entry.longVersion}`
      if (version === longVersion) {
        target = entry
      }
    }

    // Error out if the given version can't be found
    if (!target) {
      throw new Error(`Cannot find compiler version ${version}`)
    }

    file = path.join(LOCAL_SOLC_DIR, target.path)
    remote = `${EMSCRIPTEN_BUILD_PATH}/${target.path}`
  }

  try {
    // Check to see if we already have the file
    await access(file, fs.constants.F_OK)
    console.error(`${version} already downloaded`)
  } catch (e) {
    // If we don't have the file, download it
    const res = await fetch(remote)
    const bin = await res.text()
    fs.writeFileSync(file, bin)
  }
}

/**
 * Downloads all required solc versions, if not already downloaded.
 */
export const downloadAllSolcVersions = async () => {
  try {
    await mkdir(LOCAL_SOLC_DIR)
  } catch (e) {
    // directory already exists
  }

  // Keys are OVM versions.
  await Promise.all(
    Object.keys(COMPILER_VERSIONS_TO_SOLC).map(async (version) => {
      await downloadSolc(version, true)
    })
  )

  // Values are EVM versions.
  await Promise.all(
    Object.values(COMPILER_VERSIONS_TO_SOLC).map(async (version) => {
      await downloadSolc(version)
    })
  )
}
