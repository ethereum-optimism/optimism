import Vue from "vue";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, {
    password: string;
    hasTried: boolean;
}, {
    verify(): void;
}, {
    isMainPage: boolean;
    encrypt: {
        title: string;
        errorHint: string;
    };
}, {
    page: boolean;
}>;
export default _default;
