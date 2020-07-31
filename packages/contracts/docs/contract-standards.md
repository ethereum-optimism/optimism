# Contract Standards

## Style Guide
### Maximum Line Length
Maximum line length is set globally to 100 characters. For a standard reading experience, however, the following exceptions are in place:

1. Structs, enums, and events

Structs, enums, and events should be defined over several lines even if their definitions would be under 100 characters.

Ex.:
```
event SomeEvent(
    uint256 _someEventParameter
);
```

2. Function signatures

Similarly, function signatures should always be defined over several lines. This allows us to easily read the various sections of the function no matter the actual length of the signature.

Ex.:
```
function otherExternalFunction(
    uint256 _someFunctionParameter
)   
    external
    someModifier
    returns (uint256)
{
    // Some other action.
}
```

## Standard Contract Layout

```solidity
pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { WWWW } from "./path/to/contract.sol";
import { XXXX } from "./path/to/other/contract.sol";

/* Library Imports */
import { YYYY } from "./path/to/library.sol";
import { ZZZZ } from "./path/to/other/library.sol";

/**
 * @title ContractName
 * @dev Some useful description.
 */
contract ContractName {
    /*
     * Data Structures
     */
    
    struct SomeStruct {
        uint256 value;
    }

    enum SomeEnum {
        SomeEnumValue,
        OtherEnumValue
    }


    /*
     * Events
     */
    
    event SomeEvent(
        uint256 _someEventParameter,
        address _otherEventParameter
    );


    /*
     * Contract Constants
     */

    uint256 constant public SOME_SNAKE_CASE_CONSTANT;
    bytes32 constant private OTHER_SNAKE_CASE_CONSTANT;


    /*
     * Contract Variables
     */

    uint256 public someCamelCaseVariable;
    uint8 private otherCamelCaseVariable; 


    /*
     * Modifiers
     */
    
    modifier someModifier(
        uint256 _someModifierParameter
    ) {
        require(
            someRequirement,
            "Did not pass the requirement"
        );
        _;
    }


    /*
     * Constructor
     */
    
    /**
     * @param _someConstructorParameter Some constructor parameter
     */
    constructor(
        uint256 _someConstructorParameter
    )
        public
    {
        // Some constructor action.
    }


    /*
     * External Functions
     */
    
    /**
     * A function that does something.
     * @param _someFunctionParameter A function parameter.
     */
    function someExternalFunction(
        uint256 _someFunctionParameter
    )
        external
    {
        // Some action.
    }

    /**
     * A function that does another thing.
     * @param _someFunctionParameter A function parameter.
     * @return Some value.
     */
    function otherExternalFunction(
        uint256 _someFunctionParameter
    )   
        external
        someModifier
        returns (uint256)
    {
        // Some other action.
    }


    /*
     * Public Functions
     */
    
    /**
     * A public function that does something.
     * @param _someFunctionParameter A function parameter.
     */
    function somePublicFunction(
        uint256 _someFunctionParameter
    )
        public
    {
        // Some action.
    }


    /*
     * Internal Functions
     */
    
    /**
     * An internal function that does something.
     * @param _someFunctionParameter A function parameter.
     */
    function someInternalFunction(
        uint256 _someFunctionParameter
    )
        internal
    {
        // Some action.
    }


    /*
     * Private Functions
     */
    
    /**
     * A private function that does something.
     * @param _someFunctionParameter A function parameter.
     */
    function somePrivateFunction(
        uint256 _someFunctionParameter
    )
        private
    {
        // Some action.
    }
}
```