// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.12;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/utils/cryptography/MerkleProof.sol";
import "./interfaces/IMerkleDistributor.sol";

contract MerkleDistributor is IMerkleDistributor {
    address public immutable override token;
    bytes32 public immutable override merkleRoot;

    uint256 public constant ONE_YEAR_IN_SECONDS = 31_536_000;
    uint256 public immutable activationTimestamp;
    address public immutable airdropTreasury;
    bool public isActive;

    // This is a packed array of booleans.
    mapping(uint256 => uint256) private claimedBitMap;

    event Finalised(address indexed calledBy, uint256 timestamp, uint256 unclaimedAmount);

    constructor(
        address token_,
        bytes32 merkleRoot_,
        address _treasury
    ) {
        token = token_;
        merkleRoot = merkleRoot_;

        activationTimestamp = block.timestamp;
        isActive = true;
        airdropTreasury = _treasury;
    }

    function isClaimed(uint256 index) public view override returns (bool) {
        uint256 claimedWordIndex = index / 256;
        uint256 claimedBitIndex = index % 256;
        uint256 claimedWord = claimedBitMap[claimedWordIndex];
        uint256 mask = (1 << claimedBitIndex);
        return claimedWord & mask == mask;
    }

    function _setClaimed(uint256 index) private {
        uint256 claimedWordIndex = index / 256;
        uint256 claimedBitIndex = index % 256;
        claimedBitMap[claimedWordIndex] = claimedBitMap[claimedWordIndex] | (1 << claimedBitIndex);
    }

    function claim(
        uint256 index,
        address account,
        uint256 amount,
        bytes32[] calldata merkleProof
    ) external override {
        require(!isClaimed(index), "MerkleDistributor: Drop already claimed.");

        // Verify the merkle proof.
        bytes32 node = keccak256(abi.encodePacked(index, account, amount));
        require(
            MerkleProof.verify(merkleProof, merkleRoot, node),
            "MerkleDistributor: Invalid proof."
        );

        // Mark it claimed and send the token.
        _setClaimed(index);
        require(IERC20(token).transfer(account, amount), "MerkleDistributor: Transfer failed.");

        emit Claimed(index, account, amount);
    }

    /**
     * @dev Finalises the airdrop and sweeps unclaimed tokens into the Optimism multisig
     */
    function clawBack() external {
        // Airdrop can only be finalised once
        require(isActive, "Airdrop: Already finalised");
        // Airdrop will remain open for one year
        require(
            block.timestamp >= activationTimestamp + ONE_YEAR_IN_SECONDS,
            "Airdrop: Drop should remain open for one year"
        );
        // Deactivate airdrop
        isActive = false;

        // Sweep unclaimed tokens
        uint256 amount = IERC20(token).balanceOf(address(this));
        require(
            IERC20(token).transfer(airdropTreasury, amount),
            "Airdrop: Finalise transfer failed"
        );

        emit Finalised(msg.sender, block.timestamp, amount);
    }
}
