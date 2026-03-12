export const SuperchainContractTable = ({ chain, explorer }) => {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // Helper function to get config URL
  const getConfigUrl = (chain) => {
    const isTestnet = chain === '11155111';
    const network = isTestnet ? 'sepolia' : 'mainnet';
    return `https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main/superchain/configs/${network}/superchain.toml`;
  };

  // Helper function to parse TOML
  const parseSimpleToml = (tomlText) => {
    const result = {};
    const lines = tomlText.split('\n');
    
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed && !trimmed.startsWith('#') && trimmed.includes('=')) {
        const [key, value] = trimmed.split('=', 2);
        const cleanKey = key.trim();
        const cleanValue = value.trim().replace(/['"]/g, '');
        
        if (/^0x[a-fA-F0-9]{40}$/.test(cleanValue)) {
          result[cleanKey] = cleanValue;
        }
      }
    }
    
    return result;
  };

  // Helper function to extract and rename addresses
  const extractAddresses = (obj) => {
    const addresses = {};
    for (const [key, value] of Object.entries(obj)) {
      if (typeof value === 'string' && /^0x[a-fA-F0-9]{40}$/.test(value)) {
        let newKey = key;
        if (key === 'protocol_versions_addr') newKey = 'ProtocolVersions';
        if (key === 'superchain_config_addr') newKey = 'SuperchainConfig';
        if (key === 'op_contracts_manager_proxy_addr') newKey = 'OPContractsManagerProxy';
        addresses[newKey] = value;
      }
    }
    return addresses;
  };

  useEffect(() => {
    async function fetchAddresses() {
      try {
        const configUrl = getConfigUrl(chain);
        const response = await fetch(configUrl);
        if (!response.ok) {
          throw new Error(`Failed to fetch config from ${configUrl}`);
        }
        const text = await response.text();
        
        const data = parseSimpleToml(text);
        setConfig(data);
      } catch (err) {
        setError(err.message);
        console.error('Error fetching or parsing config:', err);
      } finally {
        setLoading(false);
      }
    }

    fetchAddresses();
  }, [chain]);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>Error: {error}</div>;
  }

  const addresses = extractAddresses(config || {});

  const CHAIN_CONSTANTS = {
    1: { explorer: 'https://etherscan.io' },
    10: { explorer: 'https://explorer.optimism.io' },
    11155111: { explorer: 'https://sepolia.etherscan.io' },
    11155420: { explorer: 'https://sepolia-optimism.etherscan.io' }
  }

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
        {Object.entries(addresses || {})
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
  );
}
