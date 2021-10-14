import path from 'path'

// Codehashes of OVM_ECDSAContractAccount for 0.3.0 and 0.4.0
export const EOA_CODE_HASHES = [
  '0xa73df79c90ba2496f3440188807022bed5c7e2e826b596d22bcb4e127378835a',
  '0xef2ab076db773ffc554c9f287134123439a5228e92f5b3194a28fec0a0afafe3',
]

export const UNISWAP_V3_FACTORY_ADDRESS =
  '0x1F98431c8aD98523631AE4a59f267346ea31F984'

export const UNISWAP_V3_NFPM_ADDRESS =
  '0xC36442b4a4522E871399CD717aBDD847Ab11FE88'

export const UNISWAP_V3_CONTRACT_ADDRESSES = [
  // PoolDeployer
  '0x569E8D536EC2dD5988857147c9FCC7d8a08a7DBc',
  // UniswapV3Factory
  '0x1F98431c8aD98523631AE4a59f267346ea31F984',
  // ProxyAdmin
  '0xB753548F6E010e7e680BA186F9Ca1BdAB2E90cf2',
  // TickLens
  '0xbfd8137f7d1516D3ea5cA83523914859ec47F573',
  // Quoter
  '0xb27308f9F90D607463bb33eA1BeBb41C27CE5AB6',
  // SwapRouter
  '0xE592427A0AEce92De3Edee1F18E0157C05861564',
  // NonfungiblePositionLibrary
  '0x42B24A95702b9986e82d421cC3568932790A48Ec',
  // NonfungibleTokenPositionDescriptor
  '0x91ae842A5Ffd8d12023116943e72A606179294f3',
  // TransparentUpgradeableProxy
  '0xEe6A57eC80ea46401049E92587E52f5Ec1c24785',
  // NonfungibleTokenPositionManager
  '0xC36442b4a4522E871399CD717aBDD847Ab11FE88',
  // UniswapInterfaceMulticall (OP KOVAN)
  '0x1F98415757620B543A52E61c46B32eB19261F984',
  // UniswapInterfaceMulticall (OP MAINNET)
  '0x90f872b3d8f33f305e0250db6A2761B354f7710A',
]

export const PREDEPLOY_WIPE_ADDRESSES = [
  // L2CrossDomainMessenger
  '0x4200000000000000000000000000000000000007',
  // OVM_GasPriceOracle
  '0x420000000000000000000000000000000000000F',
  // L2StandardBridge
  '0x4200000000000000000000000000000000000010',
  // OVM_SequencerFeeVault
  '0x4200000000000000000000000000000000000011',
]

export const PREDEPLOY_NO_WIPE_ADDRESSES = [
  // OVM_DeployerWhitelist
  '0x4200000000000000000000000000000000000002',
  // OVM_L2ToL1MessagePasser
  '0x4200000000000000000000000000000000000000',
]

export const PREDEPLOY_NEW_NOT_ETH_ADDRESSES = [
  // L2StandardTokenFactory
  '0x4200000000000000000000000000000000000012',
  // OVM_L1BlockNumber
  '0x4200000000000000000000000000000000000013',
]

export const OLD_ETH_ADDRESS = '0x4200000000000000000000000000000000000006'
export const NEW_ETH_ADDRESS = '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000'

export const ONEINCH_DEPLOYER_ADDRESS =
  '0xee4f7b6c39e7e87af01fb9e4cee0c893ff4d63f2'

