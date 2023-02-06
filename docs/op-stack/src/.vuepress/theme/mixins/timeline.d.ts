import Vue from "vue";
import type { PageComputed } from "@mr-hope/vuepress-types";
export interface TimelineItem {
    year: number;
    articles: PageComputed[];
}
export declare const timelineMixin: import("vue/types/vue").ExtendedVue<Vue, unknown, unknown, {
    $timelineItems: PageComputed[];
    $timeline: TimelineItem[];
}, Record<never, any>>;
