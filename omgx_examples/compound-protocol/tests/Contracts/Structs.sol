pragma solidity ^0.5.16;
pragma experimental ABIEncoderV2;

contract Structs {
    struct Outer {
        uint sentinel;
        mapping(address => Inner) inners;
    }

    struct Inner {
        uint16 a;
        uint16 b;
        uint96 c;
    }

    mapping(uint => Outer) public outers;

    function writeEach(uint id, uint16 a, uint16 b, uint96 c) public {
        Inner storage inner = outers[id].inners[msg.sender];
        inner.a = a;
        inner.b = b;
        inner.c = c;
    }

    function writeOnce(uint id, uint16 a, uint16 b, uint96 c) public {
        Inner memory inner = Inner({a: a, b: b, c: c});
        outers[id].inners[msg.sender] = inner;
    }
}