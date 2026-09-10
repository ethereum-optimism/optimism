import { interopJntV3_0, interopJntV3_1 } from '@eth-optimism/viem/chains'
import { http } from 'viem'

import { getRpcUrl, RPC_ENV_VARS } from '@/config/rpcConfig.js'
import type { Endpoints } from '@/createPonderConfig.js'
import { createPonderConfig } from '@/createPonderConfig.js'

const endpoints: Endpoints = {
  interopJntV3_0: {
    id: interopJntV3_0.id,
    rpc: http(
      getRpcUrl(
        RPC_ENV_VARS.INTEROP_JNT_V3_0,
        interopJntV3_0.rpcUrls.default.http[0],
        'InteropJntV3_0',
      ),
    ),
  },
  interopJntV3_1: {
    id: interopJntV3_1.id,
    rpc: http(
      getRpcUrl(
        RPC_ENV_VARS.INTEROP_JNT_V3_1,
        interopJntV3_1.rpcUrls.default.http[0],
        'InteropJntV3_1',
      ),
    ),
  },
}

export default createPonderConfig(endpoints)
