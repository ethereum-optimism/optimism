import { interopAlpha0, interopAlpha1 } from '@eth-optimism/viem/chains'
import { http } from 'viem'

import type { Endpoints } from '@/createPonderConfig.js'
import { createPonderConfig } from '@/createPonderConfig.js'
import { getRpcUrl, RPC_ENV_VARS } from '@/config/rpcConfig.js'

const endpoints: Endpoints = {
  interopAlpha0: {
    id: interopAlpha0.id,
    rpc: http(getRpcUrl(
      RPC_ENV_VARS.INTEROP_ALPHA0,
      interopAlpha0.rpcUrls.default.http[0],
      'InteropAlpha0'
    )),
  },
  interopAlpha1: {
    id: interopAlpha1.id,
    rpc: http(getRpcUrl(
      RPC_ENV_VARS.INTEROP_ALPHA1,
      interopAlpha1.rpcUrls.default.http[0],
      'InteropAlpha1'
    )),
  },
}

export default createPonderConfig(endpoints)
