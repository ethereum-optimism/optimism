import type { EncryptOptions } from "../types";
export declare const globalEncryptMixin: import("vue/types/vue").ExtendedVue<{
    encryptOptions: EncryptOptions;
} & Record<never, any> & import("vue").default, {
    globalEncryptPassword: string;
}, {
    checkGlobalPassword(globalPassword: string): void;
}, {
    isGlobalEncrypted: boolean;
}, Record<never, any>>;
