// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "./LinkedList.sol";

/**
 * @title Maintains a sorted list of unsigned ints keyed by bytes32.
 */
library SortedLinkedList {
    using LinkedList for LinkedList.List;

    struct List {
        LinkedList.List list;
        mapping(bytes32 => uint256) values;
    }

    /**
     * @notice Inserts an element into a doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to insert.
     * @param value The element value.
     * @param lesserKey The key of the element less than the element to insert.
     * @param greaterKey The key of the element greater than the element to insert.
     */
    function insert(List storage list, bytes32 key, uint256 value, bytes32 lesserKey, bytes32 greaterKey) internal {
        require(key != bytes32(0) && key != lesserKey && key != greaterKey && !contains(list, key), "invalid key");
        require(
            (lesserKey != bytes32(0) || greaterKey != bytes32(0)) || list.list.numElements == 0,
            "greater and lesser key zero"
        );
        require(contains(list, lesserKey) || lesserKey == bytes32(0), "invalid lesser key");
        require(contains(list, greaterKey) || greaterKey == bytes32(0), "invalid greater key");
        (lesserKey, greaterKey) = getLesserAndGreater(list, value, lesserKey, greaterKey);
        list.list.insert(key, lesserKey, greaterKey);
        list.values[key] = value;
    }

    /**
     * @notice Removes an element from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to remove.
     */
    function remove(List storage list, bytes32 key) internal {
        list.list.remove(key);
        list.values[key] = 0;
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
    function update(List storage list, bytes32 key, uint256 value, bytes32 lesserKey, bytes32 greaterKey) internal {
        remove(list, key);
        insert(list, key, value, lesserKey, greaterKey);
    }

    /**
     * @notice Inserts an element at the tail of the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to insert.
     */
    function push(List storage list, bytes32 key) internal {
        insert(list, key, 0, bytes32(0), list.list.tail);
    }

    /**
     * @notice Removes N elements from the head of the list and returns their keys.
     * @param list A storage pointer to the underlying list.
     * @param n The number of elements to pop.
     * @return The keys of the popped elements.
     */
    function popN(List storage list, uint256 n) internal returns (bytes32[] memory) {
        require(n <= list.list.numElements, "not enough elements");
        bytes32[] memory keys = new bytes32[](n);
        for (uint256 i = 0; i < n; i = i + 1) {
            bytes32 key = list.list.head;
            keys[i] = key;
            remove(list, key);
        }
        return keys;
    }

    /**
     * @notice Returns whether or not a particular key is present in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return Whether or not the key is in the sorted list.
     */
    function contains(List storage list, bytes32 key) internal view returns (bool) {
        return list.list.contains(key);
    }

    /**
     * @notice Returns Element based on key.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return Whether or not the key is in the sorted list.
     */
    function get(List storage list, bytes32 key) internal view returns (LinkedList.Element memory) {
        return list.list.get(key);
    }

    /**
     * @notice Returns the value for a particular key in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return The element value.
     */
    function getValue(List storage list, bytes32 key) internal view returns (uint256) {
        return list.values[key];
    }

    /**
     * @notice Gets all elements from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @return Array of all keys in the list.
     * @return Values corresponding to keys, which will be ordered largest to smallest.
     */
    function getElements(List storage list) internal view returns (bytes32[] memory, uint256[] memory) {
        bytes32[] memory keys = getKeys(list);
        uint256[] memory values = new uint256[](keys.length);
        for (uint256 i = 0; i < keys.length; i = i + 1) {
            values[i] = list.values[keys[i]];
        }
        return (keys, values);
    }

    /**
     * @notice Gets all element keys from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @return All element keys from head to tail.
     */
    function getKeys(List storage list) internal view returns (bytes32[] memory) {
        return list.list.getKeys();
    }

    /**
     * @notice Returns first N greatest elements of the list.
     * @param list A storage pointer to the underlying list.
     * @param n The number of elements to return.
     * @return The keys of the first n elements.
     * @dev Reverts if n is greater than the number of elements in the list.
     */
    function headN(List storage list, uint256 n) internal view returns (bytes32[] memory) {
        return list.list.headN(n);
    }

    /**
     * @notice Returns the keys of the elements greaterKey than and less than the provided value.
     * @param list A storage pointer to the underlying list.
     * @param value The element value.
     * @param lesserKey The key of the element which could be just left of the new value.
     * @param greaterKey The key of the element which could be just right of the new value.
     * @return The correct lesserKey keys.
     * @return The correct greaterKey keys.
     */
    function getLesserAndGreater(
        List storage list,
        uint256 value,
        bytes32 lesserKey,
        bytes32 greaterKey
    )
        private
        view
        returns (bytes32, bytes32)
    {
        // Check for one of the following conditions and fail if none are met:
        //   1. The value is less than the current lowest value
        //   2. The value is greater than the current greatest value
        //   3. The value is just greater than the value for `lesserKey`
        //   4. The value is just less than the value for `greaterKey`
        if (lesserKey == bytes32(0) && isValueBetween(list, value, lesserKey, list.list.tail)) {
            return (lesserKey, list.list.tail);
        } else if (greaterKey == bytes32(0) && isValueBetween(list, value, list.list.head, greaterKey)) {
            return (list.list.head, greaterKey);
        } else if (
            lesserKey != bytes32(0) && isValueBetween(list, value, lesserKey, list.list.elements[lesserKey].nextKey)
        ) {
            return (lesserKey, list.list.elements[lesserKey].nextKey);
        } else if (
            greaterKey != bytes32(0)
                && isValueBetween(list, value, list.list.elements[greaterKey].previousKey, greaterKey)
        ) {
            return (list.list.elements[greaterKey].previousKey, greaterKey);
        }

        require(false, "get lesser and greater failure");
        return (0, 0);
    }

    /**
     * @notice Returns whether or not a given element is between two other elements.
     * @param list A storage pointer to the underlying list.
     * @param value The element value.
     * @param lesserKey The key of the element whose value should be lesserKey.
     * @param greaterKey The key of the element whose value should be greaterKey.
     * @return True if the given element is between the two other elements.
     */
    function isValueBetween(
        List storage list,
        uint256 value,
        bytes32 lesserKey,
        bytes32 greaterKey
    )
        private
        view
        returns (bool)
    {
        bool isLesser = lesserKey == bytes32(0) || list.values[lesserKey] <= value;
        bool isGreater = greaterKey == bytes32(0) || list.values[greaterKey] >= value;
        return isLesser && isGreater;
    }
}
