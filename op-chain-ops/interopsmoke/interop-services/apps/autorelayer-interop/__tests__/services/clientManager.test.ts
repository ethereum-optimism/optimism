import type { PublicClient, WalletClient } from 'viem'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ClientManager } from '@/services/clientManager.js'

describe('ClientManager', () => {
  let mockPublicClient1: PublicClient
  let mockPublicClient2: PublicClient
  let mockWalletClient1: WalletClient
  let mockWalletClient2: WalletClient
  let clientManager: ClientManager

  beforeEach(() => {
    mockPublicClient1 = { uid: 'public-1' } as unknown as PublicClient
    mockPublicClient2 = { uid: 'public-2' } as unknown as PublicClient
    mockWalletClient1 = {
      account: undefined,
      getAddresses: vi.fn(),
    } as unknown as WalletClient
    mockWalletClient2 = {
      account: undefined,
      getAddresses: vi.fn(),
    } as unknown as WalletClient

    clientManager = new ClientManager(
      { 1: mockPublicClient1, 2: mockPublicClient2 },
      { 1: [mockWalletClient1], 2: [mockWalletClient2] },
    )
  })

  describe('getPublicClient', () => {
    it('returns the correct client for a known chainId', () => {
      expect(clientManager.getPublicClient(1)).toBe(mockPublicClient1)
      expect(clientManager.getPublicClient(2)).toBe(mockPublicClient2)
    })

    it('returns undefined for an unknown chainId', () => {
      expect(clientManager.getPublicClient(999)).toBeUndefined()
    })
  })

  describe('getWalletClient', () => {
    it('returns the correct wallet client for a known chainId', () => {
      expect(clientManager.getWalletClient(1)).toBe(mockWalletClient1)
      expect(clientManager.getWalletClient(2)).toBe(mockWalletClient2)
    })

    it('returns undefined for an unknown chainId', () => {
      expect(clientManager.getWalletClient(999)).toBeUndefined()
    })
  })

  describe('resolveAccount', () => {
    it('returns walletClient.account when it exists', async () => {
      const existingAccount = {
        address: '0x1234567890abcdef1234567890abcdef12345678' as const,
        type: 'local' as const,
      }
      const walletClient = {
        account: existingAccount,
        getAddresses: vi.fn(),
      } as unknown as WalletClient

      const result = await clientManager.resolveAccount(walletClient)
      expect(result).toBe(existingAccount)
      expect(walletClient.getAddresses).not.toHaveBeenCalled()
    })

    it('calls getAddresses and returns a json-rpc account when walletClient.account is undefined', async () => {
      const addresses = [
        '0xaaaa567890abcdef1234567890abcdef12345678',
        '0xbbbb567890abcdef1234567890abcdef12345678',
        '0xcccc567890abcdef1234567890abcdef12345678',
      ] as const

      const walletClient = {
        account: undefined,
        getAddresses: vi.fn().mockResolvedValue(addresses),
      } as unknown as WalletClient

      // Mock Math.random to return 0 for deterministic output
      vi.spyOn(Math, 'random').mockReturnValue(0)

      const result = await clientManager.resolveAccount(walletClient)
      expect(result).toEqual({
        address: '0xaaaa567890abcdef1234567890abcdef12345678',
        type: 'json-rpc',
      })
      expect(walletClient.getAddresses).toHaveBeenCalledOnce()

      vi.restoreAllMocks()
    })

    it('returns undefined when getAddresses returns empty array', async () => {
      const walletClient = {
        account: undefined,
        getAddresses: vi.fn().mockResolvedValue([]),
      } as unknown as WalletClient

      const result = await clientManager.resolveAccount(walletClient)
      expect(result).toBeUndefined()
    })
  })
})
