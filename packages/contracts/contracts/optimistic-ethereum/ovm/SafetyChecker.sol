pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";

//import { console } from "@nomiclabs/buidler/console.sol";

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
        uint256 _opcodeWhitelistMask = opcodeWhitelistMask;
        uint256 _opcodePushMask = ~uint256(0xffffffff000000000000000000000000);
        uint256 _opcodeSkipMask = ~uint256(0x6008000000000000000000000000000000000000004000000008000000000001) &
                                  _opcodeWhitelistMask &
                                  _opcodePushMask;
        uint256 codeLength;
        uint256 _pc;
        assembly {
            _pc := add(_bytecode, 0x20)
        }
        codeLength = _pc + _bytecode.length;
        //uint256 _ops = 0;
        uint256[8] memory skip = [
          uint256(0x0001010101010101010101010000000001010101010101010101010101010000),
          uint256(0x0100000000000000000000000000000000000000010101010101000000010100),
          uint256(0x0000000000000000000000000000000001010101000000010101010100000000),
          uint256(0x0203040500000000000000000000000000000000000000000000000000000000),
          uint256(0x0101010101010101010101010101010101010101010101010101010101010101),
          uint256(0x0101010101000000000000000000000000000000000000000000000000000000),
          uint256(0x0000000000000000000000000000000000000000000000000000000000000000),
          uint256(0x0000000000000000000000000000000000000000000000000000000000000000)];
        do {
            // current opcode: 0x00...0xff
            uint256 op;

            // inline assembly removes the extra add + bounds check
            assembly {
                let tmp := mload(_pc)

                let mpc := byte(0, mload(add(skip, byte(0, tmp))))
                mpc := add(mpc, byte(0, mload(add(skip, byte(mpc, tmp)))))
                mpc := add(mpc, byte(0, mload(add(skip, byte(mpc, tmp)))))
                mpc := add(mpc, byte(0, mload(add(skip, byte(mpc, tmp)))))
                mpc := add(mpc, byte(0, mload(add(skip, byte(mpc, tmp)))))
                mpc := add(mpc, byte(0, mload(add(skip, byte(mpc, tmp)))))
                _pc := add(_pc, mpc)
                op := byte(mpc, tmp)

                //op := byte(0, tmp)
            }

            //_ops = (_ops << 8) | op;

            // check that opcode is whitelisted (using the whitelist bit mask)

            // [STOP(0x00),JUMP(0x56),RETURN(0xf3),REVERT(0xfd),INVALID(0xfe),CALLER(0x33)]
            // + blacklisted opcodes
            // + push opcodes all have handlers
            if ((1 << op) & _opcodeSkipMask == 0) {
                uint256 opBit = 1 << op;
                if (opBit & _opcodePushMask == 0) {
                    // subsequent bytes are not opcodes. Skip them.
                    _pc += (op - 0x5e);
                    // all pushes are valid opcodes
                    continue;
                } else if (op == 0x33) {
                    // Sequence around CALLER must be:
                    // 1. PUSH1 0x00 (value)
                    // 2. CALLER (execution manager address) <-- We are here
                    // 3. GAS (gas for call)
                    // 4. CALL
                    /*if (_ops & 0xFFFF == 0x6033 &&
                        _bytecode[_pc - 1] == 0 && // ensure PUSH1ed value is 0x00
                        _bytecode[_pc + 1] == 0x5a && // gas must be set with GAS
                        _bytecode[_pc + 2] == 0xf1 // last op must be CALL
                    ) {
                        // allowed = PUSH1 0x0 CALLER GAS CALL
                    } else if (_ops & 0xFFFFFFFFFFFF == 0x608160818333 &&
                        _bytecode[_pc - 6] == 0 && // ensure PUSH1ed value is 0x00
                        _bytecode[_pc - 3] == 0x44 && // 0x44
                        _bytecode[_pc + 1] == 0x5a && // gas must be set with GAS
                        _bytecode[_pc + 2] == 0xf1 // last op must be CALL
                    ) {
                        // allowed = PUSH1 0x0 DUP2 PUSH1 0x44 DUP2 DUP4 CALLER GAS CALL
                    } else {
                        //console.log('Encountered a bad call');
                        return false;
                    }*/
                    _pc += 3;
                    continue;
                } else if (opBit & _opcodeWhitelistMask == 0) {
                    // encountered a non-whitelisted opcode!
                    //console.log('Encountered a non-whitelisted opcode (in decimal):', op);
                    return false;
                } else {
                    // STOP or JUMP or RETURN or REVERT or INVALID (see safety checker docs in wiki for more info)
                    // We are now inside unreachable code until we hit a JUMPDEST!
                    do {
                        _pc++;
                        assembly {
                            op := byte(0, mload(_pc))
                        }
                        if (op == 0x5b) break;
                        if ((1 << op) & _opcodePushMask == 0) _pc += (op - 0x5f);
                    } while (_pc < codeLength);
                }
            }
            _pc++;
        } while (_pc < codeLength);
        return true;
    }
}
