// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title Proxy
 * @notice Proxy is a transparent proxy that passes through the call
 *         if the caller is the owner or if the caller is `address(0)`,
 *         meaning that the call originated from an offchain simulation.
 */
contract Proxy {
    /**
     * @notice An event that is emitted each time the implementation is changed.
     *         This event is part of the EIP 1967 spec.
     *
     * @param implementation The address of the implementation contract
     */
    event Upgraded(address indexed implementation);

    /**
     * @notice An event that is emitted each time the owner is upgraded.
     *         This event is part of the EIP 1967 spec.
     *
     * @param previousAdmin The previous owner of the contract
     * @param newAdmin      The new owner of the contract
     */
    event AdminChanged(address previousAdmin, address newAdmin);

    /**
     * @notice The storage slot that holds the address of the implementation.
     *         bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)
     */
    bytes32 internal constant IMPLEMENTATION_KEY =
        0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    /**
     * @notice The storage slot that holds the address of the owner.
     *         bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)
     */
    bytes32 internal constant OWNER_KEY =
        0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;

    /**
     * @notice set the initial owner during contract deployment. The
     *         owner is stored at the eip1967 owner storage slot so that
     *         storage collision with the implementation is not possible.
     *
     * @param _admin Address of the initial contract owner. The owner has
     *               the ability to access the transparent proxy interface.
     */
    constructor(address _admin) {
        _changeAdmin(_admin);
    }

    // slither-disable-next-line locked-ether
    fallback() external payable {
        // Proxy call by default.
        _doProxyCall();
    }

    /**
     * @notice A modifier that reverts if not called by the owner
     *         or by `address(0)` to allow `eth_call` to interact
     *         with the proxy without needing to use low level storage
     *         inspection. It is assumed that nobody controls the private
     *         key for `address(0)`.
     */
    modifier proxyCallIfNotAdmin() {
        if (msg.sender == _getAdmin() || msg.sender == address(0)) {
            _;
        } else {
            // This WILL halt the call frame on completion.
            _doProxyCall();
        }
    }

    /**
     * @notice Set the implementation contract address. The code at this
     *         address will execute when this contract is called.
     *
     * @param _implementation The address of the implementation contract
     */
    function upgradeTo(address _implementation) external proxyCallIfNotAdmin {
        _setImplementation(_implementation);
    }

    /**
     * @notice Set the implementation and call a function in a single
     *         transaction. This is useful to ensure atomic `initialize()`
     *         based upgrades.
     *
     * @param _implementation The address of the implementation contract
     * @param _data           The calldata to delegatecall the new
     *                        implementation with
     */
    function upgradeToAndCall(address _implementation, bytes calldata _data)
        external
        payable
        proxyCallIfNotAdmin
        returns (bytes memory)
    {
        _setImplementation(_implementation);
        (bool success, bytes memory returndata) = _implementation.delegatecall(_data);
        require(success);
        return returndata;
    }

    /**
     * @notice Changes the owner of the proxy contract. Only callable by the owner.
     *
     * @param _admin New owner of the proxy contract.
     */
    function changeAdmin(address _admin) external proxyCallIfNotAdmin {
        _changeAdmin(_admin);
    }

    /**
     * @notice Gets the owner of the proxy contract.
     *
     * @return Owner address.
     */
    function admin() external proxyCallIfNotAdmin returns (address) {
        return _getAdmin();
    }

    /**
     * @notice Queries the implementation address.
     *
     * @return Implementation address.
     */
    function implementation() external proxyCallIfNotAdmin returns (address) {
        return _getImplementation();
    }

    /**
     * @notice Sets the implementation address.
     *
     * @param _implementation New implementation address.
     */
    function _setImplementation(address _implementation) internal {
        assembly {
            sstore(IMPLEMENTATION_KEY, _implementation)
        }
        emit Upgraded(_implementation);
    }

    /**
     * @notice Queries the implementation address.
     *
     * @return implementation address.
     */
    function _getImplementation() internal view returns (address) {
        address implementation;
        assembly {
            implementation := sload(IMPLEMENTATION_KEY)
        }
        return implementation;
    }

    /**
     * @notice Changes the owner of the proxy contract.
     *
     * @param _admin New owner of the proxy contract.
     */
    function _changeAdmin(address _admin) internal {
        address previous = _getAdmin();
        assembly {
            sstore(OWNER_KEY, _admin)
        }
        emit AdminChanged(previous, _admin);
    }

    /**
     * @notice Queries the owner of the proxy contract.
     *
     * @return owner address.
     */
    function _getAdmin() internal view returns (address) {
        address owner;
        assembly {
            owner := sload(OWNER_KEY)
        }
        return owner;
    }

    /**
     * @notice Performs the proxy call via a delegatecall.
     */
    function _doProxyCall() internal {
        address implementation = _getImplementation();

        require(implementation != address(0), "Proxy: implementation not initialized");

        assembly {
            // Copy calldata into memory at 0x0....calldatasize.
            calldatacopy(0x0, 0x0, calldatasize())

            // Perform the delegatecall, make sure to pass all available gas.
            let success := delegatecall(gas(), implementation, 0x0, calldatasize(), 0x0, 0x0)

            // Copy returndata into memory at 0x0....returndatasize. Note that this *will*
            // overwrite the calldata that we just copied into memory but that doesn't really
            // matter because we'll be returning in a second anyway.
            returndatacopy(0x0, 0x0, returndatasize())

            // Success == 0 means a revert. We'll revert too and pass the data up.
            if iszero(success) {
                revert(0x0, returndatasize())
            }

            // Otherwise we'll just return and pass the data up.
            return(0x0, returndatasize())
        }
    }
}
