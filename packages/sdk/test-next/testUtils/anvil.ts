import { createAnvil, CreateAnvilOptions } from '@viem/anvil'
import {
  Chain,
  createPublicClient,
  createTestClient,
  createWalletClient,
  http,
} from 'viem'
import { privateKeyToAccount } from 'viem/accounts'

// As a best practice we want to always require a fork block number
// This helps make tests more deterministic and less prone to breakage in future
type ChainOptions = CreateAnvilOptions & Required<Pick<CreateAnvilOptions, 'forkBlockNumber'>> & { chain: Chain }

type TestUtilOptions = {
  l1: ChainOptions,
  l2: ChainOptions
}

/**
 * Anvil accounts
 * (0) "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266" (10000 ETH)
 * (1) "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" (10000 ETH)
 * (2) "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC" (10000 ETH)
 * (3) "0x90F79bf6EB2c4f870365E785982E1f101E93b906" (10000 ETH)
 * (4) "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65" (10000 ETH)
 * (5) "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc" (10000 ETH)
 * (6) "0x976EA74026E726554dB657fA54763abd0C3a0aa9" (10000 ETH)
 * (7) "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955" (10000 ETH)
 * (8) "0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f" (10000 ETH)
 * (9) "0xa0Ee7A142d267C1f36714E4a8F75612F20a79720" (10000 ETH)
 * 
 * Private Keys
 * ==================
 * 
 * (0) 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
 * (1) 0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d
 * (2) 0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a
 * (3) 0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6
 * (4) 0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a
 * (5) 0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba
 * (6) 0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e
 * (7) 0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356
 * (8) 0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97
 * (9) 0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6
 */
export const anvilAccounts = [
  {
    address: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    privateKey: '0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
  },
  {
    address: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8',
    privateKey: '0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d',
  },
  {
    address: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
    privateKey: '0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a',
  },
  {
    address: '0x90F79bf6EB2c4f870365E785982E1f101E93b906',
    privateKey: '0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6',
  },
  {
    address: '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
    privateKey: '0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a',
  },
  {
    address: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
    privateKey: '0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba',
  },
  {
    address: '0x976EA74026E726554dB657fA54763abd0C3a0aa9',
    privateKey: '0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e',
  },
] as const


/**
 * Initiates test utils 
 * @important Move this to it's own package if it gets used in 3 or more places
 * Note it's already used in 2 places.  Here and the gateway
 */
export const anvilTestUtilFactory = async ({ l1, l2 }: TestUtilOptions) => {
  const anvilL1 = createAnvil({
    ...l1,
    port: l1.port ?? 8545,
  });
  const anvilL2 = createAnvil({
    ...l2,
    port: l2.port ?? 9545,
  });
  const rpcUrlL1 = `http://localhost:${anvilL1.port}`
  const rpcUrlL2 = `http://localhost:${anvilL2.port}`
  const publicClientL1 = createPublicClient({
    chain: l1.chain,
    pollingInterval: 0,
    transport: http(rpcUrlL1),
  })
  const publicClientL2 = createPublicClient({
    chain: l2.chain,
    pollingInterval: 0,
    transport: http(rpcUrlL2),
  })
  const testClientL1 = createTestClient({
    chain: l1.chain,
    mode: 'anvil',
    pollingInterval: 0,
    transport: http(rpcUrlL1),
  })
  const testClientL2 = createTestClient({
    chain: l1.chain,
    mode: 'anvil',
    pollingInterval: 0,
    transport: http(rpcUrlL2),
  })
  const accounts = anvilAccounts.map(({ privateKey }) => privateKeyToAccount(privateKey))
  const wallets = accounts.map(account => {
    return {
      l1: createWalletClient({
        chain: l1.chain,
        pollingInterval: 0,
        account,
        transport: http(rpcUrlL1),
      }),
      l2: createWalletClient({
        chain: l2.chain,
        pollingInterval: 0,
        account,
        transport: http(rpcUrlL2),
      })
    }
  })
  return {
    accounts,
    anvilL1,
    anvilL2,
    options: {
      l1,
      l2,
    },
    publicClientL1,
    publicClientL2,
    rpcUrlL1,
    rpcUrlL2,
    testClientL1,
    testClientL2,
    wallets,
  }
}

