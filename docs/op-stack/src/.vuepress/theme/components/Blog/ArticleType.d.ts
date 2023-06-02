import Vue from "vue";
interface ArticleTypeItem {
    text: string;
    path: string;
}
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, {
    navigate(path: string): void;
}, {
    types: ArticleTypeItem[];
}, Record<never, any>>;
export default _default;
