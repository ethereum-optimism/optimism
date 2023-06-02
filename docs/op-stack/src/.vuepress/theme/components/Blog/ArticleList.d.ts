import Vue from "vue";
import type { BlogOptions } from "@theme/types";
import type { PageComputed } from "@mr-hope/vuepress-types";
declare const _default: import("vue/types/vue").ExtendedVue<Vue, {
    currentPage: number;
    articleList: PageComputed[];
}, {
    getArticleList(): PageComputed[];
}, {
    blogConfig: BlogOptions;
    articlePerPage: number;
    filter: ((page: PageComputed) => boolean) | undefined;
    $articles: PageComputed[];
    articles: PageComputed[];
}, Record<never, any>>;
export default _default;
