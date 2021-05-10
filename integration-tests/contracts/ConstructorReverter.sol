// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

import { Reverter } from './Reverter.sol';

contract ConstructorReverter is Reverter {
   constructor() {
       doRevert();
   }
}
