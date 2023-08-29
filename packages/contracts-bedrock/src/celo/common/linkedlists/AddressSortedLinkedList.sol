// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "@openzeppelin/contracts/utils/math/Math.sol";

import "./SortedLinkedList.sol";

/**
 * @title Maintains a sorted list of unsigned ints keyed by address.
 */
library AddressSortedLinkedList {
    using SortedLinkedList for SortedLinkedList.List;

    /**
     * @notice Inserts an element into a doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to insert.
     * @param value The element value.
     * @param lesserKey The key of the element less than the element to insert.
     * @param greaterKey The key of the element greater than the element to insert.
     */
    function insert(
        SortedLinkedList.List storage list,
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
    function remove(SortedLinkedList.List storage list, address key) public {
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
        SortedLinkedList.List storage list,
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
    function contains(SortedLinkedList.List storage list, address key) public view returns (bool) {
        return list.contains(toBytes(key));
    }

    /**
     * @notice Returns the value for a particular key in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return The element value.
     */
    function getValue(SortedLinkedList.List storage list, address key) public view returns (uint256) {
        return list.getValue(toBytes(key));
    }

    /**
     * @notice Gets all elements from the doubly linked list.
     * @return Array of all keys in the list.
     * @return Values corresponding to keys, which will be ordered largest to smallest.
     */
    function getElements(SortedLinkedList.List storage list) public view returns (address[] memory, uint256[] memory) {
        bytes32[] memory byteKeys = list.getKeys();
        address[] memory keys = new address[](byteKeys.length);
        uint256[] memory values = new uint256[](byteKeys.length);
        for (uint256 i = 0; i < byteKeys.length; i = i + 1) {
            keys[i] = toAddress(byteKeys[i]);
            values[i] = list.values[byteKeys[i]];
        }
        return (keys, values);
    }

    /**
     * @notice Returns the minimum of `max` and the  number of elements in the list > threshold.
     * @param list A storage pointer to the underlying list.
     * @param threshold The number that the element must exceed to be included.
     * @param max The maximum number returned by this function.
     * @return The minimum of `max` and the  number of elements in the list > threshold.
     */
    function numElementsGreaterThan(
        SortedLinkedList.List storage list,
        uint256 threshold,
        uint256 max
    )
        public
        view
        returns (uint256)
    {
        uint256 revisedMax = Math.min(max, list.list.numElements);
        bytes32 key = list.list.head;
        for (uint256 i = 0; i < revisedMax; i = i + 1) {
            if (list.getValue(key) < threshold) {
                return i;
            }
            key = list.list.elements[key].previousKey;
        }
        return revisedMax;
    }

    /**
     * @notice Returns the N greatest elements of the list.
     * @param list A storage pointer to the underlying list.
     * @param n The number of elements to return.
     * @return The keys of the greatest elements.
     */
    function headN(SortedLinkedList.List storage list, uint256 n) public view returns (address[] memory) {
        bytes32[] memory byteKeys = list.headN(n);
        address[] memory keys = new address[](n);
        for (uint256 i = 0; i < n; i = i + 1) {
            keys[i] = toAddress(byteKeys[i]);
        }
        return keys;
    }

    /**
     * @notice Gets all element keys from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @return All element keys from head to tail.
     */
    function getKeys(SortedLinkedList.List storage list) public view returns (address[] memory) {
        return headN(list, list.list.numElements);
    }

    /**
     * @notice Returns the number of elements in the list.
     * @param list A storage pointer to the underlying list.
     * @return The number of elements in the list.
     */
    function getNumElements(SortedLinkedList.List storage list) public view returns (uint256) {
        return list.list.numElements;
    }

    /**
     * @notice Returns the key of the first element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the first element in the list.
     */
    function getHead(SortedLinkedList.List storage list) public view returns (address) {
        return toAddress(list.list.head);
    }

    /**
     * @notice Returns the key of the last element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the last element in the list.
     */
    function getTail(SortedLinkedList.List storage list) public view returns (address) {
        return toAddress(list.list.tail);
    }

    /**
     * @notice Gets lesser and greater for address that has increased it's value.
     * @param list A storage pointer to the underlying list.
     * @param group The original address.
     * @param newValue New value that has to be bigger or equal than the previous one.
     * @param loopLimit The max limit of loops that will be executed.
     */
    function getLesserAndGreaterOfAddressThatIncreasedValue(
        SortedLinkedList.List storage list,
        address group,
        uint256 newValue,
        uint256 loopLimit
    )
        public
        view
        returns (address previous, address next)
    {
        (, previous, next) = get(list, group);

        while (next != address(0) && loopLimit != 0 && newValue > getValue(list, next)) {
            previous = next;
            (,, next) = get(list, previous);
            loopLimit--;
        }

        if (loopLimit == 0) {
            return (address(0), address(0));
        }
    }

    /**
     * @notice Gets lesser and greater for address that has decreased it's value.
     * @param list A storage pointer to the underlying list.
     * @param group The original address.
     * @param newValue New value that has to be smaller or equal than the previous one.
     * @param loopLimit The max limit of loops that will be executed.
     */
    function getLesserAndGreaterOfAddressThatDecreasedValue(
        SortedLinkedList.List storage list,
        address group,
        uint256 newValue,
        uint256 loopLimit
    )
        public
        view
        returns (address previous, address next)
    {
        (, previous, next) = get(list, group);
        while (previous != address(0) && loopLimit != 0 && newValue < getValue(list, previous)) {
            next = previous;
            (, previous,) = get(list, next);
            loopLimit--;
        }
        if (loopLimit == 0) {
            return (address(0), address(0));
        }
    }

    function toBytes(address a) public pure returns (bytes32) {
        return bytes32(uint256(uint160(a)) << 96);
    }

    function toAddress(bytes32 b) public pure returns (address) {
        return address(uint160(uint256(b) >> 96));
    }

    /**
     * @notice Returns Element based on key.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return exists Whether or not the key exists.
     * @return previousKey Previous key.
     * @return nextKey Next key.
     */
    function get(
        SortedLinkedList.List storage list,
        address key
    )
        internal
        view
        returns (bool exists, address previousKey, address nextKey)
    {
        LinkedList.Element memory element = list.get(toBytes(key));
        exists = element.exists;
        if (element.exists) {
            previousKey = toAddress(element.previousKey);
            nextKey = toAddress(element.nextKey);
        }
    }
}
