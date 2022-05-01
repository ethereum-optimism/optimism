pragma solidity 0.8.10;

interface DepositFeed {
    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    ) external payable;
}

contract MultiDepositor {
    DepositFeed df = DepositFeed(0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001);

    constructor(address _df) {
        df = DepositFeed(_df);
    }

    function deposit(address to) external payable {
        for (uint i = 0; i < 3; i++) {
            df.depositTransaction{value : 1000000000}(
                to,
                1000,
                3000000,
                false,
                ""
            );
        }
    }
}