// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/**
 * @title Maintains a doubly linked list keyed by bytes32.
 * @dev Following the `next` pointers will lead you to the head, rather than the tail.
 */
library LinkedList {
    struct Element {
        bytes32 previousKey;
        bytes32 nextKey;
        bool exists;
    }

    struct List {
        bytes32 head;
        bytes32 tail;
        uint256 numElements;
        mapping(bytes32 => Element) elements;
    }

    /**
     * @notice Inserts an element into a doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to insert.
     * @param previousKey The key of the element that comes before the element to insert.
     * @param nextKey The key of the element that comes after the element to insert.
     */
    function insert(List storage list, bytes32 key, bytes32 previousKey, bytes32 nextKey) internal {
        require(key != bytes32(0), "Key must be defined");
        require(!contains(list, key), "Can't insert an existing element");
        require(previousKey != key && nextKey != key, "Key cannot be the same as previousKey or nextKey");

        Element storage element = list.elements[key];
        element.exists = true;

        if (list.numElements == 0) {
            list.tail = key;
            list.head = key;
        } else {
            require(previousKey != bytes32(0) || nextKey != bytes32(0), "Either previousKey or nextKey must be defined");

            element.previousKey = previousKey;
            element.nextKey = nextKey;

            if (previousKey != bytes32(0)) {
                require(contains(list, previousKey), "If previousKey is defined, it must exist in the list");
                Element storage previousElement = list.elements[previousKey];
                require(previousElement.nextKey == nextKey, "previousKey must be adjacent to nextKey");
                previousElement.nextKey = key;
            } else {
                list.tail = key;
            }

            if (nextKey != bytes32(0)) {
                require(contains(list, nextKey), "If nextKey is defined, it must exist in the list");
                Element storage nextElement = list.elements[nextKey];
                require(nextElement.previousKey == previousKey, "previousKey must be adjacent to nextKey");
                nextElement.previousKey = key;
            } else {
                list.head = key;
            }
        }

        list.numElements = list.numElements + 1;
    }

    /**
     * @notice Inserts an element at the tail of the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to insert.
     */
    function push(List storage list, bytes32 key) internal {
        insert(list, key, bytes32(0), list.tail);
    }

    /**
     * @notice Removes an element from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @param key The key of the element to remove.
     */
    function remove(List storage list, bytes32 key) internal {
        Element storage element = list.elements[key];
        require(key != bytes32(0) && contains(list, key), "key not in list");
        if (element.previousKey != bytes32(0)) {
            Element storage previousElement = list.elements[element.previousKey];
            previousElement.nextKey = element.nextKey;
        } else {
            list.tail = element.nextKey;
        }

        if (element.nextKey != bytes32(0)) {
            Element storage nextElement = list.elements[element.nextKey];
            nextElement.previousKey = element.previousKey;
        } else {
            list.head = element.previousKey;
        }

        delete list.elements[key];
        list.numElements = list.numElements - 1;
    }

    /**
     * @notice Updates an element in the list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @param previousKey The key of the element that comes before the updated element.
     * @param nextKey The key of the element that comes after the updated element.
     */
    function update(List storage list, bytes32 key, bytes32 previousKey, bytes32 nextKey) internal {
        require(key != bytes32(0) && key != previousKey && key != nextKey && contains(list, key), "key on in list");
        remove(list, key);
        insert(list, key, previousKey, nextKey);
    }

    /**
     * @notice Returns whether or not a particular key is present in the sorted list.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return Whether or not the key is in the sorted list.
     */
    function contains(List storage list, bytes32 key) internal view returns (bool) {
        return list.elements[key].exists;
    }

    /**
     * @notice Returns Element based on key.
     * @param list A storage pointer to the underlying list.
     * @param key The element key.
     * @return Whether or not the key is in the sorted list.
     */
    function get(List storage list, bytes32 key) internal view returns (Element memory) {
        return list.elements[key];
    }

    /**
     * @notice Returns the keys of the N elements at the head of the list.
     * @param list A storage pointer to the underlying list.
     * @param n The number of elements to return.
     * @return The keys of the N elements at the head of the list.
     * @dev Reverts if n is greater than the number of elements in the list.
     */
    function headN(List storage list, uint256 n) internal view returns (bytes32[] memory) {
        require(n <= list.numElements, "not enough elements");
        bytes32[] memory keys = new bytes32[](n);
        bytes32 key = list.head;
        for (uint256 i = 0; i < n; i = i + 1) {
            keys[i] = key;
            key = list.elements[key].previousKey;
        }
        return keys;
    }

    /**
     * @notice Gets all element keys from the doubly linked list.
     * @param list A storage pointer to the underlying list.
     * @return All element keys from head to tail.
     */
    function getKeys(List storage list) internal view returns (bytes32[] memory) {
        return headN(list, list.numElements);
    }
}
