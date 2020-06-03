pragma solidity >=0.5.0 <0.6.0;

/**
 MIT License
 Original author: chriseth
 */

import {D} from "./DataTypes.sol";

library Utils {
    /// combines two labels into one
    function combineLabels(D.Label memory prefix, D.Label memory suffix) internal pure returns (D.Label memory combined) {
        combined.length = prefix.length + suffix.length;
        combined.data = prefix.data | (suffix.data >> prefix.length);
    }


    /// Returns a label containing the longest common prefix of `check` and `label`
    /// and a label consisting of the remaining part of `label`.
    function splitCommonPrefix(D.Label memory label, D.Label memory check) internal pure returns (D.Label memory prefix, D.Label memory labelSuffix) {
        return splitAt(label, commonPrefixLength(check, label));
    }
    /// Splits the label at the given position and returns prefix and suffix,
    /// i.e. prefix.length == pos and prefix.data . suffix.data == l.data.
    function splitAt(D.Label memory l, uint pos) internal pure returns (D.Label memory prefix, D.Label memory suffix) {
        require(pos <= l.length, "Asked to split label at position exceeding the label length.");
        require(pos <= 256, "Asked to split label at position exceeding 256 bits.");
        prefix.length = pos;
        if (pos == 0) {
            prefix.data = bytes32(0);
        } else {
            prefix.data = l.data & ~bytes32((uint(1) << (256 - pos)) - 1);
        }
        suffix.length = l.length - pos;
        suffix.data = l.data << pos;
    }
    /// Returns the length of the longest common prefix of the two labels.
    function commonPrefixLength(D.Label memory a, D.Label memory b) internal pure returns (uint prefix) {
        uint length = a.length < b.length ? a.length : b.length;
        // TODO: This could actually use a "highestBitSet" helper
        uint diff = uint(a.data ^ b.data);
        uint mask = 1 << 255;
        for (; prefix < length; prefix++)
        {
            if ((mask & diff) != 0)
                break;
            diff += diff;
        }
    }
    /// Returns the result of removing a prefix of length `prefix` bits from the
    /// given label (i.e. shifting its data to the left).
    function removePrefix(D.Label memory l, uint prefix) internal pure returns (D.Label memory r) {
        require(prefix <= l.length, "Bad lenght");
        r.length = l.length - prefix;
        r.data = l.data << prefix;
    }
    /// Removes the first bit from a label and returns the bit and a
    /// label containing the rest of the label (i.e. shifted to the left).
    function chopFirstBit(D.Label memory l) internal pure returns (uint firstBit, D.Label memory tail) {
        require(l.length > 0, "Empty element");
        return (uint(l.data >> 255), D.Label(l.data << 1, l.length - 1));
    }
    /// Returns the first bit set in the bitfield, where the 0th bit
    /// is the least significant.
    /// Throws if bitfield is zero.
    /// More efficient the smaller the result is.
    function lowestBitSet(uint bitfield) internal pure returns (uint bit) {
        require(bitfield != 0, "Bad bitfield");
        bytes32 bitfieldBytes = bytes32(bitfield);
        // First, find the lowest byte set
        uint byteSet = 0;
        for (; byteSet < 32; byteSet++) {
            if (bitfieldBytes[31 - byteSet] != 0)
                break;
        }
        uint singleByte = uint(uint8(bitfieldBytes[31 - byteSet]));
        uint mask = 1;
        for (bit = 0; bit < 256; bit ++) {
            if ((singleByte & mask) != 0)
                return 8 * byteSet + bit;
            mask += mask;
        }
        assert(false);
        return 0;
    }
    /// Returns the value of the `bit`th bit inside `bitfield`, where
    /// the least significant is the 0th bit.
    function bitSet(uint bitfield, uint bit) internal pure returns (uint) {
        return (bitfield & (uint(1) << bit)) != 0 ? 1 : 0;
    }
}


