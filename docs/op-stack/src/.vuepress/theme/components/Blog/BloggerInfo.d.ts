import MediaLinks from "@theme/components/MediaLinks.vue";
import type { BlogOptions } from "@theme/types";
declare const _default: import("vue/types/vue").ExtendedVue<{
    $timelineItems: import("@mr-hope/vuepress-types").PageComputed[];
    $timeline: import("@theme/mixins/timeline").TimelineItem[];
} & Record<never, any> & MediaLinks, unknown, {
    navigate(url: string): void;
    jumpIntro(): void;
}, {
    blogConfig: BlogOptions;
    bloggerName: string;
    bloggerAvatar: string;
    hasIntro: boolean;
    hintAttr: string;
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
