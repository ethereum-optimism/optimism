// SPDX-License-Identifier: MIT
pragma solidity >=0.5.16 <0.8.0;

// a library for performing overflow-safe math, courtesy of DappHub (https://github.com/dapphub/ds-math)

library UniSafeMath {
    function add(uint x, uint y) internal pure returns (uint z) {
        require((z = x + y) >= x, 'ds-math-add-overflow');
    }

    function sub(uint x, uint y) internal pure returns (uint z) {
        require((z = x - y) <= x, 'ds-math-sub-underflow');
    }

    function mul(uint x, uint y) internal pure returns (uint z) {
        require(y == 0 || (z = x * y) / y == x, 'ds-math-mul-overflow');
    }
    
    // full precision multiplication. no overflow
    function fullMul (uint x, uint y)
    internal pure returns (uint l, uint h)
    {
       uint mm = mulmod (x, y, uint (-1));
       l = x * y;
       h = mm - l;
       if (mm < l) h -= 1;
    }

    // return x*y/z
    function mulDiv (uint x, uint y, uint z) internal pure returns (uint) {
      (uint l, uint h) = fullMul (x, y);
       require (h < z);
       uint mm = mulmod (x, y, z);
       if (mm > l) h -= 1;
       l -= mm;
       uint pow2 = z & -z;
       z /= pow2;
       l /= pow2;
       l += h * ((-pow2) / pow2 + 1);
       uint r = 1;
       r *= 2 - z * r;
       r *= 2 - z * r;
       r *= 2 - z * r;
       r *= 2 - z * r;
       r *= 2 - z * r;
       r *= 2 - z * r;
       r *= 2 - z * r;
       r *= 2 - z * r;
       return l * r;
    }

    // x^n
    function pow (uint x, uint n)
    internal pure returns (uint r) {
       r = 1.0;
       while (n > 0) {
           if (n % 2 == 1) {
              r *= x;
              n -= 1;
           } else {
              x *= x;
              n /= 2;
           }
       }
    }
}
