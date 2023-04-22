/// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title BondManager
/// @notice The Bond Manager serves as an escrow for permissionless output proposal bonds.
interface IBondManager {

    // The Bond Type
    struct Bond {
        address owner;
        uint64 expiration;
        bytes32 bondId;
        uint256 amount;
    }

    /// @notice Mapping from bondId to amount.
    mapping(bondId => Bond) internal bonds;

    /// @notice BondPosted is emitted when a bond is posted.
    event BondPosted(bytes32 bondId, address owner, uint64 expiration, uint256 amount);

  /// @notice Post a bond with a given id and owner.
  /// @dev This function will revert if the provided bondId is already in use.
  /// @param bondId is the id of the bond.
  /// @param owner is the address that owns the bond.
  /// @param minClaimHold is the minimum amount of time the owner must wait before reclaiming their bond.
  function post(bytes32 bondId, address owner, uint64 minClaimHold) external payable {
    require(bonds[bondId].owner == address(0), "BondManager: BondId already posted.");
    require(owner != address(0), "BondManager: Owner cannot be the zero address.");
    require(msg.value > 0, "BondManager: Value must be non-zero.");

    bonds[bondId] = Bond{
        owner: msg.sender,
        expiration: block.timestamp + minClaimHold,
        bondId: bondId,
        amount: msg.value
    };

    emit BondPosted(bondId, owner, minClaimHold, msg.value);
  }

  /// @notice Seizes the bond with the given id.
  /// @dev This function will revert if there is no bond at the given id.
  /// @param bondId is the id of the bond.
  function seize(bytes32 bondId) external {

  }

  /// @notice Seizes the bond with the given id and distributes it to recipients.
  /// @dev This function will revert if there is no bond at the given id.
  /// @param bondId is the id of the bond.
  /// @param recipients is a set of addresses to split the bond amongst.
  function seizeAndSplit(bytes32 bondId, address[] calldata recipients) external {

  }

  /// @notice Reclaims the bond of the bond owner.
  /// @dev This function will revert if there is no bond at the given id.
  /// @param bondId is the id of the bond.
  function reclaim(bytes32 bondId) external {

  }
}
