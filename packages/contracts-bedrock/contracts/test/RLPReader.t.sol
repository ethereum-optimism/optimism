// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { RLPReader } from "../libraries/rlp/RLPReader.sol";
import { CommonTest } from "./CommonTest.t.sol";

contract RLPReader_Test is CommonTest {
    function testReadBool() external {
        assertEq(
            RLPReader.readBool(hex"01"),
            true
        );

        assertEq(
            RLPReader.readBool(hex"00"),
            false
        );
    }

    function test_readBoolInvalidValue() external {
        vm.expectRevert("RLPReader: invalid RLP boolean value, must be 0 or 1");
        RLPReader.readBool(hex"02");
    }

    function test_readBoolLargeInput() external {
        vm.expectRevert("RLPReader: invalid RLP boolean value");
        RLPReader.readBool(hex"0101");
    }

    function test_readAddress() external {
        assertEq(
            RLPReader.readAddress(hex"941212121212121212121212121212121212121212"),
            address(0x1212121212121212121212121212121212121212)
        );
    }

    function test_readAddressSmall() external {
        assertEq(
            RLPReader.readAddress(hex"12"),
            address(0)
        );
    }

    function test_readAddressTooLarge() external {
        vm.expectRevert("RLPReader: invalid RLP address value");
        RLPReader.readAddress(hex"94121212121212121212121212121212121212121212121212");
    }

    function test_readAddressTooShort() external {
        vm.expectRevert("RLPReader: invalid RLP address value");
        RLPReader.readAddress(hex"94121212121212121212121212");
    }

    function test_readBytes_bytestring00() external {
        assertEq(
            RLPReader.readBytes(hex"00"),
            hex"00"
        );
    }

    function test_readBytes_bytestring01() external {
        assertEq(
            RLPReader.readBytes(hex"01"),
            hex"01"
        );
    }

    function test_readBytes_bytestring7f() external {
        assertEq(
            RLPReader.readBytes(hex"7f"),
            hex"7f"
        );
    }

    function test_readBytes_revertListItem() external {
        vm.expectRevert("RLPReader: invalid RLP bytes value");
        RLPReader.readBytes(hex"c7c0c1c0c3c0c1c0");
    }

    function test_readBytes_invalidStringLength() external {
        vm.expectRevert("RLPReader: invalid RLP long string length");
        RLPReader.readBytes(hex"b9");
    }

    function test_readBytes_invalidListLength() external {
        vm.expectRevert("RLPReader: invalid RLP long list length");
        RLPReader.readBytes(hex"ff");
    }

    function test_readBytes32_revertOnList() external {
        vm.expectRevert("RLPReader: invalid RLP bytes32 value");
        RLPReader.readBytes32(hex"c7c0c1c0c3c0c1c0");
    }

    function test_readBytes32_revertOnTooLong() external {
        vm.expectRevert("RLPReader: invalid RLP bytes32 value");
        RLPReader.readBytes32(hex"11110000000000000000000000000000000000000000000000000000000000000000");
    }

    function test_readString_emptyString() external {
        assertEq(
            RLPReader.readString(hex"80"),
            hex""
        );
    }

    function test_readString_shortString() external {
        assertEq(
            RLPReader.readString(hex"83646f67"),
            "dog"
        );
    }

    function test_readString_shortString2() external {
        assertEq(
            RLPReader.readString(hex"b74c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e7365637465747572206164697069736963696e6720656c69"),
            "Lorem ipsum dolor sit amet, consectetur adipisicing eli"
        );
    }

    function test_readString_longString() external {
        assertEq(
            RLPReader.readString(hex"b8384c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e7365637465747572206164697069736963696e6720656c6974"),
            "Lorem ipsum dolor sit amet, consectetur adipisicing elit"
        );
    }

    function test_readString_longString2() external {
        assertEq(
            RLPReader.readString(hex"b904004c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e73656374657475722061646970697363696e6720656c69742e20437572616269747572206d6175726973206d61676e612c20737573636970697420736564207665686963756c61206e6f6e2c20696163756c697320666175636962757320746f72746f722e2050726f696e20737573636970697420756c74726963696573206d616c6573756164612e204475697320746f72746f7220656c69742c2064696374756d2071756973207472697374697175652065752c20756c7472696365732061742072697375732e204d6f72626920612065737420696d70657264696574206d6920756c6c616d636f7270657220616c6971756574207375736369706974206e6563206c6f72656d2e2041656e65616e2071756973206c656f206d6f6c6c69732c2076756c70757461746520656c6974207661726975732c20636f6e73657175617420656e696d2e204e756c6c6120756c74726963657320747572706973206a7573746f2c20657420706f73756572652075726e6120636f6e7365637465747572206e65632e2050726f696e206e6f6e20636f6e76616c6c6973206d657475732e20446f6e65632074656d706f7220697073756d20696e206d617572697320636f6e67756520736f6c6c696369747564696e2e20566573746962756c756d20616e746520697073756d207072696d697320696e206661756369627573206f726369206c756374757320657420756c74726963657320706f737565726520637562696c69612043757261653b2053757370656e646973736520636f6e76616c6c69732073656d2076656c206d617373612066617563696275732c2065676574206c6163696e6961206c616375732074656d706f722e204e756c6c61207175697320756c747269636965732070757275732e2050726f696e20617563746f722072686f6e637573206e69626820636f6e64696d656e74756d206d6f6c6c69732e20416c697175616d20636f6e73657175617420656e696d206174206d65747573206c75637475732c206120656c656966656e6420707572757320656765737461732e20437572616269747572206174206e696268206d657475732e204e616d20626962656e64756d2c206e6571756520617420617563746f72207472697374697175652c206c6f72656d206c696265726f20616c697175657420617263752c206e6f6e20696e74657264756d2074656c6c7573206c65637475732073697420616d65742065726f732e20437261732072686f6e6375732c206d65747573206163206f726e617265206375727375732c20646f6c6f72206a7573746f20756c747269636573206d657475732c20617420756c6c616d636f7270657220766f6c7574706174"),
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur mauris magna, suscipit sed vehicula non, iaculis faucibus tortor. Proin suscipit ultricies malesuada. Duis tortor elit, dictum quis tristique eu, ultrices at risus. Morbi a est imperdiet mi ullamcorper aliquet suscipit nec lorem. Aenean quis leo mollis, vulputate elit varius, consequat enim. Nulla ultrices turpis justo, et posuere urna consectetur nec. Proin non convallis metus. Donec tempor ipsum in mauris congue sollicitudin. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Suspendisse convallis sem vel massa faucibus, eget lacinia lacus tempor. Nulla quis ultricies purus. Proin auctor rhoncus nibh condimentum mollis. Aliquam consequat enim at metus luctus, a eleifend purus egestas. Curabitur at nibh metus. Nam bibendum, neque at auctor tristique, lorem libero aliquet arcu, non interdum tellus lectus sit amet eros. Cras rhoncus, metus ac ornare cursus, dolor justo ultrices metus, at ullamcorper volutpat"
        );
    }

    function test_readUint256_zero() external {
        assertEq(
            RLPReader.readUint256(hex"80"),
            0
        );
    }

    function test_readUint256_smallInt() external {
        assertEq(
            RLPReader.readUint256(hex"01"),
            1
        );
    }

    function test_readUint256_smallInt2() external {
        assertEq(
            RLPReader.readUint256(hex"10"),
            16
        );
    }

    function test_readUint256_smallInt3() external {
        assertEq(
            RLPReader.readUint256(hex"4f"),
            79
        );
    }

    function test_readUint256_smallInt4() external {
        assertEq(
            RLPReader.readUint256(hex"7f"),
            127
        );
    }

    function test_readUint256_mediumInt1() external {
        assertEq(
            RLPReader.readUint256(hex"8180"),
            128
        );
    }

    function test_readUint256_mediumInt2() external {
        assertEq(
            RLPReader.readUint256(hex"8203e8"),
            1000
        );
    }

    function test_readUint256_mediumInt3() external {
        assertEq(
            RLPReader.readUint256(hex"830186a0"),
            100000
        );
    }

    function test_readList_empty() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"c0");
        assertEq(list.length, 0);
    }

    function test_readList_stringList() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"cc83646f6783676f6483636174");
        assertEq(list.length, 3);
        assertEq(RLPReader.readString(list[0]), RLPReader.readString(hex"83646f67"));
        assertEq(RLPReader.readString(list[1]), RLPReader.readString(hex"83676f64"));
        assertEq(RLPReader.readString(list[2]), RLPReader.readString(hex"83636174"));
    }

    function test_readList_multiList() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"c6827a77c10401");
        assertEq(list.length, 3);

        assertEq(RLPReader.readRawBytes(list[0]), hex"827a77");
        assertEq(RLPReader.readRawBytes(list[1]), hex"c104");
        assertEq(RLPReader.readRawBytes(list[2]), hex"01");
    }

    function test_readList_shortListMax1() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"f784617364668471776572847a78637684617364668471776572847a78637684617364668471776572847a78637684617364668471776572");

        assertEq(list.length, 11);
        assertEq(RLPReader.readRawBytes(list[0]), hex"8461736466");
        assertEq(RLPReader.readRawBytes(list[1]), hex"8471776572");
        assertEq(RLPReader.readRawBytes(list[2]), hex"847a786376");
        assertEq(RLPReader.readRawBytes(list[3]), hex"8461736466");
        assertEq(RLPReader.readRawBytes(list[4]), hex"8471776572");
        assertEq(RLPReader.readRawBytes(list[5]), hex"847a786376");
        assertEq(RLPReader.readRawBytes(list[6]), hex"8461736466");
        assertEq(RLPReader.readRawBytes(list[7]), hex"8471776572");
        assertEq(RLPReader.readRawBytes(list[8]), hex"847a786376");
        assertEq(RLPReader.readRawBytes(list[9]), hex"8461736466");
        assertEq(RLPReader.readRawBytes(list[10]), hex"8471776572");
    }

    function test_readList_longList1() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"f840cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376");

        assertEq(list.length, 4);
        assertEq(RLPReader.readRawBytes(list[0]), hex"cf84617364668471776572847a786376");
        assertEq(RLPReader.readRawBytes(list[1]), hex"cf84617364668471776572847a786376");
        assertEq(RLPReader.readRawBytes(list[2]), hex"cf84617364668471776572847a786376");
        assertEq(RLPReader.readRawBytes(list[3]), hex"cf84617364668471776572847a786376");
    }

    function test_readList_longList2() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"f90200cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376cf84617364668471776572847a786376");
        assertEq(list.length, 32);

        for (uint256 i = 0; i < 32; i++) {
            assertEq(RLPReader.readRawBytes(list[i]), hex"cf84617364668471776572847a786376");
        }
    }

    function test_readList_listOfLists() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"c4c2c0c0c0");
        assertEq(list.length, 2);
        assertEq(RLPReader.readRawBytes(list[0]), hex"c2c0c0");
        assertEq(RLPReader.readRawBytes(list[1]), hex"c0");
    }

    function test_readList_listOfLists2() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"c7c0c1c0c3c0c1c0");
        assertEq(list.length, 3);

        assertEq(RLPReader.readRawBytes(list[0]), hex"c0");
        assertEq(RLPReader.readRawBytes(list[1]), hex"c1c0");
        assertEq(RLPReader.readRawBytes(list[2]), hex"c3c0c1c0");
    }

    function test_readList_dictTest1() external {
        RLPReader.RLPItem[] memory list = RLPReader.readList(hex"ecca846b6579318476616c31ca846b6579328476616c32ca846b6579338476616c33ca846b6579348476616c34");
        assertEq(list.length, 4);

        assertEq(RLPReader.readRawBytes(list[0]), hex"ca846b6579318476616c31");
        assertEq(RLPReader.readRawBytes(list[1]), hex"ca846b6579328476616c32");
        assertEq(RLPReader.readRawBytes(list[2]), hex"ca846b6579338476616c33");
        assertEq(RLPReader.readRawBytes(list[3]), hex"ca846b6579348476616c34");
    }

    function test_readList_invalidShortList() external {
        vm.expectRevert("RLPReader: invalid RLP short list");
        RLPReader.readList(hex"efdebd");
    }

    function test_readList_longStringLength() external {
        vm.expectRevert("RLPReader: invalid RLP short list");
        RLPReader.readList(hex"efb83600");
    }

    function test_readList_notLongEnough() external {
        vm.expectRevert("RLPReader: invalid RLP short list");
        RLPReader.readList(hex"efdebdaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    }

    function test_readList_int32Overflow() external {
        vm.expectRevert("RLPReader: invalid RLP long string");
        RLPReader.readList(hex"bf0f000000000000021111");
    }

    function test_readList_int32Overflow2() external {
        vm.expectRevert("RLPReader: invalid RLP long list");
        RLPReader.readList(hex"ff0f000000000000021111");
    }

    function test_readList_incorrectLengthInArray() external {
        vm.expectRevert("RLPReader: invalid RLP list value");
        RLPReader.readList(hex"b9002100dc2b275d0f74e8a53e6f4ec61b27f24278820be3f82ea2110e582081b0565df0");
    }

    function test_readList_leadingZerosInLongLengthArray1() external {
        vm.expectRevert("RLPReader: invalid RLP list value");
        RLPReader.readList(hex"b90040000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f");
    }

    function test_readList_leadingZerosInLongLengthArray2() external {
        vm.expectRevert("RLPReader: invalid RLP list value");
        RLPReader.readList(hex"b800");
    }

    function test_readList_leadingZerosInLongLengthList1() external {
        vm.expectRevert("RLPReader: provided RLP list exceeds max list length");
        RLPReader.readList(hex"fb00000040000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f");
    }

    function test_readList_nonOptimalLongLengthArray1() external {
        vm.expectRevert("RLPReader: invalid RLP list value");
        RLPReader.readList(hex"b81000112233445566778899aabbccddeeff");
    }

    function test_readList_nonOptimalLongLengthArray2() external {
        vm.expectRevert("RLPReader: invalid RLP list value");
        RLPReader.readList(hex"b801ff");
    }

    function test_readList_invalidValue() external {
        vm.expectRevert("RLPReader: invalid RLP short string");
        RLPReader.readList(hex"91");
    }
}
