import { interopV0_0, interopV0_1 } from '@eth-optimism/viem/chains'
import { http } from 'viem'

import { getRpcUrl, RPC_ENV_VARS } from '@/config/rpcConfig.js'
import type { Endpoints } from '@/createPonderConfig.js'
import { createPonderConfig } from '@/createPonderConfig.js'

const endpoints: Endpoints = {
  interopV0_0: {
    id: interopV0_0.id,
    rpc: http(
      getRpcUrl(
        RPC_ENV_VARS.INTEROP_V0_0,
        interopV0_0.rpcUrls.default.http[0],
        'InteropV0_0',
      ),
    ),
  },
  interopV0_1: {
    id: interopV0_1.id,
    rpc: http(
      getRpcUrl(
        RPC_ENV_VARS.INTEROP_V0_1,
        interopV0_1.rpcUrls.default.http[0],
        'InteropV0_1',
      ),
    ),
  },
}

export default createPonderConfig(endpoints)
