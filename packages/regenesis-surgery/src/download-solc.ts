/**
 * Optimism PBC
 */

import axios from 'axios'
import { access } from 'fs/promises'
import path from 'path'
import * as fs from 'fs'

import {
  LOCAL_SOLC_DIR,
  EMSCRIPTEN_BUILD_LIST,
  EMSCRIPTEN_BUILD_PATH,
} from './constants'

export const downloadSolc = async (version: string) => {
  console.log(`Downloading ${version}`)
  const res = await axios.get(EMSCRIPTEN_BUILD_LIST)
  const list = await res.data.builds

  let target
  for (const entry of list) {
    const longVersion = `v${entry.longVersion}`
    if (version === longVersion) {
      target = entry
    }
  }
  if (!target) {
    throw new Error(`Cannot find compiler version ${version}`)
  }

  const file = path.join(LOCAL_SOLC_DIR, target.path)

  try {
    await access(file, fs.constants.F_OK)
    console.log(`${target.path} already downloaded`)
  } catch (e) {
    console.log(`Downloading ${target.path}`)
    const bin = await axios.get(`${EMSCRIPTEN_BUILD_PATH}/${target.path}`)
    fs.writeFileSync(file, bin.data)
  }
}
