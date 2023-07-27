/* eslint-disable @typescript-eslint/no-unused-vars */
import { Contract } from 'ethers'
import { hexStringEquals } from '@eth-optimism/core-utils'

import { AddressLike } from '../interfaces'
import { toAddress } from '../utils'
import { StandardBridgeAdapter } from './standard-bridge'

/**
 * Bridge adapter for DAI.
 */
export class DAIBridgeAdapter extends StandardBridgeAdapter {
  public async supportsTokenPair(
    l1Token: AddressLike,
    l2Token: AddressLike
  ): Promise<boolean> {
    // Just need access to this ABI for this one function.
    const l1Bridge = new Contract(
      await this.l1Bridge.getAddress(),
      [
        {
          inputs: [],
          name: 'l1Token' as const,
          outputs: [
            {
              internalType: 'address' as const,
              name: '' as const,
              type: 'address' as const,
            },
          ],
          stateMutability: 'view' as const,
          type: 'function' as const,
        },
        {
          inputs: [],
          name: 'l2Token' as const,
          outputs: [
            {
              internalType: 'address' as const,
              name: '' as const,
              type: 'address' as const,
            },
          ],
          stateMutability: 'view' as const,
          type: 'function' as const,
        },
      ],
      this.messenger.l1Provider
    )

    const allowedL1Token = await l1Bridge.l1Token()
    if (!hexStringEquals(allowedL1Token, await toAddress(l1Token))) {
      return false
    }

    const allowedL2Token = await l1Bridge.l2Token()
    if (!hexStringEquals(allowedL2Token, await toAddress(l2Token))) {
      return false
    }

    return true
  }
}
