# Findings:

## #5 Top-level V2 deploy wrapper zeroes the initial PERMISSIONED_CANNON bond

### Description

Deploy.s.sol builds a fresh OPCM V2 deployment config with PERMISSIONED_CANNON enabled but sets its initBond to 0. The same helper disables the other two dispute game types, so this zeroed bond becomes the initial respected game bond for chains deployed through the top-level wrapper.

// packages/contracts-bedrock/scripts/deploy/Deploy.s.sol:495-511
disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
enabled: true,
initBond: 0,
gameType: GameTypes.PERMISSIONED_CANNON,
gameArgs: abi.encode(...)
});
That value is not inert. Deploy.s.sol passes this config straight into OPContractsManagerV2.deploy(), which writes each configured initBond into DisputeGameFactory. DisputeGameFactory.create() later requires msg.value == initBonds[_gameType]. Therefore a fresh deployment through this wrapper programs PERMISSIONED_CANNON to open with a zero required bond.

// packages/contracts-bedrock/src/dispute/DisputeGameFactory.sol:162-163
if (msg.value != initBonds[_gameType]) revert IncorrectBondAmount();
This does not match the canonical V2 deploy script. DeployOPChain.s.sol uses DEFAULT_INIT_BOND for the same enabled PERMISSIONED_CANNON config. Its tests assert that value.

// packages/contracts-bedrock/scripts/deploy/DeployOPChain.s.sol:187-190
disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
enabled: true,
initBond: DEFAULT_INIT_BOND,
gameType: GameTypes.PERMISSIONED_CANNON,
gameArgs: abi.encode(pdgConfig)
});
The impact is narrower than a public zero-bond game because PermissionedDisputeGame still restricts initialization to the configured proposer. But the wrapper still brings the chain up with weaker dispute-game economics than intended. The designated proposer can create the initial respected game without posting the expected upfront stake.

### Recommendation

Do not maintain a second fresh-deploy dispute-game config by hand in Deploy.s.sol. Reuse the same V2 config builder as DeployOPChain.s.sol. If that refactor is not practical, set the enabled PERMISSIONED_CANNON.initBond to DEFAULT_INIT_BOND before calling OPContractsManagerV2.deploy().

Add a regression test for the top-level V2 deploy wrapper that checks disputeGameFactory.initBonds(GameTypes.PERMISSIONED_CANNON) after deployment. That test should assert the same bond value as the dedicated DeployOPChain flow.

## #7 Missing code checks can leave new L2 predeploys pointing at empty implementations

## Description

L2ContractsManagerUtils.upgradeTo() does not verify that \_implementation has runtime code. It only skips the semver downgrade check when the current implementation already has code, then writes the new address into the proxy unconditionally.

// packages/contracts-bedrock/src/libraries/L2ContractsManagerUtils.sol:42-57
address implementation = L2ProxyAdmin(Predeploys.PROXY_ADMIN).getProxyImplementation(\_proxy);

if (
implementation.code.length != 0
&& SemverComp.gt(ISemver(\_proxy).version(), ISemver(\_implementation).version())
) {
revert L2ContractsManager_DowngradeNotAllowed(address(\_proxy));
}

IProxy(payable(\_proxy)).upgradeTo(\_implementation);
That exception is exactly the case for newly introduced or conditionally deployed predeploys, where the current implementation can still be empty. If the expected implementation deployment is omitted or under-gassed, upgradeTo() can still repoint the proxy to the precomputed implementation address. The same result occurs if the deployment fails earlier in the upgrade block. This happens even though no code was ever deployed there.

upgradeToAndCall() has the same missing check for its final \_implementation. It upgrades to StorageSetter, clears initialization slots, then performs the final upgrade without validating code on \_implementation.

// packages/contracts-bedrock/src/libraries/L2ContractsManagerUtils.sol:118-126,164-165
if (
implementation.code.length != 0
&& SemverComp.gt(ISemver(\_proxy).version(), ISemver(\_implementation).version())
) {
revert L2ContractsManager_DowngradeNotAllowed(address(\_proxy));
}

IProxy(payable(\_proxy)).upgradeTo(\_storageSetterImpl);
...
IProxy(payable(\_proxy)).upgradeToAndCall(\_implementation, \_data);
Proxy does not fail closed here. upgradeTo() only stores the new implementation address. upgradeToAndCall() stores the new address before the delegatecall.

// packages/contracts-bedrock/src/universal/Proxy.sol:60-80
function upgradeTo(address \_implementation) public virtual proxyCallIfNotAdmin {
\_setImplementation(\_implementation);
}

