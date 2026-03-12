export const L1ContractTable = ({ chain, explorer, legacy }) => {
  const [addresses, setAddresses] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    async function fetchAddresses() {
      try {
        const response = await fetch('https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/superchain/extra/addresses/addresses.json')
        
        if (!response.ok) {
          throw new Error('Failed to fetch addresses')
        }
        
        const data = await response.json()
        const chainData = data[chain] || {}
        
        setAddresses(chainData)
      } catch (err) {
        setError(err.message)
        console.error('Error fetching addresses:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchAddresses()
  }, [chain, legacy])

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

  if (loading) {
    return <div>Loading...</div>
  }

  if (error) {
    return <div>Error: {error}</div>
  }

  // Filter out legacy (or non-legacy) contracts
  const filtered = Object.keys(addresses || {})
    .filter(key => LEGACY_CONTRACT_NAMES.includes(key) === legacy)
    .reduce((acc, key) => {
      acc[key] = addresses[key]
      return acc
    }, {})

  const explorerUrl = CHAIN_CONSTANTS[parseInt(explorer)]?.explorer || 'https://etherscan.io'

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
