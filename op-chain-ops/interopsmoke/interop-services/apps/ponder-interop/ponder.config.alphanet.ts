import { interopRcAlpha0, interopRcAlpha1 } from '@eth-optimism/viem/chains'
import { http } from 'viem'

import type { Endpoints } from '@/createPonderConfig.js'
import { createPonderConfig } from '@/createPonderConfig.js'
import { getRpcUrl, RPC_ENV_VARS } from '@/config/rpcConfig.js'

const endpoints: Endpoints = {
  interopRcAlpha0: {
    id: interopRcAlpha0.id,
    rpc: http(getRpcUrl(
      RPC_ENV_VARS.INTEROP_RC_ALPHA0,
      interopRcAlpha0.rpcUrls.default.http[0],
      'InteropRcAlpha0'
    )),
  },
  interopRcAlpha1: {
    id: interopRcAlpha1.id,
    rpc: http(getRpcUrl(
      RPC_ENV_VARS.INTEROP_RC_ALPHA1,
      interopRcAlpha1.rpcUrls.default.http[0],
      'InteropRcAlpha1'
    )),
  },
}

export default createPonderConfig(endpoints)
