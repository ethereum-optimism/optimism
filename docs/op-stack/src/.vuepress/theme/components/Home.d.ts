import Vue from "vue";
interface ActionConfig {
    text: string;
    link: string;
}
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, {
    navigate(link: string): void;
}, {
    actionLinks: ActionConfig[];
}, Record<never, any>>;
export default _default;
