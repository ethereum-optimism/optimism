/* Imports: External */
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { HardhatNetworkProvider } from 'hardhat/internal/hardhat-network/provider/provider'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'

/**
 * Finds the "base" Ethereum provider of the current hardhat environment.
 *
 * Basically, hardhat uses a system of nested providers where each provider wraps the next and
 * "provides" some extra features. When you're running on top of the "hardhat evm" the bottom of
 * this series of providers is the "HardhatNetworkProvider":
 * https://github.com/nomiclabs/hardhat/blob/master/packages/hardhat-core/src/internal/hardhat-network/provider/provider.ts
 * This object has direct access to the node (provider._node), which in turn has direct access to
 * the ethereumjs-vm instance (provider._node._vm). So it's quite useful to be able to find this
 * object reliably!
 * @param hre hardhat runtime environment to pull the base provider from.
 * @return base hardhat network provider
 */
export const findBaseHardhatProvider = (
  runtime: HardhatRuntimeEnvironment
): HardhatNetworkProvider => {
  // This function is pretty approximate. Haven't spent enough time figuring out if there's a more
  // reliable way to get the base provider. I can imagine a future in which there's some circular
  // references and this function ends up looping. So I'll just preempt this by capping the maximum
  // search depth.
  const maxLoopIterations = 1024
  let currentLoopIterations = 0

  // Search by looking for the internal "_wrapped" variable. Base provider doesn't have this
  // property (at least for now!).
  let provider = runtime.network.provider
  while ((provider as any)._wrapped !== undefined) {
    provider = (provider as any)._wrapped

    // Just throw if we ever end up in (what seems to be) an infinite loop.
    currentLoopIterations += 1
    if (currentLoopIterations > maxLoopIterations) {
      throw new Error(
        `[smock]: unable to find base hardhat provider. are you sure you're running locally?`
      )
    }
  }

  // TODO: Figure out a reliable way to do a type check here. Source for inspiration:
  // https://github.com/nomiclabs/hardhat/blob/master/packages/hardhat-core/src/internal/hardhat-network/provider/provider.ts
  return provider as any
}

/**
 * Converts a string into the fancy new address thing that ethereumjs-vm v5 expects while also
 * maintaining backwards compatibility with ethereumjs-vm v4.
 * @param address String address to convert into the fancy new address type.
 * @returns Fancified address.
 */
export const toFancyAddress = (address: string): any => {
  const fancyAddress = fromHexString(address)
  ;(fancyAddress as any).buf = fromHexString(address)
  ;(fancyAddress as any).toString = (encoding?: any) => {
    if (encoding === undefined) {
      return address.toLowerCase()
    } else {
      return fromHexString(address).toString(encoding)
    }
  }
  return fancyAddress
}

/**
 * Same as toFancyAddress but in the opposite direction.
 * @param fancyAddress Fancy address to turn into a string.
 * @returns Way more boring address.
 */
export const fromFancyAddress = (fancyAddress: any): string => {
  if (fancyAddress.buf) {
    return toHexString(fancyAddress.buf)
  } else {
    return toHexString(fancyAddress)
  }
}
