// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

// A trimmed-down and formatted version of https://github.com/emo-eth.
//
// This is provided to provide better UX when generating and using arrays in tests and scripts,
// since Solidity does not have great array UX.
//
// This library was generated using the `generator.py` script from the linked repo with the length
// set to 10, and then everything except the `addresses` functions was removed.
library Solarray {
    function addresses(address a) internal pure returns (address[] memory) {
        address[] memory arr = new address[](1);
        arr[0] = a;
        return arr;
    }

    function addresses(address a, address b) internal pure returns (address[] memory) {
        address[] memory arr = new address[](2);
        arr[0] = a;
        arr[1] = b;
        return arr;
    }

    function addresses(address a, address b, address c) internal pure returns (address[] memory) {
        address[] memory arr = new address[](3);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        return arr;
    }

    function addresses(address a, address b, address c, address d) internal pure returns (address[] memory) {
        address[] memory arr = new address[](4);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        return arr;
    }

    function addresses(
        address a,
        address b,
        address c,
        address d,
        address e
    )
        internal
        pure
        returns (address[] memory)
    {
        address[] memory arr = new address[](5);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        arr[4] = e;
        return arr;
    }

    function addresses(
        address a,
        address b,
        address c,
        address d,
        address e,
        address f
    )
        internal
        pure
        returns (address[] memory)
    {
        address[] memory arr = new address[](6);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        arr[4] = e;
        arr[5] = f;
        return arr;
    }

    function addresses(
        address a,
        address b,
        address c,
        address d,
        address e,
        address f,
        address g
    )
        internal
        pure
        returns (address[] memory)
    {
        address[] memory arr = new address[](7);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        arr[4] = e;
        arr[5] = f;
        arr[6] = g;
        return arr;
    }

    function addresses(
        address a,
        address b,
        address c,
        address d,
        address e,
        address f,
        address g,
        address h
    )
        internal
        pure
        returns (address[] memory)
    {
        address[] memory arr = new address[](8);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        arr[4] = e;
        arr[5] = f;
        arr[6] = g;
        arr[7] = h;
        return arr;
    }

    function addresses(
        address a,
        address b,
        address c,
        address d,
        address e,
        address f,
        address g,
        address h,
        address i
    )
        internal
        pure
        returns (address[] memory)
    {
        address[] memory arr = new address[](9);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        arr[4] = e;
        arr[5] = f;
        arr[6] = g;
        arr[7] = h;
        arr[8] = i;
        return arr;
    }

    function addresses(
        address a,
        address b,
        address c,
        address d,
        address e,
        address f,
        address g,
        address h,
        address i,
        address j
    )
        internal
        pure
        returns (address[] memory)
    {
        address[] memory arr = new address[](10);
        arr[0] = a;
        arr[1] = b;
        arr[2] = c;
        arr[3] = d;
        arr[4] = e;
        arr[5] = f;
        arr[6] = g;
        arr[7] = h;
        arr[8] = i;
        arr[9] = j;
        return arr;
    }
}