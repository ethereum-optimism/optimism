import { join } from 'path'
import { fileURLToPath } from 'url'
import { dirname } from 'path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

export const ROOT_PATH = join(__dirname, '..', '..')
export const CHAINS_OUTPUT_PATH = join(ROOT_PATH, 'src/chains.ts')
export const SUPERCHAIN_REGISTRY_PATH = join(
  ROOT_PATH,
  'node_modules/@eth-optimism/superchain-registry'
)
export const OP_GENESIS_BLOCK_PATH = join(
  SUPERCHAIN_REGISTRY_PATH,
  'superchain/configs/mainnet/op.yaml'
)
