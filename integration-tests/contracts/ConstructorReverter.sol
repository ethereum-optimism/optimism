// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Reverter } from './Reverter.sol';

contract ConstructorReverter is Reverter {
   constructor() {
       doRevert();
   }
}
