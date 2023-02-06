import ArticleIcon from "@theme/icons/ArticleIcon.vue";
declare const _default: import("vue/types/vue").ExtendedVue<{
    $starArticles: import("@mr-hope/vuepress-types").PageComputed[];
} & Record<never, any> & ArticleIcon, {
    active: string;
}, {
    setActive(name: string): void;
    navigate(path: string): void;
}, {
    i18n: {
        article: string;
        articleList: string;
        category: string;
        tag: string;
        timeline: string;
        timelineText: string;
        allText: string;
        intro: string;
        star: string;
        slides: string;
        encrypt: string;
    };
    articleNumber: number;
}, Record<never, any>>;
export default _default;