function upgradeToAndCall(address \_implementation, bytes calldata \_data) public payable virtual proxyCallIfNotAdmin returns (bytes memory) {
\_setImplementation(\_implementation);
(bool success, bytes memory returndata) = \_implementation.delegatecall(\_data);
require(success, "Proxy: delegatecall to new implementation contract failed");
return returndata;
}
Consequently, first-time installs can be left pointing at empty implementations after an otherwise successful upgrade. Relevant examples include CROSS_L2_INBOX, L2_TO_L2_CROSS_DOMAIN_MESSENGER, SUPERCHAIN_ETH_BRIDGE, plus ETH_LIQUIDITY. The upgrade transaction succeeds. The first later call into that predeploy then fails against empty code. The \_storageSetterImpl step is less severe because the next slot access should revert if that helper is missing. The final \_implementation remains unchecked.

### Recommendation

Require \_implementation.code.length != 0 before calling upgradeTo() or the final upgradeToAndCall(). Add the same check for \_storageSetterImpl for consistency. Matching the L1 assertValidContractAddress() behavior would make the L2 upgrade fail at the point where a deployment is missing instead of silently repointing the proxy first.

## #13 NUT bundle version is never enforced

### Description

The bundle includes metadata.version, but nothing uses that field to decide whether the file is actually safe to read. op-node decodes the full JSON bundle in readNUTBundle() and returns it without checking the version. On the Solidity side, NetworkUpgradeTxns.readArtifact() does not read metadata at all and parses only .transactions.

In practice, the version field is informational only. If the bundle format changes later, older readers will still try to parse the file as if nothing changed.

### Recommendation

Make the version field a real gate. Define the supported bundle version in each reader and reject any file whose metadata.version does not match before decoding transactions. If multiple bundle versions need to be supported later, branch on metadata.version explicitly and give each version its own parser. Do not rely on one permissive reader to handle every schema revision.

## #17 L2Genesis accepts mismatched interop inputs

### Description

L2Genesis does not enforce that useInterop and the OPTIMISM_PORTAL_INTEROP bit in devFeatureBitmap agree. In setL1Block, the script enables Features.INTEROP whenever useInterop is true. Earlier in the same script, the interop predeploy set is only installed when both useInterop and the interop dev bit are set.

That split lets a caller that hand-builds L2Genesis.Input generate a malformed genesis that advertises interop through L1Block but omits CrossL2Inbox, L2ToL2CrossDomainMessenger and the rest of the interop predeploy set. The standard Go deployment paths reject or avoid this mismatch, so the impact is limited to manual or helper-driven uses of the script. The script still should not accept inconsistent inputs.

### Recommendation

Fail early when the two interop inputs disagree. The smallest safe fix is to add a check near the start of run that enforces useInterop == DevFeatures.isDevFeatureEnabled(devFeatureBitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP) and reverts on mismatch.

## #19 Fork suite never revalidates post-upgrade L1Block feature state

### Description

The fork suite snapshots isInteropEnabled and isCustomGasToken before the upgrade and then reuses those cached booleans only to choose branches and expected implementation names. It never re-reads L1Block after the bundle has executed. Therefore the suite does not prove that the upgraded chain still reports the same runtime mode that it had before the fork.

The custom-gas-token migration is a clear example. L2ContractsManager.\_apply() upgrades L1Block to L1BlockCGT and then relies on a follow-up setFeature(Features.CUSTOM_GAS_TOKEN) call so that isCustomGasToken() keeps returning true. If that migration step is omitted, reordered or otherwise fails to leave the feature bit set, the bundle can still succeed and the fork suite can still go green because it only checks that the proxy points at the L1BlockCGT implementation. In that state, the upgraded chain starts reporting ether metadata again, and CGT-specific behavior such as the extra restriction in L2ToL1MessagePasserCGT can silently change. The suite has the same gap for post-upgrade INTEROP reporting.

### Recommendation

Add explicit post-upgrade assertions against L1Block. After the bundle executes, require isCustomGasToken() and isFeatureEnabled(Features.INTEROP) to match the expected fork state. On CGT chains, also assert that gasPayingTokenName() and gasPayingTokenSymbol() still resolve to the pre-upgrade values.

## #29 Move NATIVE_ASSET_LIQUIDITY upgrade down to the non-initializable pre-deploy section

### Description

The upgrade call to the native asset liquidity contract should be moved down to the section specifically detailed for non-initializable pre-deploys, unless it is required for other contracts to be deployed. This is merely a code-quality informational issue, that would help with readability.

### Recommendation

Move the upgrade down to the appropriate section.
packages/contracts-bedrock/src/L2/L2ContractsManager.sol line 337
