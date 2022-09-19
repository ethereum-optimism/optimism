/**
 * Provider Utilities
 */

import {
  Provider,
  StaticJsonRpcProvider,
  FallbackProvider as EthersFallbackProvider,
} from '@ethersproject/providers'
import { ConnectionInfo } from '@ethersproject/web'

export interface HttpHeaders {
  [key: string]: string
}

// Copied from @ethersproject/providers since it is not
// currently exported
export interface FallbackProviderConfig {
  // The Provider
  provider: Provider
  // The priority to favour this Provider; higher values are used first
  priority?: number
  // Timeout before also triggering the next provider; this does not stop
  // this provider and if its result comes back before a quorum is reached
  // it will be incorporated into the vote
  // - lower values will cause more network traffic but may result in a
  //   faster retult.
  stallTimeout?: number
  // How much this provider contributes to the quorum; sometimes a specific
  // provider may be more reliable or trustworthy than others, but usually
  // this should be left as the default
  weight?: number
}

export const FallbackProvider = (
  config: string | FallbackProviderConfig[],
  headers?: HttpHeaders
) => {
  const configs = []
  // Handle the case of a string of comma delimited urls
  if (typeof config === 'string') {
    const urls = config.split(',')
    for (const [i, url] of urls.entries()) {
      const connectionInfo: ConnectionInfo = { url }
      if (typeof headers === 'object') {
        connectionInfo.headers = headers
      }
      configs.push({
        priority: i,
        provider: new StaticJsonRpcProvider(connectionInfo),
      })
    }
    return new EthersFallbackProvider(configs)
  }

  return new EthersFallbackProvider(config)
}
