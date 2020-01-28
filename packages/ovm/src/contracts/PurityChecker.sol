pragma solidity ^0.5.0;

/**
 * @title PurityChecker
 * @notice Purity Checker contract used to check whether or not bytecode is pure, meaning:
 * 1. It uses only whitelisted opcodes
 * 2. All CALLs are to the Execution Manager and have no value set (no ETH sent)
 */
contract PurityChecker {
  uint256 public opcodeWhitelistMask;
  /**
   * @notice Construct a new Purity Checker with the specified whitelist mask
   * @param _opcodeWhitelistMask A hex number of 256 bits where each bit represents an opcode, 0 - 255, which is set if whitelisted and unset otherwise.
   */
  constructor(uint256 _opcodeWhitelistMask) public {
    opcodeWhitelistMask = _opcodeWhitelistMask;
  }

  /**
   * @notice Returns whether or not all of the provided bytecode is pure.
   * @param _bytecode The bytecode to purity check. This can be both creation bytecode (aka initcode) and runtime bytecode (aka contract code). 
   * More info on creation vs. runtime bytecode: https://medium.com/authereum/bytecode-and-init-code-and-runtime-code-oh-my-7bcd89065904
   */
  function isBytecodePure(
    bytes memory _bytecode
  ) public view returns (bool) {
    bool seenJUMP = false;
    bool insideUnreachableCode = false;
    for (uint256 i = 0; i < _bytecode.length; i++) {
      uint256 op = uint8(_bytecode[i]);
      // If this is a PUSH##, subsequent bytes are not opcodes. Skip them.
      if (op >= 0x60 && op <= 0x7f) {
        i += (op - 0x5f);
      }
      if (insideUnreachableCode) {
        // found a JUMPDEST - this bytecode is now reachable
        if (op == 0x5b) {
          insideUnreachableCode = false;
        }
      } else {
        // check that opcode is whitelisted (using the whitelist mask)
        uint256 opBit = 2 ** op;
        if (opcodeWhitelistMask & opBit != opBit) {
          return false;
        }
        // If this is a JUMP or JUMPI, set seenJUMP to true
        if (op == 0x56 || op == 0x57) {
          seenJUMP = true;
        }
        // If this is a STOP, JUMP, or REVERT, then we are entering unreachable code
        if (op == 0x00 || op == 0x56 || op == 0xfd) {
          //If there have been no JUMPs or JUMPIs seen yet, then all remaining bytecode is unreachable
          if (!seenJUMP) {
            return true;
          }
          // We are now inside unreachable code!
          insideUnreachableCode = true;
        }
      }
    }
    //TODO: check CALLs are only made to the Execution Manager and with 0 value

    return true;
  }
}
