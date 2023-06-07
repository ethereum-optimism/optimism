import Vue from "vue";
import type { PageComputed } from "@mr-hope/vuepress-types";
import type { SidebarItem } from "@theme/utils/sidebar";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, {
    openGroupIndex: number;
}, {
    refreshIndex(): void;
    toggleGroup(index: number): void;
    isActive(page: PageComputed): boolean;
}, unknown, {
    items: SidebarItem[];
    depth: number;
}>;
export default _default;
