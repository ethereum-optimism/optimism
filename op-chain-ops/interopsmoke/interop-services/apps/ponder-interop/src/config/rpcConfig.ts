/**
 * Helper function to get RPC URL with environment variable override support
 * @param envVar - Environment variable name to check
 * @param defaultUrl - Default RPC URL to use if environment variable is not set
 * @param chainName - Human-readable chain name for logging
 * @returns The RPC URL to use
 */
export function getRpcUrl(
  envVar: string,
  defaultUrl: string,
  chainName: string,
): string {
  const customUrl = process.env[envVar]

  if (customUrl) {
    console.log(`📡 Using custom RPC for ${chainName}: ${customUrl}`)
    // Basic URL validation
    if (!customUrl.startsWith('http://') && !customUrl.startsWith('https://')) {
      console.warn(
        `⚠️  Warning: RPC URL for ${chainName} does not start with http:// or https://`,
      )
    }
    return customUrl
  }

  console.log(`📡 Using default RPC for ${chainName}: ${defaultUrl}`)
  return defaultUrl
}

/**
 * Common environment variable names for different networks
 */
export const RPC_ENV_VARS = {
  // Alpha networks
  INTEROP_RC_ALPHA0: 'INTEROP_RC_ALPHA0_RPC_URL',
  INTEROP_RC_ALPHA1: 'INTEROP_RC_ALPHA1_RPC_URL',

  // Devnet networks
  INTEROP_ALPHA0: 'INTEROP_ALPHA0_RPC_URL',
  INTEROP_ALPHA1: 'INTEROP_ALPHA1_RPC_URL',

  // Supersim networks
  SUPERSIM_L2A: 'SUPERSIM_L2A_RPC_URL',
  SUPERSIM_L2B: 'SUPERSIM_L2B_RPC_URL',

  // V0 networks
  INTEROP_V0_0: 'INTEROP_V0_0_RPC_URL',
  INTEROP_V0_1: 'INTEROP_V0_1_RPC_URL',

  // JNT V3 networks
  INTEROP_JNT_V3_0: 'INTEROP_JNT_V3_0_RPC_URL',
  INTEROP_JNT_V3_1: 'INTEROP_JNT_V3_1_RPC_URL',
} as const
