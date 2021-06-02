// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/**
 * @title L1ChugSplashProxy
 * @dev Basic ChugSplash proxy contract for L1. Very close to being a normal proxy but has added
 * functions `setCode` and `setStorage` for changing the code or storage of the contract. Nifty!
 */
contract L1ChugSplashProxy {

    /*************
     * Constants *
     *************/

    // "Magic" prefix. When prepended to some arbitrary bytecode and used to create a contract, the
    // appended bytecode will be deployed as given.
    bytes13 constant internal DEPLOY_CODE_PREFIX = 0x600D380380600D6000396000f3;

    // bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)
    bytes32 constant internal IMPLEMENTATION_KEY = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    // bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)
    bytes32 constant internal OWNER_KEY = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;


    /***************
     * Constructor *
     ***************/
    
    /**
     * @param _owner Address of the initial contract owner.
     */
    constructor(
        address _owner
    ) {
        _setOwner(_owner);
    }


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Makes a proxy call instead of triggering the given function when the caller is either the
     * owner or the zero address. Caller can only ever be the zero address if this function is
     * being called off-chain via eth_call, which is totally fine and can be convenient for
     * client-side tooling. Avoids situations where the proxy and implementation share a sighash
     * and the proxy function ends up being called instead of the implementation one.
     */
    modifier proxyCallIfNotOwner() {
        if (msg.sender == _getOwner() || msg.sender == address(0)) {
            _;
        } else {
            // This WILL halt the call frame on completion.
            _doProxyCall();
        }
    }


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        // Proxy call by default.
        _doProxyCall();
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Sets the code that should be running behind this proxy. Note that this scheme is a bit
     * different from the standard proxy scheme where one would typically deploy the code
     * separately and then set the implementation address. We're doing it this way because it gives
     * us a lot more freedom on the client side. Can only be triggered by the contract owner.
     * @param _code New contract code to run inside this contract.
     */
    function setCode(
        bytes memory _code
    )
        proxyCallIfNotOwner
        public
    {
        // Get the code hash of the current implementation.
        address implementation = _getImplementation();
        bytes32 currentCodeHash;
        assembly {
            currentCodeHash := extcodehash(implementation)
        }

        // If the code hash matches the new implementation then we return early.
        if (keccak256(_code) == currentCodeHash) {
            return;
        }

        // Create the deploycode by appending the magic prefix.
        bytes memory deploycode = abi.encodePacked(
            DEPLOY_CODE_PREFIX,
            _code
        );

        // Deploy the code and set the new implementation address.
        address newImplementation;
        assembly {
            newImplementation := create(0x0, add(deploycode, 0x20), mload(deploycode))
        }
        _setImplementation(newImplementation);
    }

    /**
     * Modifies some storage slot within the proxy contract. Gives us a lot of power to perform
     * upgrades in a more transparent way. Only callable by the owner.
     * @param _key Storage key to modify.
     * @param _value New value for the storage key.
     */
    function setStorage(
        bytes32 _key,
        bytes32 _value
    )
        proxyCallIfNotOwner
        public
    {
        assembly {
            sstore(_key, _value)
        }
    }

    /**
     * Changes the owner of the proxy contract. Only callable by the owner.
     * @param _owner New owner of the proxy contract.
     */
    function setOwner(
        address _owner
    )
        proxyCallIfNotOwner
        public
    {
        _setOwner(_owner);
    }

    /**
     * Queries the owner of the proxy contract. Can only be called by the owner OR by making an
     * eth_call and setting the "from" address to address(0).
     * @return Owner address.
     */
    function getOwner()
        proxyCallIfNotOwner
        public
        returns (
            address
        )
    {
        return _getOwner();
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Sets the implementation address.
     * @param _implementation New implementation address.
     */
    function _setImplementation(
        address _implementation
    )
        internal
    {
        assembly {
            sstore(IMPLEMENTATION_KEY, _implementation)
        }
    }

    /**
     * Queries the implementation address.
     * @return Implementation address.
     */
    function _getImplementation()
        internal
        view
        returns (
            address
        )
    {
        address implementation;
        assembly {
            implementation := sload(IMPLEMENTATION_KEY)
        }
        return implementation;
    }

    /**
     * Changes the owner of the proxy contract.
     * @param _owner New owner of the proxy contract.
     */
    function _setOwner(
        address _owner
    )
        internal
    {
        assembly {
            sstore(OWNER_KEY, _owner)
        }
    }

    /**
     * Queries the owner of the proxy contract.
     * @return Owner address.
     */
    function _getOwner()
        internal
        view 
        returns (
            address
        )
    {
        address owner;
        assembly {
            owner := sload(OWNER_KEY)
        }
        return owner;
    }

    /**
     * Performs the proxy call via a delegatecall.
     */
    function _doProxyCall()
        internal
    {
        address implementation = _getImplementation();

        assembly {
            calldatacopy(0x0, 0x0, calldatasize())
            let result := delegatecall(gas(), implementation, 0x0, calldatasize(), 0x0, 0x0)
            returndatacopy(0x0, 0x0, returndatasize())
            switch result
            case 0x0 {
                revert(0x0, returndatasize())
            }
            default {
                return (0x0, returndatasize())
            }
        }
    }
}
