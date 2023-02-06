import Vue from "vue";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, {
    darkmode: "auto" | "off" | "on";
}, {
    setDarkmode(status: "on" | "off" | "auto"): void;
    toggleDarkmode(isDarkmode: boolean): void;
}, {
    darkmodeConfig: "auto" | "auto-switch" | "switch" | "disable";
}, Record<never, any>>;
export default _default;
