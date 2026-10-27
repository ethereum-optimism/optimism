// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title PrivateChainSettings
/// @notice Names and key derivation for the OP Private private chain settings stored in
///         `SystemConfig.keyValue`. A setting for a private chain is stored under
///         `keccak256(abi.encode(name, chainId))`, where `name` is one of the constants below and
///         `chainId` is the private chain's chain ID. Off-chain readers derive keys the same way.
///         Nodes apply a default for any setting that has no entry, and they validate values.
library PrivateChainSettings {
    /// @notice Address that signs the private chain's unsafe blocks. Value: `abi.encode(address)`.
    bytes32 internal constant UNSAFE_BLOCK_SIGNER = "unsafeBlockSigner";

    /// @notice Minimum base fee of the private chain, in wei. Value: `abi.encode(uint64)`.
    bytes32 internal constant MIN_BASE_FEE = "minBaseFee";

    /// @notice EIP-1559 denominator of the private chain. Value: `abi.encode(uint32)`.
    bytes32 internal constant EIP1559_DENOMINATOR = "eip1559Denominator";

    /// @notice EIP-1559 elasticity of the private chain. Value: `abi.encode(uint32)`.
    bytes32 internal constant EIP1559_ELASTICITY = "eip1559Elasticity";

    /// @notice Returns the `SystemConfig.keyValue` key of a setting for a private chain.
    /// @param _name    Name of the setting, one of the constants above.
    /// @param _chainId Chain ID of the private chain.
    /// @return key_ The key under which the setting is stored.
    function key(bytes32 _name, uint256 _chainId) internal pure returns (bytes32 key_) {
        key_ = keccak256(abi.encode(_name, _chainId));
    }
}
