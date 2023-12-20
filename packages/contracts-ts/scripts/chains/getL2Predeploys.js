/**
 * @typedef {ReturnType<typeof getL2Predeploys>} L2Predeploys
 */

/** Gets the l2 predeploy contracts shaped as an extension of viem
 * @param {number} genesisBlock
 */
export const getL2Predeploys = (genesisBlock = 0) => ({
  weth9: {
    address: '0x4200000000000000000000000000000000000006',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: false,
  },
  l2CrossDomainMessenger: {
    address: '0x4200000000000000000000000000000000000007',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  l2StandardBridge: {
    address: '0x4200000000000000000000000000000000000010',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  sequencerFeeVault: {
    address: '0x4200000000000000000000000000000000000011',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  optimismMintableERC20Factory: {
    address: '0x4200000000000000000000000000000000000012',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  gasPriceOracle: {
    address: '0x420000000000000000000000000000000000000F',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  governanceToken: {
    address: '0x4200000000000000000000000000000000000042',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: false,
  },
  l1Block: {
    address: '0x4200000000000000000000000000000000000015',
    blockCreated: genesisBlock,
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  l2ToL1MessagePasser: {
    address: '0x4200000000000000000000000000000000000016',
    blockCreated: genesisBlock,
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  l2ERC721Bridge: {
    address: '0x4200000000000000000000000000000000000014',
    blockCreated: genesisBlock,
    introduced: 'Legacy',
    deprecated: false,
    proxied: true,
  },
  optimismMintableERC721Factory: {
    address: '0x4200000000000000000000000000000000000017',
    blockCreated: genesisBlock,
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  proxyAdmin: {
    address: '0x4200000000000000000000000000000000000018',
    blockCreated: genesisBlock,
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  baseFeeVault: {
    address: '0x4200000000000000000000000000000000000019',
    blockCreated: genesisBlock,
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
  l1FeeVault: {
    address: '0x420000000000000000000000000000000000001a',
    blockCreated: genesisBlock,
    introduced: 'Bedrock',
    deprecated: false,
    proxied: true,
  },
})
