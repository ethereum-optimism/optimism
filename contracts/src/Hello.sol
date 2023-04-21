pragma solidity ^0.7.6;

// Hello-world contract to test source maps and instrumentation
contract Hello {

    function helloWorld(uint32 x) public returns (uint32) {
        require(x > 3, "test");
        return x + 100;
    }
}