contract UtilsTest {
    function test() public pure {
        testLowestBitSet();
        testChopFirstBit();
        testRemovePrefix();
        testCommonPrefix();
        testSplitAt();
        testSplitCommonPrefix();
    }
    function testLowestBitSet() internal pure {
        require(Utils.lowestBitSet(0x123) == 0, "testLowestBitSet 1");
        require(Utils.lowestBitSet(0x124) == 2, "testLowestBitSet 2");
        require(Utils.lowestBitSet(0x11 << 30) == 30, "testLowestBitSet 3");
        require(Utils.lowestBitSet(1 << 255) == 255, "testLowestBitSet 4");
    }
    function testChopFirstBit() internal pure {
        D.Label memory l;
        l.data = hex"ef1230";
        l.length = 20;
        uint bit1;
        uint bit2;
        uint bit3;
        uint bit4;
        (bit1, l) = Utils.chopFirstBit(l);
        (bit2, l) = Utils.chopFirstBit(l);
        (bit3, l) = Utils.chopFirstBit(l);
        (bit4, l) = Utils.chopFirstBit(l);
        require(bit1 == 1, "testChopFirstBit 1");
        require(bit2 == 1, "testChopFirstBit 2");
        require(bit3 == 1, "testChopFirstBit 3");
        require(bit4 == 0, "testChopFirstBit 4");
        require(l.length == 16, "testChopFirstBit 5");
        require(l.data == hex"F123", "testChopFirstBit 6");

        l.data = hex"80";
        l.length = 1;
        (bit1, l) = Utils.chopFirstBit(l);
        require(bit1 == 1, "Fail 7");
        require(l.length == 0, "Fail 8");
        require(l.data == 0, "Fail 9");
    }
    function testRemovePrefix() internal pure {
        D.Label memory l;
        l.data = hex"ef1230";
        l.length = 20;
        l = Utils.removePrefix(l, 4);
        require(l.length == 16, "testRemovePrefix 1");
        require(l.data == hex"f123", "testRemovePrefix 2");
        l = Utils.removePrefix(l, 15);
        require(l.length == 1, "testRemovePrefix 3");
        require(l.data == hex"80", "testRemovePrefix 4");
        l = Utils.removePrefix(l, 1);
        require(l.length == 0, "testRemovePrefix 5");
        require(l.data == 0, "testRemovePrefix 6");
    }
    function testCommonPrefix() internal pure {
        D.Label memory a;
        D.Label memory b;
        a.data = hex"abcd";
        a.length = 16;
        b.data = hex"a000";
        b.length = 16;
        require(Utils.commonPrefixLength(a, b) == 4, "testCommonPrefix 1");

        b.length = 0;
        require(Utils.commonPrefixLength(a, b) == 0, "testCommonPrefix 2");

        b.data = hex"bbcd";
        b.length = 16;
        require(Utils.commonPrefixLength(a, b) == 3, "testCommonPrefix 3");
        require(Utils.commonPrefixLength(b, b) == b.length, "testCommonPrefix 4");
    }
    function testSplitAt() internal pure {
        D.Label memory a;
        a.data = hex"abcd";
        a.length = 16;
        (D.Label memory x, D.Label memory y) = Utils.splitAt(a, 0);
        require(x.length == 0, "testSplitAt 1");
        require(y.length == a.length, "testSplitAt 2");
        require(y.data == a.data, "testSplitAt 3");

        (x, y) = Utils.splitAt(a, 4);
        require(x.length == 4, "testSplitAt 4");
        require(x.data == hex"a0", "testSplitAt 5");
        require(y.length == 12, "testSplitAt 6");
        require(y.data == hex"bcd0", "testSplitAt 7");

        (x, y) = Utils.splitAt(a, 16);
        require(y.length == 0, "testSplitAt 8");
        require(x.length == a.length, "testSplitAt 9");
        require(x.data == a.data, "testSplitAt 10");
    }
    function testSplitCommonPrefix() internal pure {
        D.Label memory a;
        D.Label memory b;
        a.data = hex"abcd";
        a.length = 16;
        b.data = hex"a0f570";
        b.length = 20;
        (D.Label memory prefix, D.Label memory suffix) = Utils.splitCommonPrefix(b, a);
        require(prefix.length == 4, "testSplitCommonPrefix 1");
        require(prefix.data == hex"a0", "testSplitCommonPrefix 2");
        require(suffix.length == 16, "testSplitCommonPrefix 3");
        require(suffix.data == hex"0f57", "testSplitCommonPrefix 4");
    }
}

