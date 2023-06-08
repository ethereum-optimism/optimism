import type { EncryptOptions } from "../types";
export declare const pathEncryptMixin: import("vue/types/vue").ExtendedVue<{
    encryptOptions: EncryptOptions;
} & Record<never, any> & import("vue").default, {
    encryptPasswordConfig: Record<string, string>;
}, {
    checkPathPassword(password: string): void;
}, {
    pathEncryptMatchKeys: string[];
    isPathEncrypted: boolean;
}, Record<never, any>>;
