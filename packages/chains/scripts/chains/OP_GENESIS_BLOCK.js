import { readFileSync } from 'fs'
import YAML from 'yaml'
import { OP_GENESIS_BLOCK_PATH } from './paths.js'

/**
 * @type {number}
 */
export const OP_GENESIS_BLOCK = YAML.parse(
  readFileSync(OP_GENESIS_BLOCK_PATH, 'utf8')
).genesis.l2.number
