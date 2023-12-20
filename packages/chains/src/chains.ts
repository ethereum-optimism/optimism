export const opMainnet = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 105235063,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 105235063,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0xdfe97868233d1aa22e815a266982f2cf17685a27',
    },
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      blockCreated: 4286263,
    },
    portal: {
      address: '0xbEb5Fc579115071764c7423A4f12eDde41f106Ed',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 105235063,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 105235063,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x543bA4AADBAb8f9025686Bd03993043599c6fB04',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 105235063,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 105235063,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0xdE1FCfB0851916CA5101820A69b13a4E276bd81F',
    },
    l1ERC721Bridge: {
      address: '0x5a7749f83b81B301cAb5f48EB8516B986DAef23D',
    },
    l1StandardBridge: {
      address: '0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1',
    },
    l1CrossDomainMessenger: {
      address: '0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1',
    },
    l2ERC20Factory: {
      address: '0x75505a97BD334E7BD3C476893285569C4136Fa0F',
    },
  },
  id: 10,
  name: 'OP Mainnet',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://mainnet.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Optimism Explorer',
      url: 'https://optimistic.etherscan.io',
      apiUrl: 'https://api-optimistic.etherscan.io',
    },
  },
  sourceId: 1,
} as const
export const opstack291 = {
  id: 291,
  name: 'opstack291',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xb570F4aD27e7De879A2E4F2F3DE27dBaBc20E9B9',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0x91493a61ab83b62943E6dCAa5475Dd330704Cc84',
    },
    addressManager: {
      address: '0x87630a802a3789463eC4b00f89b27b1e9f6b92e9',
    },
    l1ERC721Bridge: {
      address: '0x934Ab59Ef14b638653b1C0FEf7aB9a72186393DC',
    },
    l2OutputOracle: {
      address: '0x5e76821C3c1AbB9fD6E310224804556C61D860e0',
    },
    l1StandardBridge: {
      address: '0xe07eA0436100918F157DF35D01dCE5c11b16D1F1',
    },
    l1CrossDomainMessenger: {
      address: '0xc76543A64666d9a073FaEF4e75F651c88e7DBC08',
    },
    l2ERC20Factory: {
      address: '0x7a69a90d8ea11E9618855da55D09E6F953730686',
    },
  },
  sourceId: 1,
} as const
export const optimismGoerli = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0',
    },
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      blockCreated: 49461,
    },
    portal: {
      address: '0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x01d3670863c3F4b24D7b107900f0b75d4BbC6e0d',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0xa6f73589243a6A7a9023b1Fa0651b1d89c177111',
    },
    l1ERC721Bridge: {
      address: '0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9',
    },
    l1StandardBridge: {
      address: '0x636Af16bf2f682dD3109e60102b8E1A089FedAa8',
    },
    l1CrossDomainMessenger: {
      address: '0x5086d1eEF304eb5284A0f6720f79403b4e9bE294',
    },
    l2ERC20Factory: {
      address: '0x883dcF8B05364083D849D8bD226bC8Cb4c42F9C5',
    },
  },
  id: 420,
  name: 'Optimism Goerli',
  nativeCurrency: {
    name: 'Goerli Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://goerli.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Etherscan',
      url: 'https://goerli-optimism.etherscan.io',
      apiUrl: 'https://goerli-optimism.etherscan.io/api',
    },
  },
  testnet: true,
  sourceId: 5,
} as const
export const pgn = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  id: 424,
  network: 'pgn',
  name: 'PGN',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://rpc.publicgoods.network'],
    },
  },
  blockExplorers: {
    default: {
      name: 'PGN Explorer',
      url: 'https://explorer.publicgoods.network',
      apiUrl: 'https://explorer.publicgoods.network/api',
    },
    blocksout: {
      name: 'PGN Explorer',
      url: 'https://explorer.publicgoods.network',
      apiUrl: 'https://explorer.publicgoods.network/api',
    },
  },
  contracts: {
    l2OutputOracle: {
      address: '0xA38d0c4E6319F9045F20318BA5f04CDe94208608',
    },
    multicall3: {
      address: '0xcA11bde05977b3631167028862bE2a173976CA11',
      blockCreated: 3380209,
    },
    portal: {
      address: '0xb26Fd985c5959bBB382BAFdD0b879E149e48116c',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xc6A8d2c5d0F068BE745f6A770378F01ca1714cc4',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0x09d5DbA52F0ee2C4A5E94FD5C802bD74Ca9cAD3e',
    },
    l1ERC721Bridge: {
      address: '0xaFF0F8aaB6Cc9108D34b3B8423C76d2AF434d115',
    },
    l1StandardBridge: {
      address: '0xD0204B9527C1bA7bD765Fa5CCD9355d38338272b',
    },
    l1CrossDomainMessenger: {
      address: '0x97BAf688E5d0465E149d1d5B497Ca99392a6760e',
    },
    l2ERC20Factory: {
      address: '0x8A04c7e5b182eb3470073E681bE54b2aB48FBbE8',
    },
  },
  sourceId: 1,
} as const
export const opstack888 = {
  id: 888,
  name: 'opstack888',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x64d1E91BD7B80354e77C05c7FBff3Ad00E05946a',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0x1566c8Eea4A255C07Ef58edF91431c8A73ae0B62',
    },
    addressManager: {
      address: '0x41E2A82Ddf1311D74c898Bb825c8D0eafaea2432',
    },
    l1ERC721Bridge: {
      address: '0x058BBf091232afE99BC2481F809254cD15e64Df5',
    },
    l2OutputOracle: {
      address: '0x7D00A03f180d8C07B88d8c1384a15326c38FF9Ff',
    },
    l1StandardBridge: {
      address: '0x60859421Ed85C0B11071230cf61dcEeEf54630Ff',
    },
    l1CrossDomainMessenger: {
      address: '0xfc428D28D197fFf99A5EbAc6be8B761FEd8718Da',
    },
    l2ERC20Factory: {
      address: '0x526920419b61153c1F80fD306B5Ab52b69110A6C',
    },
  },
  sourceId: 1,
} as const
export const opstack957 = {
  id: 957,
  name: 'opstack957',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x35d5D43271548c984662d4879FBc8e041Bc1Ff93',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0x85eA9c11cf3D4786027F7FD08F4406b15777e5f8',
    },
    addressManager: {
      address: '0xC845F9C4004EB35a8bde8ad89C4760a9c0e65CAB',
    },
    l1ERC721Bridge: {
      address: '0x6CC3268794c5d3E3d9d52adEfC748B59d536cb22',
    },
    l2OutputOracle: {
      address: '0x1145E7848c8B64c6cab86Fd6D378733385c5C3Ba',
    },
    l1StandardBridge: {
      address: '0x61E44dC0dae6888B5a301887732217d5725B0bFf',
    },
    l1CrossDomainMessenger: {
      address: '0x5456f02c08e9A018E42C39b351328E5AA864174A',
    },
    l2ERC20Factory: {
      address: '0x08Dea366F26C25a08C8D1C3568ad07d1e587136d',
    },
  },
  sourceId: 1,
} as const
export const opstack997 = {
  id: 997,
  name: 'opstack997',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xD98bD7a1F2384D890d0D6153Cb6F813ab6cCFcCF',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0xc6170e048b7daef6d0b6cbbad9aca0e5370ddbbc',
    },
    addressManager: {
      address: '0xf3a31b72d030e1916afeb3abba90e7e104818b9b',
    },
    l1ERC721Bridge: {
      address: '0xab598ffd07bdf497fce58e36138573ccba6b7a8b',
    },
    l2OutputOracle: {
      address: '0xddb2e0c86ae08f1249d528f1a810cebd1b4c4d72',
    },
    l1StandardBridge: {
      address: '0x0178b1f72eb1e61e1847f8fd36c791822623fb42',
    },
    l1CrossDomainMessenger: {
      address: '0x12371d047382bb3a4b1891e8474ddaee983d08ec',
    },
    l2ERC20Factory: {
      address: '0x00b75ed2e46c4c29bc363a75a6d97791018b3903',
    },
  },
  sourceId: 1,
} as const
export const base = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0x56315b90c40730925ec5485cf004d835058518A0',
    },
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      blockCreated: 5022,
    },
    portal: {
      address: '0x49048044D57e1C92A77f79988d21Fa8fAF74E97e',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x0475cBCAebd9CE8AfA5025828d5b98DFb67E059E',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0x8EfB6B5c4767B09Dc9AA6Af4eAA89F749522BaE2',
    },
    l1ERC721Bridge: {
      address: '0x608d94945A64503E642E6370Ec598e519a2C1E53',
    },
    l1StandardBridge: {
      address: '0x3154Cf16ccdb4C6d922629664174b904d80F2C35',
    },
    l1CrossDomainMessenger: {
      address: '0x866E82a600A1414e583f7F13623F1aC5d58b0Afa',
    },
    l2ERC20Factory: {
      address: '0x05cc379EBD9B30BbA19C6fA282AB29218EC61D84',
    },
  },
  id: 8453,
  name: 'Base',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://mainnet.base.org'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Basescan',
      url: 'https://basescan.org',
      apiUrl: 'https://api.basescan.org/api',
    },
  },
  sourceId: 1,
} as const
export const opstack34443 = {
  id: 34443,
  name: 'opstack34443',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x470d87b1dae09a454A43D1fD772A561a03276aB7',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0x8B34b14c7c7123459Cf3076b8Cb929BE097d0C07',
    },
    addressManager: {
      address: '0x50eF494573f28Cad6B64C31b7a00Cdaa48306e15',
    },
    l1ERC721Bridge: {
      address: '0x2901dA832a4D0297FF0691100A8E496626cc626D',
    },
    l2OutputOracle: {
      address: '0x4317ba146D4933D889518a3e5E11Fe7a53199b04',
    },
    l1StandardBridge: {
      address: '0x735aDBbE72226BD52e818E7181953f42E3b0FF21',
    },
    l1CrossDomainMessenger: {
      address: '0x95bDCA6c8EdEB69C98Bd5bd17660BaCef1298A6f',
    },
    l2ERC20Factory: {
      address: '0x69216395A62dFb243C05EF4F1C27AF8655096a95',
    },
  },
  sourceId: 1,
} as const
export const pgnTestnet = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  id: 58008,
  network: 'pgn-testnet',
  name: 'PGN ',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://sepolia.publicgoods.network'],
    },
  },
  blockExplorers: {
    default: {
      name: 'PGN Testnet Explorer',
      url: 'https://explorer.sepolia.publicgoods.network',
      apiUrl: 'https://explorer.sepolia.publicgoods.network/api',
    },
    blocksout: {
      name: 'PGN Testnet Explorer',
      url: 'https://explorer.sepolia.publicgoods.network',
      apiUrl: 'https://explorer.sepolia.publicgoods.network/api',
    },
  },
  contracts: {
    l2OutputOracle: {
      address: '0xD5bAc3152ffC25318F848B3DD5dA6C85171BaEEe',
    },
    multicall3: {
      address: '0xcA11bde05977b3631167028862bE2a173976CA11',
      blockCreated: 3754925,
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x5f336973dabad13409ea93416b8487d92769e457',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0xF04BdD5353Bb0EFF6CA60CfcC78594278eBfE179',
    },
    addressManager: {
      address: '0x0Ad91488288BBe60ff38258785568A6D1EB3B983',
    },
    l1ERC721Bridge: {
      address: '0xBA8397B6f255618D5985d0fB427D8c0496F3a5FA',
    },
    l1StandardBridge: {
      address: '0xFaE6abCAF30D23e233AC7faF747F2fC3a5a6Bfa3',
    },
    l1CrossDomainMessenger: {
      address: '0x97f3558Ce48FE71B8CeFA5497708A49531D5A8E1',
    },
    l2ERC20Factory: {
      address: '0x0167EF3188FDaa2661e4530A4623Ee1aB4555683',
    },
  },
  sourceId: 11155111,
  testnet: true,
} as const
export const baseGoerli = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0x2A35891ff30313CcFa6CE88dcf3858bb075A2298',
    },
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      blockCreated: 1376988,
    },
    portal: {
      address: '0xe93c8cD0D409341205A592f8c4Ac1A5fe5585cfA',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xbc0Fc544736b7d610D9b05F31B182C8154BEf336',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0x4Cf6b56b14c6CFcB72A75611080514F94624c54e',
    },
    l1ERC721Bridge: {
      address: '0x5E0c967457347D5175bF82E8CCCC6480FCD7e568',
    },
    l1StandardBridge: {
      address: '0xfA6D8Ee5BE770F84FC001D098C4bD604Fe01284a',
    },
    l1CrossDomainMessenger: {
      address: '0x8e5693140eA606bcEB98761d9beB1BC87383706D',
    },
    l2ERC20Factory: {
      address: '0xa88530E2DD811363cA3Ef479dBab3C0BF73d90b1',
    },
  },
  id: 84531,
  name: 'Base Goerli',
  nativeCurrency: {
    name: 'Goerli Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://goerli.base.org'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Basescan',
      url: 'https://goerli.basescan.org',
      apiUrl: 'https://goerli.basescan.org/api',
    },
  },
  testnet: true,
  sourceId: 5,
} as const
export const baseSepolia = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0x84457ca9D0163FbC4bbfe4Dfbb20ba46e48DF254',
    },
    portal: {
      address: '0x49f53e41452C74589E85cA1677426Ba426459e85',
    },
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      blockCreated: 1059647,
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x0389E59Aa0a41E4A413Ae70f0008e76CAA34b1F3',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0x709c2B8ef4A9feFc629A8a2C1AF424Dc5BD6ad1B',
    },
    l1ERC721Bridge: {
      address: '0x21eFD066e581FA55Ef105170Cc04d74386a09190',
    },
    l1StandardBridge: {
      address: '0xfd0Bf71F60660E2f608ed56e1659C450eB113120',
    },
    l1CrossDomainMessenger: {
      address: '0xC34855F4De64F1840e5686e64278da901e261f20',
    },
    l2ERC20Factory: {
      address: '0xb1efB9650aD6d0CC1ed3Ac4a0B7f1D5732696D37',
    },
  },
  id: 84532,
  network: 'base-sepolia',
  name: 'Base Sepolia',
  nativeCurrency: {
    name: 'Sepolia Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://sepolia.base.org'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Blockscout',
      url: 'https://base-sepolia.blockscout.com',
      apiUrl: 'https://base-sepolia.blockscout.com/api',
    },
  },
  testnet: true,
  sourceId: 11155111,
} as const
export const zora = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0x9E6204F750cD866b299594e2aC9eA824E2e5f95c',
    },
    multicall3: {
      address: '0xcA11bde05977b3631167028862bE2a173976CA11',
      blockCreated: 5882,
    },
    portal: {
      address: '0x1a0ad011913A150f69f6A19DF447A0CfD9551054',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xD4ef175B9e72cAEe9f1fe7660a6Ec19009903b49',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0xEF8115F2733fb2033a7c756402Fc1deaa56550Ef',
    },
    l1ERC721Bridge: {
      address: '0x83A4521A3573Ca87f3a971B169C5A0E1d34481c3',
    },
    l1StandardBridge: {
      address: '0x3e2Ea9B92B7E48A52296fD261dc26fd995284631',
    },
    l1CrossDomainMessenger: {
      address: '0xdC40a14d9abd6F410226f1E6de71aE03441ca506',
    },
    l2ERC20Factory: {
      address: '0xc52BC7344e24e39dF1bf026fe05C4e6E23CfBcFf',
    },
  },
  id: 7777777,
  name: 'Zora',
  nativeCurrency: {
    decimals: 18,
    name: 'Ether',
    symbol: 'ETH',
  },
  rpcUrls: {
    default: {
      http: ['https://rpc.zora.energy'],
      webSocket: ['wss://rpc.zora.energy'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Explorer',
      url: 'https://explorer.zora.energy',
      apiUrl: 'https://explorer.zora.energy/api',
    },
  },
  sourceId: 1,
} as const
export const optimismSepolia = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F',
    },
    multicall3: {
      address: '0xca11bde05977b3631167028862be2a173976ca11',
      blockCreated: 1620204,
    },
    portal: {
      address: '0x16Fc5058F25648194471939df75CF27A2fdC48BC',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0x9bFE9c5609311DF1c011c47642253B78a4f33F4B',
    },
    l1ERC721Bridge: {
      address: '0xd83e03D576d23C9AEab8cC44Fa98d058D2176D1f',
    },
    l1StandardBridge: {
      address: '0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1',
    },
    l1CrossDomainMessenger: {
      address: '0x58Cc85b8D04EA49cC6DBd3CbFFd00B4B8D6cb3ef',
    },
    l2ERC20Factory: {
      address: '0x868D59fF9710159C2B330Cc0fBDF57144dD7A13b',
    },
  },
  id: 11155420,
  name: 'Optimism Sepolia',
  nativeCurrency: {
    name: 'Sepolia Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: ['https://sepolia.optimism.io'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Blockscout',
      url: 'https://optimism-sepolia.blockscout.com',
      apiUrl: 'https://optimism-sepolia.blockscout.com/api',
    },
  },
  testnet: true,
  sourceId: 11155111,
} as const
export const opstack11763071 = {
  id: 11763071,
  name: 'opstack11763071',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0x4d56E97228bBF10DcB2ED7E8F455c57AbE247404',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0x61A7dc680a0f3F67aDc357453d3f51bDc70fAE1B',
    },
    addressManager: {
      address: '0x098492Ef1F4Bf26F305F25826CA0F4e4Be6d45f4',
    },
    l1ERC721Bridge: {
      address: '0x8e5B1fF0C5afB207Ac447B31e149996E053D9C22',
    },
    l2OutputOracle: {
      address: '0x805fbEDB43E814b2216ce6926A0A19bdeDb0C8Cd',
    },
    l1StandardBridge: {
      address: '0x21E0Cc91D566cfF3edC500F8012D6105f889d2b0',
    },
    l1CrossDomainMessenger: {
      address: '0x548531f9E60e75726F6f6ec1E5F0A181B9d2c1C0',
    },
    l2ERC20Factory: {
      address: '0x92210e86f7e71606394FD57Be284Ef46Eced62Da',
    },
  },
  sourceId: 1,
} as const
export const zoraSepolia = {
  formatters: {
    block: {
      type: 'block',
    },
    transaction: {
      type: 'transaction',
    },
    transactionReceipt: {
      type: 'transactionReceipt',
    },
  },
  serializers: {},
  contracts: {
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2Erc721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2OutputOracle: {
      address: '0x2615B481Bd3E5A1C0C7Ca3Da1bdc663E8615Ade9',
    },
    multicall3: {
      address: '0xcA11bde05977b3631167028862bE2a173976CA11',
      blockCreated: 83160,
    },
    portal: {
      address: '0xeffE2C6cA9Ab797D418f0D91eA60807713f3536f',
    },
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xE17071F4C216Eb189437fbDBCc16Bb79c4efD9c2',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    addressManager: {
      address: '0x27c9392144DFcB6dab113F737356C32435cD1D55',
    },
    l1ERC721Bridge: {
      address: '0x16B0a4f451c4CB567703367e587E15Ac108e4311',
    },
    l1StandardBridge: {
      address: '0x5376f1D543dcbB5BD416c56C189e4cB7399fCcCB',
    },
    l1CrossDomainMessenger: {
      address: '0x1bDBC0ae22bEc0c2f08B4dd836944b3E28fe9b7A',
    },
    l2ERC20Factory: {
      address: '0x5F3bdd57f01e88cE2F88f00685D30D6eb51A187c',
    },
  },
  id: 999999999,
  name: 'Zora Sepolia',
  network: 'zora-sepolia',
  nativeCurrency: {
    decimals: 18,
    name: 'Zora Sepolia',
    symbol: 'ETH',
  },
  rpcUrls: {
    default: {
      http: ['https://sepolia.rpc.zora.energy'],
      webSocket: ['wss://sepolia.rpc.zora.energy'],
    },
  },
  blockExplorers: {
    default: {
      name: 'Zora Sepolia Explorer',
      url: 'https://sepolia.explorer.zora.energy/',
      apiUrl: 'https://sepolia.explorer.zora.energy/api',
    },
  },
  sourceId: 11155111,
  testnet: true,
} as const
export const opstack129831238013 = {
  id: 129831238013,
  name: 'opstack129831238013',
  nativeCurrency: {
    name: 'Ether',
    symbol: 'ETH',
    decimals: 18,
  },
  rpcUrls: {
    default: {
      http: [],
    },
    public: {
      http: [],
    },
  },
  contracts: {
    weth9: {
      address: '0x4200000000000000000000000000000000000006',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l2CrossDomainMessenger: {
      address: '0x4200000000000000000000000000000000000007',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    l2StandardBridge: {
      address: '0x4200000000000000000000000000000000000010',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    sequencerFeeVault: {
      address: '0x4200000000000000000000000000000000000011',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC20Factory: {
      address: '0x4200000000000000000000000000000000000012',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    gasPriceOracle: {
      address: '0x420000000000000000000000000000000000000F',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    governanceToken: {
      address: '0x4200000000000000000000000000000000000042',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: false,
    },
    l1Block: {
      address: '0x4200000000000000000000000000000000000015',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ToL1MessagePasser: {
      address: '0x4200000000000000000000000000000000000016',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l2ERC721Bridge: {
      address: '0x4200000000000000000000000000000000000014',
      blockCreated: 0,
      introduced: 'Legacy',
      deprecated: false,
      proxied: true,
    },
    optimismMintableERC721Factory: {
      address: '0x4200000000000000000000000000000000000017',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    proxyAdmin: {
      address: '0xf592f1730154cE9e3F1793b583582B31A00EFBf1',
    },
    baseFeeVault: {
      address: '0x4200000000000000000000000000000000000019',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    l1FeeVault: {
      address: '0x420000000000000000000000000000000000001a',
      blockCreated: 0,
      introduced: 'Bedrock',
      deprecated: false,
      proxied: true,
    },
    portal: {
      address: '0x0550548367B568C129b1dC7b2B5C6273Cbd2da76',
    },
    addressManager: {
      address: '0x099bDAB5005747B8098237b655a808d2bdb28787',
    },
    l1ERC721Bridge: {
      address: '0xf029DB8aC06031984A2829DFed81BE18d382EE9C',
    },
    l2OutputOracle: {
      address: '0x689258a0dc1D421f6e884d0325B71e778ceda1DD',
    },
    l1StandardBridge: {
      address: '0xBf5cE0a8C1F9926d410Af9b159a11Ea430Cd5ffE',
    },
    l1CrossDomainMessenger: {
      address: '0x91363E5E3B9544a361FB473fBfe42665e16aa436',
    },
    l2ERC20Factory: {
      address: '0x8e3ECF806D0921d69D646D5E5647308050640416',
    },
  },
  sourceId: 1,
} as const
