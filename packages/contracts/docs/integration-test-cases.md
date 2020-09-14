# Execution Managager Integration/State Tests

## General notes
- run everything below with invalid state accesses automatically and assert invalid state access handled in ALL cases
- run everything below through a state manager proxy which consumes a different amount of gas and check that the **OVM** gas values are not different


## Test Cases
 - CALL-types
     - for all: call an undeployed contract and make sure it errors or whatevs (or maybe that's just a unit test)
     - ovmCALL 
         - -> ovmCALLER
         - -> ovmADDRESS
         - -> SLOAD
         - -> SSTORE
         - -> CREATE/2
     - ovmSTATICCALL 
         - -> ovmCALLER
         - -> ovmADDRESS
         - -> SLOAD
         - -> SSTORE (fail)
         - -> CREATE/2 (fail)
         - -> ovmCALL -> ovmSSTORE
         - -> ovmCALL -> ovmCREATE
         - -> ovmSTATICCALL -> RETURN -> SLOAD (fails)
     - ovmDELEGATECALL 
         - -> ovmCALLER
         - -> ovmADDRESS
         - -> SLOAD
         - -> SSTORE
         - -> CREATE/2
         - -> ovmDELEGATECALL -> ovmCALLER
         - -> ovmDELEGATECALL -> ovmADDRESS
 - Code-related

 - CREATE-types
     - do we just duplicate these exactly for CREATE and CREATE2?  Probably
     - ovmCREATE -> success -> ovmEXTCODE{SIZE,HASH,COPY}
     - ovmCREATE -> fail (ovmREVERT, NOT out of gas/invalid jump) -> ovmEXTCODE{SIZE,HASH,COPY}
     - ovmCREATE -> fail -> ovmCALL what was attempted to be created (fail)
     - ovmCREATE -> ovmCREATE (during constructor) -> success -> success (check right address for inner deployment)
     - ovmCREATE -> ovmCALL(in constructor) -> ovmSSTORE, return -> ovmREVERT (deployment fails, storage not modified, but state access gas correctly increased)
     - ovmCREATE -> ovmCREATE (during constructor) -> success -> fail (outer contract)
     - "creator" does ovmCREATE -> invalid jumpdest -> creator out-of-gasses (or at least, appears to--really it will revert with no data, so there will be some gas left)
     - "creator" does ovmCREATE -> initcode does ovmCREATE -> invalid jumpdest -> creator out-of-gasses (or at least, appears to--really it will revert with no data, so there will be some gas left) AKA same as above but nested CREATEs

- OVM gas metering
    - do everything for both queue origins/flip flopped roles:
    - blocks transactions whose gas limit would put the cumulative gas over the max for the current epoch
    - starts a new epoch and allows tx through if the above would have held true, but new epoch has begun
    - allows transaction through queue B even if queue A cumulative gas would have blocked it

- out of gas
    - ovmCALL -> [ovmCALL(gas()/2) -> out of gas] -> SSTORE (does not out of gas parent)


- State access limiting logic
    - ovmCALL(gas()/2) -> ovmCALL(gas()) -> out of gas -> return(someVal) -> SSTORE(someVal)
        - this one shows that if a subcall out-of-gasses, but you do not do any further state accesses, then you can still return, and if your parent has a bigger allocation left, they can still ovmSSTORE
    - ovmSSTORE, repeated max times, ovmCALL(gas()) -> ovmSSTORE -> fails (even though max gas allotment was given, the parent already used them up)
    - ovmCALL(gas/2) -> ovmCREATE, out of gas -> SSTORE succeeds
    - ovmCALL(gas) -> ovmCREATE, out of gas -> SSTORE fails
    - ovmCALL(gas) -> ovmCREATE, ovmREVERT (in init) -> SSTORE succeeds
    - ovmCALL(gas) -> ovmCREATE, ovmSSTORE(max times), ovmREVERT -> ovmSSTORE fails (max allocated in reverting CREATE)
    - ovmCALL(gas) -> ovmCREATE -> ovmCREATE ovmSSTORE(max times), ovmREVERT -> deploy -> ovmSSTORE fails (propogates through a failed CREATE inside a successful CREATE
    - ovmCALL(gas) -> ovmCREATE -> ovmCREATE, ovmSLOAD(max times), inner deploy success -> outer deploy fail -> ovmSSTORE fails (propogates through a successful create inside a failed create)
    - ovmCREATE -> ovmCALL, ovmSSTORE (max), return -> ovmSSTORE fails
    - ovmCREATE -> ovmCALL(gas/2) -> ovmCREATE, out of gas, call reverts (as if out of gas) -> ovmSSTORE (success in constructor)

- Explicit invalid state access tests
    - CALL -> CALL, ISA
    - CALL -> CALL, CALL, ISA
    - CREATE -> CREATE, ISA
    - CREATE -> CREATE -> CREATE ISA
    - CREATE -> CALL, ISA
    - CALL -> CREATE, ISA
    - CALL -> CREATE -> CALL, ISA
    - CREATE -> CALL -> CREATE, ISA