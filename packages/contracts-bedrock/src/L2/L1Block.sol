// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { Constants } from "src/libraries/Constants.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";
import { GasPayingToken, IGasToken } from "src/libraries/GasPayingToken.sol";
import { IFeeVault, Types as ITypes } from "src/L2/interfaces/IFeeVault.sol";
import { ICrossDomainMessenger } from "src/universal/interfaces/ICrossDomainMessenger.sol";
import { IStandardBridge } from "src/universal/interfaces/IStandardBridge.sol";
import { IERC721Bridge } from "src/universal/interfaces/IERC721Bridge.sol";
import { IOptimismMintableERC721Factory } from "src/L2/interfaces/IOptimismMintableERC721Factory.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Storage } from "src/libraries/Storage.sol";
import { Types } from "src/libraries/Types.sol";
import "src/libraries/L1BlockErrors.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1Block
/// @notice The L1Block predeploy gives users access to information about the last known L1 block.
///         Values within this contract are updated once per epoch (every L1 block) and can only be
///         set by the "depositor" account, a special system address. Depositor account transactions
///         are created by the protocol whenever we move to a new epoch.
contract L1Block is ISemver, IGasToken {
    /// @notice Event emitted when the gas paying token is set.
    event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol);

    /// @notice
    bytes32 internal constant BASE_FEE_VAULT_CONFIG_SLOT = bytes32(uint256(keccak256("opstack.basefeevaultconfig")) - 1);
    /// @notice
    bytes32 internal constant L1_FEE_VAULT_CONFIG_SLOT = bytes32(uint256(keccak256("opstack.l1feevaultconfig")) - 1);
    /// @notice
    bytes32 internal constant SEQUENCER_FEE_VAULT_CONFIG_SLOT =
        bytes32(uint256(keccak256("opstack.sequencerfeevaultconfig")) - 1);
    /// @notice
    bytes32 internal constant L1_CROSS_DOMAIN_MESSENGER_ADDRESS_SLOT =
        bytes32(uint256(keccak256("opstack.l1crossdomainmessengeraddress")) - 1);
    /// @notice
    bytes32 internal constant L1_ERC_721_BRIDGE_ADDRESS_SLOT =
        bytes32(uint256(keccak256("opstack.l1erc721bridgeaddress")) - 1);
    /// @notice
    bytes32 internal constant L1_STANDARD_BRIDGE_ADDRESS_SLOT =
        bytes32(uint256(keccak256("opstack.l1standardbridgeaddress")) - 1);
    /// @notice
    bytes32 internal constant REMOTE_CHAIN_ID_SLOT = bytes32(uint256(keccak256("opstack.remotechainid")) - 1);

    /// @notice Address of the special depositor account.
    function DEPOSITOR_ACCOUNT() public pure returns (address addr_) {
        addr_ = Constants.DEPOSITOR_ACCOUNT;
    }

    /// @notice The latest L1 block number known by the L2 system.
    uint64 public number;

    /// @notice The latest L1 timestamp known by the L2 system.
    uint64 public timestamp;

    /// @notice The latest L1 base fee.
    uint256 public basefee;

    /// @notice The latest L1 blockhash.
    bytes32 public hash;

    /// @notice The number of L2 blocks in the same epoch.
    uint64 public sequenceNumber;

    /// @notice The scalar value applied to the L1 blob base fee portion of the blob-capable L1 cost func.
    uint32 public blobBaseFeeScalar;

    /// @notice The scalar value applied to the L1 base fee portion of the blob-capable L1 cost func.
    uint32 public baseFeeScalar;

    /// @notice The versioned hash to authenticate the batcher by.
    bytes32 public batcherHash;

    /// @notice The overhead value applied to the L1 portion of the transaction fee.
    /// @custom:legacy
    uint256 public l1FeeOverhead;

    /// @notice The scalar value applied to the L1 portion of the transaction fee.
    /// @custom:legacy
    uint256 public l1FeeScalar;

    /// @notice The latest L1 blob base fee.
    uint256 public blobBaseFee;

    /// @custom:semver 1.5.1-beta.2
    function version() public pure virtual returns (string memory) {
        return "1.5.1-beta.2";
    }

    /// @notice Returns the gas paying token, its decimals, name and symbol.
    ///         If nothing is set in state, then it means ether is used.
    function gasPayingToken() public view returns (address addr_, uint8 decimals_) {
        (addr_, decimals_) = GasPayingToken.getToken();
    }

    /// @notice Returns the gas paying token name.
    ///         If nothing is set in state, then it means ether is used.
    function gasPayingTokenName() public view returns (string memory name_) {
        name_ = GasPayingToken.getName();
    }

    /// @notice Returns the gas paying token symbol.
    ///         If nothing is set in state, then it means ether is used.
    function gasPayingTokenSymbol() public view returns (string memory symbol_) {
        symbol_ = GasPayingToken.getSymbol();
    }

    /// @notice Getter for custom gas token paying networks. Returns true if the
    ///         network uses a custom gas token.
    function isCustomGasToken() public view returns (bool) {
        (address token,) = gasPayingToken();
        return token != Constants.ETHER;
    }

    /// @custom:legacy
    /// @notice Updates the L1 block values.
    /// @param _number         L1 blocknumber.
    /// @param _timestamp      L1 timestamp.
    /// @param _basefee        L1 basefee.
    /// @param _hash           L1 blockhash.
    /// @param _sequenceNumber Number of L2 blocks since epoch start.
    /// @param _batcherHash    Versioned hash to authenticate batcher by.
    /// @param _l1FeeOverhead  L1 fee overhead.
    /// @param _l1FeeScalar    L1 fee scalar.
    function setL1BlockValues(
        uint64 _number,
        uint64 _timestamp,
        uint256 _basefee,
        bytes32 _hash,
        uint64 _sequenceNumber,
        bytes32 _batcherHash,
        uint256 _l1FeeOverhead,
        uint256 _l1FeeScalar
    )
        external
    {
        require(msg.sender == DEPOSITOR_ACCOUNT(), "L1Block: only the depositor account can set L1 block values");

        number = _number;
        timestamp = _timestamp;
        basefee = _basefee;
        hash = _hash;
        sequenceNumber = _sequenceNumber;
        batcherHash = _batcherHash;
        l1FeeOverhead = _l1FeeOverhead;
        l1FeeScalar = _l1FeeScalar;
    }

    /// @notice Updates the L1 block values for an Ecotone upgraded chain.
    /// Params are packed and passed in as raw msg.data instead of ABI to reduce calldata size.
    /// Params are expected to be in the following order:
    ///   1. _baseFeeScalar      L1 base fee scalar
    ///   2. _blobBaseFeeScalar  L1 blob base fee scalar
    ///   3. _sequenceNumber     Number of L2 blocks since epoch start.
    ///   4. _timestamp          L1 timestamp.
    ///   5. _number             L1 blocknumber.
    ///   6. _basefee            L1 base fee.
    ///   7. _blobBaseFee        L1 blob base fee.
    ///   8. _hash               L1 blockhash.
    ///   9. _batcherHash        Versioned hash to authenticate batcher by.
    function setL1BlockValuesEcotone() public {
        _setL1BlockValuesEcotone();
    }

    /// @notice Updates the L1 block values for an Ecotone upgraded chain.
    /// Params are packed and passed in as raw msg.data instead of ABI to reduce calldata size.
    /// Params are expected to be in the following order:
    ///   1. _baseFeeScalar      L1 base fee scalar
    ///   2. _blobBaseFeeScalar  L1 blob base fee scalar
    ///   3. _sequenceNumber     Number of L2 blocks since epoch start.
    ///   4. _timestamp          L1 timestamp.
    ///   5. _number             L1 blocknumber.
    ///   6. _basefee            L1 base fee.
    ///   7. _blobBaseFee        L1 blob base fee.
    ///   8. _hash               L1 blockhash.
    ///   9. _batcherHash        Versioned hash to authenticate batcher by.
    function _setL1BlockValuesEcotone() internal {
        address depositor = DEPOSITOR_ACCOUNT();
        assembly {
            // Revert if the caller is not the depositor account.
            if xor(caller(), depositor) {
                mstore(0x00, 0x3cc50b45) // 0x3cc50b45 is the 4-byte selector of "NotDepositor()"
                revert(0x1C, 0x04) // returns the stored 4-byte selector from above
            }
            // sequencenum (uint64), blobBaseFeeScalar (uint32), baseFeeScalar (uint32)
            sstore(sequenceNumber.slot, shr(128, calldataload(4)))
            // number (uint64) and timestamp (uint64)
            sstore(number.slot, shr(128, calldataload(20)))
            sstore(basefee.slot, calldataload(36)) // uint256
            sstore(blobBaseFee.slot, calldataload(68)) // uint256
            sstore(hash.slot, calldataload(100)) // bytes32
            sstore(batcherHash.slot, calldataload(132)) // bytes32
        }
    }

    /// @notice Sets static configuration options for the L2 system. Can only be called by the special
    ///         depositor account.
    /// @param _type  The type of configuration to set.
    /// @param _value The encoded value with which to set the configuration.
    function setConfig(Types.ConfigType _type, bytes calldata _value) public virtual {
        if (msg.sender != DEPOSITOR_ACCOUNT()) revert NotDepositor();

        // TODO: sort out StaticType library encoding/decoding vs just using abi.encode/decode
        if (_type == Types.ConfigType.SET_GAS_PAYING_TOKEN) {
            _setGasPayingToken(_value);
        } else if (_type == Types.ConfigType.SET_BASE_FEE_VAULT_CONFIG) {
            Storage.setBytes32(BASE_FEE_VAULT_CONFIG_SLOT, abi.decode(_value, (bytes32)));
        } else if (_type == Types.ConfigType.SET_L1_ERC_721_BRIDGE_ADDRESS) {
            Storage.setAddress(L1_ERC_721_BRIDGE_ADDRESS_SLOT, abi.decode(_value, (address)));
        } else if (_type == Types.ConfigType.SET_REMOTE_CHAIN_ID) {
            Storage.setUint(REMOTE_CHAIN_ID_SLOT, abi.decode(_value, (uint256)));
        } else if (_type == Types.ConfigType.SET_L1_CROSS_DOMAIN_MESSENGER_ADDRESS) {
            Storage.setAddress(L1_CROSS_DOMAIN_MESSENGER_ADDRESS_SLOT, abi.decode(_value, (address)));
        } else if (_type == Types.ConfigType.SET_L1_STANDARD_BRIDGE_ADDRESS) {
            Storage.setAddress(L1_STANDARD_BRIDGE_ADDRESS_SLOT, abi.decode(_value, (address)));
        } else if (_type == Types.ConfigType.SET_SEQUENCER_FEE_VAULT_CONFIG) {
            Storage.setBytes32(SEQUENCER_FEE_VAULT_CONFIG_SLOT, abi.decode(_value, (bytes32)));
        }
    }

    /// @notice Internal method to set the gas paying token.
    /// @param _value The encoded value with which to set the gas paying token.
    function _setGasPayingToken(bytes calldata _value) internal {
        (address token, uint8 decimals, bytes32 name, bytes32 symbol) = StaticConfig.decodeSetGasPayingToken(_value);

        GasPayingToken.set({ _token: token, _decimals: decimals, _name: name, _symbol: symbol });

        emit GasPayingTokenSet({ token: token, decimals: decimals, name: name, symbol: symbol });
    }

    /// @notice Sets the gas paying token for the L2 system. Can only be called by the special
    ///         depositor account, initiated by a deposit transaction from L1.
    function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external {
        if (msg.sender != DEPOSITOR_ACCOUNT()) revert NotDepositor();

        GasPayingToken.set({ _token: _token, _decimals: _decimals, _name: _name, _symbol: _symbol });

        emit GasPayingTokenSet({ token: _token, decimals: _decimals, name: _name, symbol: _symbol });
    }

    /// @notice
    function setHolocene() external {
        if (msg.sender != DEPOSITOR_ACCOUNT()) revert NotDepositor();
        // TODO set holocene activation to true

        Storage.setBytes32(BASE_FEE_VAULT_CONFIG_SLOT, _feeVaultConfig(Predeploys.BASE_FEE_VAULT));
        Storage.setBytes32(L1_FEE_VAULT_CONFIG_SLOT, _feeVaultConfig(Predeploys.L1_FEE_VAULT));
        Storage.setBytes32(SEQUENCER_FEE_VAULT_CONFIG_SLOT, _feeVaultConfig(Predeploys.SEQUENCER_FEE_WALLET));
        Storage.setAddress(
            L1_CROSS_DOMAIN_MESSENGER_ADDRESS_SLOT,
            address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger())
        );
        Storage.setAddress(
            L1_STANDARD_BRIDGE_ADDRESS_SLOT,
            address(IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).otherBridge())
        );
        Storage.setAddress(
            L1_ERC_721_BRIDGE_ADDRESS_SLOT, address(IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).otherBridge())
        );
        Storage.setUint(
            REMOTE_CHAIN_ID_SLOT, IOptimismMintableERC721Factory(Predeploys.L2_ERC721_BRIDGE).remoteChainId()
        );
    }

    /// @notice
    function baseFeeVaultConfig()
        public
        view
        returns (address recipient_, uint256 amount_, Types.WithdrawalNetwork network_)
    {
        (recipient_, amount_, network_) = Encoding.decodeFeeVaultConfig(Storage.getBytes32(BASE_FEE_VAULT_CONFIG_SLOT));
    }

    /// @notice
    function l1FeeVaultConfig()
        public
        view
        returns (address recipient_, uint256 amount_, Types.WithdrawalNetwork network_)
    {
        (recipient_, amount_, network_) = Encoding.decodeFeeVaultConfig(Storage.getBytes32(L1_FEE_VAULT_CONFIG_SLOT));
    }

    /// @notice
    function sequencerFeeVaultConfig()
        public
        view
        returns (address recipient_, uint256 amount_, Types.WithdrawalNetwork network_)
    {
        (recipient_, amount_, network_) =
            Encoding.decodeFeeVaultConfig(Storage.getBytes32(SEQUENCER_FEE_VAULT_CONFIG_SLOT));
    }

    /// @notice
    function l1CrossDomainMessenger() public view returns (address) {
        return Storage.getAddress(L1_CROSS_DOMAIN_MESSENGER_ADDRESS_SLOT);
    }

    /// @notice
    function l1StandardBridge() public view returns (address) {
        return Storage.getAddress(L1_STANDARD_BRIDGE_ADDRESS_SLOT);
    }

    /// @notice
    function l1ERC721Bridge() public view returns (address) {
        return Storage.getAddress(L1_ERC_721_BRIDGE_ADDRESS_SLOT);
    }

    /// @notice
    function remoteChainId() public view returns (uint256) {
        return Storage.getUint(REMOTE_CHAIN_ID_SLOT);
    }

    /// @notice perhaps rename this to something migration related
    function _feeVaultConfig(address _addr) internal view returns (bytes32) {
        // compiler issue with type here
        (address recipient, uint256 amount, ITypes.WithdrawalNetwork network) = IFeeVault(payable(_addr)).config();
        return Encoding.encodeFeeVaultConfig(recipient, amount, Types.WithdrawalNetwork(uint8(network)));
    }
}
