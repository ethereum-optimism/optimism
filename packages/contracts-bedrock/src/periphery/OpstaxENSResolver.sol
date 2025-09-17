// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { LibString } from "@solady/utils/LibString.sol";

/// @notice Minimal interface for the ENSIP-10 ExtendedResolver.
interface IExtendedResolver {
    function resolve(bytes calldata _name, bytes calldata _data) external view returns (bytes memory);
}

/// @notice Minimal subset of ERC-165 for interface detection.
interface IERC165 {
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
}

/// @title OpstaxENSResolver
/// @notice Resolves names of the form `<chain>-<contract>.opstax.eth` into addresses.
///         A privileged owner can register the `SystemConfig` for each chain by its name.
///         Contract addresses are then resolved by name via the registered `SystemConfig`.
contract OpstaxENSResolver is Owned, ISemver, IExtendedResolver, IERC165 {
    /// @notice Emitted when a SystemConfig is set for a chain.
    /// @param chainName The lowercase chain name label (e.g. "op", "base").
    /// @param systemConfig The SystemConfig address set for the chain.
    event ChainSystemConfigSet(string indexed chainName, address indexed systemConfig);

    /// @notice Thrown when attempting to resolve a name that does not match `<chain>-<contract>.opstax.eth`.
    error OpstaxENSResolver_InvalidName();

    /// @notice Thrown when a chain does not have a registered `SystemConfig`.
    error OpstaxENSResolver_UnknownChain();

    /// @notice Thrown when a requested contract label is unknown.
    error OpstaxENSResolver_UnknownContract();

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    function version() public pure virtual returns (string memory) {
        return "1.0.0";
    }

    /// @notice Mapping from lowercase chain name to SystemConfig address.
    mapping(string => address) public chainNameToSystemConfig;

    /// @param _owner Initial owner of the resolver.
    constructor(address _owner) Owned(_owner) { }

    /// @notice Sets the `SystemConfig` for a given chain name. Only callable by the owner.
    /// @param _chainName Lowercase chain label (e.g. "op", "base").
    /// @param _systemConfig Address of the `SystemConfig` for the chain.
    function setChainSystemConfig(string calldata _chainName, address _systemConfig) external onlyOwner {
        chainNameToSystemConfig[_chainName] = _systemConfig;
        emit ChainSystemConfigSet(_chainName, _systemConfig);
    }

    /// @inheritdoc IExtendedResolver
    function resolve(bytes calldata _name, bytes calldata _data) external view returns (bytes memory) {
        // Expecting `addr(bytes32)` calls as per ENS resolver ABI
        if (_data.length < 4) revert OpstaxENSResolver_InvalidName();
        bytes4 selector;
        assembly {
            selector := calldataload(_data.offset)
        }

        // addr(bytes32) selector
        if (selector != 0x3b3b57de) revert OpstaxENSResolver_InvalidName();

        (string memory contractLabel, string memory chainLabel) = _parseName(_name);

        address systemConfigAddr = chainNameToSystemConfig[chainLabel];
        if (systemConfigAddr == address(0)) revert OpstaxENSResolver_UnknownChain();

        address resolved = _resolveContractAddress(contractLabel, SystemConfig(systemConfigAddr));
        if (resolved == address(0)) revert OpstaxENSResolver_UnknownContract();

        return abi.encode(resolved);
    }

    /// @notice Parses a DNS-encoded ENS name into `<chain>` and `<contract>` labels.
    ///         Expects format `<chain>-<contract>.opstax.eth` and splits the first label on '-'.
    function _parseName(bytes calldata _name) internal pure returns (string memory contractLabel_, string memory chainLabel_) {
        // The name is DNS-encoded: [len][label]...[0]
        // We minimally parse into labels and validate suffix.
        bytes[] memory labels = _splitDnsName(_name);
        if (labels.length != 3) revert OpstaxENSResolver_InvalidName();

        // labels are ordered left-to-right: [chain-contract, opstax, eth]
        string memory l1 = LibString.lower(string(labels[1]));
        string memory l2 = LibString.lower(string(labels[2]));
        if (!LibString.eq(l1, "opstax") || !LibString.eq(l2, "eth")) {
            revert OpstaxENSResolver_InvalidName();
        }

        // Split the first label on '-' to get chain and contract
        string memory firstLabel = LibString.lower(string(labels[0]));
        string[] memory parts = LibString.split(firstLabel, "-");
        if (parts.length != 2) revert OpstaxENSResolver_InvalidName();

        chainLabel_ = parts[0];
        contractLabel_ = parts[1];
    }

    /// @notice Resolves a contract label via SystemConfig getters.
    function _resolveContractAddress(string memory _contractLabel, SystemConfig _systemConfig)
        internal
        view
        returns (address)
    {
        bytes32 h = keccak256(bytes(_contractLabel));

        // Special-case: the `systemconfig` label resolves to the SystemConfig itself
        if (h == keccak256("systemconfig")) return address(_systemConfig);
        if (h == keccak256("superchainconfig")) return address(_systemConfig.superchainConfig());
        if (h == keccak256("l1crossdomainmessenger")) return _systemConfig.l1CrossDomainMessenger();
        if (h == keccak256("l1erc721bridge")) return _systemConfig.l1ERC721Bridge();
        if (h == keccak256("l1standardbridge")) return _systemConfig.l1StandardBridge();
        if (h == keccak256("optimismportal")) return _systemConfig.optimismPortal();
        if (h == keccak256("optimismmintableerc20factory")) return _systemConfig.optimismMintableERC20Factory();
        if (h == keccak256("disputegamefactory")) return _systemConfig.disputeGameFactory();

        return address(0);
    }

    /// @notice Minimal DNS name splitter for ENS-encoded names.
    ///         Returns an array of raw label byte-slices (not lowercased).
    function _splitDnsName(bytes calldata name)
        internal
        pure
        returns (bytes[] memory labels_)
    {
        // Count labels
        uint256 i = 0;
        uint256 count = 0;
        while (true) {
            if (i >= name.length) revert OpstaxENSResolver_InvalidName();
            uint256 len = uint8(name[i]);
            i++;
            if (len == 0) break;
            if (i + len > name.length) revert OpstaxENSResolver_InvalidName();
            count++;
            i += len;
        }

        labels_ = new bytes[](count);

        // Extract labels
        i = 0;
        uint256 idx = 0;
        while (true) {
            uint256 len = uint8(name[i]);
            i++;
            if (len == 0) break;
            labels_[idx] = name[i:i+len];
            idx++;
            i += len;
        }
    }



    /// @inheritdoc IERC165
    function supportsInterface(bytes4 interfaceId) external pure returns (bool) {
        // IExtendedResolver
        if (interfaceId == 0x9061b923) return true;
        // ERC165
        if (interfaceId == 0x01ffc9a7) return true;
        return false;
    }
}

