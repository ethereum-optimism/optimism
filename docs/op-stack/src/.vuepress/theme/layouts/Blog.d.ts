import BlogInfo from "@BlogInfo";
declare const _default: import("vue/types/vue").ExtendedVue<Record<never, any> & {
    globalEncryptPassword: string;
} & {
    checkGlobalPassword(globalPassword: string): void;
} & {
    isGlobalEncrypted: boolean;
} & {
    encryptOptions: import("../types").EncryptOptions;
} & BlogInfo, unknown, unknown, unknown, Record<never, any>>;
export default _default;
