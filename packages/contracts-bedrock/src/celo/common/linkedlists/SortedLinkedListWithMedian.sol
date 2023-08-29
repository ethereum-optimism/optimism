// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "./LinkedList.sol";
import "./SortedLinkedList.sol";

/**
 * @title Maintains a sorted list of unsigned ints keyed by bytes32.
 */
library SortedLinkedListWithMedian {
    using SortedLinkedList for SortedLinkedList.List;

    enum MedianAction {
        None,
        Lesser,
        Greater
    }

    enum MedianRelation {
        Undefined,
        Lesser,
        Greater,
        Equal
    }

    struct List {
        SortedLinkedList.List list;
        bytes32 median;
        mapping(bytes32 => MedianRelation) relation;
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
        list.list.insert(key, value, lesserKey, greaterKey);
        LinkedList.Element storage element = list.list.list.elements[key];

        MedianAction action = MedianAction.None;
        if (list.list.list.numElements == 1) {
            list.median = key;
            list.relation[key] = MedianRelation.Equal;
        } else if (list.list.list.numElements % 2 == 1) {
            // When we have an odd number of elements, and the element that we inserted is less than
            // the previous median, we need to slide the median down one element, since we had previously
            // selected the greater of the two middle elements.
            if (element.previousKey == bytes32(0) || list.relation[element.previousKey] == MedianRelation.Lesser) {
                action = MedianAction.Lesser;
                list.relation[key] = MedianRelation.Lesser;
            } else {
                list.relation[key] = MedianRelation.Greater;
            }
        } else {
            // When we have an even number of elements, and the element that we inserted is greater than
            // the previous median, we need to slide the median up one element, since we always select
            // the greater of the two middle elements.
            if (element.nextKey == bytes32(0) || list.relation[element.nextKey] == MedianRelation.Greater) {
                action = MedianAction.Greater;
                list.relation[key] = MedianRelation.Greater;
            } else {
                list.relation[key] = MedianRelation.Lesser;
            }
        }
        updateMedian(list, action);
    }

    /**
     * @notice Removes an element from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to remove.
     */
    function remove(List storage list, bytes32 key) internal {
        MedianAction action = MedianAction.None;
        if (list.list.list.numElements == 0) {
            list.median = bytes32(0);
        } else if (list.list.list.numElements % 2 == 0) {
            // When we have an even number of elements, we always choose the higher of the two medians.
            // Thus, if the element we're removing is greaterKey than or equal to the median we need to
            // slide the median left by one.
            if (list.relation[key] == MedianRelation.Greater || list.relation[key] == MedianRelation.Equal) {
                action = MedianAction.Lesser;
            }
        } else {
            // When we don't have an even number of elements, we just choose the median value.
            // Thus, if the element we're removing is less than or equal to the median, we need to slide
            // median right by one.
            if (list.relation[key] == MedianRelation.Lesser || list.relation[key] == MedianRelation.Equal) {
                action = MedianAction.Greater;
            }
        }
        updateMedian(list, action);

        list.list.remove(key);
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
        insert(list, key, 0, bytes32(0), list.list.list.tail);
    }

    /**
     * @notice Removes N elements from the head of the list and returns their keys.
     * @param list A storage pointer to the underlying list.
     * @param n The number of elements to pop.
     * @return The keys of the popped elements.
     */
    function popN(List storage list, uint256 n) internal returns (bytes32[] memory) {
        require(n <= list.list.list.numElements, "not enough elements");
        bytes32[] memory keys = new bytes32[](n);
        for (uint256 i = 0; i < n; i++) {
            bytes32 key = list.list.list.head;
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
     * @notice Returns the value for a particular key in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return The element value.
     */
    function getValue(List storage list, bytes32 key) internal view returns (uint256) {
        return list.list.values[key];
    }

    /**
     * @notice Returns the median value of the sorted list.
     * @param list A storage pointer to the underlying list.
     * @return The median value.
     */
    function getMedianValue(List storage list) internal view returns (uint256) {
        return getValue(list, list.median);
    }

    /**
     * @notice Returns the key of the first element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the first element in the list.
     */
    function getHead(List storage list) internal view returns (bytes32) {
        return list.list.list.head;
    }

    /**
     * @notice Returns the key of the median element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the median element in the list.
     */
    function getMedian(List storage list) internal view returns (bytes32) {
        return list.median;
    }

    /**
     * @notice Returns the key of the last element in the list.
     * @param list A storage pointer to the underlying list.
     * @return The key of the last element in the list.
     */
    function getTail(List storage list) internal view returns (bytes32) {
        return list.list.list.tail;
    }

    /**
     * @notice Returns the number of elements in the list.
     * @param list A storage pointer to the underlying list.
     * @return The number of elements in the list.
     */
    function getNumElements(List storage list) internal view returns (uint256) {
        return list.list.list.numElements;
    }

    /**
     * @notice Gets all elements from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @return Array of all keys in the list.
     * @return Values corresponding to keys, which will be ordered largest to smallest.
     * @return Array of relations to median of corresponding list elements.
     */
    function getElements(List storage list)
        internal
        view
        returns (bytes32[] memory, uint256[] memory, MedianRelation[] memory)
    {
        bytes32[] memory keys = getKeys(list);
        uint256[] memory values = new uint256[](keys.length);
        MedianRelation[] memory relations = new MedianRelation[](keys.length);
        for (uint256 i = 0; i < keys.length; i++) {
            values[i] = list.list.values[keys[i]];
            relations[i] = list.relation[keys[i]];
        }
        return (keys, values, relations);
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
     * @notice Moves the median pointer right or left of its current value.
     * @param list A storage pointer to the underlying list.
     * @param action Which direction to move the median pointer.
     */
    function updateMedian(List storage list, MedianAction action) private {
        LinkedList.Element storage previousMedian = list.list.list.elements[list.median];
        if (action == MedianAction.Lesser) {
            list.relation[list.median] = MedianRelation.Greater;
            list.median = previousMedian.previousKey;
        } else if (action == MedianAction.Greater) {
            list.relation[list.median] = MedianRelation.Lesser;
            list.median = previousMedian.nextKey;
        }
        list.relation[list.median] = MedianRelation.Equal;
    }
}
