import Vue from "vue";
import type { GitContributor } from "@mr-hope/vuepress-plugin-git";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, {
    createEditLink(): string;
}, {
    i18n: {
        contributor: string;
        editLink: string;
        updateTime: string;
    };
    contributors: GitContributor[];
    contributorsText: string;
    updateTime: string;
    updateTimeText: string;
    editLink: string | false;
    editLinkText: string;
}, Record<never, any>>;
export default _default;
