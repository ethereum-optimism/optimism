import { DeployConfig } from '../src/deploy-config'
import mainnetJson from './mainnet.json'

// NOTE: The 'mainnet' network is currently being used for bedrock migration rehearsals.
// The system configured below is not yet live on mainnet, and many of the addresses used are
// unsafe for a production system.

// Re-export the mainnet json as a DeployConfig object.
//
// Notice, the following roles in the system are assigned to the:
// Optimism Foundation Mulitisig:
// - finalSystemOwner
// - controller
// - portalGuardian
// - proxyAdminOwner
// - l2OutputOracleChallenger
//
// The following roles are assigned to the same fee recipient:
// - baseFeeVaultRecipient
// - l1FeeVaultRecipient
// - sequencerFeeVaultRecipient
//
// The following role is assigned to the Mint Manager contract:
// - governanceTokenOwner
const config: DeployConfig = mainnetJson

export default config
