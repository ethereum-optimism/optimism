import Anchor from "@theme/components/Anchor.vue";
import type { SidebarHeader } from "@theme/utils/groupHeader";
declare const _default: import("vue/types/vue").ExtendedVue<{
    $timelineItems: import("@mr-hope/vuepress-types").PageComputed[];
    $timeline: import("@theme/mixins/timeline").TimelineItem[];
} & Record<never, any> & Anchor, unknown, {
    navigate(url: string): void;
}, {
    hint: string;
    anchorConfig: SidebarHeader[];
}, Record<never, any>>;
export default _default;
