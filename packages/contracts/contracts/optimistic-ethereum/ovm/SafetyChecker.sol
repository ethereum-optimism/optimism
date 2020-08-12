pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { ExecutionManager } from "./ExecutionManager.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { console } from "@nomiclabs/buidler/console.sol";
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
        uint256 prevOp = 0;
        for (uint256 pc = 0; pc < _bytecode.length; pc++) {
            // current opcode: 0x00...0xff
            uint256 op = uint8(_bytecode[pc]);

            // PUSH##
            if (op >= 0x60 && op <= 0x7f) {
                // subsequent bytes are not opcodes. Skip them.
                pc += (op - 0x5f);
            }
            // If we're in between a STOP or REVERT or JUMP and a JUMPDEST
            if (insideUnreachableCode) {
                // JUMPDEST
                if (op == 0x5b) {
                    // this bytecode is now reachable via JUMP or JUMPI
                    insideUnreachableCode = false;
                }
            } else {
                // check that opcode is whitelisted (using the whitelist bit mask)
                uint256 opBit = 1 << op;
                if (opcodeWhitelistMask & opBit != opBit) {
                    // encountered a non-whitelisted opcode!
                    console.log('Encountered a non-whitelisted opcode (in decimal):', op);
                    return false;
                }
                // append this opcode to a list of ops
                // [STOP(0x00),JUMP(0x56),RETURN(0xf3),REVERT(0xfd),INVALID(0xfe),CALLER(0x33)] all have handlers
                if (opBit & 0x6008000000000000000000000000000000000000004000000008000000000001 != 0) {
                    // STOP or JUMP or RETURN or REVERT or INVALID (see safety checker docs in wiki for more info)
                    if (opBit & 0x6008000000000000000000000000000000000000004000000000000000000001 != 0) {
                        // We are now inside unreachable code until we hit a JUMPDEST!
                        insideUnreachableCode = true;
                    // CALL
                    } else if (op == 0x33) {
                        // Sequence around CALLER must be:
                        // 1. PUSH1 0x00 (value)
                        // 2. CALLER (execution manager address) <-- We are here
                        // 3. GAS (gas for call)
                        // 4. CALL
                        if (
                            prevOp != 0x60 || // value must be set with a PUSH1
                            _bytecode[pc - 1] != 0 || // ensure PUSH1ed value is 0x00
                            _bytecode[++pc] != 0x5a || // gas must be set with GAS
                            _bytecode[++pc] != 0xf1 // last op must be CALL
                        ) {
                            return false;
                        }
                    }
                }
                prevOp = op;
            }
        }
        return true;
    }


    /*
     * Internal Functions
     */

    /**
     * Converts the 20 bytes at _start of _bytes into an address.
     * @param _bytes The bytes to extract the address from.
     * @param _start The start index from which to extract the address from
     *               (e.g. 0 if _bytes starts with the address).
     * @return Bytes converted to an address.
     */
    function toAddress(
        bytes memory _bytes,
        uint256 _start
    )
        internal
        pure
        returns (address addr)
    {
        require(_bytes.length >= (_start + 20), "Addresses must be at least 20 bytes");

        assembly {
            addr := mload(add(add(_bytes, 20), _start))
        }
    }


    /*
     * Contract Resolution
     */

    function resolveExecutionManager()
        internal
        view
        returns (ExecutionManager)
    {
        return ExecutionManager(resolveContract("ExecutionManager"));
    }
}
