pragma solidity ^0.5.0;

/**
 * @title OpcodeWhitelist
 * @notice Opcode whitelist contract used to check whether or not code uses only whitelisted opcodes.
 */
contract OpcodeWhitelist {
  uint256 public opcodeWhitelistMask;

  constructor(uint256 _opcodeWhitelistMask) public {
    opcodeWhitelistMask = _opcodeWhitelistMask;
  }

  // Returns whether or not all of the opcodes in the provided bytecode are whitelisted.
  function isBytecodeWhitelisted(
    bytes memory _bytecode
  ) public view returns (bool) {
    for (uint256 i = 0; i < _bytecode.length; i++) {
      uint256 op = uint8(_bytecode[i]);
      uint256 opBit = 2 ** op;
      if (opcodeWhitelistMask & opBit != opBit) {
        return false;
      }

      // If this is a PUSH##, subsequent bytes are not opcodes. Skip them.
      if (op >= 0x60 && op <= 0x7f) {
        i += (op - 0x5f);
      }
    }

    return true;
  }
}
