import Vue from "vue";
import type { BlogOptions } from "@theme/types";
import type { SidebarItem } from "@theme/utils/sidebar";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, unknown, {
    blogConfig: BlogOptions;
    sidebarDisplay: "always" | "mobile" | "none";
}, {
    items: SidebarItem[];
}>;
export default _default;
