import { supersimL2A, supersimL2B } from '@eth-optimism/viem/chains'
import { http } from 'viem'

import type { Endpoints } from '@/createPonderConfig.js'
import { createPonderConfig } from '@/createPonderConfig.js'
import { getRpcUrl, RPC_ENV_VARS } from '@/config/rpcConfig.js'

const endpoints: Endpoints = {
  supersimL2A: {
    id: supersimL2A.id,
    rpc: http(getRpcUrl(
      RPC_ENV_VARS.SUPERSIM_L2A,
      supersimL2A.rpcUrls.default.http[0],
      'SupersimL2A'
    )),
    disableCache: true,
  },
  supersimL2B: {
    id: supersimL2B.id,
    rpc: http(getRpcUrl(
      RPC_ENV_VARS.SUPERSIM_L2B,
      supersimL2B.rpcUrls.default.http[0],
      'SupersimL2B'
    )),
    disableCache: true,
  },
}

export default createPonderConfig(endpoints)
