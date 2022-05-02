pragma solidity ^0.8.10;

interface IL2StandardTokenFactory {
    event StandardL2TokenCreated(address indexed _l1Token, address indexed _l2Token);

    function createStandardL2Token(
        address _l1Token,
        string memory _name,
        string memory _symbol
    ) external;
}