export const DELETE_CONTRACTS = [
  // 1inch aggregator
  '0x11111112542D85B3EF69AE05771c2dCCff4fAa26',
  // OVM_L1MessageSender
  '0x4200000000000000000000000000000000000001',
  // OVM v1 System Contract
  '0xDEADDEaDDeAddEADDeaDDEADdeaDdeAddeAd0005',
  // OVM v1 System Contract
  '0xDEADdeAdDeAddEAdDEaDdEaddEAddeaDdEaD0006',
  // OVM v1 System Contract
  '0xDeaDDeaDDeaddEADdeaDdEadDeaDdeADDEad0007',
  // Uniswap Position
  '0x18F7E3ae7202e93984290e1195810c66e1E276FF',
  // Uniswap Oracle
  '0x17b0f5e5850e7230136df66c5d49497b8c3be0c1',
  // Uniswap Tick
  '0x47405b0d5f88e16701be6dc8ae185fefaa5dca2f',
  // Uniswap TickBitmap
  '0x01d95165c3c730d6b40f55c37e24c7aac73d5e6f',
  // Uniswap TickMath
  '0x308c3e60585ad4eab5b7677be0566fead4cb4746',
  // Uniswap SwapMath
  '0x198dcc7cd919dd33dd72c3f981df653750901d75',
  // Uniswap UniswapV3PoolDeployer
  '0x569e8d536ec2dd5988857147c9fcc7d8a08a7dbc',
  // Uniswap NFTDescriptor
  '0x042f51014b152c2d2fc9b57e36b16bc744065d8c',
]

export const WETH_TRANSFER_ADDRESSES = [
  // Rubicon Mainnet bathETH
  '0xB0bE5d911E3BD4Ee2A8706cF1fAc8d767A550497',
  // Rubicon Mainnet bathETH-USDC
  '0x87a7Eed69eaFA78D30344001D0baFF99FC005Dc8',
  // Rubicon Mainnet bathETH-DAI
  '0x314eC4Beaa694264746e1ae324A5edB913a6F7C6',
  // Rubicon Mainnet bathETH-USDT
  '0xF6A47B24e80D12Ac7d3b5Cef67B912BCd3377333',
  // Rubicon Mainnet exchange
  '0x7a512d3609211e719737E82c7bb7271eC05Da70d',
  // Rubicon Kovan bathETH
  '0x5790AedddfB25663f7dd58261De8E96274A82BAd',
  // Rubicon Kovan bathETH-USDC
  '0x52fBa53c876a47a64A10F111fbeA7Ed506dCc7e7',
  // Rubicon Kovan bathETH-DAI
  '0xA92E4Bd9f61e90757Cd8806D236580698Fc20C91',
  // Rubicon Kovan bathETH-USDT
  '0x80D94a6f6b0335Bfed8D04B92423B6Cd14b5d31C',
  // Rubicon Kovan market
  '0x5ddDa7DF721272106af1904abcc64E76AB2019d2',
  // Hop Kovan AMM Wrapper
  '0xc9E6628791cdD4ad568550fcc6f378cEF27e98fd',
  // Hop Kovan Swap
  '0xD6E31cE884DFf44c4600fD9D36BcC9af447C28d5',
]

// TODO: confirm OVM/EVM mapps with ben-chain
export const COMPILER_VERSIONS_TO_SOLC = {
  'v0.5.16': 'v0.5.16+commit.9c3226ce',
  'v0.5.16-alpha.7': 'v0.5.16+commit.9c3226ce',
  'v0.6.12': 'v0.6.12+commit.27d51765',
  'v0.7.6': 'v0.7.6+commit.7338295f',
  'v0.7.6+commit.3b061308': 'v0.7.6+commit.7338295f',
  'v0.7.6-allow_kall': 'v0.7.6+commit.7338295f',
  'v0.7.6-no_errors': 'v0.7.6+commit.7338295f',
  'v0.8.4': 'v0.8.4+commit.c7e474f2',
}

export const SOLC_BIN_PATH = 'https://binaries.soliditylang.org'
export const EMSCRIPTEN_BUILD_PATH = `${SOLC_BIN_PATH}/emscripten-wasm32`
export const EMSCRIPTEN_BUILD_LIST = `${EMSCRIPTEN_BUILD_PATH}/list.json`
export const LOCAL_SOLC_DIR = path.join(__dirname, '..', 'solc-bin')
