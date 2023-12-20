import { readFileSync } from 'fs'
import YAML from 'yaml'

/**
 * @type {number}
 */
export const OP_GENESIS_BLOCK = YAML.parse(readFileSync('../../node_modules/@eth-optimism/superchain-registry/superchain/configs/mainnet/op.yaml', 'utf8')).genesis.l2.number

