// SPDX-License-Identifier: MIT
pragma solidity 0.8.12;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./GovernanceToken.sol";

/**
 * @dev Set as `owner` of the OP token and responsible for the token inflation schedule.
 * Contract acts as the token "mint manager" with permission to the `mint` function only.
 * Currently permitted to mint once per year of up to 2% of the total token supply.
 * Upgradable to allow changes in the inflation schedule.
 */
contract MintManager is Ownable {
    GovernanceToken public governanceToken;

    uint256 public constant MINT_CAP = 200; // 2%
    uint256 public constant MINT_PERIOD = 365 days;
    uint256 public mintPermittedAfter;

    constructor(address _upgrader, address _governanceToken) {
        transferOwnership(_upgrader);
        governanceToken = GovernanceToken(_governanceToken);
    }

    /**
     * @param _account Address to mint new tokens to.
     * @param _amount Amount of tokens to be minted.
     * @notice Only the token owner is allowed to mint.
     */
    function mint(address _account, uint256 _amount) public onlyOwner {
        if (mintPermittedAfter > 0) {
            require(mintPermittedAfter <= block.timestamp, "OP: minting not permitted yet");

            require(
                _amount <= (governanceToken.totalSupply() * MINT_CAP) / 1000,
                "OP: mint amount exceeds cap"
            );
        }

        governanceToken.mint(_account, _amount);

        mintPermittedAfter = block.timestamp + MINT_PERIOD;
    }

    function upgrade(address _newMintManager) public onlyOwner {
        require(_newMintManager != address(0), "OP: Mint manager cannot be empty");

        governanceToken.transferOwnership(_newMintManager);
    }
}
