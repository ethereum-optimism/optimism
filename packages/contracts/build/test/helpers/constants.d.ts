export declare const DEFAULT_ACCOUNTS: {
    balance: string;
    secretKey: string;
}[];
export declare const DEFAULT_ACCOUNTS_BUIDLER: {
    balance: string;
    privateKey: string;
}[];
export declare const OVM_TX_GAS_LIMIT = 10000000;
export declare const RUN_OVM_TEST_GAS = 20000000;
export declare const FORCE_INCLUSION_PERIOD_SECONDS = 600;
export declare const NULL_BYTES32: string;
export declare const NON_NULL_BYTES32: string;
export declare const ZERO_ADDRESS: string;
export declare const NON_ZERO_ADDRESS: string;
export declare const VERIFIED_EMPTY_CONTRACT_HASH = "0x00004B1DC0DE000000004B1DC0DE000000004B1DC0DE000000004B1DC0DE0000";
export declare const NUISANCE_GAS_COSTS: {
    NUISANCE_GAS_SLOAD: number;
    NUISANCE_GAS_SSTORE: number;
    MIN_NUISANCE_GAS_PER_CONTRACT: number;
    NUISANCE_GAS_PER_CONTRACT_BYTE: number;
    MIN_GAS_FOR_INVALID_STATE_ACCESS: number;
};
export declare const Helper_TestRunner_BYTELEN = 3654;
export declare const STORAGE_XOR = "0xfeedfacecafebeeffeedfacecafebeeffeedfacecafebeeffeedfacecafebeef";
export declare const getStorageXOR: (key: string) => string;
