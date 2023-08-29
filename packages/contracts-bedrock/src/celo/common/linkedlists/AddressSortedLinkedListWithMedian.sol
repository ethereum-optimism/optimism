// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "./SortedLinkedListWithMedian.sol";

/**
 * @title Maintains a sorted list of unsigned ints keyed by address.
 */
library AddressSortedLinkedListWithMedian {
    using SortedLinkedListWithMedian for SortedLinkedListWithMedian.List;

    function toBytes(address a) public pure returns (bytes32) {
        return bytes32(uint256(uint160(a)) << 96);
    }

    function toAddress(bytes32 b) public pure returns (address) {
        return address(uint160(uint256(b) >> 96));
    }

    /**
     * @notice Inserts an element into a doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to insert.
     * @param value The element value.
     * @param lesserKey The key of the element less than the element to insert.
     * @param greaterKey The key of the element greater than the element to insert.
     */
    function insert(
        SortedLinkedListWithMedian.List storage list,
        address key,
        uint256 value,
        address lesserKey,
        address greaterKey
    )
        public
    {
        list.insert(toBytes(key), value, toBytes(lesserKey), toBytes(greaterKey));
    }

    /**
     * @notice Removes an element from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to remove.
     */
    function remove(SortedLinkedListWithMedian.List storage list, address key) public {
        list.remove(toBytes(key));
    }

    /**
     * @notice Updates an element in the list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @param value The element value.
     * @param lesserKey The key of the element will be just left of `key` after the update.
     * @param greaterKey The key of the element will be just right of `key` after the update.
     * @dev Note that only one of "lesserKey" or "greaterKey" needs to be correct to reduce friction.
     */
    function update(
        SortedLinkedListWithMedian.List storage list,
        address key,
        uint256 value,
        address lesserKey,
        address greaterKey
    )
        public
    {
        list.update(toBytes(key), value, toBytes(lesserKey), toBytes(greaterKey));
    }

    /**
     * @notice Returns whether or not a particular key is present in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return Whether or not the key is in the sorted list.
     */
    function contains(SortedLinkedListWithMedian.List storage list, address key) public view returns (bool) {
        return list.contains(toBytes(key));
    }

    /**
     * @notice Returns the value for a particular key in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return The element value.
     */
    function getValue(SortedLinkedListWithMedian.List storage list, address key) public view returns (uint256) {
        return list.getValue(toBytes(key));
    }

    /**
     * @notice Returns the median value of the sorted list.
     * @param list A storage pointer to the underlying list.
     * @return The median value.
     */
    function getMedianValue(SortedLinkedListWithMedian.List storage list) public view returns (uint256) {
        return list.getValue(list.median);
    }

    /**
     * @notice Returns the key of the first element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the first element in the list.
     */
    function getHead(SortedLinkedListWithMedian.List storage list) external view returns (address) {
        return toAddress(list.getHead());
    }

    /**
     * @notice Returns the key of the median element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the median element in the list.
     */
    function getMedian(SortedLinkedListWithMedian.List storage list) external view returns (address) {
        return toAddress(list.getMedian());
    }

    /**
     * @notice Returns the key of the last element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the last element in the list.
     */
    function getTail(SortedLinkedListWithMedian.List storage list) external view returns (address) {
        return toAddress(list.getTail());
    }

    /**
     * @notice Returns the number of elements in the list.
     * @param list A storage pointer to the underlying list.
     * @return The number of elements in the list.
     */
    function getNumElements(SortedLinkedListWithMedian.List storage list) external view returns (uint256) {
        return list.getNumElements();
    }

    /**
     * @notice Gets all elements from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @return Array of all keys in the list.
     * @return Values corresponding to keys, which will be ordered largest to smallest.
     * @return Array of relations to median of corresponding list elements.
     */
    function getElements(SortedLinkedListWithMedian.List storage list)
        public
        view
        returns (address[] memory, uint256[] memory, SortedLinkedListWithMedian.MedianRelation[] memory)
    {
        bytes32[] memory byteKeys = list.getKeys();
        address[] memory keys = new address[](byteKeys.length);
        uint256[] memory values = new uint256[](byteKeys.length);
        // prettier-ignore
        SortedLinkedListWithMedian.MedianRelation[] memory relations =
            new SortedLinkedListWithMedian.MedianRelation[](keys.length);
        for (uint256 i = 0; i < byteKeys.length; i++) {
            keys[i] = toAddress(byteKeys[i]);
            values[i] = list.getValue(byteKeys[i]);
            relations[i] = list.relation[byteKeys[i]];
        }
        return (keys, values, relations);
    }
}
