pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

/**
 * @title SafetyChecker
 * @notice Safety Checker contract used to check whether or not bytecode is
 *         safe, meaning:
 *              1. It uses only whitelisted opcodes.
 *              2. All CALLs are to the Execution Manager and have no value.
 */
contract SafetyChecker is ContractResolver {
    /*
     * Contract Variables
     */

    uint256 public opcodeWhitelistMask;


    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     * @param _opcodeWhitelistMask Whitelist mask of allowed opcodes.
     */
    constructor(
        address _addressResolver,
        uint256 _opcodeWhitelistMask
    )
        public
        ContractResolver(_addressResolver)
    {
        opcodeWhitelistMask = _opcodeWhitelistMask;
    }


    /*
     * Public Functions
     */

    /**
     * Returns whether or not all of the provided bytecode is safe.
     * @dev More info on creation vs. runtime bytecode:
     * https://medium.com/authereum/bytecode-and-init-code-and-runtime-code-oh-my-7bcd89065904.
     * @param _bytecode The bytecode to safety check. This can be either
     *                  creation bytecode (aka initcode) or runtime bytecode
     *                  (aka cont
     * More info on creation vs. runtime bytecode:
     * https://medium.com/authereum/bytecode-and-init-code-and-runtime-code-oh-my-7bcd89065904ract code).
     * @return `true` if the bytecode is safe, `false` otherwise.
     */
    function isBytecodeSafe(
        bytes memory _bytecode
    )
        public
        view
        returns (bool)
    {
        bool insideUnreachableCode = false;
        uint256 codeLength = _bytecode.length;
        uint256 _opcodeBlacklistMask = ~opcodeWhitelistMask;
        uint256 _opcodeProcessMask = 0x6008000000000000000000000000000000000000004000000008000000000001;
        uint256 _bytecode32;
        assembly {
          _bytecode32 := add(_bytecode, 0x20)
        }
        for (uint256 _pc = 0; _pc < codeLength; _pc++) {
            // current opcode: 0x00...0xff
            uint256 op; // = uint8(_bytecode[_pc]);

            // inline assembly removes the extra add + bounds check
            assembly {
              op := shr(0xf8, mload(add(_bytecode32, _pc)))
            }

            // PUSH##
            if ((op - 0x60) <= 0x1f) {
                // subsequent bytes are not opcodes. Skip them.
                _pc += (op - 0x5f);
                // all pushes are valid opcodes
                continue;
            }

            // If we're in between a STOP or REVERT or JUMP and a JUMPDEST
            if (insideUnreachableCode) {
                // JUMPDEST
                insideUnreachableCode = (op != 0x5b);
                // exit unreachable code on the next go around
                continue;
            } 

            // check that opcode is whitelisted (using the whitelist bit mask)
            uint256 opBit = 1 << op;
            if (opBit & _opcodeBlacklistMask != 0) {
                // encountered a non-whitelisted opcode!
                return false;
            }
            // append this opcode to a list of ops
            // [STOP(0x00),JUMP(0x56),RETURN(0xf3),REVERT(0xfd),INVALID(0xfe),CALLER(0x33)] all have handlers
            if (opBit & _opcodeProcessMask != 0) {
                // CALLER (how CALL must be used)
                if (op == 0x33) {
                    // Sequence around CALLER must be:
                    // 1. PUSH1 0x00 (value)
                    // 2. CALLER (execution manager address) <-- We are here
                    // 3. GAS (gas for call)
                    // 4. CALL
                    if (
                        _pc < 2 ||
                        _bytecode[_pc - 2] != 0x60 || // value must be set with a PUSH1
                        _bytecode[_pc - 1] != 0 || // ensure PUSH1ed value is 0x00
                        _bytecode[++_pc] != 0x5a || // gas must be set with GAS
                        _bytecode[++_pc] != 0xf1 // last op must be CALL
                    ) {
                        return false;
                    }
                } else {
                    // STOP or JUMP or RETURN or REVERT or INVALID (see safety checker docs in wiki for more info)
                    // We are now inside unreachable code until we hit a JUMPDEST!
                    insideUnreachableCode = true;
                }
            }
        }
        return true;
    }
}
