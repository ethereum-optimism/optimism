# OPCM Verification for Developers

This guide is for developers who have created a new feature for the OP Stack that includes an OPContractsManager (OPCM) deployment. Before signers can execute your OPCM transaction, they must be certain they're using the correct OPCM and not a malicious one.

The `VerifyOPCM.s.sol` script provides automated verification that your deployed OPCM matches locally built artifacts from your audited commit. This guide explains how to modify the script for your custom deployment.

## Why Verification Matters

The verification process protects against:
- **Malicious OPCM deployments**: Ensuring the deployed contracts match audited source code
- **Supply chain attacks**: Preventing compromised or unauthorized contract deployments
- **Configuration errors**: Validating that contract addresses and parameters are correct

As detailed in the [trust establishment process](https://github.com/ethereum-optimism/optimism/blob/develop/docs/op-stack/src/docs/security/opcm-verification.md), verification is a critical step in ensuring that signers can confidently execute upgrade transactions.

## Prerequisites

- Your feature has been audited and you have a specific commit hash
- You have deployed your OPCM with your new contracts
- You have access to the OPCM address and related contract addresses
- You understand which contracts are part of your feature

## Understanding VerifyOPCM Structure

The VerifyOPCM script verifies three main categories of contracts:

1. **OPCM Properties**: Contracts referenced via property getters (e.g., `opcmDeployer()`)
2. **Implementations**: Contracts in the `implementations()` struct
3. **Blueprints**: Contracts in the `blueprints()` struct (for large contracts split across multiple parts)

For more background on the OPCM architecture, see the [OPCM guide](./opcm.md).

## Step-by-Step Modification Guide

### 1. Identify Your New Contracts

First, determine which contracts your feature adds and where they fit:

```bash
# Check what's in your OPCM implementations
cast call $OPCM_ADDRESS "implementations()" --rpc-url $ETH_RPC_URL

# Check what's in your OPCM blueprints
cast call $OPCM_ADDRESS "blueprints()" --rpc-url $ETH_RPC_URL

# Check for any new property getters
cast abi-decode "tuple(address,address,address,address,address)" $(cast call $OPCM_ADDRESS "implementations()" --rpc-url $ETH_RPC_URL)
```

### 2. Add Field Name Overrides

If your contract field names don't cleanly map to contract names, add overrides in the `setUp()` function:

```solidity
// In VerifyOPCM.s.sol setUp() function
function setUp() public {
    // ... existing overrides ...

    // Add your custom field name overrides
    fieldNameOverrides["yourFieldName"] = "YourActualContractName";
    fieldNameOverrides["anotherFieldImpl"] = "AnotherContract";

    // For blueprint contracts that are split into parts
    fieldNameOverrides["largeContract1"] = "LargeContract";
    fieldNameOverrides["largeContract2"] = "LargeContract";
}
```

**Examples of when you need field name overrides:**
- Field: `myFeatureImpl` → Contract: `MyFeature`
- Field: `customGame` → Contract: `CustomDisputeGame`
- Field: `newValidator` → Contract: `NewValidatorContract`

### 3. Add Source Name Overrides

If multiple contracts are defined in the same source file, add source name overrides:

```solidity
// In VerifyOPCM.s.sol setUp() function
function setUp() public {
    // ... existing overrides ...

    // Add your source name overrides
    sourceNameOverrides["YourContractA"] = "SharedSourceFile";
    sourceNameOverrides["YourContractB"] = "SharedSourceFile";
}
```

**Example**: If you have `MyFeatureValidator` and `MyFeatureDeployer` both defined in `MyFeature.sol`

### 4. Test Individual Contract Verification

Before testing the full OPCM, verify individual contracts work:

```bash
cd packages/contracts-bedrock

# Set up environment
export ETH_RPC_URL=<your_rpc_url>
export ETHERSCAN_API_KEY=<your_api_key>

# Test individual contract verification
forge script scripts/deploy/VerifyOPCM.s.sol:VerifyOPCM \
  --sig "runSingle(string,address,bool)" \
  "YourContractName" \
  "0xYourContractAddress" \
  true \
  --rpc-url $ETH_RPC_URL
```

### 5. Verify Your Complete OPCM

Once individual contracts verify, test the full OPCM:

```bash
export OPCM_ADDRESS=<your_opcm_address>
forge script scripts/deploy/VerifyOPCM.s.sol:VerifyOPCM --rpc-url $ETH_RPC_URL
```

### 6. Validate Constructor Arguments

Ensure your OPCM's constructor arguments match expected values from the superchain registry:

```bash
# Check upgradeController (should match ProxyAdminOwner)
cast call $OPCM_ADDRESS "upgradeController()(address)"

# Check superchainProxyAdmin (should match ProxyAdmin)
cast call $OPCM_ADDRESS "superchainProxyAdmin()(address)"

# Check superchainConfig
cast call $OPCM_ADDRESS "superchainConfig()(address)"

# Check protocolVersions
cast call $OPCM_ADDRESS "protocolVersions()(address)"

# Check l1ContractsRelease (should match your release version)
cast call $OPCM_ADDRESS "l1ContractsRelease()(string)"
```

## Common Issues and Solutions

### Issue: Field Name Not Found
**Error**: Function call fails or unexpected results
**Solution**: Add a field name override mapping the actual field name to your contract name

### Issue: Bytecode Mismatch
**Error**: `[FAIL] ERROR: Bytecode difference found`
**Solution**:
- Ensure you're building from the exact same commit
- Check if immutables are different (this may be expected)
- Verify you have the correct contract name mapping

### Issue: Blueprint Part Mismatch
**Error**: `VerifyOPCM_UnexpectedPart()`
**Solution**: Blueprint contracts are automatically split. Ensure your field names end with `1` or `2` to indicate the part number.

### Issue: Artifact File Not Found
**Error**: `VerifyOPCM_EmptyArtifactFile` or file not found
**Solution**:
- Run `forge build` to generate artifacts
- Check if you need a source name override
- Verify the contract name is correct

## Advanced Customization

### Adding New Property Verification

If your OPCM has new property getters beyond the standard ones, you may need to extend the verification logic:

```solidity
// Example: If you added a new property getter like myCustomProperty()
// The script will automatically detect functions starting with "opcm"
// For other property getters, you might need custom logic
```

### Handling Custom Constructor Arguments

If your contracts have complex constructor arguments, you may need to extend the `_isValidConstructorArgs` function:

```solidity
// Override constructor validation for specific contracts if needed
function _isValidConstructorArgs(
    string memory _contractName,
    bytes memory _constructorArgs
) internal returns (bool) {
    // Add custom validation logic for your contracts
    if (LibString.eq(_contractName, "YourCustomContract")) {
        // Custom validation logic here
        return true;
    }

    // Fall back to default validation
    return super._isValidConstructorArgs(_contractName, _constructorArgs);
}
```

## Integration with Audit Process

### Before Audit
1. Complete your feature implementation
2. Update VerifyOPCM with necessary overrides
3. Test verification against your deployment
4. Document any custom verification requirements

### During Audit
- Include VerifyOPCM modifications in audit scope
- Ensure auditors can reproduce your verification process
- Document any assumptions or manual verification steps

### After Audit
- Provide clear instructions for signers to verify your OPCM
- Include the audited commit hash in your documentation
- Test the full verification process on the target network

## Example Checklist for New Features

- [ ] Identified all new contracts and their locations (implementations/blueprints/properties)
- [ ] Added necessary field name overrides
- [ ] Added necessary source name overrides
- [ ] Tested individual contract verification
- [ ] Tested full OPCM verification
- [ ] Validated constructor arguments
- [ ] Documented verification process for signers
- [ ] Included VerifyOPCM changes in audit scope

## Best Practices

1. **Use clear naming**: Choose field names that map cleanly to contract names when possible
2. **Test early**: Don't wait until deployment to test verification
3. **Document overrides**: Comment why each override is necessary
4. **Version control**: Commit VerifyOPCM changes with your feature
5. **Audit inclusion**: Always include verification script changes in audit scope

## Support and Resources

- [OP Stack Documentation](https://docs.optimism.io/)
- [Optimism Monorepo Contributing Guide](https://github.com/ethereum-optimism/optimism/blob/develop/CONTRIBUTING.md)
- [Superchain Registry](https://github.com/ethereum-optimism/superchain-registry)

For questions about verification requirements or issues with the script, consult with the OP Stack team during the audit process.
