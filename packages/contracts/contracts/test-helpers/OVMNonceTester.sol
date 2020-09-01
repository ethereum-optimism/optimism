pragma solidity ^0.5.0;

contract IExecutionManager {
    function ovmSETNONCE(uint256 _nonce) public;
    function ovmGETNONCE() public returns (uint256);
}

contract OVMNonceTester {
    address executionManagerAddress;

    constructor(
        address _executionManagerAddress
    ) public {
        executionManagerAddress = _executionManagerAddress;
    }

    function setNonce(
        uint256 _nonce
    )
        public
    {
        IExecutionManager(executionManagerAddress).ovmSETNONCE(_nonce);
    }

    function getNonce()
        public
    {
        uint256 nonce = IExecutionManager(executionManagerAddress).ovmGETNONCE();

        // Increment by one to make sure that we're actually getting a value
        // here. Otherwise we really have no way to check the returned value.
        setNonce(
            nonce + 1
        );
    }
}