export const L2ContractTable = ({ chain, legacy }) => {
  // L2 predeploy addresses are standard across all OP Stack chains
  const PREDEPLOYS = {
    'L2ToL1MessagePasser': '0x4200000000000000000000000000000000000016',
    'L2CrossDomainMessenger': '0x4200000000000000000000000000000000000007',
    'L2StandardBridge': '0x4200000000000000000000000000000000000010',
    'L2ERC721Bridge': '0x4200000000000000000000000000000000000014',
    'SequencerFeeVault': '0x4200000000000000000000000000000000000011',
    'OptimismMintableERC20Factory': '0x4200000000000000000000000000000000000012',
    'OptimismMintableERC721Factory': '0x4200000000000000000000000000000000000017',
    'L1Block': '0x4200000000000000000000000000000000000015',
    'GasPriceOracle': '0x420000000000000000000000000000000000000F',
    'ProxyAdmin': '0x4200000000000000000000000000000000000018',
    'BaseFeeVault': '0x4200000000000000000000000000000000000019',
    'L1FeeVault': '0x420000000000000000000000000000000000001A',
    'GovernanceToken': '0x4200000000000000000000000000000000000042',
    'SchemaRegistry': '0x4200000000000000000000000000000000000020',
    'EAS': '0x4200000000000000000000000000000000000021',
    'L1MessageSender': '0x4200000000000000000000000000000000000001',
    'DeployerWhitelist': '0x4200000000000000000000000000000000000002',
    'LegacyERC20ETH': '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000',
    'L1BlockNumber': '0x4200000000000000000000000000000000000013',
    'LegacyMessagePasser': '0x4200000000000000000000000000000000000000'
  }

  const LEGACY_CONTRACT_NAMES = [
    'AddressManager',
    'DeployerWhitelist', 
    'L1MessageSender',
    'LegacyERC20ETH',
    'LegacyMessagePasser',
    'L1BlockNumber',
  ]

  const CHAIN_CONSTANTS = {
    1: { explorer: 'https://etherscan.io' },
    10: { explorer: 'https://explorer.optimism.io' },
    11155111: { explorer: 'https://sepolia.etherscan.io' },
    11155420: { explorer: 'https://sepolia-optimism.etherscan.io' }
  }

  // Filter out legacy (or non-legacy) contracts
  const filtered = Object.keys(PREDEPLOYS || {})
    .filter(key => LEGACY_CONTRACT_NAMES.includes(key) === legacy)
    .reduce((acc, key) => {
      acc[key] = PREDEPLOYS[key]
      return acc
    }, {})

  const explorerUrl = CHAIN_CONSTANTS[parseInt(chain)]?.explorer || 'https://explorer.optimism.io'

  return (
    <table>
      <thead>
        <tr>
          <th>Contract Name</th>
          <th>Contract Address</th>
        </tr>
      </thead>
      <tbody>
        {Object.entries(filtered)
          .filter(([name, address]) => address && address !== '')
          .map(([contract, address]) => (
            <tr key={`${chain}.${contract}.${address}`}>
              <td>
                <code>{contract}</code>
              </td>
              <td>
                <a href={`${explorerUrl}/address/${address}`} target="_blank" rel="noreferrer">
                  <code>{address}</code>
                </a>
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}
