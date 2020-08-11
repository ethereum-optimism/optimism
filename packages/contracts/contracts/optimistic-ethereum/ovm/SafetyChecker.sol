pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { ExecutionManager } from "./ExecutionManager.sol";

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
        uint256 ops = 0;
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
                    return false;
                }
                // append this opcode to a list of ops
                ops <<= 8;
                ops |= op;
                // [0x56, 0x00, 0xfd, 0xfe, 0xf3, 0xf1] all have handlers
                if (opBit & 0x600a00000000000000000000000000000000000000c000000000000000000001 != 0) {
                    // STOP or JUMP or REVERT or INVALID or RETURN (see safety checker docs in wiki for more info)
                    if (op == 0x00 || op == 0x56 || op == 0xfd || op == 0xfe || op == 0xf3) {
                        // We are now inside unreachable code until we hit a JUMPDEST!
                        insideUnreachableCode = true;
                    // CALL
                    } else if (op == 0xf1) {
                        // Minimum 4 total ops:
                        // 1. PUSH1 Value (must be 0x00)
                        // 2. CALLER (execution manager address)
                        // 3. GAS
                        // 4. CALL

                        // if opIndex < 3, there would be 0s here, so we don't need the check
                        uint256 gasOp = (ops >> 8) & 0xFF;
                        uint256 addressOp = (ops >> 16) & 0xFF;
                        uint256 valueOp = (ops >> 24) & 0xFF;
                        if (
                            gasOp < 0x60 || // PUSHes are 0x60...0x7f
                            gasOp > 0x8f || // DUPs are 0x80...0x8f
                            addressOp != 0x73 || // address must be set with a PUSH20
                            valueOp != 0x60 // value must be set with a PUSH1
                        ) {
                            return false;
                        } else {
                            uint256 pushedBytes;
                            // gas is set with a PUSH##
                            if (gasOp >= 0x60 && gasOp <= 0x7f) {
                                pushedBytes = gasOp - 0x5f;
                            }

                            // 23 is from 1 + PUSH20 + 20 bytes of address + PUSH or DUP gas
                            byte callValue = _bytecode[pc - (23 + pushedBytes)];

                            // 21 is from 1 + 19 bytes of address + PUSH or DUP gas
                            address callAddress = toAddress(_bytecode, (pc - (21 + pushedBytes)));

                            // CALL is made to the execution manager with msg.value of 0 ETH
                            if (callAddress != address(resolveExecutionManager()) || callValue != 0 ) {
                                return false;
                            }
                        }
                    }
                }
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
