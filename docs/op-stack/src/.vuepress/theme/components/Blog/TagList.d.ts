import Vue from "vue";
interface TagOption {
    name: string;
    path: string;
}
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, {
    isActive(name: string): boolean;
    clickTag(path: string): void;
}, {
    tagList: TagOption[];
}, Record<never, any>>;
export default _default;
